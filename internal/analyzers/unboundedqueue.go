package analyzers

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
	insppass "golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// AnalyzerUnboundedQueue (K8S013) flags usage of workqueue without rate limiting
// or without max-depth guards.
var AnalyzerUnboundedQueue = &analysis.Analyzer{
	Name:     "k8s013_unboundedqueue",
	Doc:      "flags unbounded workqueue usage without rate limiting",
	Run:      runUnboundedQueue,
	Requires: []*analysis.Analyzer{insppass.Analyzer},
}

func runUnboundedQueue(pass *analysis.Pass) (any, error) {
	isFromWorkqueue := func(obj types.Object, names ...string) bool {
		if obj == nil || obj.Pkg() == nil {
			return false
		}
		if obj.Pkg().Path() != "k8s.io/client-go/util/workqueue" {
			return false
		}
		on := obj.Name()
		for _, n := range names {
			if on == n {
				return true
			}
		}
		return false
	}
	insp := pass.ResultOf[insppass.Analyzer].(*inspector.Inspector)
	insp.Preorder([]ast.Node{(*ast.CallExpr)(nil)}, func(n ast.Node) {
		ce := n.(*ast.CallExpr)
		if obj := pass.TypesInfo.Uses[calleeIdent(ce.Fun)]; obj != nil {
			if isFromWorkqueue(obj, "New", "NewNamed") {
				pass.Reportf(ce.Lparen, "Workqueue constructed without a rate limiter; use NewRateLimitingQueue or a RateLimitingInterface")
			}
		}
	})
	return nil, nil
}

func calleeIdent(expr ast.Expr) *ast.Ident {
	switch x := expr.(type) {
	case *ast.Ident:
		return x
	case *ast.SelectorExpr:
		if x.Sel != nil {
			return x.Sel
		}
	}
	return nil
}
