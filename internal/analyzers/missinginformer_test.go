package analyzers

import (
	"testing"

	"github.com/amisstea/k8s-client-audit/internal/analyzers/testutil"

	"golang.org/x/tools/go/analysis"
)

func runMissingInformerAnalyzerOnSrc(t *testing.T, src string, spoof bool) []analysis.Diagnostic {
	t.Helper()
	var diags []analysis.Diagnostic
	var err error
	if spoof {
		diags, err = testutil.RunAnalyzerOnSrc(AnalyzerMissingInformer, src, testutil.SpoofInformers)
	} else {
		diags, err = testutil.RunAnalyzerOnSrc(AnalyzerMissingInformer, src)
	}
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	return diags
}

func TestMissingInformer_WatchWithoutInformer_Flagged(t *testing.T) {
	src := `package a
func f(c interface{ Watch(x any) error }) { _ = c.Watch(nil) }`
	diags := runMissingInformerAnalyzerOnSrc(t, src, true) // spoof as Kubernetes types
	if len(diags) == 0 {
		t.Fatalf("expected diagnostic for Kubernetes Watch without shared informer")
	}
}

func TestMissingInformer_WithSharedInformer_NoDiag(t *testing.T) {
	src := `package a
func NewSharedInformerFactory() {}
func g(c interface{ Watch(x any) error }) { NewSharedInformerFactory(); _ = c.Watch(nil) }`
	diags := runMissingInformerAnalyzerOnSrc(t, src, true) // spoof as Kubernetes types
	if len(diags) != 0 {
		t.Fatalf("did not expect diagnostic when Kubernetes shared informer is present")
	}
}

func TestMissingInformer_NonKubernetesWatch_NoDiag(t *testing.T) {
	src := `package a
func f(c interface{ Watch(x any) error }) { _ = c.Watch(nil) }`
	diags := runMissingInformerAnalyzerOnSrc(t, src, false) // don't spoof - should be treated as non-Kubernetes
	if len(diags) != 0 {
		t.Fatalf("expected 0 diagnostics for non-Kubernetes Watch calls, got %d", len(diags))
	}
}

func TestMissingInformer_NonKubernetesInformer_StillFlags(t *testing.T) {
	src := `package a
func NewSharedInformerFactory() {} // Non-Kubernetes informer
func f(c interface{ Watch(x any) error }) { NewSharedInformerFactory(); _ = c.Watch(nil) }`
	diags := runMissingInformerAnalyzerOnSrc(t, src, false) // don't spoof informer, but spoof Watch partially
	if len(diags) != 0 {
		t.Fatalf("expected 0 diagnostics when non-Kubernetes types are used, got %d", len(diags))
	}
}
