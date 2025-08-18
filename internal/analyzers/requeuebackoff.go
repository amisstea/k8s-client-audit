package analyzers

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
	insppass "golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// AnalyzerRequeueBackoff flags controller-runtime Reconcile paths that
// requeue immediately without a backoff (e.g., returning requeue=true without RequeueAfter).
var AnalyzerRequeueBackoff = &analysis.Analyzer{
	Name:     "requeuebackoff",
	Doc:      "flags requeue without backoff in Reconcile",
	Run:      runRequeueBackoff,
	Requires: []*analysis.Analyzer{insppass.Analyzer},
}

func runRequeueBackoff(pass *analysis.Pass) (any, error) {
	insp := pass.ResultOf[insppass.Analyzer].(*inspector.Inspector)

	// Check if a type is a controller-runtime Result type
	isControllerRuntimeResult := func(t types.Type) bool {
		if named, ok := t.(*types.Named); ok {
			if named.Obj() != nil && named.Obj().Pkg() != nil {
				pkg := named.Obj().Pkg().Path()
				name := named.Obj().Name()

				// Check for controller-runtime reconcile.Result
				if name == "Result" {
					switch {
					case pkg == "sigs.k8s.io/controller-runtime/pkg/reconcile":
						return true
					}
				}
			}
		}
		return false
	}

	nodes := []ast.Node{(*ast.ReturnStmt)(nil)}
	insp.Nodes(nodes, func(n ast.Node, push bool) bool {
		if !push {
			return true
		}

		ret := n.(*ast.ReturnStmt)

		// Look for return statements with controller-runtime Result composite literals
		for _, res := range ret.Results {
			cl, ok := res.(*ast.CompositeLit)
			if !ok {
				continue
			}

			// Use type information to verify this is a controller-runtime Result
			if t := pass.TypesInfo.TypeOf(cl); t != nil {
				if isControllerRuntimeResult(t) {
					hasRequeue := false
					hasRequeueAfter := false

					// Check the fields of the composite literal
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
						pass.Reportf(ret.Return, "controller-runtime Requeue without backoff; prefer RequeueAfter with delay or rate-limited queue")
					}
				}
			}
		}

		return true
	})

	return nil, nil
}
