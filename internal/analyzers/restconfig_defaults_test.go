package analyzers

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"golang.org/x/tools/go/analysis"
)

func runRestConfigDefaultsAnalyzerOnSrc(t *testing.T, src string) []analysis.Diagnostic {
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
	pass := &analysis.Pass{Analyzer: AnalyzerRestConfigDefaults, Fset: fset, Files: files, TypesInfo: info, TypesSizes: types.SizesFor("gc", "amd64"), Report: func(d analysis.Diagnostic) { diags = append(diags, d) }, ResultOf: map[*analysis.Analyzer]interface{}{}}
	_, _ = AnalyzerRestConfigDefaults.Run(pass)
	return diags
}

func TestRestConfigDefaults_MissingFields_Flagged(t *testing.T) {
	src := `package a
type Config struct{ Timeout int; UserAgent string }
var _ = Config{}`
	diags := runRestConfigDefaultsAnalyzerOnSrc(t, src)
	if len(diags) == 0 {
		t.Fatalf("expected diagnostic for missing Timeout/UserAgent")
	}
}

func TestRestConfigDefaults_ZeroTimeout_EmptyUA_Flagged(t *testing.T) {
	src := `package a
type Config struct{ Timeout int; UserAgent string }
var _ = Config{Timeout:0, UserAgent:""}`
	diags := runRestConfigDefaultsAnalyzerOnSrc(t, src)
	if len(diags) < 1 {
		t.Fatalf("expected diagnostics for zero Timeout/empty UA")
	}
}

func TestRestConfigDefaults_WithValues_NoDiag(t *testing.T) {
	src := `package a
type Config struct{ Timeout int; UserAgent string }
var _ = Config{Timeout:10, UserAgent:"my-agent"}`
	diags := runRestConfigDefaultsAnalyzerOnSrc(t, src)
	if len(diags) != 0 {
		t.Fatalf("did not expect diagnostic when fields are set")
	}
}
