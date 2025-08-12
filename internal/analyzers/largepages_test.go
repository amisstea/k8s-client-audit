package analyzers

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"golang.org/x/tools/go/analysis"
)

func runLargePagesAnalyzerOnSrc(t *testing.T, src string) []analysis.Diagnostic {
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
	pass := &analysis.Pass{Analyzer: AnalyzerLargePageSizes, Fset: fset, Files: files, TypesInfo: info, TypesSizes: types.SizesFor("gc", "amd64"), Report: func(d analysis.Diagnostic) { diags = append(diags, d) }, ResultOf: map[*analysis.Analyzer]interface{}{}}
	_, _ = AnalyzerLargePageSizes.Run(pass)
	return diags
}

func TestLargePages_LimitLarge_Flagged(t *testing.T) {
	src := `package a
type ListOptions struct{ Limit int64 }
type IFace interface{ List(x any, opts ListOptions) error }
func f(c IFace){ _ = c.List(nil, ListOptions{Limit:2000}) }`
	diags := runLargePagesAnalyzerOnSrc(t, src)
	if len(diags) == 0 {
		t.Fatalf("expected diagnostic for large page size")
	}
}

func TestLargePages_LimitReasonable_NoDiag(t *testing.T) {
	src := `package a
type ListOptions struct{ Limit int64 }
type IFace interface{ List(x any, opts ListOptions) error }
func f(c IFace){ _ = c.List(nil, ListOptions{Limit:100}) }`
	diags := runLargePagesAnalyzerOnSrc(t, src)
	if len(diags) != 0 {
		t.Fatalf("did not expect diagnostic for reasonable page size")
	}
}
