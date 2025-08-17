package analyzers

import (
	"testing"

	"github.com/amisstea/k8s-client-audit/internal/analyzers/testutil"

	"golang.org/x/tools/go/analysis"
)

func runAnalyzerOnSrc(t *testing.T, src string) []analysis.Diagnostic {
	t.Helper()
	diags, err := testutil.RunAnalyzerOnSrc(AnalyzerQPSBurst, src)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	return diags
}

func TestAnalyzer_ConfigLiteralMissingOrBad(t *testing.T) {
	src := `package a
type Config struct{ QPS float32; Burst int }
var _ = Config{}
var _ = Config{QPS: 0}
var _ = Config{Burst: 0}
var _ = Config{QPS: 200000.0, Burst: 1}
`
	diags := runAnalyzerOnSrc(t, src)
	if len(diags) < 3 {
		t.Fatalf("expected at least 3 diagnostics, got %d", len(diags))
	}
}

func TestAnalyzer_AssignmentsBad(t *testing.T) {
	src := `package a
type Config struct{ QPS float32; Burst int }
func f(){ var cfg Config; cfg.QPS = 0; cfg.Burst = 0; cfg.QPS = 200000.0; cfg.Burst = 1000000 }
`
	diags := runAnalyzerOnSrc(t, src)
	if len(diags) < 3 {
		t.Fatalf("expected multiple diagnostics, got %d", len(diags))
	}
}

func TestAnalyzer_GoodValues_NoDiag(t *testing.T) {
	src := `package a
type Config struct{ QPS float32; Burst int }
func f(){ var cfg Config; cfg.QPS = 30; cfg.Burst = 100 }
`
	diags := runAnalyzerOnSrc(t, src)
	if len(diags) != 0 {
		t.Fatalf("expected 0 diagnostics, got %d", len(diags))
	}
}
