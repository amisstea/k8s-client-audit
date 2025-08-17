package analyzers

import (
	"testing"

	"github.com/amisstea/k8s-client-audit/internal/analyzers/testutil"

	"golang.org/x/tools/go/analysis"
)

func runMissingCtxAnalyzerOnSrc(t *testing.T, src string, spoof bool) []analysis.Diagnostic {
	t.Helper()
	var diags []analysis.Diagnostic
	var err error
	if spoof {
		diags, err = testutil.RunAnalyzerOnSrc(AnalyzerMissingContext, src, testutil.SpoofCommonK8s)
	} else {
		diags, err = testutil.RunAnalyzerOnSrc(AnalyzerMissingContext, src)
	}
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	return diags
}

func TestMissingContext_BackgroundFlagged(t *testing.T) {
	src := `package a
import "context"
type C interface{ Get(ctx any) }
func f(c C){ c.Get(context.Background()) }`
	diags := runMissingCtxAnalyzerOnSrc(t, src, true) // spoof as Kubernetes types
	if len(diags) == 0 {
		t.Fatalf("expected diagnostic for Background")
	}
}

func TestMissingContext_Propagated_NoDiag(t *testing.T) {
	src := `package a
type C interface{ Get(ctx any) }
func f(c C, ctx any){ c.Get(ctx) }`
	diags := runMissingCtxAnalyzerOnSrc(t, src, true) // spoof as Kubernetes types
	if len(diags) != 0 {
		t.Fatalf("expected 0 diagnostics, got %d", len(diags))
	}
}

func TestMissingContext_GitHubClient_NoDiag(t *testing.T) {
	src := `package a
import "context"
type GitHubAppsService struct{}
func (g *GitHubAppsService) Get(ctx context.Context, slug string) (interface{}, interface{}, error) { return nil, nil, nil }
type GitHubClient struct{ Apps *GitHubAppsService }
func f(client *GitHubClient){ client.Apps.Get(context.Background(), "") }`
	diags := runMissingCtxAnalyzerOnSrc(t, src, false) // don't spoof - should be treated as non-Kubernetes
	if len(diags) != 0 {
		t.Fatalf("expected 0 diagnostics for GitHub client calls, got %d", len(diags))
	}
}

func TestMissingContext_GenericClient_NoDiag(t *testing.T) {
	src := `package a
import "context"
type HTTPClient interface{ Get(ctx context.Context, url string) error }
func f(client HTTPClient){ client.Get(context.Background(), "https://api.github.com") }`
	diags := runMissingCtxAnalyzerOnSrc(t, src, false) // don't spoof - should be treated as non-Kubernetes
	if len(diags) != 0 {
		t.Fatalf("expected 0 diagnostics for non-Kubernetes client calls, got %d", len(diags))
	}
}

func TestMissingContext_KubernetesClient_Flagged(t *testing.T) {
	src := `package a
import "context"
type KubernetesClient interface{ Get(ctx context.Context, name string) error }
func f(client KubernetesClient){ client.Get(context.Background(), "my-pod") }`
	diags := runMissingCtxAnalyzerOnSrc(t, src, true) // spoof as Kubernetes types
	if len(diags) == 0 {
		t.Fatalf("expected diagnostic for Kubernetes client with Background context")
	}
}
