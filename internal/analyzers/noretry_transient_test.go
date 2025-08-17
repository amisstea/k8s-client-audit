package analyzers

import (
	"testing"

	"github.com/amisstea/k8s-client-audit/internal/analyzers/testutil"

	"golang.org/x/tools/go/analysis"
)

func runNoRetryTransientAnalyzerOnSrc(t *testing.T, src string) []analysis.Diagnostic {
	t.Helper()
	diags, err := testutil.RunAnalyzerOnSrc(AnalyzerNoRetryTransient, src)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	return diags
}

func TestNoRetryTransient_TransientWithoutRetry_Flagged(t *testing.T) {
	src := `package a
import "k8s.io/client-go/kubernetes"
func f(err any){ if Timeout { /* no retry */ } }`
	diags := runNoRetryTransientAnalyzerOnSrc(t, src)
	if len(diags) == 0 {
		t.Fatalf("expected diagnostic for Kubernetes-related transient error without retry")
	}
}

func TestNoRetryTransient_WithRetry_NoDiag(t *testing.T) {
	src := `package a
import "sigs.k8s.io/controller-runtime/pkg/client"
func Backoff(){}
func f(err any){ if Temporary { Backoff() } }`
	diags := runNoRetryTransientAnalyzerOnSrc(t, src)
	if len(diags) != 0 {
		t.Fatalf("did not expect diagnostic when retry present")
	}
}

func TestNoRetryTransient_NonKubernetesCode_NoDiag(t *testing.T) {
	src := `package a
func f(err any){ if Timeout { /* no retry */ } }`
	diags := runNoRetryTransientAnalyzerOnSrc(t, src)
	if len(diags) != 0 {
		t.Fatalf("expected 0 diagnostics for non-Kubernetes code, got %d", len(diags))
	}
}

func TestNoRetryTransient_NoTransientError_NoDiag(t *testing.T) {
	src := `package a
import "k8s.io/client-go/kubernetes"
func f(err any){ if SomeOtherError { /* no retry needed */ } }`
	diags := runNoRetryTransientAnalyzerOnSrc(t, src)
	if len(diags) != 0 {
		t.Fatalf("expected 0 diagnostics when no transient error mentioned, got %d", len(diags))
	}
}
