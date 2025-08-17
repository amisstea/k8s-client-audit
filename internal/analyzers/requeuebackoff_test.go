package analyzers

import (
	"testing"

	"github.com/amisstea/k8s-client-audit/internal/analyzers/testutil"

	"golang.org/x/tools/go/analysis"
)

func runRequeueBackoffAnalyzerOnSrc(t *testing.T, src string, spoofControllerRuntimeTypes bool) []analysis.Diagnostic {
	t.Helper()
	var diags []analysis.Diagnostic
	var err error
	if spoofControllerRuntimeTypes {
		diags, err = testutil.RunAnalyzerOnSrc(AnalyzerRequeueBackoff, src, testutil.SpoofControllerRuntimeResult)
	} else {
		diags, err = testutil.RunAnalyzerOnSrc(AnalyzerRequeueBackoff, src)
	}
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	return diags
}

func TestRequeueBackoff_RequeueWithoutAfter_Flagged(t *testing.T) {
	src := `package a
type Result struct{ Requeue bool; RequeueAfter int }
func f() (Result, error) { return Result{Requeue:true}, nil }`
	diags := runRequeueBackoffAnalyzerOnSrc(t, src, true) // spoof as controller-runtime types
	if len(diags) == 0 {
		t.Fatalf("expected diagnostic for controller-runtime requeue without backoff")
	}
}

func TestRequeueBackoff_WithRequeueAfter_NoDiag(t *testing.T) {
	src := `package a
type Result struct{ Requeue bool; RequeueAfter int }
func f() (Result, error) { return Result{Requeue:true, RequeueAfter:5}, nil }`
	diags := runRequeueBackoffAnalyzerOnSrc(t, src, true) // spoof as controller-runtime types
	if len(diags) != 0 {
		t.Fatalf("did not expect diagnostic when RequeueAfter is set")
	}
}

func TestRequeueBackoff_NonControllerRuntimeResult_NoDiag(t *testing.T) {
	src := `package a
type Result struct{ Requeue bool; RequeueAfter int }
func f() (Result, error) { return Result{Requeue:true}, nil }`
	diags := runRequeueBackoffAnalyzerOnSrc(t, src, false) // don't spoof - should be treated as non-controller-runtime
	if len(diags) != 0 {
		t.Fatalf("expected 0 diagnostics for non-controller-runtime Result types, got %d", len(diags))
	}
}

func TestRequeueBackoff_NoRequeue_NoDiag(t *testing.T) {
	src := `package a
type Result struct{ Requeue bool; RequeueAfter int }
func f() (Result, error) { return Result{}, nil }`
	diags := runRequeueBackoffAnalyzerOnSrc(t, src, true) // spoof as controller-runtime types
	if len(diags) != 0 {
		t.Fatalf("expected 0 diagnostics when Requeue is not set, got %d", len(diags))
	}
}
