package analyzers

import (
	"testing"

	"cursor-experiment/internal/analyzers/testutil"

	"golang.org/x/tools/go/analysis"
)

func runIgnoring429AnalyzerOnSrc(t *testing.T, src string) []analysis.Diagnostic {
	t.Helper()
	diags, err := testutil.RunAnalyzerOnSrc(AnalyzerIgnoring429, src, nil)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	return diags
}

func TestIgnoring429_NoBackoff_Flagged(t *testing.T) {
	src := `package a
import "net/http"
func f(code int){ if code == http.StatusTooManyRequests { /* retry immediately */ } }`
	diags := runIgnoring429AnalyzerOnSrc(t, src)
	if len(diags) == 0 {
		t.Fatalf("expected diagnostic for 429 without backoff")
	}
}

func TestIgnoring429_WithSleep_NoDiag(t *testing.T) {
	src := `package a
import "time"
const StatusTooManyRequests = 429
func f(code int){ if code == StatusTooManyRequests { time.Sleep(10) } }`
	diags := runIgnoring429AnalyzerOnSrc(t, src)
	if len(diags) != 0 {
		t.Fatalf("did not expect diagnostic when backoff present")
	}
}
