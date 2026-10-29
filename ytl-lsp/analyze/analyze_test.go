package analyze

import (
	"strings"
	"testing"

	"ytl-lsp/parser"
)

func TestNoErrors(t *testing.T) {
	d := Analyze("A:\n    x 1\n    s \"привет\"\n    os[1, 2]\n")
	if len(d) != 0 {
		t.Fatalf("expected no errors: %+v", d)
	}
}

func TestInvalidValue(t *testing.T) {
	d := Analyze("ip 999.1.1.1\n")
	if len(d) != 1 {
		t.Fatalf("expected 1 error, got %+v", d)
	}
	want := parser.Range{
		Start: parser.Position{Line: 0, Col: 3},
		End:   parser.Position{Line: 0, Col: 12},
	}
	if d[0].Range != want {
		t.Fatalf("position %+v, expected %+v", d[0].Range, want)
	}
}

func TestParserErrorIncluded(t *testing.T) {
	d := Analyze("os[\n1\n") // the array is never closed
	if len(d) != 1 {
		t.Fatalf("expected 1 parser error, got %+v", d)
	}
}

func TestKeyWithoutValueIsAllowed(t *testing.T) {
	d := Analyze("A:\n    lonely\n    other\n")
	if len(d) != 0 {
		t.Fatalf("keys without values must not be errors: %+v", d)
	}
}

func TestUTF16Columns(t *testing.T) {
	// In bytes the element 999.1.1.1 starts at column 15, in characters at 11.
	d := Analyze("os[\"прив\", 999.1.1.1]\n")
	if len(d) != 1 {
		t.Fatalf("expected 1 error, got %+v", d)
	}
	if d[0].Range.Start.Col != 11 || d[0].Range.End.Col != 20 {
		t.Fatalf("columns %+v, expected 11..20", d[0].Range)
	}
}

func TestInheritanceValid(t *testing.T) {
	d := Analyze("Animal:\n    x 1\nDog from Animal:\n    y 1\nPuppy from Dog:\n    z 1\n")
	if len(d) != 0 {
		t.Fatalf("expected no errors: %+v", d)
	}
}

func TestInheritanceFromItself(t *testing.T) {
	d := Analyze("dal from dal:\n    x 1\n")
	if len(d) != 1 || !strings.Contains(d[0].Message, "itself") {
		t.Fatalf("expected a self-inheritance error, got %+v", d)
	}
	want := parser.Range{
		Start: parser.Position{Line: 0, Col: 9},
		End:   parser.Position{Line: 0, Col: 12},
	}
	if d[0].Range != want {
		t.Fatalf("position %+v, expected %+v", d[0].Range, want)
	}
}

func TestInheritanceMissingParent(t *testing.T) {
	d := Analyze("Dog from Animal:\n    x 1\n")
	if len(d) != 1 || !strings.Contains(d[0].Message, "not defined") {
		t.Fatalf("expected a missing-parent error, got %+v", d)
	}
	if d[0].Range.Start.Col != 9 || d[0].Range.End.Col != 15 {
		t.Fatalf("position %+v, expected columns 9..15", d[0].Range)
	}
}

func TestInheritanceCycle(t *testing.T) {
	d := Analyze("A from B:\n    x 1\nB from A:\n    y 1\n")
	if len(d) != 2 {
		t.Fatalf("expected an error on each member of the cycle, got %+v", d)
	}
	for _, x := range d {
		if !strings.Contains(x.Message, "Circular") {
			t.Fatalf("unexpected message: %q", x.Message)
		}
	}
}

func TestInheritanceChainIntoCycle(t *testing.T) {
	// C only leads into the cycle A <-> B; only A and B must be reported.
	d := Analyze("C from A:\n    x 1\nA from B:\n    x 1\nB from A:\n    x 1\n")
	if len(d) != 2 {
		t.Fatalf("expected 2 errors (A and B), got %+v", d)
	}
}

func TestKeyNames(t *testing.T) {
	cases := []struct{ text, want string }{
		{"42Name \"x\"\n", "cannot start with a digit"},
		{"Name-With-Dash \"x\"\n", "invalid character '-'"},
		{"Name.With.Dot \"x\"\n", "invalid character '.'"},
		{"\"Name\" \"x\"\n", "without quotes"},
		{"A:\n    9lives \"x\"\n", "cannot start with a digit"},
		{"1sec:\n    x 1\n", "cannot start with a digit"}, // a section name
	}
	for _, c := range cases {
		d := Analyze(c.text)
		if len(d) != 1 || !strings.Contains(d[0].Message, c.want) {
			t.Errorf("%q: expected one error containing %q, got %+v", c.text, c.want, d)
		}
	}
}

func TestKeyNamesValid(t *testing.T) {
	d := Analyze("project_name \"x\"\nIPv4_Address 127.0.0.1\n_private 1\nOS2[1, 2]\n")
	if len(d) != 0 {
		t.Fatalf("expected no errors: %+v", d)
	}
}
