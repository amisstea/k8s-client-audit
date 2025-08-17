package analyzers

import (
	"testing"

	"github.com/amisstea/k8s-client-audit/internal/analyzers/testutil"

	"golang.org/x/tools/go/analysis"
)

func runRBACScopeAnalyzerOnSrc(t *testing.T, src string, which *analysis.Analyzer, spoof bool) []analysis.Diagnostic {
	t.Helper()
	var diags []analysis.Diagnostic
	var err error
	if spoof {
		diags, err = testutil.RunAnalyzerOnSrc(which, src, testutil.SpoofRBACTypes)
	} else {
		diags, err = testutil.RunAnalyzerOnSrc(which, src)
	}
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	return diags
}

func TestExcessiveClusterScope_Flagged(t *testing.T) {
	src := `package a
type ClusterRole struct{}
var _ = ClusterRole{}`
	diags := runRBACScopeAnalyzerOnSrc(t, src, AnalyzerExcessiveClusterScope, true) // spoof as Kubernetes RBAC types
	if len(diags) == 0 {
		t.Fatalf("expected diagnostic for Kubernetes cluster-scoped RBAC")
	}
}

func TestExcessiveClusterScope_NamespaceRole_NoDiag(t *testing.T) {
	src := `package a
type Role struct{}
var _ = Role{}`
	diags := runRBACScopeAnalyzerOnSrc(t, src, AnalyzerExcessiveClusterScope, true) // spoof as Kubernetes RBAC types
	if len(diags) != 0 {
		t.Fatalf("did not expect diagnostic for namespaced role")
	}
}

func TestExcessiveClusterScope_NonKubernetesRole_NoDiag(t *testing.T) {
	src := `package a
type ClusterRole struct{}
var _ = ClusterRole{}`
	diags := runRBACScopeAnalyzerOnSrc(t, src, AnalyzerExcessiveClusterScope, false) // don't spoof - should be treated as non-Kubernetes
	if len(diags) != 0 {
		t.Fatalf("expected 0 diagnostics for non-Kubernetes ClusterRole, got %d", len(diags))
	}
}

func TestWildcardVerbs_Flagged(t *testing.T) {
	src := `package a
type PolicyRule struct{ Verbs []string }
var _ = PolicyRule{Verbs: []string{"*"}}`
	diags := runRBACScopeAnalyzerOnSrc(t, src, AnalyzerWildcardVerbs, true) // spoof as Kubernetes RBAC types
	if len(diags) == 0 {
		t.Fatalf("expected diagnostic for Kubernetes wildcard verbs")
	}
}

func TestWildcardVerbs_Specific_NoDiag(t *testing.T) {
	src := `package a
type PolicyRule struct{ Verbs []string }
var _ = PolicyRule{Verbs: []string{"get","list"}}`
	diags := runRBACScopeAnalyzerOnSrc(t, src, AnalyzerWildcardVerbs, true) // spoof as Kubernetes RBAC types
	if len(diags) != 0 {
		t.Fatalf("did not expect diagnostic for specific verbs")
	}
}

func TestWildcardVerbs_NonKubernetesRule_NoDiag(t *testing.T) {
	src := `package a
type PolicyRule struct{ Verbs []string }
var _ = PolicyRule{Verbs: []string{"*"}}`
	diags := runRBACScopeAnalyzerOnSrc(t, src, AnalyzerWildcardVerbs, false) // don't spoof - should be treated as non-Kubernetes
	if len(diags) != 0 {
		t.Fatalf("expected 0 diagnostics for non-Kubernetes PolicyRule, got %d", len(diags))
	}
}
