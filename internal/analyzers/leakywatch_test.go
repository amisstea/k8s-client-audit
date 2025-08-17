package analyzers

import (
	"testing"

	"github.com/amisstea/k8s-client-audit/internal/analyzers/testutil"

	"golang.org/x/tools/go/analysis"
)

func runLeakyWatchAnalyzerOnSrc(t *testing.T, src string, spoof bool) []analysis.Diagnostic {
	t.Helper()
	var diags []analysis.Diagnostic
	var err error
	if spoof {
		diags, err = testutil.RunAnalyzerOnSrc(AnalyzerLeakyWatch, src, testutil.CommonK8sSpoof)
	} else {
		diags, err = testutil.RunAnalyzerOnSrc(AnalyzerLeakyWatch, src)
	}
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	return diags
}

func TestLeakyWatch_NoStop_Flagged(t *testing.T) {
	src := `package a
type W interface{ ResultChan() chan int; Stop() }
func f(w W){ ch := w.ResultChan(); _ = ch }`
	diags := runLeakyWatchAnalyzerOnSrc(t, src, true) // spoof as Kubernetes types
	if len(diags) == 0 {
		t.Fatalf("expected diagnostic for missing Stop/Cancel on watch")
	}
}

func TestLeakyWatch_WithStop_NoDiag(t *testing.T) {
	src := `package a
type W interface{ ResultChan() chan int; Stop() }
func f(w W){ ch := w.ResultChan(); _ = ch; w.Stop() }`
	diags := runLeakyWatchAnalyzerOnSrc(t, src, true) // spoof as Kubernetes types
	if len(diags) != 0 {
		t.Fatalf("did not expect diagnostic when Stop is called")
	}
}

func TestLeakyWatch_NonKubernetesClient_NoDiag(t *testing.T) {
	src := `package a
type EventChannel interface{ ResultChan() chan string }
func f(ec EventChannel){ ch := ec.ResultChan(); _ = ch }`
	diags := runLeakyWatchAnalyzerOnSrc(t, src, false) // don't spoof - should be treated as non-Kubernetes
	if len(diags) != 0 {
		t.Fatalf("expected 0 diagnostics for non-Kubernetes ResultChan calls, got %d", len(diags))
	}
}
