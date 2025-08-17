package analyzers

import (
	"testing"

	"github.com/amisstea/k8s-client-audit/internal/analyzers/testutil"

	"golang.org/x/tools/go/analysis"
)

func runLargePagesAnalyzerOnSrc(t *testing.T, src string, spoof bool) []analysis.Diagnostic {
	t.Helper()
	var diags []analysis.Diagnostic
	var err error
	if spoof {
		diags, err = testutil.RunAnalyzerOnSrc(AnalyzerLargePageSizes, src, testutil.CommonK8sSpoof, testutil.SpoofListOptionsType)
	} else {
		diags, err = testutil.RunAnalyzerOnSrc(AnalyzerLargePageSizes, src)
	}
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	return diags
}

func TestLargePages_LimitLarge_Flagged(t *testing.T) {
	src := `package a
type ListOptions struct{ Limit int64 }
type IFace interface{ List(x any, opts ListOptions) error }
func f(c IFace){ _ = c.List(nil, ListOptions{Limit:2000}) }`
	diags := runLargePagesAnalyzerOnSrc(t, src, true) // spoof as Kubernetes types
	if len(diags) == 0 {
		t.Fatalf("expected diagnostic for large Kubernetes page size")
	}
}

func TestLargePages_LimitReasonable_NoDiag(t *testing.T) {
	src := `package a
type ListOptions struct{ Limit int64 }
type IFace interface{ List(x any, opts ListOptions) error }
func f(c IFace){ _ = c.List(nil, ListOptions{Limit:100}) }`
	diags := runLargePagesAnalyzerOnSrc(t, src, true) // spoof as Kubernetes types
	if len(diags) != 0 {
		t.Fatalf("did not expect diagnostic for reasonable page size")
	}
}

func TestLargePages_NonKubernetesListOptions_NoDiag(t *testing.T) {
	src := `package a
type ListOptions struct{ Limit int64 }
type DatabaseClient interface{ List(opts ListOptions) error }
func f(c DatabaseClient){ _ = c.List(ListOptions{Limit:2000}) }`
	diags := runLargePagesAnalyzerOnSrc(t, src, false) // don't spoof - should be treated as non-Kubernetes
	if len(diags) != 0 {
		t.Fatalf("expected 0 diagnostics for non-Kubernetes ListOptions, got %d", len(diags))
	}
}

func TestLargePages_NonKubernetesList_NoDiag(t *testing.T) {
	src := `package a
type ListOptions struct{ Limit int64 }
type APIClient interface{ List(opts ListOptions) error }
func f(c APIClient){ _ = c.List(ListOptions{Limit:2000}) }`
	diags := runLargePagesAnalyzerOnSrc(t, src, false) // don't spoof - should be treated as non-Kubernetes
	if len(diags) != 0 {
		t.Fatalf("expected 0 diagnostics for non-Kubernetes List calls, got %d", len(diags))
	}
}
