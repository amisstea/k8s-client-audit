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

func runUnstructuredEverywhereAnalyzerOnSrc(t *testing.T, src string) []analysis.Diagnostic {
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
	pass := &analysis.Pass{Analyzer: AnalyzerUnstructuredEverywhere, Fset: fset, Files: files, TypesInfo: info, TypesSizes: types.SizesFor("gc", "amd64"), Report: func(d analysis.Diagnostic) { diags = append(diags, d) }, ResultOf: map[*analysis.Analyzer]interface{}{insppass.Analyzer: inspector.New(files)}}
	_, _ = AnalyzerUnstructuredEverywhere.Run(pass)
	return diags
}

func TestUnstructuredEverywhere_ManyUsages_Flagged(t *testing.T) {
	src := `package a
type Unstructured struct{}
var _ = Unstructured{}
func a1(){ _ = Unstructured{} }
func a2(){ _ = Unstructured{} }`
	diags := runUnstructuredEverywhereAnalyzerOnSrc(t, src)
	if len(diags) == 0 {
		t.Fatalf("expected diagnostic for heavy unstructured usage")
	}
}

func TestUnstructuredEverywhere_SparseUsage_NoDiag(t *testing.T) {
	src := `package a
type Unstructured struct{}
func a1(){ _ = Unstructured{} }`
	diags := runUnstructuredEverywhereAnalyzerOnSrc(t, src)
	if len(diags) != 0 {
		t.Fatalf("did not expect diagnostic for light unstructured usage")
	}
}
