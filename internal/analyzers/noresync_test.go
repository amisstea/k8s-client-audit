package analyzers

import (
	"testing"

	"cursor-experiment/internal/analyzers/testutil"

	"golang.org/x/tools/go/analysis"
)

func runNoResyncAnalyzerOnSrc(t *testing.T, src string) []analysis.Diagnostic {
	t.Helper()
	diags, err := testutil.RunAnalyzerOnSrc(AnalyzerNoResync, src, nil)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	return diags
}

func TestNoResync_ZeroResync_Flagged(t *testing.T) {
	src := `package a
type Inf interface{}
func NewSharedIndexInformer(a,b,c any, resync int) Inf { return nil }
func f(){ _ = NewSharedIndexInformer(nil,nil,nil,0) }`
	diags := runNoResyncAnalyzerOnSrc(t, src)
	if len(diags) == 0 {
		t.Fatalf("expected diagnostic for zero resync period")
	}
}

func TestNoResync_PositiveResync_NoDiag(t *testing.T) {
	src := `package a
type Inf interface{}
func NewSharedIndexInformer(a,b,c any, resync int) Inf { return nil }
func f(){ _ = NewSharedIndexInformer(nil,nil,nil,10) }`
	diags := runNoResyncAnalyzerOnSrc(t, src)
	if len(diags) != 0 {
		t.Fatalf("did not expect diagnostic for positive resync period")
	}
}
