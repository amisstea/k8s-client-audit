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

func runIgnoring429AnalyzerOnSrc(t *testing.T, src string) []analysis.Diagnostic {
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
	pass := &analysis.Pass{Analyzer: AnalyzerIgnoring429, Fset: fset, Files: files, TypesInfo: info, TypesSizes: types.SizesFor("gc", "amd64"), Report: func(d analysis.Diagnostic) { diags = append(diags, d) }, ResultOf: map[*analysis.Analyzer]interface{}{insppass.Analyzer: inspector.New(files)}}
	_, _ = AnalyzerIgnoring429.Run(pass)
	return diags
}

func TestIgnoring429_NoBackoff_Flagged(t *testing.T) {
	src := `package a
import "net/http"
func f(code int){ if code == http.StatusTooManyRequests { /* retry immediately */ } }`
	diags := runIgnoring429AnalyzerOnSrc(t, src)
	if len(diags) == 0 {
		t.Fatalf("expected diagnostic for 429 without backoff")
	}
}

func TestIgnoring429_WithSleep_NoDiag(t *testing.T) {
	src := `package a
import "time"
const StatusTooManyRequests = 429
func f(code int){ if code == StatusTooManyRequests { time.Sleep(10) } }`
	diags := runIgnoring429AnalyzerOnSrc(t, src)
	if len(diags) != 0 {
		t.Fatalf("did not expect diagnostic when backoff present")
	}
}
