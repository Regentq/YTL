package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

// frame wraps a message the same way VS Code does.
func frame(v any) string {
	body, _ := json.Marshal(v)
	return fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(body), body)
}

func TestSession(t *testing.T) {
	in := frame(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": map[string]any{}}) +
		frame(map[string]any{"jsonrpc": "2.0", "method": "initialized", "params": map[string]any{}}) +
		// open a file with an error
		frame(map[string]any{"jsonrpc": "2.0", "method": "textDocument/didOpen", "params": map[string]any{
			"textDocument": map[string]any{"uri": "file:///t.ytl", "languageId": "ytl", "version": 1, "text": "ip 999.1.1.1\n"},
		}}) +
		// fix the error
		frame(map[string]any{"jsonrpc": "2.0", "method": "textDocument/didChange", "params": map[string]any{
			"textDocument":   map[string]any{"uri": "file:///t.ytl", "version": 2},
			"contentChanges": []any{map[string]any{"text": "ip 127.0.0.1\n"}},
		}}) +
		// an unknown request must get an error response
		frame(map[string]any{"jsonrpc": "2.0", "id": 5, "method": "textDocument/hover", "params": map[string]any{}}) +
		frame(map[string]any{"jsonrpc": "2.0", "id": 2, "method": "shutdown"}) +
		frame(map[string]any{"jsonrpc": "2.0", "method": "exit"})

	var out bytes.Buffer
	s := newServer(bufio.NewReader(strings.NewReader(in)), &out)

	if code := s.run(); code != 0 {
		t.Fatalf("exit code %d, expected 0", code)
	}

	got := out.String()

	if !strings.Contains(got, `"textDocumentSync"`) {
		t.Errorf("the initialize response has no textDocumentSync:\n%s", got)
	}
	if n := strings.Count(got, "textDocument/publishDiagnostics"); n != 2 {
		t.Errorf("expected 2 diagnostics publications, got %d", n)
	}
	if !strings.Contains(got, "out of range (0-255)") {
		t.Errorf("the IPv4 error was not found:\n%s", got)
	}
	if !strings.Contains(got, `"diagnostics":[]`) {
		t.Errorf("after the fix an empty error list must be sent:\n%s", got)
	}
	if !strings.Contains(got, `"code":-32601`) {
		t.Errorf("an unknown request must be answered with an error:\n%s", got)
	}
	if !strings.Contains(got, `"result":null`) {
		t.Errorf("shutdown must be answered with result:null:\n%s", got)
	}
}

// completionSession opens a document and asks for completion at (line, char).
func completionSession(text string, line, char int) string {
	in := frame(map[string]any{"jsonrpc": "2.0", "method": "textDocument/didOpen", "params": map[string]any{
		"textDocument": map[string]any{"uri": "file:///t.ytl", "languageId": "ytl", "version": 1, "text": text},
	}}) + frame(map[string]any{"jsonrpc": "2.0", "id": 7, "method": "textDocument/completion", "params": map[string]any{
		"textDocument": map[string]any{"uri": "file:///t.ytl"},
		"position":     map[string]any{"line": line, "character": char},
	}})

	var out bytes.Buffer
	s := newServer(bufio.NewReader(strings.NewReader(in)), &out)
	s.run()
	return out.String()
}

func TestCompletionFrom(t *testing.T) {
	got := completionSession("Animal:\n    x \"1\"\nDog fr\n", 2, 6)
	if !strings.Contains(got, `"label":"from"`) {
		t.Errorf("expected the suggestion \"from\":\n%s", got)
	}
}

func TestCompletionParents(t *testing.T) {
	got := completionSession("Animal:\n    x \"1\"\nDog from \n", 2, 9)
	if !strings.Contains(got, `"label":"Animal"`) {
		t.Errorf("expected the suggestion \"Animal\":\n%s", got)
	}
}

func TestCompletionEmptyListIsNotNull(t *testing.T) {
	got := completionSession("Animal:\n    x \"1\"\n", 1, 3)
	if !strings.Contains(got, `"result":[]`) {
		t.Errorf("an empty completion result must be [] and not null:\n%s", got)
	}
}

func TestOnTypeFormattingSpawnsParentKeys(t *testing.T) {
	text := "Animal:\n    name \"x\"\nDog from Animal:\n"
	in := frame(map[string]any{"jsonrpc": "2.0", "method": "textDocument/didOpen", "params": map[string]any{
		"textDocument": map[string]any{"uri": "file:///t.ytl", "languageId": "ytl", "version": 1, "text": text},
	}}) + frame(map[string]any{"jsonrpc": "2.0", "id": 9, "method": "textDocument/onTypeFormatting", "params": map[string]any{
		"textDocument": map[string]any{"uri": "file:///t.ytl"},
		"position":     map[string]any{"line": 3, "character": 0},
		"ch":           "\n",
		"options":      map[string]any{"tabSize": 4, "insertSpaces": true},
	}})

	var out bytes.Buffer
	s := newServer(bufio.NewReader(strings.NewReader(in)), &out)
	s.run()

	if !strings.Contains(out.String(), `"newText":"    name"`) {
		t.Errorf("expected the key \"name\" to be spawned:\n%s", out.String())
	}
}
