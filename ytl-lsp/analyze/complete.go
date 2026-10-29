package analyze

import (
	"regexp"
	"strings"

	"ytl-lsp/parser"
)

// CompletionKind tells the editor which icon to show for a suggestion.
type CompletionKind int

const (
	KindKeyword CompletionKind = iota // the word "from"
	KindSection                       // the name of a section
)

// Completion is one autocomplete suggestion.
type Completion struct {
	Label  string
	Kind   CompletionKind
	Detail string // small grey text next to the label
}

var reIdent = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// Complete returns suggestions for the cursor position (line and character are
// 0-based, character is counted in UTF-16 units, as in LSP).
//
// Only the start of a top-level section header is handled:
//
//	Dae |        -> from
//	Dae fr|      -> from
//	Dae from |   -> names of sections that can be inherited
//	Dae from Da| -> the same list (the editor filters it by what was typed)
//
// The first word always exists, so the context is easy to recognise.
func Complete(text string, line, character int) []Completion {
	lines := strings.Split(text, "\n")
	if line < 0 || line >= len(lines) {
		return nil
	}
	cur := strings.TrimRight(lines[line], "\r")
	before := cur[:fromUTF16(cur, character)] // the part of the line left of the cursor

	// Only top-level headers (no indentation), not comments, not inside quotes.
	if before == "" || before[0] == ' ' || before[0] == '\t' {
		return nil
	}
	if strings.Contains(before, "//") || strings.Count(before, `"`)%2 == 1 {
		return nil
	}

	words := strings.Fields(before)
	trailingSpace := strings.HasSuffix(before, " ") || strings.HasSuffix(before, "\t")

	if len(words) == 0 || !reIdent.MatchString(words[0]) || words[0] == "enum" {
		return nil
	}
	name := words[0]

	switch {
	// "Dae |"
	case len(words) == 1 && trailingSpace:
		return []Completion{fromKeyword()}

	// "Dae fr|" - offer "from" only if what was typed can become "from"
	case len(words) == 2 && !trailingSpace:
		typed := strings.ToLower(words[1])
		if typed != "from" && strings.HasPrefix("from", typed) {
			return []Completion{fromKeyword()}
		}
		return nil

	// "Dae from |"
	case len(words) == 2 && trailingSpace && words[1] == "from":
		return parentCandidates(text, name)

	// "Dae from Da|"
	case len(words) == 3 && !trailingSpace && words[1] == "from":
		return parentCandidates(text, name)
	}
	return nil
}

func fromKeyword() Completion {
	return Completion{Label: "from", Kind: KindKeyword, Detail: "inherit from another section"}
}

// parentCandidates lists top-level sections that "self" may inherit from:
// not itself, and not a section that already inherits from self (that would be a cycle).
func parentCandidates(text, self string) []Completion {
	doc := parser.Parse(text)

	byName := map[string]*parser.Section{}
	var order []string // names in file order, without duplicates
	for _, n := range doc.Children {
		if sec, ok := n.(*parser.Section); ok {
			if _, exists := byName[sec.Name]; !exists {
				byName[sec.Name] = sec
				order = append(order, sec.Name)
			}
		}
	}

	var out []Completion
	for _, name := range order {
		if name == self || inheritsFrom(byName, name, self) {
			continue
		}
		detail := "section"
		if p := byName[name].Parent; p != "" {
			detail = "extends " + p
		}
		out = append(out, Completion{Label: name, Kind: KindSection, Detail: detail})
	}
	return out
}

// inheritsFrom reports whether following the "from" chain starting at "start" reaches "target".
func inheritsFrom(byName map[string]*parser.Section, start, target string) bool {
	seen := map[string]bool{}
	for cur := start; cur != ""; {
		if cur == target {
			return true
		}
		if seen[cur] {
			return false // a cycle that does not contain target
		}
		seen[cur] = true

		sec, ok := byName[cur]
		if !ok {
			return false
		}
		cur = sec.Parent
	}
	return false
}

// fromUTF16 converts a column in UTF-16 units into a byte offset inside the line.
func fromUTF16(line string, units int) int {
	count := 0
	for i, r := range line {
		if count >= units {
			return i
		}
		if r >= 0x10000 {
			count += 2
		} else {
			count++
		}
	}
	return len(line)
}
