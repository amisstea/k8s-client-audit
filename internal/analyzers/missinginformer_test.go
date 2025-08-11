package analyzers

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"golang.org/x/tools/go/analysis"
)

func runMissingInformerAnalyzerOnSrc(t *testing.T, src string) []analysis.Diagnostic {
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
	pass := &analysis.Pass{Analyzer: AnalyzerMissingInformer, Fset: fset, Files: files, TypesInfo: info, TypesSizes: types.SizesFor("gc", "amd64"), Report: func(d analysis.Diagnostic) { diags = append(diags, d) }, ResultOf: map[*analysis.Analyzer]interface{}{}}
	_, _ = AnalyzerMissingInformer.Run(pass)
	return diags
}

func TestMissingInformer_WatchWithoutInformer_Flagged(t *testing.T) {
	src := `package a
func f(c interface{ Watch(x any) error }) { _ = c.Watch(nil) }`
	diags := runMissingInformerAnalyzerOnSrc(t, src)
	if len(diags) == 0 {
		t.Fatalf("expected diagnostic for Watch without shared informer")
	}
}

func TestMissingInformer_WithSharedInformer_NoDiag(t *testing.T) {
	src := `package a
func NewSharedInformerFactory() {}
func g(c interface{ Watch(x any) error }) { NewSharedInformerFactory(); _ = c.Watch(nil) }`
	diags := runMissingInformerAnalyzerOnSrc(t, src)
	if len(diags) != 0 {
		t.Fatalf("did not expect diagnostic when shared informer is present")
	}
}
