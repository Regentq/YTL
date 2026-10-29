// Package validate checks YTL values.
//
// The type of a value is determined by its SHAPE (like in the tmLanguage grammar):
// if a value looks like IPv4/MAC/date/... but is written incorrectly, it is an error,
// and the message says exactly what is wrong (see explain.go).
// Text must be in quotes. The only exception is the name after enum.
package validate

import (
	"fmt"
	"net/netip"
	"regexp"
	"strings"
	"time"
)

// Issue is a problem found in a value. Start/End are byte offsets INSIDE the value string.
type Issue struct {
	Start, End int
	Message    string
}

// Value checks a whole value. The key is needed because bare words are allowed
// after the enum key. Comma-separated values ("14.5s, 14d, 14ms") are checked one by one.
func Value(key, value string) []Issue {
	// Multi-line values are not checked yet.
	if strings.Contains(value, "\n") || strings.HasPrefix(value, `"""`) {
		return nil
	}

	var issues []Issue
	for _, p := range splitList(value) {
		if msg := checkOne(key, p.text); msg != "" {
			issues = append(issues, Issue{Start: p.off, End: p.off + len(p.text), Message: msg})
		}
	}
	return issues
}

type part struct {
	text string
	off  int
}

// splitList splits on commas, ignoring commas inside "quoted strings".
func splitList(s string) []part {
	var parts []part
	inStr := false
	start := 0

	flush := func(end int) {
		raw := s[start:end]
		trimmed := strings.TrimSpace(raw)
		if trimmed != "" {
			parts = append(parts, part{text: trimmed, off: start + strings.Index(raw, trimmed)})
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

var (
	reColor = regexp.MustCompile(`^#([0-9A-Fa-f]{3,4}|[0-9A-Fa-f]{6}|[0-9A-Fa-f]{8})$`)
	reIdent = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
	reHex   = regexp.MustCompile(`^0[xX][0-9A-Fa-f]+$`)

	// "Looks like an IPv4 address": 4+ numbers, or 3+ numbers followed by /prefix or :port.
	// (Exactly 3 numbers without a suffix is a version, like 1.2.3.)
	reIPv4Loose = regexp.MustCompile(`^\d+(\.\d+){2,}[/:]\S*$|^\d+(\.\d+){3,}([/:]\S*)?$`)

	// "Looks like a MAC address": 4+ groups of exactly two characters, separated by
	// ':' or '-' (or even '::', which is a typo in a MAC, not an IPv6 address).
	reMACLoose      = regexp.MustCompile(`^[0-9A-Za-z]{2}(?:(?:::|[:-])[0-9A-Za-z]{2}){3,}$`)
	reMACSep        = regexp.MustCompile(`::|[:-]`)
	reMACShort      = regexp.MustCompile(`^[0-9A-Fa-f]{4}(\.[0-9A-Fa-f]{4}){2}$`)
	reMACShortLoose = regexp.MustCompile(`^[0-9A-Fa-f]{1,4}\.[0-9A-Fa-f]{1,4}\.[0-9A-Fa-f]{1,4}$`)

	reIPv6Charset = regexp.MustCompile(`^[0-9A-Za-z:.%/]+$`)

	reDateLike   = regexp.MustCompile(`^\d{4}-\d{1,2}-\d{1,2}`)
	reDMYLike    = regexp.MustCompile(`^\d{1,2}[./]\d{1,2}[./]\d{4}$`)
	reTimeLike   = regexp.MustCompile(`^\d{1,2}:\d{2}`)
	reDuration   = regexp.MustCompile(`^\d+(\.\d+)?(ms|s|m|h|d)$`)
	reDurLike    = regexp.MustCompile(`^(\d+(?:\.\d+)?)([a-zA-Z]+)$`)
	reNumber     = regexp.MustCompile(`^-?\d+(\.\d+)?$`)
	reSemverLike = regexp.MustCompile(`^\d+\.\d+\.\d+`)
	reSemver     = regexp.MustCompile(`^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(-[0-9A-Za-z.-]+)?(\+[0-9A-Za-z.-]+)?$`)
)

var constants = map[string]bool{
	"@true": true, "@false": true,
	"@null": true, "@nil": true, "@none": true,
	"@NaN": true, "@Inf": true,
}

var isoLayouts = []string{
	time.RFC3339, "2006-01-02T15:04:05", "2006-01-02T15:04",
	"2006-01-02 15:04:05", "2006-01-02 15:04", "2006-01-02",
}

// checkOne checks a single value. An empty result means "all good".
func checkOne(key, s string) string {
	// enum Name is the only place where a word may be written without quotes.
	if key == "enum" {
		if !reIdent.MatchString(s) {
			return "enum needs a name: letters, digits and _, not starting with a digit"
		}
		return ""
	}

	switch {
	case strings.HasPrefix(s, `"`):
		return checkString(s)

	case strings.HasPrefix(s, "@"):
		if !constants[s] {
			return "Unknown constant " + s + " (valid: @true, @false, @null, @nil, @none, @NaN, @Inf)"
		}
		return ""

	case strings.HasPrefix(s, "#"):
		if !reColor.MatchString(s) {
			return explainColor(s)
		}
		return ""

	case strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X"):
		if !reHex.MatchString(s) {
			return explainHex(s)
		}
		return ""

	case strings.HasPrefix(s, "["):
		return checkIPv6Socket(s)

	case strings.HasPrefix(s, "://"):
		return "The URL is missing its scheme: start it with http:// or https://"

	case strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://"):
		return checkURL(s)

	case reIPv4Loose.MatchString(s):
		return checkIPv4(s)
	}

	// MAC is checked before dates and times: they look similar.
	if msg, isMAC := checkMAC(s); isMAC {
		return msg
	}

	if reDateLike.MatchString(s) {
		return checkLayouts(s, isoLayouts, explainISO)
	}

	if reDMYLike.MatchString(s) {
		return checkLayouts(s, []string{"2.1.2006", "2/1/2006"}, explainDMY)
	}

	if reTimeLike.MatchString(s) {
		return checkLayouts(s, []string{"15:04", "15:04:05", "15:04:05Z07:00"}, explainClockValue)
	}

	// IPv6 handles anything with a colon that isn't caught by time layouts above
	if looksLikeIPv6(s) || strings.Contains(s, ":") {
		return checkIPv6(s)
	}

	if reDuration.MatchString(s) || reNumber.MatchString(s) {
		return ""
	}
	if m := reDurLike.FindStringSubmatch(s); m != nil {
		return fmt.Sprintf("Unknown unit %q (allowed: ms, s, m, h, d)", m[2])
	}

	if reSemver.MatchString(s) {
		return ""
	}
	if looksLikeShortMAC(s) {
		return "Short MAC address needs 3 groups of 4 hex digits (example: 001a.2b3c.4d5e)"
	}
	if reSemverLike.MatchString(s) {
		return explainVersion(s)
	}

	// Everything else is text, and text must be in quotes.
	switch s {
	case "true", "false", "null", "nil", "none", "NaN", "Inf":
		return "Constants start with @: @" + s
	}
	return `Text must be in quotes: "` + s + `"`
}

func checkString(s string) string {
	for i := 1; i < len(s); i++ {
		if s[i] == '\\' {
			i++
			continue
		}
		if s[i] == '"' {
			if i != len(s)-1 {
				return "Extra text after the closing quote"
			}
			return ""
		}
	}
	return `Unclosed string: missing closing "`
}

func checkIPv4(s string) string {
	valid := false
	switch {
	case strings.Contains(s, "/"):
		p, err := netip.ParsePrefix(s)
		valid = err == nil && p.Addr().Is4()
	case strings.Contains(s, ":"):
		ap, err := netip.ParseAddrPort(s)
		valid = err == nil && ap.Addr().Is4()
	default:
		a, err := netip.ParseAddr(s)
		valid = err == nil && a.Is4()
	}
	if valid {
		return ""
	}
	return explainIPv4(s)
}

// looksLikeIPv6 decides whether to treat a value as an IPv6 attempt
// (so that mistakes get an IPv6-specific message instead of "put it in quotes").
func looksLikeIPv6(s string) bool {
	if !reIPv6Charset.MatchString(s) {
		return false
	}
	addr, _, _ := strings.Cut(s, "/")
	if !strings.Contains(addr, "::") && strings.Count(addr, ":") < 3 {
		return false // 17:43:12 and similar are not IPv6
	}
	if strings.Trim(addr, ":") == "" {
		return true // "::"
	}
	// At least one group must be written in hex; otherwise it is just text like foo:bar:baz:qux.
	groups := strings.FieldsFunc(addr, func(r rune) bool { return r == ':' || r == '%' })
	for _, g := range groups {
		if isHex(g) {
			return true
		}
	}
	return false
}

func checkIPv6(s string) string {
	valid := false
	if strings.Contains(s, "/") {
		p, err := netip.ParsePrefix(s)
		valid = err == nil && p.Addr().Is6()
	} else {
		a, err := netip.ParseAddr(s)
		valid = err == nil && a.Is6()
	}
	if valid {
		return ""
	}
	return explainIPv6(s)
}

func checkIPv6Socket(s string) string {
	ap, err := netip.ParseAddrPort(s)
	if err == nil && ap.Addr().Is6() {
		return ""
	}
	return explainIPv6Socket(s)
}

// checkMAC: the second result is true if the value is treated as a MAC address.
func checkMAC(s string) (string, bool) {
	if reMACShort.MatchString(s) {
		return "", true // 001a.2b3c.4d5e
	}
	if !reMACLoose.MatchString(s) {
		return "", false
	}

	groups := reMACSep.Split(s, -1) // the two-character groups
	n := len(groups)
	valid := 0
	bad := ""
	for _, g := range groups {
		if isHex(g) {
			valid++
		} else if bad == "" {
			bad = g
		}
	}
	if valid*2 < n {
		return "", false // mostly not hex: this is just text like to-do-it-now
	}
	if strings.Contains(s, "::") {
		return "A MAC address cannot contain '::': write all 6 pairs separated by a single ':' or '-'", true
	}
	if bad != "" {
		return fmt.Sprintf("%q is not a valid hex pair in a MAC address (allowed: 0-9 and a-f)", bad), true
	}
	if n != 6 {
		return fmt.Sprintf("A MAC address needs 6 pairs, found %d", n), true
	}
	sep := s[2]
	for i := 2; i < len(s); i += 3 {
		if s[i] != sep {
			return "Mixed separators in a MAC address: use only ':' or only '-'", true
		}
	}
	return "", true
}

// looksLikeShortMAC recognises a broken 001a.2b3c.4d5e (for example 001A.2B3C.4D5).
func looksLikeShortMAC(s string) bool {
	if !reMACShortLoose.MatchString(s) || !strings.ContainsAny(s, "abcdefABCDEF") {
		return false
	}
	for _, g := range strings.Split(s, ".") {
		if len(g) == 4 {
			return true
		}
	}
	return false
}

// checkLayouts accepts the value if it parses with any layout; otherwise it asks
// explain for a message that says what is wrong.
func checkLayouts(s string, layouts []string, explain func(string) string) string {
	for _, l := range layouts {
		if _, err := time.Parse(l, s); err == nil {
			return ""
		}
	}
	return explain(s)
}
