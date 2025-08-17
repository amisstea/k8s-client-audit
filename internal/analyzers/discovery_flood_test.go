package analyzers

import (
	"testing"

	"github.com/amisstea/k8s-client-audit/internal/analyzers/testutil"

	"golang.org/x/tools/go/analysis"
)

func runDiscoveryFloodAnalyzerOnSrc(t *testing.T, src string) []analysis.Diagnostic {
	t.Helper()
	diags, err := testutil.RunAnalyzerOnSrc(AnalyzerDiscoveryFlood, src, testutil.CommonK8sSpoof)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	return diags
}

func TestDiscoveryFlood_RepeatedInLoop_Flagged(t *testing.T) {
	src := `package a
func NewDiscoveryClientForConfig(x any) any { return nil }
func f(){ for { _ = NewDiscoveryClientForConfig(nil) } }`
	diags := runDiscoveryFloodAnalyzerOnSrc(t, src)
	if len(diags) == 0 {
		t.Fatalf("expected diagnostic for discovery in loop")
	}
}

func TestDiscoveryFlood_OutsideLoop_NoDiag(t *testing.T) {
	src := `package a
func NewDiscoveryClientForConfig(x any) any { return nil }
func f(){ _ = NewDiscoveryClientForConfig(nil) }`
	diags := runDiscoveryFloodAnalyzerOnSrc(t, src)
	if len(diags) != 0 {
		t.Fatalf("did not expect diagnostic outside loop")
	}
}
