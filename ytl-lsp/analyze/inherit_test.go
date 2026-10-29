package analyze

import (
	"strings"
	"testing"
)

const animal = "Animal:\n    name \"x\"\n    legs 4\n    tags[\n\"a\"\n]\n"

func TestStubBasic(t *testing.T) {
	text := animal + "Dog from Animal:\n\n" // line 7 is the new blank line
	e, ok := InheritedStub(text, 7, "    ")
	if !ok {
		t.Fatal("expected keys to be spawned")
	}
	want := "    name\n    legs\n    tags[]"
	if e.NewText != want || e.Line != 7 || e.StartCol != 0 || e.EndCol != 0 {
		t.Fatalf("got %+v, expected text %q on line 7", e, want)
	}
}

func TestStubReplacesAutoIndent(t *testing.T) {
	// the editor already put 4 spaces on the new line: they must be replaced
	text := animal + "Dog from Animal:\n    "
	e, ok := InheritedStub(text, 7, "    ")
	if !ok || e.EndCol != 4 || !strings.HasPrefix(e.NewText, "    name") {
		t.Fatalf("got %+v ok=%v", e, ok)
	}
}

func TestStubNestedHeader(t *testing.T) {
	text := "Animal:\n    name \"x\"\nNet:\n    Dog from Animal:\n        "
	e, ok := InheritedStub(text, 4, "    ")
	if !ok || e.NewText != "        name" {
		t.Fatalf("got %+v ok=%v", e, ok)
	}
}

func TestStubTabs(t *testing.T) {
	text := "Animal:\n\tname \"x\"\nDog from Animal:\n"
	e, ok := InheritedStub(text, 3, "\t")
	if !ok || e.NewText != "\tname" {
		t.Fatalf("got %+v ok=%v", e, ok)
	}
}

func TestStubAncestorsOldestFirst(t *testing.T) {
	text := "A:\n    a 1\nB from A:\n    b 1\nC from B:\n"
	e, ok := InheritedStub(text, 5, "    ")
	if !ok || e.NewText != "    a\n    b" {
		t.Fatalf("got %+v ok=%v", e, ok)
	}
}

func TestStubNotSpawned(t *testing.T) {
	cases := map[string]struct {
		text string
		line int
	}{
		"section already has content": {"Animal:\n    name \"x\"\nDog from Animal:\n\n    own 1\n", 3},
		"plain section, no from":      {"Animal:\n    name \"x\"\nDog:\n", 3},
		"unknown parent":              {"Dog from Animal:\n", 1},
		"parent has no keys":          {"Animal:\nDog from Animal:\n", 2},
		"new line is not blank":       {"Animal:\n    name \"x\"\nDog from Animal:\nabc", 3},
		"self inheritance":            {"dal from dal:\n", 1},
	}
	for name, c := range cases {
		if e, ok := InheritedStub(c.text, c.line, "    "); ok {
			t.Errorf("%s: nothing should be spawned, got %+v", name, e)
		}
	}
}

func TestStubCycleDoesNotHang(t *testing.T) {
	text := "A from B:\n    a 1\nB from A:\n    b 1\nC from A:\n"
	e, ok := InheritedStub(text, 5, "    ")
	if !ok || !strings.Contains(e.NewText, "a") || !strings.Contains(e.NewText, "b") {
		t.Fatalf("got %+v ok=%v", e, ok)
	}
}

func TestStubNestedSections(t *testing.T) {
	text := "Net:\n    ip 1\n    Hw:\n        mac 2\n    cidr 3\nDas from Net:\n"
	e, ok := InheritedStub(text, 6, "    ")
	want := "    ip\n    Hw:\n        mac\n    cidr"
	if !ok || e.NewText != want {
		t.Fatalf("got %+v ok=%v, expected text %q", e, ok, want)
	}
}

// The layout from the real test file: a repeated key and a section that is
// indented deeper than the key before it, but still belongs to Net.
func TestStubRepeatedKeyAndDeepSection(t *testing.T) {
	text := "Net:\n    ip 1\n    cidr 2\n\n    cidr 3\n\n        Hw:\n            mac 4\nDas from Net:\n"
	e, ok := InheritedStub(text, 9, "    ")
	want := "    ip\n    cidr\n    Hw:\n        mac"
	if !ok || e.NewText != want {
		t.Fatalf("got %+v ok=%v, expected text %q", e, ok, want)
	}
}

func TestStubMergesSectionsOfAncestors(t *testing.T) {
	text := "A:\n    S:\n        a 1\nB from A:\n    S:\n        b 1\nC from B:\n"
	e, ok := InheritedStub(text, 7, "    ")
	want := "    S:\n        a\n        b"
	if !ok || e.NewText != want {
		t.Fatalf("got %+v ok=%v, expected text %q", e, ok, want)
	}
}

func TestStubEmptyNestedSection(t *testing.T) {
	text := "A:\n    x 1\n    Empty:\nB from A:\n"
	e, ok := InheritedStub(text, 4, "    ")
	want := "    x\n    Empty:"
	if !ok || e.NewText != want {
		t.Fatalf("got %+v ok=%v, expected text %q", e, ok, want)
	}
}
