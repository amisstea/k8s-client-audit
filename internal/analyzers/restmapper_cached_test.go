package analyzers

import (
	"testing"

	"github.com/amisstea/k8s-client-audit/internal/analyzers/testutil"

	"golang.org/x/tools/go/analysis"
)

func runRESTMapperNotCachedAnalyzerOnSrc(t *testing.T, src string) []analysis.Diagnostic {
	t.Helper()
	diags, err := testutil.RunAnalyzerOnSrc(AnalyzerRESTMapperNotCached, src, testutil.SpoofRestMapper)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	return diags
}

func TestRESTMapperNotCached_NoCache_Flagged(t *testing.T) {
	src := `package a
func NewDeferredDiscoveryRESTMapper(x any) any { return nil }
func f(){ _ = NewDeferredDiscoveryRESTMapper(nil) }`
	diags := runRESTMapperNotCachedAnalyzerOnSrc(t, src)
	if len(diags) == 0 {
		t.Fatalf("expected diagnostic for RESTMapper without caching")
	}
}

func TestRESTMapperNotCached_WithCache_NoDiag(t *testing.T) {
	src := `package a
func NewDeferredDiscoveryRESTMapper(x any) any { return nil }
func NewShortcutExpander(x any, y any) any { return nil }
func f(){ _ = NewShortcutExpander(NewDeferredDiscoveryRESTMapper(nil), nil) }`
	diags := runRESTMapperNotCachedAnalyzerOnSrc(t, src)
	if len(diags) != 0 {
		t.Fatalf("did not expect diagnostic when cached wrapper present")
	}
}
