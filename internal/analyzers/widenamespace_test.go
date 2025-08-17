package analyzers

import (
	"testing"

	"github.com/amisstea/k8s-client-audit/internal/analyzers/testutil"

	"golang.org/x/tools/go/analysis"
)

func runWideNamespaceAnalyzerOnSrc(t *testing.T, src string, spoof bool) []analysis.Diagnostic {
	t.Helper()
	var diags []analysis.Diagnostic
	var err error
	if spoof {
		diags, err = testutil.RunAnalyzerOnSrc(AnalyzerWideNamespace, src, testutil.SpoofCommonK8s)
	} else {
		diags, err = testutil.RunAnalyzerOnSrc(AnalyzerWideNamespace, src)
	}
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	return diags
}

func TestWideNamespace_InNamespaceEmpty_Flagged(t *testing.T) {
	src := `package a
type Opts struct{}
func InNamespace(ns string) Opts { return Opts{} }
type Client interface{ List(ctx any, obj any, opts ...Opts) error }
func f(c Client){ var o struct{}; _ = c.List(nil, &o, InNamespace("")) }`
	diags := runWideNamespaceAnalyzerOnSrc(t, src, true) // spoof as Kubernetes types
	if len(diags) == 0 {
		t.Fatalf("expected diagnostic for InNamespace(\"\")")
	}
}

func TestWideNamespace_TypedChain_PodsEmpty_Flagged(t *testing.T) {
	src := `package a
type PodsIFace interface{ List(ctx any) error }
type CoreV1 interface{ Pods(ns string) PodsIFace }
func f(c CoreV1){ _ = c.Pods("").List(nil) }`
	diags := runWideNamespaceAnalyzerOnSrc(t, src, true) // spoof as Kubernetes types
	if len(diags) == 0 {
		t.Fatalf("expected diagnostic for typed Pods(\"\").List")
	}
}

func TestWideNamespace_NonKubernetesClient_NoDiag(t *testing.T) {
	src := `package a
type DatabaseOpts struct{}
func InNamespace(ns string) DatabaseOpts { return DatabaseOpts{} }
type DatabaseClient interface{ List(opts ...DatabaseOpts) error }
func f(c DatabaseClient){ _ = c.List(InNamespace("")) }`
	diags := runWideNamespaceAnalyzerOnSrc(t, src, false) // don't spoof - should be treated as non-Kubernetes
	if len(diags) != 0 {
		t.Fatalf("expected 0 diagnostics for non-Kubernetes InNamespace calls, got %d", len(diags))
	}
}

func TestWideNamespace_InNamespaceWithValue_NoDiag(t *testing.T) {
	src := `package a
type Opts struct{}
func InNamespace(ns string) Opts { return Opts{} }
type Client interface{ List(ctx any, obj any, opts ...Opts) error }
func f(c Client){ var o struct{}; _ = c.List(nil, &o, InNamespace("default")) }`
	diags := runWideNamespaceAnalyzerOnSrc(t, src, true) // spoof as Kubernetes types
	if len(diags) != 0 {
		t.Fatalf("expected 0 diagnostics when InNamespace has a value, got %d", len(diags))
	}
}
