package analyzers

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"golang.org/x/tools/go/analysis"
)

func runRBACScopeAnalyzerOnSrc(t *testing.T, src string, which *analysis.Analyzer) []analysis.Diagnostic {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", src, 0)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	files := []*ast.File{f}
	info := &types.Info{Types: map[ast.Expr]types.TypeAndValue{}, Defs: map[*ast.Ident]types.Object{}, Uses: map[*ast.Ident]types.Object{}, Selections: map[*ast.SelectorExpr]*types.Selection{}}
	var conf types.Config
	_, _ = conf.Check("p", fset, files, info)
	var diags []analysis.Diagnostic
	pass := &analysis.Pass{Analyzer: which, Fset: fset, Files: files, TypesInfo: info, TypesSizes: types.SizesFor("gc", "amd64"), Report: func(d analysis.Diagnostic) { diags = append(diags, d) }, ResultOf: map[*analysis.Analyzer]interface{}{}}
	_, _ = which.Run(pass)
	return diags
}

func TestExcessiveClusterScope_Flagged(t *testing.T) {
	src := `package a
type ClusterRole struct{}
var _ = ClusterRole{}`
	diags := runRBACScopeAnalyzerOnSrc(t, src, AnalyzerExcessiveClusterScope)
	if len(diags) == 0 {
		t.Fatalf("expected diagnostic for cluster-scoped RBAC")
	}
}

func TestExcessiveClusterScope_NamespaceRole_NoDiag(t *testing.T) {
	src := `package a
type Role struct{}
var _ = Role{}`
	diags := runRBACScopeAnalyzerOnSrc(t, src, AnalyzerExcessiveClusterScope)
	if len(diags) != 0 {
		t.Fatalf("did not expect diagnostic for namespaced role")
	}
}

func TestWildcardVerbs_Flagged(t *testing.T) {
	src := `package a
type PolicyRule struct{ Verbs []string }
var _ = PolicyRule{Verbs: []string{"*"}}`
	diags := runRBACScopeAnalyzerOnSrc(t, src, AnalyzerWildcardVerbs)
	if len(diags) == 0 {
		t.Fatalf("expected diagnostic for wildcard verbs")
	}
}

func TestWildcardVerbs_Specific_NoDiag(t *testing.T) {
	src := `package a
type PolicyRule struct{ Verbs []string }
var _ = PolicyRule{Verbs: []string{"get","list"}}`
	diags := runRBACScopeAnalyzerOnSrc(t, src, AnalyzerWildcardVerbs)
	if len(diags) != 0 {
		t.Fatalf("did not expect diagnostic for specific verbs")
	}
}
