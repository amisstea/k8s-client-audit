package analyzers

import (
	"testing"

	"github.com/amisstea/k8s-client-audit/internal/analyzers/testutil"

	"golang.org/x/tools/go/analysis"
)

func runUnstructuredEverywhereAnalyzerOnSrc(t *testing.T, src string, spoof bool) []analysis.Diagnostic {
	t.Helper()
	var diags []analysis.Diagnostic
	var err error
	if spoof {
		diags, err = testutil.RunAnalyzerOnSrc(AnalyzerUnstructuredEverywhere, src, testutil.SpoofUnstructuredTypes)
	} else {
		diags, err = testutil.RunAnalyzerOnSrc(AnalyzerUnstructuredEverywhere, src)
	}
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	return diags
}

func TestUnstructuredEverywhere_ManyUsages_Flagged(t *testing.T) {
	src := `package a
type Unstructured struct{}
var _ = Unstructured{}
func a1(){ _ = Unstructured{} }
func a2(){ _ = Unstructured{} }`
	diags := runUnstructuredEverywhereAnalyzerOnSrc(t, src, true) // spoof as Kubernetes Unstructured types
	if len(diags) == 0 {
		t.Fatalf("expected diagnostic for heavy Kubernetes unstructured usage")
	}
}

func TestUnstructuredEverywhere_SparseUsage_NoDiag(t *testing.T) {
	src := `package a
type Unstructured struct{}
func a1(){ _ = Unstructured{} }`
	diags := runUnstructuredEverywhereAnalyzerOnSrc(t, src, true) // spoof as Kubernetes Unstructured types
	if len(diags) != 0 {
		t.Fatalf("did not expect diagnostic for light Kubernetes unstructured usage")
	}
}

func TestUnstructuredEverywhere_NonKubernetesUnstructured_NoDiag(t *testing.T) {
	src := `package a
type Unstructured struct{}
var _ = Unstructured{}
func a1(){ _ = Unstructured{} }
func a2(){ _ = Unstructured{} }
func a3(){ _ = Unstructured{} }`
	diags := runUnstructuredEverywhereAnalyzerOnSrc(t, src, false) // don't spoof - should be treated as non-Kubernetes
	if len(diags) != 0 {
		t.Fatalf("expected 0 diagnostics for non-Kubernetes Unstructured usage, got %d", len(diags))
	}
}

func TestUnstructuredEverywhere_MixedUsage_OnlyKubernetesUsage_Flagged(t *testing.T) {
	src := `package a
type Unstructured struct{}
type SomeOtherType struct{}
var _ = Unstructured{}
var _ = SomeOtherType{}
func a1(){ _ = Unstructured{} }
func a2(){ _ = Unstructured{} }`
	diags := runUnstructuredEverywhereAnalyzerOnSrc(t, src, true) // spoof only Unstructured as Kubernetes
	if len(diags) == 0 {
		t.Fatalf("expected diagnostic for heavy Kubernetes unstructured usage in mixed code")
	}
}
