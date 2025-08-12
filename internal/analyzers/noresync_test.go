package analyzers

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"golang.org/x/tools/go/analysis"
)

func runNoResyncAnalyzerOnSrc(t *testing.T, src string) []analysis.Diagnostic {
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
	pass := &analysis.Pass{Analyzer: AnalyzerNoResync, Fset: fset, Files: files, TypesInfo: info, TypesSizes: types.SizesFor("gc", "amd64"), Report: func(d analysis.Diagnostic) { diags = append(diags, d) }, ResultOf: map[*analysis.Analyzer]interface{}{}}
	_, _ = AnalyzerNoResync.Run(pass)
	return diags
}

func TestNoResync_ZeroResync_Flagged(t *testing.T) {
	src := `package a
type Inf interface{}
func NewSharedIndexInformer(a,b,c any, resync int) Inf { return nil }
func f(){ _ = NewSharedIndexInformer(nil,nil,nil,0) }`
	diags := runNoResyncAnalyzerOnSrc(t, src)
	if len(diags) == 0 {
		t.Fatalf("expected diagnostic for zero resync period")
	}
}

func TestNoResync_PositiveResync_NoDiag(t *testing.T) {
	src := `package a
type Inf interface{}
func NewSharedIndexInformer(a,b,c any, resync int) Inf { return nil }
func f(){ _ = NewSharedIndexInformer(nil,nil,nil,10) }`
	diags := runNoResyncAnalyzerOnSrc(t, src)
	if len(diags) != 0 {
		t.Fatalf("did not expect diagnostic for positive resync period")
	}
}
