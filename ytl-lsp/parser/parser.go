package parser

import "strings"

// frame is one open section on the stack.
type frame struct {
	indent  int
	section *Section // nil only for the root
}

// Parse parses YTL text line by line. Nesting is defined by indentation.
func Parse(text string) *Document {
	doc := &Document{}
	stack := []frame{{indent: -1}} // element 0 is the "file root"
	lines := strings.Split(text, "\n")

	// Manual counter: multi-line values (""" and arrays) consume several lines.
	for i := 0; i < len(lines); i++ {
		line := stripComment(strings.TrimRight(lines[i], "\r"))
		line = strings.TrimRight(line, " \t")
		if line == "" {
			continue
		}

		indent := len(line) - len(strings.TrimLeft(line, " \t"))
		content := line[indent:]
		lineRange := Range{
			Start: Position{i, indent},
			End:   Position{i, len(line)},
		}

		// 1. Close sections whose children have ended.
		for len(stack) > 1 && stack[len(stack)-1].indent >= indent {
			stack = stack[:len(stack)-1]
		}

		// 2. Section or pair?
		var node Node
		isSection := false

		if sec := parseSection(i, indent, content, lineRange); sec != nil {
			node = sec
			isSection = true
		} else {
			pair, extraLines := parsePair(doc, lines, i, indent, content, lineRange)
			node = pair
			i += extraLines // skip the lines consumed by a multi-line value
		}

		// 3. Attach the node to the current parent.
		parent := stack[len(stack)-1]
		if parent.section == nil {
			doc.Children = append(doc.Children, node)
		} else {
			parent.section.Children = append(parent.section.Children, node)
		}

		// 4. A section is opened (pushed onto the stack).
		if isSection {
			stack = append(stack, frame{indent: indent, section: node.(*Section)})
		}

		// 5. All open sections extend at least to the end of this node.
		end := node.Span().End
		for _, f := range stack[1:] {
			f.section.Range.End = end
		}
	}

	return doc
}

// parseSection handles "Name:" and "Name from Parent:". Returns nil if the line is not a section.
func parseSection(lineNo, indent int, content string, lineRange Range) *Section {
	if !strings.HasSuffix(content, ":") {
		return nil
	}
	words := strings.Fields(strings.TrimSuffix(content, ":"))

	switch {
	case len(words) == 1:
		name := words[0]
		return &Section{
			Name: name,
			NameRange: Range{
				Start: Position{lineNo, indent},
				End:   Position{lineNo, indent + len(name)},
			},
			Range: lineRange,
		}

	case len(words) == 3 && words[1] == "from":
		name, parent := words[0], words[2]
		parentCol := indent + strings.LastIndex(content, parent)
		return &Section{
			Name: name,
			NameRange: Range{
				Start: Position{lineNo, indent},
				End:   Position{lineNo, indent + len(name)},
			},
			Parent: parent,
			ParentRange: Range{
				Start: Position{lineNo, parentCol},
				End:   Position{lineNo, parentCol + len(parent)},
			},
			Range: lineRange,
		}
	}
	return nil
}

// parsePair handles "key value". The second result is the number of EXTRA lines
// consumed by a multi-line value or array.
func parsePair(doc *Document, lines []string, lineNo, indent int, content string, lineRange Range) (*Pair, int) {
	// Multi-line array: a line like "os[" - elements are read raw up to the "]" line.
	if strings.HasSuffix(content, "[") {
		if pair, extra, ok := parseArray(doc, lines, lineNo, indent, content, lineRange); ok {
			return pair, extra
		}
	}

	// Single-line array: os[1.0, 100]. The "[" must follow the key directly.
	if k := strings.Index(content, "["); k > 0 && strings.HasSuffix(content, "]") &&
		!strings.ContainsAny(content[:k], " \t") {
		return parseInlineArray(lineNo, indent, content, lineRange), 0
	}

	keyEnd := strings.IndexAny(content, " \t")

	// A key without a value is allowed: developers often write the keys first
	// and fill in the values later.
	if keyEnd == -1 {
		end := lineRange.End
		return &Pair{
			Key:        content,
			KeyRange:   lineRange,
			ValueRange: Range{Start: end, End: end},
			Range:      lineRange,
		}, 0
	}

	key := content[:keyEnd]
	rest := content[keyEnd:]
	valueOffset := keyEnd + (len(rest) - len(strings.TrimLeft(rest, " \t")))
	value := content[valueOffset:]

	pair := &Pair{
		Key: key,
		KeyRange: Range{
			Start: Position{lineNo, indent},
			End:   Position{lineNo, indent + len(key)},
		},
		Value: value,
		ValueRange: Range{
			Start: Position{lineNo, indent + valueOffset},
			End:   Position{lineNo, lineRange.End.Col},
		},
		Range: lineRange,
	}

	// Multi-line string: opened """ and not closed on the same line.
	if strings.HasPrefix(value, `"""`) && !strings.Contains(value[3:], `"""`) {
		var b strings.Builder
		b.WriteString(value)

		for j := lineNo + 1; j < len(lines); j++ {
			l := strings.TrimRight(lines[j], "\r")
			b.WriteString("\n")
			b.WriteString(l)

			if idx := strings.Index(l, `"""`); idx >= 0 {
				end := Position{j, idx + 3}
				pair.Value = b.String()
				pair.ValueRange.End = end
				pair.Range.End = end
				return pair, j - lineNo
			}
		}

		// End of file reached and the closing """ is missing.
		doc.Errors = append(doc.Errors, ParseError{
			Range:   pair.ValueRange,
			Message: `Unclosed """ string`,
		})
		return pair, len(lines) - 1 - lineNo
	}

	return pair, 0
}

// stripComment cuts off a "//..." comment. Rules:
//   - "//" inside a "quoted string" is not a comment;
//   - "//" must be at the start of the line or after a space,
//     so https://site.com stays intact.
func stripComment(line string) string {
	if strings.Contains(line, `"""`) {
		return line // lines with """ are left alone so multi-line strings are not broken
	}

	inString := false
	for i := 0; i < len(line); i++ {
		c := line[i]

		if inString {
			if c == '\\' {
				i++ // skip the escaped character
			} else if c == '"' {
				inString = false
			}
			continue
		}

		if c == '"' {
			inString = true
			continue
		}

		if c == '/' && i+1 < len(line) && line[i+1] == '/' &&
			(i == 0 || line[i-1] == ' ' || line[i-1] == '\t') {
			return line[:i]
		}
	}
	return line
}

// parseArray parses a multi-line array:
//
//	os[
//	1.0
//	100
//	]
//
// Indentation inside does not matter, so the lines are read here and not by the
// main loop. Returns the pair, the number of extra lines consumed, and ok=false
// if this is not an array.
func parseArray(doc *Document, lines []string, lineNo, indent int, content string, lineRange Range) (*Pair, int, bool) {
	key := strings.TrimSpace(strings.TrimSuffix(content, "["))
	if key == "" || strings.ContainsAny(key, " \t") {
		return nil, 0, false
	}

	keyRange := Range{
		Start: Position{lineNo, indent},
		End:   Position{lineNo, indent + len(key)},
	}
	pair := &Pair{
		Key:      key,
		KeyRange: keyRange,
		IsArray:  true,
		Range:    lineRange,
		ValueRange: Range{
			Start: Position{lineNo, indent + len(content) - 1}, // position of "["
			End:   lineRange.End,
		},
	}

	for j := lineNo + 1; j < len(lines); j++ {
		line := stripComment(strings.TrimRight(lines[j], "\r"))
		line = strings.TrimRight(line, " \t")
		text := strings.TrimSpace(line)
		if text == "" {
			continue
		}
		col := len(line) - len(strings.TrimLeft(line, " \t"))

		// The closing bracket.
		if text == "]" {
			end := Position{j, col + 1}
			pair.ValueRange.End = end
			pair.Range.End = end

			values := make([]string, len(pair.Items))
			for k, it := range pair.Items {
				values[k] = it.Value
			}
			pair.Value = strings.Join(values, ", ")
			return pair, j - lineNo, true
		}

		// An element. A trailing comma is allowed but not required.
		text = strings.TrimSpace(strings.TrimSuffix(text, ","))
		if text == "" {
			continue
		}
		pair.Items = append(pair.Items, Item{
			Value: text,
			Range: Range{
				Start: Position{j, col},
				End:   Position{j, col + len(text)},
			},
		})
	}

	// The file ended and "]" was never found.
	doc.Errors = append(doc.Errors, ParseError{
		Range:   keyRange,
		Message: "Unclosed array: closing ] not found",
	})
	return pair, len(lines) - 1 - lineNo, true
}

// parseInlineArray parses a single-line array: os[1.0, 100, "a, b"]
func parseInlineArray(lineNo, indent int, content string, lineRange Range) *Pair {
	k := strings.Index(content, "[")
	key := content[:k]
	inner := content[k+1 : len(content)-1]
	innerCol := indent + k + 1 // column where the bracket contents begin

	pair := &Pair{
		Key: key,
		KeyRange: Range{
			Start: Position{lineNo, indent},
			End:   Position{lineNo, indent + len(key)},
		},
		Value:   strings.TrimSpace(inner),
		IsArray: true,
		Range:   lineRange,
		ValueRange: Range{
			Start: Position{lineNo, indent + k},
			End:   lineRange.End,
		},
	}

	for _, p := range splitCommas(inner) {
		pair.Items = append(pair.Items, Item{
			Value: p.text,
			Range: Range{
				Start: Position{lineNo, innerCol + p.off},
				End:   Position{lineNo, innerCol + p.off + len(p.text)},
			},
		})
	}
	return pair
}

type listPart struct {
	text string
	off  int // offset inside the original string
}

// splitCommas splits on commas, ignoring commas inside "quoted strings".
func splitCommas(s string) []listPart {
	var parts []listPart
	inStr := false
	start := 0

	flush := func(end int) {
		raw := s[start:end]
		trimmed := strings.TrimSpace(raw)
		if trimmed != "" {
			parts = append(parts, listPart{text: trimmed, off: start + strings.Index(raw, trimmed)})
		}
	}

	for i := 0; i < len(s); i++ {
		c := s[i]
		if inStr {
			if c == '\\' {
				i++
			} else if c == '"' {
				inStr = false
			}
			continue
		}
		switch c {
		case '"':
			inStr = true
		case ',':
			flush(i)
			start = i + 1
		}
	}
	flush(len(s))
	return parts
}
