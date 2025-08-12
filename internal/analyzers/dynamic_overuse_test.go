package analyzers

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"golang.org/x/tools/go/analysis"
)

func runDynamicOveruseAnalyzerOnSrc(t *testing.T, src string) []analysis.Diagnostic {
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
	pass := &analysis.Pass{Analyzer: AnalyzerDynamicOveruse, Fset: fset, Files: files, TypesInfo: info, TypesSizes: types.SizesFor("gc", "amd64"), Report: func(d analysis.Diagnostic) { diags = append(diags, d) }, ResultOf: map[*analysis.Analyzer]interface{}{}}
	_, _ = AnalyzerDynamicOveruse.Run(pass)
	return diags
}

func TestDynamicOveruse_DynamicWithTypedPresent_Flagged(t *testing.T) {
	src := `package a
func NewForConfig(x any) any { return nil }
func NewDynamicClientForConfig(x any) any { return nil }
func f(){ _ = NewForConfig(nil); _ = NewDynamicClientForConfig(nil) }`
	diags := runDynamicOveruseAnalyzerOnSrc(t, src)
	if len(diags) == 0 {
		t.Fatalf("expected diagnostic when dynamic used but typed available")
	}
}

func TestDynamicOveruse_OnlyDynamic_NoDiag(t *testing.T) {
	src := `package a
func NewDynamicClientForConfig(x any) any { return nil }
func f(){ _ = NewDynamicClientForConfig(nil) }`
	diags := runDynamicOveruseAnalyzerOnSrc(t, src)
	if len(diags) != 0 {
		t.Fatalf("did not expect diagnostic when only dynamic exists")
	}
}
