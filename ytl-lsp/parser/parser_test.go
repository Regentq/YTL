package parser

import "testing"

func TestNested(t *testing.T) {
	src := "Net:\n" +
		"    CIDR 10.0.0.0/24\n" +
		"    Hardware:\n" +
		"        mac 00:1A:2B:3C:4D:5E\n" +
		"    port 80\n" +
		"Other:\n" +
		"    x 1\n"

	doc := Parse(src)
	if len(doc.Errors) != 0 {
		t.Fatalf("unexpected errors: %+v", doc.Errors)
	}
	if len(doc.Children) != 2 {
		t.Fatalf("expected 2 top-level sections, got %d", len(doc.Children))
	}

	net := doc.Children[0].(*Section)
	if net.Name != "Net" || len(net.Children) != 3 {
		t.Fatalf("Net: name=%q children=%d (expected 3)", net.Name, len(net.Children))
	}

	hw := net.Children[1].(*Section)
	if hw.Name != "Hardware" || len(hw.Children) != 1 {
		t.Fatalf("Hardware parsed wrongly: %+v", hw)
	}

	// "port 80" must go back to Net and not stay inside Hardware
	if p := net.Children[2].(*Pair); p.Key != "port" || p.Value != "80" {
		t.Fatalf("port parsed wrongly: %+v", p)
	}
}

func TestPositions(t *testing.T) {
	doc := Parse("A:\n    key  some value\n")
	sec := doc.Children[0].(*Section)
	p := sec.Children[0].(*Pair)

	if p.Key != "key" || p.Value != "some value" {
		t.Fatalf("key/value: %q / %q", p.Key, p.Value)
	}
	if p.KeyRange.Start != (Position{1, 4}) || p.KeyRange.End != (Position{1, 7}) {
		t.Fatalf("KeyRange: %+v", p.KeyRange)
	}
	if p.ValueRange.Start != (Position{1, 9}) || p.ValueRange.End != (Position{1, 19}) {
		t.Fatalf("ValueRange: %+v", p.ValueRange)
	}
	if sec.Range.End != (Position{1, 19}) {
		t.Fatalf("a section must end where its last line ends: %+v", sec.Range)
	}
}

func TestValueWithSpacesAndList(t *testing.T) {
	doc := Parse("A:\n    name \"YTL (Template Language)\"\n    uptime 14.5s, 14d, 14ms\n")
	sec := doc.Children[0].(*Section)
	if v := sec.Children[0].(*Pair).Value; v != "\"YTL (Template Language)\"" {
		t.Fatalf("value with parentheses: %q", v)
	}
	if v := sec.Children[1].(*Pair).Value; v != "14.5s, 14d, 14ms" {
		t.Fatalf("list: %q", v)
	}
}

func TestKeyWithoutValue(t *testing.T) {
	// a key without a value is allowed, it is not an error
	doc := Parse("A:\n    lonely\n    next 1\n")
	if len(doc.Errors) != 0 {
		t.Fatalf("unexpected errors: %+v", doc.Errors)
	}
	sec := doc.Children[0].(*Section)
	p := sec.Children[0].(*Pair)
	if p.Key != "lonely" || p.Value != "" {
		t.Fatalf("parsed wrongly: %+v", p)
	}
	if len(sec.Children) != 2 {
		t.Fatalf("expected 2 children, got %d", len(sec.Children))
	}
}

func TestBlankLinesAndCRLF(t *testing.T) {
	doc := Parse("A:\r\n\r\n    x 1\r\n\r\n")
	if len(doc.Errors) != 0 {
		t.Fatalf("errors: %+v", doc.Errors)
	}
	p := doc.Children[0].(*Section).Children[0].(*Pair)
	if p.Value != "1" {
		t.Fatalf("\\r must not end up in the value: %q", p.Value)
	}
}

func TestComments(t *testing.T) {
	doc := Parse("A:\n    x 1 // comment\n    site https://example.com/a\n    s \"a // b\"\n")
	sec := doc.Children[0].(*Section)
	if v := sec.Children[0].(*Pair).Value; v != "1" {
		t.Fatalf("comment must be stripped: %q", v)
	}
	if v := sec.Children[1].(*Pair).Value; v != "https://example.com/a" {
		t.Fatalf("URL must stay intact: %q", v)
	}
	if v := sec.Children[2].(*Pair).Value; v != "\"a // b\"" {
		t.Fatalf("// inside a string is not a comment: %q", v)
	}
}

func TestInheritance(t *testing.T) {
	doc := Parse("Dog from Animal:\n    x 1\n")
	sec := doc.Children[0].(*Section)
	if sec.Name != "Dog" || sec.Parent != "Animal" {
		t.Fatalf("name/parent: %q / %q", sec.Name, sec.Parent)
	}
	if sec.ParentRange.Start != (Position{0, 9}) || sec.ParentRange.End != (Position{0, 15}) {
		t.Fatalf("ParentRange: %+v", sec.ParentRange)
	}
}

func TestMultilineString(t *testing.T) {
	doc := Parse("A:\n    bio \"\"\"\nline one\nline two\n\"\"\"\n    next 1\n")
	if len(doc.Errors) != 0 {
		t.Fatalf("errors: %+v", doc.Errors)
	}
	sec := doc.Children[0].(*Section)
	if len(sec.Children) != 2 {
		t.Fatalf("expected 2 children (bio and next), got %d", len(sec.Children))
	}
}

func TestInlineArray(t *testing.T) {
	doc := Parse("os[1.0, 100, \"a, b\"]\n")
	if len(doc.Errors) != 0 {
		t.Fatalf("errors: %+v", doc.Errors)
	}
	p := doc.Children[0].(*Pair)
	if !p.IsArray || p.Key != "os" || len(p.Items) != 3 {
		t.Fatalf("parsed wrongly: %+v", p)
	}
	if p.Items[2].Value != `"a, b"` {
		t.Fatalf("a comma inside quotes must not split: %q", p.Items[2].Value)
	}
	if p.Items[1].Range.Start != (Position{0, 8}) || p.Items[1].Range.End != (Position{0, 11}) {
		t.Fatalf("item position: %+v", p.Items[1].Range)
	}
}

func TestMultilineArray(t *testing.T) {
	doc := Parse("os[\n1.0 // comment\n100\n]\nx 1\n")
	if len(doc.Errors) != 0 {
		t.Fatalf("errors: %+v", doc.Errors)
	}
	if len(doc.Children) != 2 {
		t.Fatalf("expected 2 nodes (array and x), got %d", len(doc.Children))
	}
	arr := doc.Children[0].(*Pair)
	if len(arr.Items) != 2 || arr.Items[0].Value != "1.0" || arr.Items[1].Value != "100" {
		t.Fatalf("items: %+v", arr.Items)
	}
}

func TestUnclosedArray(t *testing.T) {
	doc := Parse("os[\n1\n")
	if len(doc.Errors) != 1 {
		t.Fatalf("expected 1 error, got %+v", doc.Errors)
	}
}

func TestSectionDeeperThanPair(t *testing.T) {
	// a section indented deeper than the previous pair is fine: no errors expected
	doc := Parse("Net:\n    CIDR \"x\"\n        Hw:\n            a \"y\"\n")
	if len(doc.Errors) != 0 {
		t.Fatalf("errors: %+v", doc.Errors)
	}
	net := doc.Children[0].(*Section)
	if len(net.Children) != 2 {
		t.Fatalf("Net should have 2 children, got %d", len(net.Children))
	}
	if hw := net.Children[1].(*Section); len(hw.Children) != 1 {
		t.Fatalf("Hw should have 1 child, got %d", len(hw.Children))
	}
}

func TestEnumIsPair(t *testing.T) {
	doc := Parse("enum Color\n")
	p := doc.Children[0].(*Pair)
	if p.Key != "enum" || p.Value != "Color" {
		t.Fatalf("enum: %+v", p)
	}
}
