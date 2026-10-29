package parser

// Position is a place in the file. Numbering starts at ZERO (as LSP requires).
// Col is counted in bytes (UTF-8); the analyze package converts it to UTF-16.
type Position struct {
	Line int
	Col  int
}

// Range is a span of text from Start to End.
type Range struct {
	Start Position
	End   Position
}

// Node is any node of the tree: a section or a pair.
type Node interface {
	Span() Range
}

// Section is "Name:" or "Name from Parent:" together with everything inside it.
type Section struct {
	Name        string
	NameRange   Range
	Parent      string // for "Name from Parent:", otherwise ""
	ParentRange Range
	Range       Range // the whole section, children included
	Children    []Node
}

func (s *Section) Span() Range { return s.Range }

// Item is one element of an array.
type Item struct {
	Value string
	Range Range
}

// Pair is "key value" or an array "key[ ... ]".
type Pair struct {
	Key        string
	KeyRange   Range
	Value      string
	ValueRange Range
	Range      Range

	IsArray bool   // true for os[ ... ]
	Items   []Item // array elements
}

func (p *Pair) Span() Range { return p.Range }

// ParseError is a syntax error.
type ParseError struct {
	Range   Range
	Message string
}

// Document is the result of parsing a whole file.
type Document struct {
	Children []Node       // top level (YTL:, NetworkEntities:, ...)
	Errors   []ParseError // the parser does not stop at the first error
}
