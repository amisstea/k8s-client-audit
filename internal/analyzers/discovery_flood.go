package analyzers

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
)

// AnalyzerDiscoveryFlood (K8S070) flags repeated discovery client creations or
// RESTMapper resets in loops, which can flood the API server.
var AnalyzerDiscoveryFlood = &analysis.Analyzer{
	Name: "k8s070_discoveryflood",
	Doc:  "flags repeated discovery or RESTMapper rebuilds",
	Run:  runDiscoveryFlood,
}

func runDiscoveryFlood(pass *analysis.Pass) (any, error) {
	for _, f := range pass.Files {
		ast.Inspect(f, func(n ast.Node) bool {
			fl, ok := n.(*ast.ForStmt)
			if !ok {
				return true
			}
			repeated := false
			ast.Inspect(fl.Body, func(m ast.Node) bool {
				ce, ok := m.(*ast.CallExpr)
				if !ok {
					return true
				}
				if sel, ok := ce.Fun.(*ast.SelectorExpr); ok && sel.Sel != nil {
					if sel.Sel.Name == "NewDiscoveryClientForConfig" || sel.Sel.Name == "ResetRESTMapper" || sel.Sel.Name == "NewDeferredDiscoveryRESTMapper" {
						repeated = true
					}
				}
				if id, ok := ce.Fun.(*ast.Ident); ok {
					if id.Name == "NewDiscoveryClientForConfig" || id.Name == "ResetRESTMapper" || id.Name == "NewDeferredDiscoveryRESTMapper" {
						repeated = true
					}
				}
				return true
			})
			if repeated {
				pass.Reportf(fl.For, "Repeated discovery/RESTMapper setup inside loop; cache and reuse to avoid API server flood")
			}
			return true
		})
	}
	return nil, nil
}
