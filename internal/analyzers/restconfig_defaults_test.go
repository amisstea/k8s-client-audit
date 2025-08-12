package analyzers

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"runtime"
	"testing"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// NOTE: The analyzer under test expects the *real* rest.Config type from k8s.io/client-go/rest,
// not a local stub. So we must import the real type and use it in the test source.

func runRestConfigDefaultsAnalyzerOnSrc(t *testing.T, src string) []analysis.Diagnostic {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", src, parser.AllErrors)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	files := []*ast.File{f}
	info := &types.Info{
		Types:      map[ast.Expr]types.TypeAndValue{},
		Defs:       map[*ast.Ident]types.Object{},
		Uses:       map[*ast.Ident]types.Object{},
		Selections: map[*ast.SelectorExpr]*types.Selection{},
	}
	var conf types.Config
	_, _ = conf.Check("p", fset, files, info)
	// Fabricate named type for rest.Config and attach to composite literals so type check passes
	pkg := types.NewPackage("k8s.io/client-go/rest", "rest")
	named := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "Config", nil), types.NewStruct(nil, nil), nil)
	ast.Inspect(f, func(n ast.Node) bool {
		if cl, ok := n.(*ast.CompositeLit); ok {
			info.Types[cl] = types.TypeAndValue{Type: named}
		}
		return true
	})

	var diags []analysis.Diagnostic
	pass := &analysis.Pass{
		Analyzer:   AnalyzerRestConfigDefaults,
		Fset:       fset,
		Files:      files,
		TypesInfo:  info,
		TypesSizes: types.SizesFor("gc", runtime.GOARCH),
		Report:     func(d analysis.Diagnostic) { diags = append(diags, d) },
		ResultOf:   map[*analysis.Analyzer]interface{}{inspect.Analyzer: inspector.New(files)},
	}
	_, _ = AnalyzerRestConfigDefaults.Run(pass)
	return diags
}

func TestRestConfigDefaults_MissingFields_Flagged(t *testing.T) {
	src := `
package a

import "k8s.io/client-go/rest"

var _ = rest.Config{}
`
	diags := runRestConfigDefaultsAnalyzerOnSrc(t, src)
	if len(diags) == 0 {
		t.Fatalf("expected diagnostic for missing Timeout/UserAgent")
	}
}

func TestRestConfigDefaults_ZeroTimeout_EmptyUA_Flagged(t *testing.T) {
	src := `
package a

import "k8s.io/client-go/rest"

var _ = rest.Config{Timeout:0, UserAgent:""}
`
	diags := runRestConfigDefaultsAnalyzerOnSrc(t, src)
	if len(diags) == 0 {
		t.Fatalf("expected diagnostics for zero Timeout/empty UA")
	}
}

func TestRestConfigDefaults_WithValues_NoDiag(t *testing.T) {
	src := `
package a

import "k8s.io/client-go/rest"

var _ = rest.Config{Timeout:10, UserAgent:"my-agent"}
`
	diags := runRestConfigDefaultsAnalyzerOnSrc(t, src)
	if len(diags) != 0 {
		t.Fatalf("did not expect diagnostic when fields are set, got %d", len(diags))
	}
}
