package scanner

// Position indicates where in source code a finding occurred.
type Position struct {
	Filename string
	Line     int
	Column   int
}

// Issue is a single static analysis finding.
type Issue struct {
	RuleID      string
	Title       string
	Description string
	PackagePath string
	Position    Position
	Suggestion  string
}
