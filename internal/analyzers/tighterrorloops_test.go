package analyzers

import (
	"testing"

	"github.com/amisstea/k8s-client-audit/internal/analyzers/testutil"

	"golang.org/x/tools/go/analysis"
)

func runTightAnalyzerOnSrc(t *testing.T, src string, spoof bool) []analysis.Diagnostic {
	t.Helper()
	var diags []analysis.Diagnostic
	var err error
	if spoof {
		diags, err = testutil.RunAnalyzerOnSrc(AnalyzerTightErrorLoops, src, testutil.CommonK8sSpoof, testutil.CommonStdLibSpoof)
	} else {
		diags, err = testutil.RunAnalyzerOnSrc(AnalyzerTightErrorLoops, src)
	}
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	return diags
}

func TestTightErrorLoops_WithAPICall_NoSleep(t *testing.T) {
	src := `package a
type Client interface{ List(ctx any) error }
func f(c Client){ var err error; for { if err != nil { _ = c.List(nil) } else { break } } }`
	diags := runTightAnalyzerOnSrc(t, src, true) // spoof as Kubernetes types
	if len(diags) == 0 {
		t.Fatalf("expected diagnostic for tight loop without sleep around API call")
	}
}

func TestTightErrorLoops_NoAPICall_NoDiag(t *testing.T) {
	src := `package a
func f(){ var err error; for { if err != nil { _ = 1+1 } else { break } } }`
	diags := runTightAnalyzerOnSrc(t, src, false) // no spoofing needed
	if len(diags) != 0 {
		t.Fatalf("did not expect diagnostic without API call, got %d", len(diags))
	}
}

func TestTightErrorLoops_WithSleep_NoDiag(t *testing.T) {
	src := `package a
import "time"
type Client interface{ List(ctx any) error }
func f(c Client){ var err error; for { if err != nil { _ = c.List(nil); time.Sleep(100) } else { break } } }`
	diags := runTightAnalyzerOnSrc(t, src, true) // spoof as Kubernetes/time types
	if len(diags) != 0 {
		t.Fatalf("did not expect diagnostic when sleep present, got %d", len(diags))
	}
}

func TestTightErrorLoops_NonKubernetesClient_NoDiag(t *testing.T) {
	src := `package a
type DatabaseClient interface{ List() []string }
func f(c DatabaseClient){ var err error; for { if err != nil { _ = c.List() } else { break } } }`
	diags := runTightAnalyzerOnSrc(t, src, false) // don't spoof - should be treated as non-Kubernetes
	if len(diags) != 0 {
		t.Fatalf("expected 0 diagnostics for non-Kubernetes client calls, got %d", len(diags))
	}
}
