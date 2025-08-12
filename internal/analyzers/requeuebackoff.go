package analyzers

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
)

// AnalyzerRequeueBackoff (K8S014) flags controller-runtime Reconcile paths that
// requeue immediately without a backoff (e.g., returning requeue=true without RequeueAfter).
var AnalyzerRequeueBackoff = &analysis.Analyzer{
	Name: "k8s014_requeuebackoff",
	Doc:  "flags requeue without backoff in Reconcile",
	Run:  runRequeueBackoff,
}

func runRequeueBackoff(pass *analysis.Pass) (any, error) {
	for _, f := range pass.Files {
		ast.Inspect(f, func(n ast.Node) bool {
			ret, ok := n.(*ast.ReturnStmt)
			if !ok {
				return true
			}
			// Look for named return style: return ctrl.Result{Requeue:true}, nil
			for _, res := range ret.Results {
				cl, ok := res.(*ast.CompositeLit)
				if !ok {
					continue
				}
				isResult := false
				if sel, ok := cl.Type.(*ast.SelectorExpr); ok && sel.Sel != nil {
					if sel.Sel.Name == "Result" {
						isResult = true
					}
				}
				if id, ok := cl.Type.(*ast.Ident); ok {
					if id.Name == "Result" {
						isResult = true
					}
				}
				if isResult {
					hasRequeue := false
					hasRequeueAfter := false
					for _, el := range cl.Elts {
						if kv, ok := el.(*ast.KeyValueExpr); ok {
							if k, ok := kv.Key.(*ast.Ident); ok {
								if k.Name == "Requeue" {
									hasRequeue = true
								}
								if k.Name == "RequeueAfter" {
									hasRequeueAfter = true
								}
							}
						}
					}
					if hasRequeue && !hasRequeueAfter {
						pass.Reportf(ret.Return, "Requeue without backoff; prefer RequeueAfter with delay or rate-limited queue")
					}
				}
			}
			return true
		})
	}
	return nil, nil
}
