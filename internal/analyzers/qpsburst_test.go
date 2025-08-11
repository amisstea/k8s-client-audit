package analyzers

import (
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"golang.org/x/tools/go/analysis"
)

func runAnalyzerOnSrc(t *testing.T, src string) []analysis.Diagnostic {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "a.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	files := []*ast.File{f}
	conf := types.Config{Importer: nil}
	info := &types.Info{
		Types:      map[ast.Expr]types.TypeAndValue{},
		Defs:       map[*ast.Ident]types.Object{},
		Uses:       map[*ast.Ident]types.Object{},
		Selections: map[*ast.SelectorExpr]*types.Selection{},
	}
	// type-check minimally; we are using synthetic types in tests
	_, _ = conf.Check("p", fset, files, info)
	var diags []analysis.Diagnostic
	pass := &analysis.Pass{
		Analyzer:   AnalyzerQPSBurst,
		Fset:       fset,
		Files:      files,
		Pkg:        nil,
		TypesInfo:  info,
		TypesSizes: types.SizesFor("gc", build.Default.GOARCH),
		Report:     func(d analysis.Diagnostic) { diags = append(diags, d) },
		ResultOf:   map[*analysis.Analyzer]interface{}{},
	}
	_, _ = AnalyzerQPSBurst.Run(pass)
	return diags
}

func TestAnalyzer_ConfigLiteralMissingOrBad(t *testing.T) {
	src := `package a
type Config struct{ QPS float32; Burst int }
var _ = Config{}
var _ = Config{QPS: 0}
var _ = Config{Burst: 0}
var _ = Config{QPS: 200000.0, Burst: 1}
`
	diags := runAnalyzerOnSrc(t, src)
	if len(diags) < 3 {
		t.Fatalf("expected at least 3 diagnostics, got %d", len(diags))
	}
}

func TestAnalyzer_AssignmentsBad(t *testing.T) {
	src := `package a
type Config struct{ QPS float32; Burst int }
func f(){ var cfg Config; cfg.QPS = 0; cfg.Burst = 0; cfg.QPS = 200000.0; cfg.Burst = 1000000 }
`
	diags := runAnalyzerOnSrc(t, src)
	if len(diags) < 3 {
		t.Fatalf("expected multiple diagnostics, got %d", len(diags))
	}
}

func TestAnalyzer_GoodValues_NoDiag(t *testing.T) {
	src := `package a
type Config struct{ QPS float32; Burst int }
func f(){ var cfg Config; cfg.QPS = 30; cfg.Burst = 100 }
`
	diags := runAnalyzerOnSrc(t, src)
	if len(diags) != 0 {
		t.Fatalf("expected 0 diagnostics, got %d", len(diags))
	}
}
