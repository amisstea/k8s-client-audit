package analyzers

import (
	"testing"

	"github.com/amisstea/k8s-client-audit/internal/analyzers/testutil"

	"golang.org/x/tools/go/analysis"
)

func runDynamicOveruseAnalyzerOnSrc(t *testing.T, src string) []analysis.Diagnostic {
	t.Helper()
	diags, err := testutil.RunAnalyzerOnSrc(AnalyzerDynamicOveruse, src, testutil.CommonK8sSpoof)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	return diags
}

func TestDynamicOveruse_DynamicWithTypedPresent_Flagged(t *testing.T) {
	src := `package a
func NewForConfig(x any) any { return nil }
func NewDynamicClientForConfig(x any) any { return nil }
func f(){ _ = NewForConfig(nil); _ = NewDynamicClientForConfig(nil) }`
	diags := runDynamicOveruseAnalyzerOnSrc(t, src)
	if len(diags) == 0 {
		t.Fatalf("expected diagnostic when dynamic used but typed available")
	}
}

func TestDynamicOveruse_OnlyDynamic_NoDiag(t *testing.T) {
	src := `package a
func NewDynamicClientForConfig(x any) any { return nil }
func f(){ _ = NewDynamicClientForConfig(nil) }`
	diags := runDynamicOveruseAnalyzerOnSrc(t, src)
	if len(diags) != 0 {
		t.Fatalf("did not expect diagnostic when only dynamic exists")
	}
}
