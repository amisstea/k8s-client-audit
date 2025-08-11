package analyzers

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"golang.org/x/tools/go/analysis"
)

func runWideNamespaceAnalyzerOnSrc(t *testing.T, src string) []analysis.Diagnostic {
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
	pass := &analysis.Pass{Analyzer: AnalyzerWideNamespace, Fset: fset, Files: files, TypesInfo: info, TypesSizes: types.SizesFor("gc", "amd64"), Report: func(d analysis.Diagnostic) { diags = append(diags, d) }, ResultOf: map[*analysis.Analyzer]interface{}{}}
	_, _ = AnalyzerWideNamespace.Run(pass)
	return diags
}

func TestWideNamespace_InNamespaceEmpty_Flagged(t *testing.T) {
	src := `package a
type Opts struct{}
func InNamespace(ns string) Opts { return Opts{} }
type Client interface{ List(ctx any, obj any, opts ...Opts) error }
func f(c Client){ var o struct{}; _ = c.List(nil, &o, InNamespace("")) }`
	diags := runWideNamespaceAnalyzerOnSrc(t, src)
	if len(diags) == 0 {
		t.Fatalf("expected diagnostic for InNamespace(\"\")")
	}
}

func TestWideNamespace_TypedChain_PodsEmpty_Flagged(t *testing.T) {
	src := `package a
type PodsIFace interface{ List(ctx any) error }
type CoreV1 interface{ Pods(ns string) PodsIFace }
func f(c CoreV1){ _ = c.Pods("").List(nil) }`
	diags := runWideNamespaceAnalyzerOnSrc(t, src)
	if len(diags) == 0 {
		t.Fatalf("expected diagnostic for typed Pods(\"\").List")
	}
}
