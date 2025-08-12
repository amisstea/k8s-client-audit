package analyzers

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
)

// AnalyzerUnboundedQueue (K8S013) flags usage of workqueue without rate limiting
// or without max-depth guards.
var AnalyzerUnboundedQueue = &analysis.Analyzer{
	Name: "k8s013_unboundedqueue",
	Doc:  "flags unbounded workqueue usage without rate limiting",
	Run:  runUnboundedQueue,
}

func runUnboundedQueue(pass *analysis.Pass) (any, error) {
	hasRateLimiter := false
	// detect NewItemFastSlowRateLimiter/NewItemExponentialFailureRateLimiter etc.
	for _, f := range pass.Files {
		ast.Inspect(f, func(n ast.Node) bool {
			ce, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			if sel, ok := ce.Fun.(*ast.SelectorExpr); ok && sel.Sel != nil {
				switch sel.Sel.Name {
				case "NewItemExponentialFailureRateLimiter", "NewItemFastSlowRateLimiter", "NewMaxOfRateLimiter", "NewWithMaxWaitRateLimiter":
					hasRateLimiter = true
				}
			}
			if id, ok := ce.Fun.(*ast.Ident); ok {
				switch id.Name {
				case "NewItemExponentialFailureRateLimiter", "NewItemFastSlowRateLimiter", "NewMaxOfRateLimiter", "NewWithMaxWaitRateLimiter":
					hasRateLimiter = true
				}
			}
			return true
		})
	}
	for _, f := range pass.Files {
		ast.Inspect(f, func(n ast.Node) bool {
			ce, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			if sel, ok := ce.Fun.(*ast.SelectorExpr); ok && sel.Sel != nil {
				if sel.Sel.Name == "NewNamed" || sel.Sel.Name == "New" {
					// Rough heuristic: if queue constructed and no rate limiter observed in package, warn
					if !hasRateLimiter {
						pass.Reportf(sel.Sel.Pos(), "Workqueue constructed without a rate limiter; use workqueue.RateLimitingInterface")
					}
				}
			}
			if id, ok := ce.Fun.(*ast.Ident); ok {
				if id.Name == "NewNamed" || id.Name == "New" {
					if !hasRateLimiter {
						pass.Reportf(id.Pos(), "Workqueue constructed without a rate limiter; use workqueue.RateLimitingInterface")
					}
				}
			}
			return true
		})
	}
	return nil, nil
}
