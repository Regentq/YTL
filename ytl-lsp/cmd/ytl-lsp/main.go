// ytl-lsp is the language server for YTL. It talks to VS Code over stdin/stdout.
//
// The LSP protocol is JSON messages with a Content-Length header:
//
//	Content-Length: 52\r\n
//	\r\n
//	{"jsonrpc":"2.0","method":"initialized","params":{}}
//
// IMPORTANT: never print anything extra to stdout (fmt.Println etc.) - it is the
// protocol channel. Use log.Println for logs (it goes to stderr).
package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"

	"ytl-lsp/analyze"
)

const (
	lsName    = "ytl-lsp"
	lsVersion = "0.1.0"
)

func main() {
	log.SetFlags(0)
	log.SetPrefix("[ytl-lsp] ")

	s := newServer(bufio.NewReader(os.Stdin), os.Stdout)
	os.Exit(s.run())
}

// ---------- Message types ----------

// message is any incoming message: a request, a notification or a response.
type message struct {
	ID     json.RawMessage `json:"id,omitempty"`     // present on requests (they need an answer)
	Method string          `json:"method,omitempty"` // for example "textDocument/didOpen"
	Params json.RawMessage `json:"params,omitempty"`
}

type responseOK struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result"` // no omitempty: null is a valid result too
}

type responseErr struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Error   rpcError        `json:"error"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type notification struct {
	JSONRPC string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  any    `json:"params"`
}

// Parameters of the document notifications (only the fields we need).
type didOpenParams struct {
	TextDocument struct {
		URI  string `json:"uri"`
		Text string `json:"text"`
	} `json:"textDocument"`
}

type didChangeParams struct {
	TextDocument struct {
		URI string `json:"uri"`
	} `json:"textDocument"`
	ContentChanges []struct {
		Text string `json:"text"`
	} `json:"contentChanges"`
}

type didCloseParams struct {
	TextDocument struct {
		URI string `json:"uri"`
	} `json:"textDocument"`
}

// Diagnostics (the red squiggles).
type lspPosition struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

type lspRange struct {
	Start lspPosition `json:"start"`
	End   lspPosition `json:"end"`
}

type lspDiagnostic struct {
	Range    lspRange `json:"range"`
	Severity int      `json:"severity"` // 1 = Error
	Source   string   `json:"source"`
	Message  string   `json:"message"`
}

type publishParams struct {
	URI         string          `json:"uri"`
	Diagnostics []lspDiagnostic `json:"diagnostics"`
}

// ---------- Server ----------

type server struct {
	in       *bufio.Reader
	out      io.Writer
	shutdown bool              // whether "shutdown" was received before "exit"
	docs     map[string]string // text of every open document, by URI
}

func newServer(in *bufio.Reader, out io.Writer) *server {
	return &server{in: in, out: out, docs: map[string]string{}}
}

// run reads messages until the client sends "exit". Returns the process exit code.
func (s *server) run() int {
	for {
		msg, err := s.readMessage()
		if err != nil {
			if err != io.EOF {
				log.Println("read error:", err)
			}
			if s.shutdown {
				return 0
			}
			return 1 // the client vanished without saying goodbye
		}
		if stop, code := s.handle(msg); stop {
			return code
		}
	}
}

// readMessage reads one message: headers, an empty line, then the body.
func (s *server) readMessage() (*message, error) {
	contentLength := -1

	for {
		line, err := s.in.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break // an empty line means the headers are over
		}
		if name, value, ok := strings.Cut(line, ":"); ok && strings.EqualFold(name, "Content-Length") {
			n, err := strconv.Atoi(strings.TrimSpace(value))
			if err != nil {
				return nil, err
			}
			contentLength = n
		}
	}

	if contentLength < 0 {
		return nil, errors.New("message has no Content-Length header")
	}

	body := make([]byte, contentLength)
	if _, err := io.ReadFull(s.in, body); err != nil {
		return nil, err
	}

	var msg message
	if err := json.Unmarshal(body, &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

// handle processes one message. stop=true means "shut down now".
func (s *server) handle(msg *message) (stop bool, code int) {
	if msg.Method == "" {
		return false, 0 // a client response to our request; we never send requests
	}

	switch msg.Method {
	case "initialize":
		s.reply(msg.ID, map[string]any{
			"capabilities": map[string]any{
				"textDocumentSync": map[string]any{
					"openClose": true, // send didOpen / didClose
					"change":    1,    // 1 = Full: send the whole text on every change
				},
				// Autocomplete also pops up after a space ("Dae |" -> from).
				"completionProvider": map[string]any{"triggerCharacters": []string{" "}},
				// After Enter below "Dog from Animal:" the parent's keys are inserted.
				"documentOnTypeFormattingProvider": map[string]any{"firstTriggerCharacter": "\n"},
			},
			"serverInfo": map[string]any{"name": lsName, "version": lsVersion},
		})

	case "initialized":
		// nothing to do

	case "shutdown":
		s.shutdown = true
		s.reply(msg.ID, nil)

	case "exit":
		if s.shutdown {
			return true, 0
		}
		return true, 1

	case "textDocument/didOpen":
		var p didOpenParams
		if err := json.Unmarshal(msg.Params, &p); err == nil {
			s.docs[p.TextDocument.URI] = p.TextDocument.Text
			s.publish(p.TextDocument.URI, p.TextDocument.Text)
		}

	case "textDocument/didChange":
		var p didChangeParams
		if err := json.Unmarshal(msg.Params, &p); err == nil && len(p.ContentChanges) > 0 {
			// Full sync: the last change carries the whole text.
			last := p.ContentChanges[len(p.ContentChanges)-1]
			s.docs[p.TextDocument.URI] = last.Text
			s.publish(p.TextDocument.URI, last.Text)
		}

	case "textDocument/didClose":
		var p didCloseParams
		if err := json.Unmarshal(msg.Params, &p); err == nil {
			delete(s.docs, p.TextDocument.URI)
			// The file was closed - clear its errors from the Problems panel.
			s.send(notification{
				JSONRPC: "2.0",
				Method:  "textDocument/publishDiagnostics",
				Params:  publishParams{URI: p.TextDocument.URI, Diagnostics: []lspDiagnostic{}},
			})
		}

	case "textDocument/completion":
		items := []lspCompletionItem{} // an empty list, not null
		var p completionParams
		if err := json.Unmarshal(msg.Params, &p); err == nil {
			text := s.docs[p.TextDocument.URI]
			for _, c := range analyze.Complete(text, p.Position.Line, p.Position.Character) {
				items = append(items, toLSPItem(c))
			}
		}
		s.reply(msg.ID, items)

	case "textDocument/onTypeFormatting":
		edits := []lspTextEdit{} // an empty list, not null
		var p onTypeParams
		if err := json.Unmarshal(msg.Params, &p); err == nil && p.Ch == "\n" {
			unit := "\t"
			if p.Options.InsertSpaces {
				size := p.Options.TabSize
				if size <= 0 {
					size = 4
				}
				unit = strings.Repeat(" ", size)
			}
			// After Enter the cursor is on the NEW line.
			text := s.docs[p.TextDocument.URI]
			if e, ok := analyze.InheritedStub(text, p.Position.Line, unit); ok {
				edits = append(edits, lspTextEdit{
					Range: lspRange{
						Start: lspPosition{Line: e.Line, Character: e.StartCol},
						End:   lspPosition{Line: e.Line, Character: e.EndCol},
					},
					NewText: e.NewText,
				})
			}
		}
		s.reply(msg.ID, edits)

	default:
		// An unknown request (it has an id) must be rejected, otherwise the client
		// waits for an answer forever. Unknown notifications are simply ignored.
		if len(msg.ID) > 0 {
			s.send(responseErr{
				JSONRPC: "2.0",
				ID:      msg.ID,
				Error:   rpcError{Code: -32601, Message: "method not supported: " + msg.Method},
			})
		}
	}
	return false, 0
}

// publish analyzes the text and sends the list of errors to the editor.
func (s *server) publish(uri, text string) {
	// An empty slice, not nil: otherwise JSON gets null and the old errors
	// would not disappear after the user fixes them.
	diags := []lspDiagnostic{}

	for _, d := range analyze.Analyze(text) {
		diags = append(diags, lspDiagnostic{
			Range: lspRange{
				Start: lspPosition{Line: d.Range.Start.Line, Character: d.Range.Start.Col},
				End:   lspPosition{Line: d.Range.End.Line, Character: d.Range.End.Col},
			},
			Severity: 1,
			Source:   lsName,
			Message:  d.Message,
		})
	}

	s.send(notification{
		JSONRPC: "2.0",
		Method:  "textDocument/publishDiagnostics",
		Params:  publishParams{URI: uri, Diagnostics: diags},
	})
}

func (s *server) reply(id json.RawMessage, result any) {
	s.send(responseOK{JSONRPC: "2.0", ID: id, Result: result})
}

// send wraps a message into JSON, adds the header and writes it to stdout.
func (s *server) send(v any) {
	body, err := json.Marshal(v)
	if err != nil {
		log.Println("could not marshal JSON:", err)
		return
	}
	// One write: header + empty line + body.
	packet := fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(body), body)
	if _, err := io.WriteString(s.out, packet); err != nil {
		log.Println("could not write response:", err)
	}
}

// ---------- Autocomplete ----------

type completionParams struct {
	TextDocument struct {
		URI string `json:"uri"`
	} `json:"textDocument"`
	Position lspPosition `json:"position"`
}

type lspCompletionItem struct {
	Label  string `json:"label"`
	Kind   int    `json:"kind"` // 14 = Keyword, 7 = Class
	Detail string `json:"detail,omitempty"`
}

func toLSPItem(c analyze.Completion) lspCompletionItem {
	kind := 14 // Keyword
	if c.Kind == analyze.KindSection {
		kind = 7 // Class (a section looks like a class in the editor)
	}
	return lspCompletionItem{Label: c.Label, Kind: kind, Detail: c.Detail}
}

// ---------- Typing help ----------

type onTypeParams struct {
	TextDocument struct {
		URI string `json:"uri"`
	} `json:"textDocument"`
	Position lspPosition `json:"position"`
	Ch       string      `json:"ch"`
	Options  struct {
		TabSize      int  `json:"tabSize"`
		InsertSpaces bool `json:"insertSpaces"`
	} `json:"options"`
}

type lspTextEdit struct {
	Range   lspRange `json:"range"`
	NewText string   `json:"newText"`
}
