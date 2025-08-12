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

func runListInLoopAnalyzerOnSrc(t *testing.T, src string) []analysis.Diagnostic {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", src, 0)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	files := []*ast.File{f}
	info := &types.Info{Types: map[ast.Expr]types.TypeAndValue{}, Defs: map[*ast.Ident]types.Object{}, Uses: map[*ast.Ident]types.Object{}, Selections: map[*ast.SelectorExpr]*types.Selection{}}
	var conf types.Config
	_, err = conf.Check("p", fset, files, info)
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	var diags []analysis.Diagnostic
	pass := &analysis.Pass{Analyzer: AnalyzerListInLoop, Fset: fset, Files: files, TypesInfo: info, TypesSizes: types.SizesFor("gc", "amd64"), Report: func(d analysis.Diagnostic) { diags = append(diags, d) }, ResultOf: map[*analysis.Analyzer]interface{}{insppass.Analyzer: inspector.New(files)}}
	_, err = AnalyzerListInLoop.Run(pass)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	return diags
}

func TestListInLoop_FlagsCalls(t *testing.T) {
	src := `package a
type C interface{ List() error; Watch() error }
func f(c C){ for i:=0;i<2;i++{ _ = c.List(); _ = c.Watch() } }`
	diags := runListInLoopAnalyzerOnSrc(t, src)
	if len(diags) < 2 {
		t.Fatalf("expected diagnostics for List/Watch in loop, got %d", len(diags))
	}
}

func TestListInLoop_NoLoop_NoDiag(t *testing.T) {
	src := `package a
type C interface{ List() error }
func f(c C){ _ = c.List() }`
	diags := runListInLoopAnalyzerOnSrc(t, src)
	if len(diags) != 0 {
		t.Fatalf("expected 0 diagnostics, got %d", len(diags))
	}
}
