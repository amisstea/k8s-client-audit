package analyzers

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"golang.org/x/tools/go/analysis"
	insppass "golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

func runRBACScopeAnalyzerOnSrc(t *testing.T, src string, which *analysis.Analyzer, spoofKubernetesTypes bool) []analysis.Diagnostic {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", src, 0)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	files := []*ast.File{f}
	info := &types.Info{Types: map[ast.Expr]types.TypeAndValue{}, Defs: map[*ast.Ident]types.Object{}, Uses: map[*ast.Ident]types.Object{}, Selections: map[*ast.SelectorExpr]*types.Selection{}}
	var conf types.Config
	_, err = conf.Check("p", fset, files, info)
	if err != nil {
		// Expected for test files with incomplete type information
	}

	// Optionally spoof type info to mark types as coming from Kubernetes RBAC packages
	if spoofKubernetesTypes {
		pkgRBAC := types.NewPackage("k8s.io/api/rbac/v1", "v1")

		// Find type declarations and mark them as Kubernetes RBAC types
		ast.Inspect(f, func(n ast.Node) bool {
			if ts, ok := n.(*ast.TypeSpec); ok {
				name := ts.Name.Name
				if name == "ClusterRole" || name == "ClusterRoleBinding" || name == "PolicyRule" || name == "Rule" {
					// Create a named type for Kubernetes RBAC
					rbacType := types.NewNamed(types.NewTypeName(token.NoPos, pkgRBAC, name, nil), types.NewStruct(nil, nil), nil)
					info.Defs[ts.Name] = rbacType.Obj()
				}
			}
			return true
		})

		// Find composite literals and associate them with the Kubernetes RBAC types
		ast.Inspect(f, func(n ast.Node) bool {
			if cl, ok := n.(*ast.CompositeLit); ok {
				if id, ok := cl.Type.(*ast.Ident); ok {
					name := id.Name
					if name == "ClusterRole" || name == "ClusterRoleBinding" || name == "PolicyRule" || name == "Rule" {
						// Create the Kubernetes RBAC type
						rbacType := types.NewNamed(types.NewTypeName(token.NoPos, pkgRBAC, name, nil), types.NewStruct(nil, nil), nil)
						info.Types[cl] = types.TypeAndValue{Type: rbacType}
					}
				}
			}
			return true
		})
	}

	var diags []analysis.Diagnostic
	pass := &analysis.Pass{Analyzer: which, Fset: fset, Files: files, TypesInfo: info, TypesSizes: types.SizesFor("gc", "amd64"), Report: func(d analysis.Diagnostic) { diags = append(diags, d) }, ResultOf: map[*analysis.Analyzer]interface{}{insppass.Analyzer: inspector.New(files)}}
	_, _ = which.Run(pass)
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
