package analyzers

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"golang.org/x/tools/go/analysis"
)

func runMissingCtxAnalyzerOnSrc(t *testing.T, src string) []analysis.Diagnostic {
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
	pass := &analysis.Pass{Analyzer: AnalyzerMissingContext, Fset: fset, Files: files, TypesInfo: info, TypesSizes: types.SizesFor("gc", "amd64"), Report: func(d analysis.Diagnostic) { diags = append(diags, d) }, ResultOf: map[*analysis.Analyzer]interface{}{}}
	_, _ = AnalyzerMissingContext.Run(pass)
	return diags
}

func TestMissingContext_BackgroundFlagged(t *testing.T) {
	src := `package a
import "context"
type C interface{ Get(ctx any) }
func f(c C){ c.Get(context.Background()) }`
	diags := runMissingCtxAnalyzerOnSrc(t, src)
	if len(diags) == 0 {
		t.Fatalf("expected diagnostic for Background")
	}
}

func TestMissingContext_Propagated_NoDiag(t *testing.T) {
	src := `package a
type C interface{ Get(ctx any) }
func f(c C, ctx any){ c.Get(ctx) }`
	diags := runMissingCtxAnalyzerOnSrc(t, src)
	if len(diags) != 0 {
		t.Fatalf("expected 0 diagnostics, got %d", len(diags))
	}
}
