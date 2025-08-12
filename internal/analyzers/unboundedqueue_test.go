package analyzers

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"golang.org/x/tools/go/analysis"
)

func runUnboundedQueueAnalyzerOnSrc(t *testing.T, src string) []analysis.Diagnostic {
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
	pass := &analysis.Pass{Analyzer: AnalyzerUnboundedQueue, Fset: fset, Files: files, TypesInfo: info, TypesSizes: types.SizesFor("gc", "amd64"), Report: func(d analysis.Diagnostic) { diags = append(diags, d) }, ResultOf: map[*analysis.Analyzer]interface{}{}}
	_, _ = AnalyzerUnboundedQueue.Run(pass)
	return diags
}

func TestUnboundedQueue_NewWithoutRateLimiter_Flagged(t *testing.T) {
	src := `package a
type Q interface{}
func New() Q { return nil }
func f(){ _ = New() }`
	diags := runUnboundedQueueAnalyzerOnSrc(t, src)
	if len(diags) == 0 {
		t.Fatalf("expected diagnostic for New queue without ratelimiter")
	}
}

func TestUnboundedQueue_WithRateLimiter_NoDiag(t *testing.T) {
	src := `package a
type RL interface{}
func NewItemExponentialFailureRateLimiter() RL { return nil }
type Q interface{}
func NewNamed() Q { return nil }
func f(){ _ = NewItemExponentialFailureRateLimiter(); _ = NewNamed() }`
	diags := runUnboundedQueueAnalyzerOnSrc(t, src)
	if len(diags) != 0 {
		t.Fatalf("did not expect diagnostic when rate limiter is present")
	}
}
