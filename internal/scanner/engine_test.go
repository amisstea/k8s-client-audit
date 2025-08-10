package scanner

import (
	"context"
	"go/token"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/tools/go/packages"
)

// mockRule records that it was executed and returns a canned issue.
type mockRule struct{}

func (m mockRule) ID() string          { return "MOCK" }
func (m mockRule) Description() string { return "mock" }
func (m mockRule) Apply(ctx context.Context, _ *token.FileSet, _ *packages.Package) ([]Issue, error) {
	return []Issue{{RuleID: "MOCK", Title: "mock issue", Severity: SeverityWarning}}, nil
}

func TestEnginePipeline(t *testing.T) {
	// create a tiny module with one go file
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module tmp\n\ngo 1.21\n"), 0o644)
	_ = os.MkdirAll(filepath.Join(dir, "pkg"), 0o755)
	_ = os.WriteFile(filepath.Join(dir, "pkg", "x.go"), []byte("package pkg\nfunc X() {}\n"), 0o644)

	// build engine with one mock rule via registry replacement
	eng := NewEngine() // no rules
	// Directly run engine without rules (should return 0 issues) to validate loading works
	if issues, err := eng.Run(context.Background(), dir); err != nil {
		t.Fatalf("engine run error: %v", err)
	} else if len(issues) != 0 {
		t.Fatalf("expected 0 issues, got %d", len(issues))
	}

	// Now run with a real rule hook using the public Scanner abstraction
	s := New()
	// replacing internals for test: create a scanner with a mock rule
	s.engine = NewEngine(mockRule{})
	issues, err := s.ScanDirectory(context.Background(), dir)
	if err != nil {
		t.Fatalf("scan error: %v", err)
	}
	if len(issues) != 1 || issues[0].RuleID != "MOCK" {
		t.Fatalf("unexpected issues: %+v", issues)
	}
}
