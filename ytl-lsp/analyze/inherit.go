package analyze

import (
	"strings"

	"ytl-lsp/parser"
)

// LineEdit replaces columns [StartCol, EndCol) of one line with NewText.
type LineEdit struct {
	Line     int
	StartCol int
	EndCol   int
	NewText  string
}

// InheritedStub is called after the user pressed Enter. newLine is the number of
// the line that was just created. If the line above it is a header like
//
//	da from YTL:
//
// and the section is still empty, it returns an edit that fills the new line with
// the structure of the parent section (and of its ancestors): the keys WITHOUT
// values, and the nested sections together with their keys:
//
//	da from YTL:
//	    project_name
//	    Hardware:
//	        Classic_MAC
//
// A key without a value is not an error, so whatever the user does not need can
// simply be left as it is. indentUnit is one level of indentation ("    " or "\t").
func InheritedStub(text string, newLine int, indentUnit string) (LineEdit, bool) {
	lines := strings.Split(text, "\n")
	if newLine < 1 || newLine >= len(lines) {
		return LineEdit{}, false
	}

	// The new line must be blank (it may contain auto-indentation).
	cur := strings.TrimRight(lines[newLine], "\r")
	if strings.TrimSpace(cur) != "" {
		return LineEdit{}, false
	}

	doc := parser.Parse(text)
	header := findHeader(doc.Children, newLine-1)
	if header == nil || header.Parent == "" || len(header.Children) != 0 {
		return LineEdit{}, false // not an inheriting header, or it already has content
	}

	rawStubs := inheritedStubs(doc, header.Parent, header.Name)
	if len(rawStubs) == 0 {
		return LineEdit{}, false
	}

	prev := strings.TrimRight(lines[newLine-1], "\r")
	headerIndent := prev[:len(prev)-len(strings.TrimLeft(prev, " \t"))]

	// Render starting at headerIndent + indentUnit so the first level fields
	// get the correct child indentation level.
	rendered := renderStubs(rawStubs, headerIndent+indentUnit, indentUnit)

	return LineEdit{
		Line:     newLine,
		StartCol: 0,
		EndCol:   len(cur),
		NewText:  strings.Join(rendered, "\n"),
	}, true
}

// findHeader finds the section whose name is written on the given line.
func findHeader(nodes []parser.Node, line int) *parser.Section {
	for _, n := range nodes {
		sec, ok := n.(*parser.Section)
		if !ok {
			continue
		}
		if sec.NameRange.Start.Line == line {
			return sec
		}
		if found := findHeader(sec.Children, line); found != nil {
			return found
		}
	}
	return nil
}

// stub is one line of the generated text, with the lines nested under it.
type stub struct {
	text     string // "key", "key[]" or "Section:"
	section  bool
	children []*stub
}

// inheritedStubs collects the structure of the parent and of all its ancestors.
// The oldest ancestor comes first. A key that appears several times is listed once,
// and sections with the same name are merged.
func inheritedStubs(doc *parser.Document, parent, self string) []*stub {
	byName := map[string]*parser.Section{}
	for _, n := range doc.Children {
		if sec, ok := n.(*parser.Section); ok {
			if _, exists := byName[sec.Name]; !exists {
				byName[sec.Name] = sec
			}
		}
	}

	// The chain of ancestors, nearest first. "seen" protects against cycles.
	var chain []*parser.Section
	seen := map[string]bool{self: true}
	for name := parent; name != "" && !seen[name]; {
		seen[name] = true
		sec, ok := byName[name]
		if !ok {
			break
		}
		chain = append(chain, sec)
		name = sec.Parent
	}

	var stubs []*stub
	for i := len(chain) - 1; i >= 0; i-- {
		stubs = addStubs(stubs, chain[i].Children)
	}
	return stubs
}

// addStubs adds the nodes to the list, skipping what is already there.
func addStubs(dst []*stub, nodes []parser.Node) []*stub {
	for _, n := range nodes {
		switch v := n.(type) {
		case *parser.Pair:
			text := v.Key
			if v.IsArray {
				text += "[]"
			}
			if findStub(dst, text, false) == nil {
				dst = append(dst, &stub{text: text})
			}

		case *parser.Section:
			text := v.Name + ":"
			s := findStub(dst, text, true)
			if s == nil {
				s = &stub{text: text, section: true}
				dst = append(dst, s)
			}
			s.children = addStubs(s.children, v.Children)
		}
	}
	return dst
}

func findStub(list []*stub, text string, section bool) *stub {
	for _, s := range list {
		if s.text == text && s.section == section {
			return s
		}
	}
	return nil
}

// renderStubs turns the stubs into indented lines.
func renderStubs(list []*stub, indent, unit string) []string {
	var lines []string
	for _, s := range list {
		lines = append(lines, indent+s.text)
		lines = append(lines, renderStubs(s.children, indent+unit, unit)...)
	}
	return lines
}
