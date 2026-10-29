package validate

import (
	"strings"
	"testing"
)

func TestValid(t *testing.T) {
	valid := []string{
		`"text"`, `"a, b"`, `@true`, `@null`, `#88ff00`, `0x59544C00`,
		`127.0.0.1`, `127.0.0.1:8080`, `10.0.0.0/24`,
		`2001:db8::/32`, `2001:0db8:85a3:0000:0000:8a2e:0370:7334`,
		`[2001:db8:85a3::8a2e:370:7334]:8080`,
		`00:1A:2B:3C:4D:5E`, `00-0a-2b-3c-4d-5e`, `001a.2b3c.4d5e`,
		`2026-10-26T17:43:12Z`, `2026-10-26T20:43:12+03:00`, `2026-10-26`,
		`26.10.2026`, `17:43:12`, `12:30`,
		`14.5s, 14d, 14ms`, `102`, `10.2`, `1.0`, `1.2.0-rs`,
		`https://example.com/a/b`,
	}
	for _, v := range valid {
		if issues := Value("key", v); len(issues) != 0 {
			t.Errorf("%q: expected no errors, got %+v", v, issues)
		}
	}
}

func TestInvalid(t *testing.T) {
	invalid := []string{
		`999.1.1.1`, `10.0.0.0/99`, `127.0.0.1:99999`,
		`00:1A:2B:3C:4D`, `00:1A:2B:3C:4D:5E:6F`,
		`"abc`, `"abc" xyz`, `@tru`, `#12345`, `0x`,
		`2026-13-45`, `32.01.2026`, `25:99`,
		`14x`, `14.5s, 14x`,
		`2001:db8:::1`, `[2001:db8::1]:99999`, `01.2.3`,
		`hello`, `true`, `"ok", bad`, `https://`,
	}
	for _, v := range invalid {
		if issues := Value("key", v); len(issues) == 0 {
			t.Errorf("%q: expected an error, got none", v)
		}
	}
}

func TestListOffsets(t *testing.T) {
	// the error is in the second element: "14x" starts at offset 7
	issues := Value("key", "14.5s, 14x")
	if len(issues) != 1 || issues[0].Start != 7 || issues[0].End != 10 {
		t.Fatalf("wrong offsets: %+v", issues)
	}
}

func TestEnum(t *testing.T) {
	if issues := Value("enum", "Color"); len(issues) != 0 {
		t.Errorf("enum Color must be valid: %+v", issues)
	}
	if issues := Value("enum", "1bad"); len(issues) == 0 {
		t.Errorf("enum 1bad must be an error")
	}
	// the same bare word under a normal key is an error
	if issues := Value("name", "Color"); len(issues) == 0 {
		t.Errorf("Color without quotes under a normal key must be an error")
	}
}

// Every broken value must get a message that names the real problem.
func TestMessages(t *testing.T) {
	cases := []struct{ value, want string }{
		// IPv4
		{`192.168.1.300/24`, "300 is out of range (0-255)"},
		{`999.1.1.1`, "999 is out of range (0-255)"},
		{`10.0.0/16`, "needs 4 numbers, found 3"},
		{`172.16.0.1.2/24`, "needs 4 numbers, found 5"},
		{`10.0.0.0/33`, "Prefix /33 is out of range (0-32)"},
		{`192.168.1.1:80bc`, `Port "80bc" is not a number`},
		{`192.168.1.1:`, "Missing port"},
		{`127.0.0.1:99999`, "Port 99999 is out of range"},
		// IPv6
		{`2001:db8::1::2/64`, "'::' may appear only once"},
		{`:::1/64`, "Too many colons"},
		{`2001:db8:g123::/64`, "not hexadecimal"},
		{`2001:db8:00005::/64`, "more than 4 hex digits"},
		{`2001:db8:1234:5678:90ab:cdef:1111/64`, "needs 8 groups, found 7"},
		{`2001:db8:1234:5678:90ab:cdef:1111:2222:3333/64`, "Too many groups"},
		{`2001:db8:1234:5678:90ab:cdef:192.168.1.300/64`, "300 is out of range (0-255)"},
		{`2001:db8::1/129`, "Prefix /129 is out of range (0-128)"},
		{`[2001:db8::1]:65536`, "Port 65536 is out of range"},
		{`[2001:db8::1]`, "Missing port"},
		// MAC
		{`00:1A:2B:3C:4D`, "needs 6 pairs, found 5"},
		{`00:1A:2B:3C:4D:5E:6F`, "needs 6 pairs, found 7"},
		{`00:1A:2B:3C:4D:5G`, `"5G" is not a valid hex pair`},
		{`00:1A:2B-3C:4D:5E`, "Mixed separators"},
		{`00:1A:2B:3C::4D`, "cannot contain '::'"},
		{`00:1A:2B:3C:4D:5E::6F`, "cannot contain '::'"},
		{`001A.2B3C.4D5`, "Short MAC"},
		// URL
		{`http://`, "no host"},
		{`https://google..com`, "Empty part"},
		{`https://-google.com`, "cannot start or end with '-'"},
		{`https://google-.com`, "cannot start or end with '-'"},
		{`https://google.com:abcd`, `Port "abcd" is not a number`},
		{`://google.com`, "missing its scheme"},
		// dates and times
		{`2026-13-45`, "Month 13 is out of range"},
		{`2026-02-30`, "Day 30 is out of range for month 2 (1-28)"},
		{`32.01.2026`, "Day 32 is out of range"},
		{`2026-1-5`, "two digits"},
		{`25:99`, "Hour 25 is out of range"},
		{`12:75`, "Minute 75 is out of range"},
		// the rest
		{`14x`, `Unknown unit "x"`},
		{`01.2.3`, "leading zero"},
		{`#12345`, "5 hex digits"},
		{`#12G456`, "non-hex"},
		{`hello`, "must be in quotes"},
		{`true`, "@true"},
	}

	for _, c := range cases {
		issues := Value("key", c.value)
		if len(issues) == 0 {
			t.Errorf("%q: expected an error, got none", c.value)
			continue
		}
		if !strings.Contains(issues[0].Message, c.want) {
			t.Errorf("%q: message %q does not contain %q", c.value, issues[0].Message, c.want)
		}
	}
}

// Values that merely look unusual must stay valid.
func TestStillValid(t *testing.T) {
	valid := []string{
		`::`, `::1`, `fe80::1%eth0`, `2001:db8::`, `::ffff:192.168.1.1`,
		`2001:db8:1234:5678:90ab:cdef:192.168.1.1/64`,
		`https://example.com`, `http://localhost:8080/path?x=1#top`, `https://127.0.0.1:8443/`,
		`https://user@example.com/`, `http://[::1]:8080/`,
		`1.2.3`, `10.20.30-beta.1`, `0.0.0`,
		`12:30`,
		// a real IPv6 address is not taken for a MAC: its groups are not all two characters
		`00:1A:2B:3C:4D::5`, `fe80::1`, `2001:db8::1`, `fe80::`,
	}
	for _, v := range valid {
		if issues := Value("key", v); len(issues) != 0 {
			t.Errorf("%q: expected no errors, got %+v", v, issues)
		}
	}
}

func TestKey(t *testing.T) {
	for _, ok := range []string{"name", "IPv4_Address", "_x", "a1", "X"} {
		if msg := Key(ok); msg != "" {
			t.Errorf("%q must be a valid name, got %q", ok, msg)
		}
	}
	bad := []struct{ name, want string }{
		{"42Name", "cannot start with a digit"},
		{"Name-With-Dash", "invalid character '-'"},
		{"Name.With.Dot", "invalid character '.'"},
		{`"Name"`, "without quotes"},
	}
	for _, c := range bad {
		if msg := Key(c.name); !strings.Contains(msg, c.want) {
			t.Errorf("%q: message %q does not contain %q", c.name, msg, c.want)
		}
	}
}
