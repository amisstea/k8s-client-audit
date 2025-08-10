package scanner

// Severity indicates how severe a finding is.
type Severity string

const (
	SeverityInfo    Severity = "INFO"
	SeverityWarning Severity = "WARNING"
	SeverityError   Severity = "ERROR"
)

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
	Severity    Severity
	PackagePath string
	Position    Position
	Suggestion  string
}
