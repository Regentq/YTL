package validate

import "fmt"

// Key checks the name of a key or of a section.
//
// The rule is the same as in the syntax highlighting: letters, digits and _,
// not starting with a digit. If dashes or dots should be allowed one day,
// change it here (and in tools/syntax.js, the "ident" piece).
func Key(name string) string {
	if name == "" || reIdent.MatchString(name) {
		return ""
	}
	if name[0] == '"' || name[0] == '\'' {
		return "Names of keys and sections are written without quotes"
	}
	if name[0] >= '0' && name[0] <= '9' {
		return fmt.Sprintf("Name %q cannot start with a digit", name)
	}
	for _, c := range name {
		ok := c == '_' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9'
		if !ok {
			return fmt.Sprintf("Name %q contains the invalid character %q (allowed: letters, digits and _)", name, c)
		}
	}
	return ""
}
