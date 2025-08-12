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
	// Spoof workqueue and ratelimiter symbols to proper packages so analyzer type checks pass
	pkgWQ := types.NewPackage("k8s.io/client-go/util/workqueue", "workqueue")
	pkgWQR := types.NewPackage("k8s.io/client-go/util/workqueue", "workqueue")
	ast.Inspect(f, func(n ast.Node) bool {
		if ce, ok := n.(*ast.CallExpr); ok {
			switch fun := ce.Fun.(type) {
			case *ast.Ident:
				// do not map bare identifiers; only selector-based calls to workqueue are recognized
			case *ast.SelectorExpr:
				if fun.Sel != nil {
					switch fun.Sel.Name {
					case "New", "NewNamed":
						info.Uses[fun.Sel] = types.NewFunc(token.NoPos, pkgWQ, fun.Sel.Name, nil)
					case "NewItemExponentialFailureRateLimiter", "NewItemFastSlowRateLimiter", "NewMaxOfRateLimiter", "NewWithMaxWaitRateLimiter":
						info.Uses[fun.Sel] = types.NewFunc(token.NoPos, pkgWQR, fun.Sel.Name, nil)
					}
				}
			}
		}
		return true
	})
	var diags []analysis.Diagnostic
	pass := &analysis.Pass{Analyzer: AnalyzerUnboundedQueue, Fset: fset, Files: files, TypesInfo: info, TypesSizes: types.SizesFor("gc", "amd64"), Report: func(d analysis.Diagnostic) { diags = append(diags, d) }, ResultOf: map[*analysis.Analyzer]interface{}{insppass.Analyzer: inspector.New(files)}}
	_, _ = AnalyzerUnboundedQueue.Run(pass)
	return diags
}

func TestUnboundedQueue_WorkqueueNew_Flagged(t *testing.T) {
	src := `package a
import wq "k8s.io/client-go/util/workqueue"
func f(){ _ = wq.New() }`
	diags := runUnboundedQueueAnalyzerOnSrc(t, src)
	if len(diags) == 0 {
		t.Fatalf("expected diagnostic for workqueue.New without rate limiter")
	}
}

func TestUnboundedQueue_RateLimitingQueue_NoDiag(t *testing.T) {
	src := `package a
import wq "k8s.io/client-go/util/workqueue"
func NewItemExponentialFailureRateLimiter() any { return nil }
func f(){ rl := NewItemExponentialFailureRateLimiter(); _ = wq.NewRateLimitingQueue(rl) }`
	diags := runUnboundedQueueAnalyzerOnSrc(t, src)
	if len(diags) != 0 {
		t.Fatalf("did not expect diagnostic for NewRateLimitingQueue")
	}
}

func TestUnboundedQueue_NonWorkqueue_New_NoDiag(t *testing.T) {
	src := `package a
func New() any { return nil }
func f(){ _ = New() }`
	diags := runUnboundedQueueAnalyzerOnSrc(t, src)
	if len(diags) != 0 {
		t.Fatalf("did not expect diagnostic for non-workqueue New")
	}
}
