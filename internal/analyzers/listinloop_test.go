package analyzers

import (
	"testing"

	"cursor-experiment/internal/analyzers/testutil"

	"golang.org/x/tools/go/analysis"
)

func runListInLoopAnalyzerOnSrc(t *testing.T, src string) []analysis.Diagnostic {
	t.Helper()
	diags, err := testutil.RunAnalyzerOnSrc(AnalyzerListInLoop, src, nil)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	return diags
}

func TestListInLoop_FlagsCalls(t *testing.T) {
	src := `package a
type C interface{ List() error; Watch() error }
func f(c C){ for i:=0;i<2;i++{ _ = c.List(); _ = c.Watch() } }`
	diags := runListInLoopAnalyzerOnSrc(t, src)
	if len(diags) < 2 {
		t.Fatalf("expected diagnostics for List/Watch in loop, got %d", len(diags))
	}
}

func TestListInLoop_NoLoop_NoDiag(t *testing.T) {
	src := `package a
type C interface{ List() error }
func f(c C){ _ = c.List() }`
	diags := runListInLoopAnalyzerOnSrc(t, src)
	if len(diags) != 0 {
		t.Fatalf("expected 0 diagnostics, got %d", len(diags))
	}
}
