// Package analyze glues the parser and the validator together: file text -> list of errors.
// It knows nothing about LSP, which makes it easy to test.
package analyze

import (
	"fmt"
	"strings"

	"ytl-lsp/parser"
	"ytl-lsp/validate"
)

// Diagnostic is one error. The columns in Range are already in UTF-16, as LSP requires.
type Diagnostic struct {
	Range   parser.Range
	Message string
}

// Analyze parses the text and returns every error found.
func Analyze(text string) []Diagnostic {
	doc := parser.Parse(text)

	var out []Diagnostic

	// 1. Syntax errors from the parser.
	for _, e := range doc.Errors {
		out = append(out, Diagnostic{Range: e.Range, Message: e.Message})
	}

	// 2. Errors in values.
	out = append(out, checkNodes(doc.Children)...)

	// 3. Errors in inheritance ("Name from Parent:").
	out = append(out, checkInheritance(doc)...)

	// 4. The parser counts columns in bytes (UTF-8) but LSP wants UTF-16.
	//    For Latin text there is no difference, but with Cyrillic or emoji in a
	//    string the columns would drift.
	lines := strings.Split(text, "\n")
	for i := range out {
		out[i].Range.Start = toUTF16(lines, out[i].Range.Start)
		out[i].Range.End = toUTF16(lines, out[i].Range.End)
	}
	return out
}

func checkNodes(nodes []parser.Node) []Diagnostic {
	var out []Diagnostic
	for _, n := range nodes {
		switch v := n.(type) {
		case *parser.Section:
			if msg := validate.Key(v.Name); msg != "" {
				out = append(out, Diagnostic{Range: v.NameRange, Message: msg})
			}
			out = append(out, checkNodes(v.Children)...)

		case *parser.Pair:
			if msg := validate.Key(v.Key); msg != "" {
				out = append(out, Diagnostic{Range: v.KeyRange, Message: msg})
			}
			if v.IsArray {
				for _, it := range v.Items {
					out = append(out, valueIssues(v.Key, it.Value, it.Range.Start)...)
				}
			} else if v.Value != "" {
				out = append(out, valueIssues(v.Key, v.Value, v.ValueRange.Start)...)
			}
		}
	}
	return out
}

// valueIssues checks one value. start is where the value begins in the file.
func valueIssues(key, value string, start parser.Position) []Diagnostic {
	var out []Diagnostic
	for _, is := range validate.Value(key, value) {
		out = append(out, Diagnostic{
			Range: parser.Range{
				Start: parser.Position{Line: start.Line, Col: start.Col + is.Start},
				End:   parser.Position{Line: start.Line, Col: start.Col + is.End},
			},
			Message: is.Message,
		})
	}
	return out
}

// checkInheritance validates every "Name from Parent:" section:
//   - a section cannot inherit from itself;
//   - the parent must exist (parents are top-level sections);
//   - inheritance must not form a cycle (A from B, B from A).
func checkInheritance(doc *parser.Document) []Diagnostic {
	// Top-level sections by name. For duplicates the first one wins.
	byName := map[string]*parser.Section{}
	for _, n := range doc.Children {
		if sec, ok := n.(*parser.Section); ok {
			if _, exists := byName[sec.Name]; !exists {
				byName[sec.Name] = sec
			}
		}
	}

	var out []Diagnostic
	var walk func(nodes []parser.Node)
	walk = func(nodes []parser.Node) {
		for _, n := range nodes {
			sec, ok := n.(*parser.Section)
			if !ok {
				continue
			}
			if sec.Parent != "" {
				out = append(out, checkParent(sec, byName)...)
			}
			walk(sec.Children)
		}
	}
	walk(doc.Children)
	return out
}

func checkParent(sec *parser.Section, byName map[string]*parser.Section) []Diagnostic {
	report := func(msg string) []Diagnostic {
		return []Diagnostic{{Range: sec.ParentRange, Message: msg}}
	}

	if sec.Parent == sec.Name {
		return report(fmt.Sprintf("Section %q cannot inherit from itself", sec.Name))
	}

	cur, ok := byName[sec.Parent]
	if !ok {
		return report(fmt.Sprintf("Parent section %q is not defined", sec.Parent))
	}

	// Follow the chain of parents. If it leads back to sec, that is a cycle.
	chain := []string{sec.Name}
	visited := map[string]bool{}
	for {
		chain = append(chain, cur.Name)
		if cur.Name == sec.Name {
			return report("Circular inheritance: " + strings.Join(chain, " -> "))
		}
		if visited[cur.Name] {
			return nil // a cycle that does not include sec; its own members report it
		}
		visited[cur.Name] = true

		if cur.Parent == "" {
			return nil
		}
		next, ok := byName[cur.Parent]
		if !ok {
			return nil // the missing parent is reported on that section itself
		}
		cur = next
	}
}

// toUTF16 converts a position from bytes of the line to UTF-16 code units.
func toUTF16(lines []string, p parser.Position) parser.Position {
	if p.Line < 0 || p.Line >= len(lines) {
		return p
	}
	line := strings.TrimRight(lines[p.Line], "\r")

	n := p.Col
	if n > len(line) {
		n = len(line)
	}

	col := 0
	for _, r := range line[:n] {
		if r >= 0x10000 {
			col += 2 // characters outside the basic plane (emoji) take 2 units
		} else {
			col++
		}
	}
	return parser.Position{Line: p.Line, Col: col}
}
