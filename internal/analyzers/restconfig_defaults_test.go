package analyzers

import (
	"testing"

	"github.com/amisstea/k8s-client-audit/internal/analyzers/testutil"

	"golang.org/x/tools/go/analysis"
)

// NOTE: The analyzer under test expects the *real* rest.Config type from k8s.io/client-go/rest,
// not a local stub. So we must import the real type and use it in the test source.

func runRestConfigDefaultsAnalyzerOnSrc(t *testing.T, src string) []analysis.Diagnostic {
	t.Helper()
	diags, err := testutil.RunAnalyzerOnSrc(AnalyzerRestConfigDefaults, src, testutil.SpoofRestConfig)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	return diags
}

func TestRestConfigDefaults_MissingFields_Flagged(t *testing.T) {
	src := `
package a

import "k8s.io/client-go/rest"

var _ = rest.Config{}
`
	diags := runRestConfigDefaultsAnalyzerOnSrc(t, src)
	if len(diags) == 0 {
		t.Fatalf("expected diagnostic for missing Timeout/UserAgent")
	}
}

func TestRestConfigDefaults_ZeroTimeout_EmptyUA_Flagged(t *testing.T) {
	src := `
package a

import "k8s.io/client-go/rest"

var _ = rest.Config{Timeout:0, UserAgent:""}
`
	diags := runRestConfigDefaultsAnalyzerOnSrc(t, src)
	if len(diags) == 0 {
		t.Fatalf("expected diagnostics for zero Timeout/empty UA")
	}
}

func TestRestConfigDefaults_WithValues_NoDiag(t *testing.T) {
	src := `
package a

import "k8s.io/client-go/rest"

var _ = rest.Config{Timeout:10, UserAgent:"my-agent"}
`
	diags := runRestConfigDefaultsAnalyzerOnSrc(t, src)
	if len(diags) != 0 {
		t.Fatalf("did not expect diagnostic when fields are set, got %d", len(diags))
	}
}
