package analyzers

import (
	"testing"

	"github.com/amisstea/k8s-client-audit/internal/analyzers/testutil"

	"golang.org/x/tools/go/analysis"
)

func runManualPollingAnalyzerOnSrc(t *testing.T, src string, spoof bool) []analysis.Diagnostic {
	t.Helper()
	var diags []analysis.Diagnostic
	var err error
	if spoof {
		diags, err = testutil.RunAnalyzerOnSrc(AnalyzerManualPolling, src, testutil.SpoofCommonK8s, testutil.SpoofCommonStdLib)
	} else {
		diags, err = testutil.RunAnalyzerOnSrc(AnalyzerManualPolling, src)
	}
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	return diags
}

func TestManualPolling_ListWithSleep_Flagged(t *testing.T) {
	src := `package a
import "time"
type Client interface{ List(ctx any, obj any) error }
func f(c Client){ for { var o struct{}; _ = c.List(nil, &o); time.Sleep(100) } }`
	diags := runManualPollingAnalyzerOnSrc(t, src, true) // spoof as Kubernetes/time types
	if len(diags) == 0 {
		t.Fatalf("expected diagnostic for manual polling with List + Sleep")
	}
}

func TestManualPolling_Watch_NoDiag(t *testing.T) {
	src := `package a
import "time"
type IFace interface{ Watch(x any) error }
func f(c IFace){ for { _ = c.Watch(nil); time.Sleep(100) } }`
	diags := runManualPollingAnalyzerOnSrc(t, src, false) // don't spoof - Watch shouldn't be flagged anyway
	if len(diags) != 0 {
		t.Fatalf("did not expect diagnostic when using Watch")
	}
}

func TestManualPolling_NonKubernetesClient_NoDiag(t *testing.T) {
	src := `package a
import "time"
type DatabaseClient interface{ List() []string }
func f(c DatabaseClient){ for { _ = c.List(); time.Sleep(100) } }`
	diags := runManualPollingAnalyzerOnSrc(t, src, false) // don't spoof - should be treated as non-Kubernetes
	if len(diags) != 0 {
		t.Fatalf("expected 0 diagnostics for non-Kubernetes client calls, got %d", len(diags))
	}
}
