package analyzers

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"golang.org/x/tools/go/analysis"
)

func runRequeueBackoffAnalyzerOnSrc(t *testing.T, src string) []analysis.Diagnostic {
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
	pass := &analysis.Pass{Analyzer: AnalyzerRequeueBackoff, Fset: fset, Files: files, TypesInfo: info, TypesSizes: types.SizesFor("gc", "amd64"), Report: func(d analysis.Diagnostic) { diags = append(diags, d) }, ResultOf: map[*analysis.Analyzer]interface{}{}}
	_, _ = AnalyzerRequeueBackoff.Run(pass)
	return diags
}

func TestRequeueBackoff_RequeueWithoutAfter_Flagged(t *testing.T) {
	src := `package a
type Result struct{ Requeue bool; RequeueAfter int }
func f() (Result, error) { return Result{Requeue:true}, nil }`
	diags := runRequeueBackoffAnalyzerOnSrc(t, src)
	if len(diags) == 0 {
		t.Fatalf("expected diagnostic for requeue without backoff")
	}
}

func TestRequeueBackoff_WithRequeueAfter_NoDiag(t *testing.T) {
	src := `package a
type Result struct{ Requeue bool; RequeueAfter int }
func f() (Result, error) { return Result{Requeue:true, RequeueAfter:5}, nil }`
	diags := runRequeueBackoffAnalyzerOnSrc(t, src)
	if len(diags) != 0 {
		t.Fatalf("did not expect diagnostic when RequeueAfter is set")
	}
}
