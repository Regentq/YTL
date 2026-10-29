package analyze

import "testing"

const base = "Animal:\n    x \"1\"\nDog from Animal:\n    y \"1\"\n"

func labels(items []Completion) []string {
	var out []string
	for _, c := range items {
		out = append(out, c.Label)
	}
	return out
}

func TestCompleteFrom(t *testing.T) {
	cases := []struct {
		last string
		char int
	}{
		{"Dae ", 4},
		{"Dae f", 5},
		{"Dae fr", 6},
	}
	for _, c := range cases {
		got := Complete(base+c.last, 4, c.char)
		if len(got) != 1 || got[0].Label != "from" {
			t.Errorf("%q: expected [from], got %v", c.last, labels(got))
		}
	}
}

func TestCompleteNothing(t *testing.T) {
	cases := []struct {
		text string
		line int
		char int
	}{
		{base + "Dae x", 4, 5},             // "x" cannot become "from"
		{base + "Dae", 4, 3},               // still typing the first word
		{base + "    y fr", 4, 8},          // indented lines are not headers
		{base + "enum fr", 4, 7},           // enum is a normal pair
		{base + "Dae from Animal ", 4, 16}, // the header is already complete
		{base + "Dae // fr", 4, 9},         // inside a comment
	}
	for _, c := range cases {
		if got := Complete(c.text, c.line, c.char); len(got) != 0 {
			t.Errorf("%q: expected nothing, got %v", c.text, labels(got))
		}
	}
}

func TestCompleteParents(t *testing.T) {
	for _, last := range []string{"Dae from ", "Dae from Da"} {
		got := Complete(base+last, 4, len(last))
		l := labels(got)
		if len(l) != 2 || l[0] != "Animal" || l[1] != "Dog" {
			t.Errorf("%q: expected [Animal Dog], got %v", last, l)
		}
	}
}

func TestCompleteParentsShowInheritance(t *testing.T) {
	got := Complete(base+"Dae from ", 4, 9)
	if got[1].Detail != "extends Animal" {
		t.Errorf("Dog should be shown as 'extends Animal', got %q", got[1].Detail)
	}
}

func TestCompleteParentsNoSelfAndNoCycles(t *testing.T) {
	// Animal may not inherit from itself, nor from Dog (Dog already extends Animal).
	got := Complete(base+"Animal from ", 4, 12)
	if len(got) != 0 {
		t.Errorf("expected no candidates, got %v", labels(got))
	}
}

func TestFromUTF16(t *testing.T) {
	// "Пёс" is 3 characters (3 UTF-16 units) but 6 bytes.
	if got := fromUTF16("Пёс fr", 3); got != 6 {
		t.Errorf("Cyrillic: expected byte offset 6, got %d", got)
	}
	// An emoji is 1 character but takes 2 UTF-16 units and 4 bytes.
	if got := fromUTF16("a😀b", 3); got != 5 {
		t.Errorf("emoji: expected byte offset 5, got %d", got)
	}
	// A column past the end of the line is clamped to the line length.
	if got := fromUTF16("abc", 10); got != 3 {
		t.Errorf("past the end: expected 3, got %d", got)
	}
}
