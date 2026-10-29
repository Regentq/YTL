package validate

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	reISODate = regexp.MustCompile(`^(\d{4})-(\d{1,2})-(\d{1,2})(?:[T ](\d{1,2}):(\d{2})(?::(\d{2})(?:\.\d+)?)?)?(?:Z|[+-]\d{2}:\d{2})?$`)
	reDMY     = regexp.MustCompile(`^(\d{1,2})[./](\d{1,2})[./](\d{4})$`)
	reClock   = regexp.MustCompile(`^(\d{1,2}):(\d{2})(?::(\d{2}))?(?:Z|[+-]\d{2}:\d{2})?$`)
)

// ---------- small helpers ----------

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func isHex(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F') {
			return false
		}
	}
	return true
}

func isLetters(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z') {
			return false
		}
	}
	return true
}

// number converts a string of digits to an int. Very long strings become a huge number.
func number(s string) int {
	if len(s) > 9 {
		return 1 << 30
	}
	n, _ := strconv.Atoi(s)
	return n
}

// ---------- ports and prefixes ----------

func explainPort(port string) string {
	if port == "" {
		return "Missing port after ':'"
	}
	if !isDigits(port) {
		return fmt.Sprintf("Port %q is not a number", port)
	}
	if number(port) > 65535 {
		return fmt.Sprintf("Port %s is out of range (0-65535)", port)
	}
	return ""
}

func explainPrefix(prefix string, max int) string {
	if prefix == "" {
		return "Missing prefix length after '/'"
	}
	if !isDigits(prefix) {
		return fmt.Sprintf("Prefix %q is not a number", prefix)
	}
	if number(prefix) > max {
		return fmt.Sprintf("Prefix /%s is out of range (0-%d)", prefix, max)
	}
	return ""
}

// ---------- IPv4 ----------

// explainIPv4Addr checks only the address part (four numbers separated by dots).
func explainIPv4Addr(addr string) string {
	parts := strings.Split(addr, ".")
	if len(parts) != 4 {
		return fmt.Sprintf("An IPv4 address needs 4 numbers, found %d", len(parts))
	}
	for _, p := range parts {
		if !isDigits(p) {
			return fmt.Sprintf("%q is not a number in an IPv4 address", p)
		}
		if number(p) > 255 {
			return fmt.Sprintf("%s is out of range (0-255) in an IPv4 address", p)
		}
		if len(p) > 1 && p[0] == '0' {
			return fmt.Sprintf("Leading zero in %q: write the number without leading zeros", p)
		}
	}
	return ""
}

// explainIPv4 explains a broken "address", "address/prefix" or "address:port".
func explainIPv4(s string) string {
	addr, rest := s, ""
	if i := strings.IndexAny(s, "/:"); i >= 0 {
		addr, rest = s[:i], s[i:]
	}
	if msg := explainIPv4Addr(addr); msg != "" {
		return msg
	}
	switch {
	case strings.HasPrefix(rest, "/"):
		if msg := explainPrefix(rest[1:], 32); msg != "" {
			return msg
		}
	case strings.HasPrefix(rest, ":"):
		if msg := explainPort(rest[1:]); msg != "" {
			return msg
		}
	}
	return "Invalid IPv4 address"
}

// ---------- IPv6 ----------

func splitGroups(side string) []string {
	if side == "" {
		return nil
	}
	return strings.Split(side, ":")
}

// explainIPv6Addr checks only the address part (no /prefix). Empty result = looks fine.
func explainIPv6Addr(addr string) string {
	if at := strings.Index(addr, "%"); at >= 0 {
		if at == len(addr)-1 {
			return "Missing zone name after '%'"
		}
		addr = addr[:at]
	}
	if addr == "" {
		return "Missing IPv6 address"
	}
	if strings.Contains(addr, ":::") {
		return "Too many colons in a row (':::'); '::' may be used only once"
	}
	if strings.Count(addr, "::") > 1 {
		return "'::' may appear only once in an IPv6 address"
	}

	compressed := strings.Contains(addr, "::")
	var groups []string
	if compressed {
		left, right, _ := strings.Cut(addr, "::")
		groups = append(groups, splitGroups(left)...)
		groups = append(groups, splitGroups(right)...)
	} else {
		groups = strings.Split(addr, ":")
	}

	total := 0 // how many 16-bit groups are written explicitly
	for i, g := range groups {
		switch {
		case g == "":
			return "Stray ':' in the IPv6 address"

		case strings.Contains(g, "."):
			// An embedded IPv4 address, allowed only at the very end.
			if i != len(groups)-1 {
				return "An embedded IPv4 address is only allowed at the end of an IPv6 address"
			}
			if msg := explainIPv4Addr(g); msg != "" {
				return msg
			}
			total += 2

		case !isHex(g):
			return fmt.Sprintf("Group %q is not hexadecimal (allowed: 0-9 and a-f)", g)

		case len(g) > 4:
			return fmt.Sprintf("Group %q has more than 4 hex digits", g)

		default:
			total++
		}
	}

	switch {
	case compressed && total > 7:
		return fmt.Sprintf("Too many groups: with '::' at most 7 can be written, found %d", total)
	case !compressed && total < 8:
		return fmt.Sprintf("An IPv6 address needs 8 groups, found %d (use '::' to skip zeros)", total)
	case !compressed && total > 8:
		return fmt.Sprintf("Too many groups: an IPv6 address has 8, found %d", total)
	}
	return ""
}

// explainIPv6 explains a broken "address" or "address/prefix".
func explainIPv6(s string) string {
	addr, prefix, hasPrefix := strings.Cut(s, "/")
	if msg := explainIPv6Addr(addr); msg != "" {
		return msg
	}
	if hasPrefix {
		if msg := explainPrefix(prefix, 128); msg != "" {
			return msg
		}
	}
	return "Invalid IPv6 address"
}

// explainIPv6Socket explains a broken "[address]:port".
func explainIPv6Socket(s string) string {
	closing := strings.Index(s, "]")
	if closing < 0 {
		return "Missing closing ']' in [IPv6]:port"
	}
	inner, rest := s[1:closing], s[closing+1:]

	if strings.Contains(inner, "/") {
		return "A prefix length is not allowed inside [address]:port"
	}
	if msg := explainIPv6Addr(inner); msg != "" {
		return msg
	}
	if rest == "" {
		return "Missing port: write it as [address]:port"
	}
	if !strings.HasPrefix(rest, ":") {
		return fmt.Sprintf("Unexpected text %q after ']'", rest)
	}
	if msg := explainPort(rest[1:]); msg != "" {
		return msg
	}
	return "Invalid [IPv6]:port (example: [2001:db8::1]:8080)"
}

// ---------- URL ----------

func isDigitsAndDots(s string) bool {
	if !strings.Contains(s, ".") {
		return false
	}
	for _, c := range s {
		if c != '.' && (c < '0' || c > '9') {
			return false
		}
	}
	return true
}

// explainHost checks a host name or an IPv4 address.
func explainHost(host string) string {
	if isDigitsAndDots(host) {
		return explainIPv4Addr(host)
	}

	labels := strings.Split(host, ".")
	for _, l := range labels {
		switch {
		case l == "":
			return "Empty part in the host name (two dots in a row, or a dot at the start or end)"
		case len(l) > 63:
			return "A part of the host name is longer than 63 characters"
		case l[0] == '-' || l[len(l)-1] == '-':
			return fmt.Sprintf("Host name part %q cannot start or end with '-'", l)
		}
		for _, c := range l {
			ok := c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-' || c == '_'
			if !ok {
				return fmt.Sprintf("Invalid character %q in the host name", c)
			}
		}
	}

	if len(labels) > 1 {
		tld := labels[len(labels)-1]
		if len(tld) < 2 || !isLetters(tld) {
			return fmt.Sprintf("The top-level domain %q must be at least 2 letters", tld)
		}
	}
	return ""
}

// checkURL validates http:// and https:// addresses. Empty result = valid.
func checkURL(s string) string {
	scheme, rest, _ := strings.Cut(s, "://")

	// The authority is everything up to the first / ? or #
	authority := rest
	if end := strings.IndexAny(rest, "/?#"); end >= 0 {
		authority = rest[:end]
	}
	if at := strings.LastIndex(authority, "@"); at >= 0 {
		authority = authority[at+1:] // drop user:password@
	}
	if authority == "" {
		return fmt.Sprintf("The URL has no host after \"%s://\"", scheme)
	}

	// [IPv6]:port
	if strings.HasPrefix(authority, "[") {
		closing := strings.Index(authority, "]")
		if closing < 0 {
			return "Missing closing ']' in the URL host"
		}
		if msg := explainIPv6Addr(authority[1:closing]); msg != "" {
			return msg
		}
		tail := authority[closing+1:]
		if tail == "" {
			return ""
		}
		if !strings.HasPrefix(tail, ":") {
			return "Unexpected text after ']' in the URL host"
		}
		return explainPort(tail[1:])
	}

	host := authority
	if i := strings.LastIndex(authority, ":"); i >= 0 {
		host = authority[:i]
		if msg := explainPort(authority[i+1:]); msg != "" {
			return msg
		}
	}
	if host == "" {
		return "The URL has no host name"
	}
	return explainHost(host)
}

// ---------- dates and times ----------

// explainDay checks the month and the day of the month.
func explainDay(year, month, day int) string {
	if month < 1 || month > 12 {
		return fmt.Sprintf("Month %d is out of range (1-12)", month)
	}
	last := time.Date(year, time.Month(month)+1, 0, 0, 0, 0, 0, time.UTC).Day()
	if day < 1 || day > last {
		return fmt.Sprintf("Day %d is out of range for month %d (1-%d)", day, month, last)
	}
	return ""
}

// explainClock checks hours, minutes and (optional) seconds given as strings.
func explainClock(h, m, sec string) string {
	if number(h) > 23 {
		return fmt.Sprintf("Hour %s is out of range (0-23)", h)
	}
	if number(m) > 59 {
		return fmt.Sprintf("Minute %s is out of range (0-59)", m)
	}
	if sec != "" && number(sec) > 59 {
		return fmt.Sprintf("Second %s is out of range (0-59)", sec)
	}
	return ""
}

func explainISO(s string) string {
	m := reISODate.FindStringSubmatch(s)
	if m == nil {
		return "Invalid date/time (example: 2026-10-26T17:43:12Z)"
	}
	if msg := explainDay(number(m[1]), number(m[2]), number(m[3])); msg != "" {
		return msg
	}
	if len(m[2]) != 2 || len(m[3]) != 2 {
		return "Use two digits for the month and the day (example: 2026-01-05)"
	}
	if m[4] != "" {
		if msg := explainClock(m[4], m[5], m[6]); msg != "" {
			return msg
		}
	}
	return "Invalid date/time (example: 2026-10-26T17:43:12Z)"
}

func explainDMY(s string) string {
	m := reDMY.FindStringSubmatch(s)
	if m == nil {
		return "Invalid date (example: 26.10.2026)"
	}
	if msg := explainDay(number(m[3]), number(m[2]), number(m[1])); msg != "" {
		return msg
	}
	return "Invalid date (example: 26.10.2026)"
}

func explainClockValue(s string) string {
	m := reClock.FindStringSubmatch(s)
	if m == nil {
		return "Invalid time (example: 17:43:12)"
	}
	if msg := explainClock(m[1], m[2], m[3]); msg != "" {
		return msg
	}
	return "Invalid time (example: 17:43:12)"
}

// ---------- numbers, colors, versions ----------

func explainColor(s string) string {
	body := s[1:]
	if body == "" {
		return "Invalid color: no digits after '#'"
	}
	for _, c := range body {
		if !isHex(string(c)) {
			return fmt.Sprintf("Invalid color: contains the non-hex character %q", c)
		}
	}
	return fmt.Sprintf("Invalid color: %d hex digits, expected 3, 4, 6 or 8", len(body))
}

func explainHex(s string) string {
	body := s[2:]
	if body == "" {
		return "Invalid hex number: no digits after 0x"
	}
	for _, c := range body {
		if !isHex(string(c)) {
			return fmt.Sprintf("Invalid hex number: %q is not a hex digit", c)
		}
	}
	return "Invalid hex number"
}

func explainVersion(s string) string {
	// Redirect to IPv4 only if it is clearly a broken IP address with letters at the end (e.g., 192.168.1.abcd)
	if strings.Contains(s, ".") {
		parts := strings.Split(s, ".")
		// If the first two octets are digits, but the tail contains non-digits
		if len(parts) >= 3 && isDigits(parts[0]) && isDigits(parts[1]) {
			for _, p := range parts[2:] {
				if !isDigits(p) {
					return explainIPv4(s)
				}
			}
		}
	}

	core := s
	if i := strings.IndexAny(core, "-+"); i >= 0 {
		core = core[:i]
	}
	for _, p := range strings.Split(core, ".") {
		if len(p) > 1 && p[0] == '0' {
			return fmt.Sprintf("Invalid version: %q has a leading zero", p)
		}
	}
	return "Invalid version (example: 1.2.3 or 1.2.3-rs)"
}
