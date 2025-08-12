package analyzers

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"golang.org/x/tools/go/analysis"
)

func runRESTMapperNotCachedAnalyzerOnSrc(t *testing.T, src string) []analysis.Diagnostic {
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
	pass := &analysis.Pass{Analyzer: AnalyzerRESTMapperNotCached, Fset: fset, Files: files, TypesInfo: info, TypesSizes: types.SizesFor("gc", "amd64"), Report: func(d analysis.Diagnostic) { diags = append(diags, d) }, ResultOf: map[*analysis.Analyzer]interface{}{}}
	_, _ = AnalyzerRESTMapperNotCached.Run(pass)
	return diags
}

func TestRESTMapperNotCached_NoCache_Flagged(t *testing.T) {
	src := `package a
func NewDeferredDiscoveryRESTMapper(x any) any { return nil }
func f(){ _ = NewDeferredDiscoveryRESTMapper(nil) }`
	diags := runRESTMapperNotCachedAnalyzerOnSrc(t, src)
	if len(diags) == 0 {
		t.Fatalf("expected diagnostic for RESTMapper without caching")
	}
}

func TestRESTMapperNotCached_WithCache_NoDiag(t *testing.T) {
	src := `package a
func NewDeferredDiscoveryRESTMapper(x any) any { return nil }
func NewShortcutExpander(x any, y any) any { return nil }
func f(){ _ = NewShortcutExpander(NewDeferredDiscoveryRESTMapper(nil), nil) }`
	diags := runRESTMapperNotCachedAnalyzerOnSrc(t, src)
	if len(diags) != 0 {
		t.Fatalf("did not expect diagnostic when cached wrapper present")
	}
}
