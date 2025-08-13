package analyzers

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
	insppass "golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// AnalyzerDiscoveryFlood flags repeated discovery client creations or
// RESTMapper resets in loops, which can flood the API server.
var AnalyzerDiscoveryFlood = &analysis.Analyzer{
	Name:     "discoveryflood",
	Doc:      "flags repeated discovery or RESTMapper rebuilds",
	Run:      runDiscoveryFlood,
	Requires: []*analysis.Analyzer{insppass.Analyzer},
}

func runDiscoveryFlood(pass *analysis.Pass) (any, error) {
	insp := pass.ResultOf[insppass.Analyzer].(*inspector.Inspector)
	loopDepth := 0
	repeated := false
	nodes := []ast.Node{(*ast.ForStmt)(nil), (*ast.RangeStmt)(nil), (*ast.CallExpr)(nil)}
	insp.Nodes(nodes, func(n ast.Node, push bool) bool {
		switch x := n.(type) {
		case *ast.ForStmt, *ast.RangeStmt:
			if push {
				loopDepth++
				repeated = false
			} else {
				if loopDepth > 0 && repeated {
					pass.Reportf(x.Pos(), "Repeated discovery/RESTMapper setup inside loop; cache and reuse to avoid API server flood")
				}
				loopDepth--
			}
		case *ast.CallExpr:
			if !push || loopDepth == 0 {
				return true
			}
			if id := calleeIdent(x.Fun); id != nil {
				if obj := pass.TypesInfo.Uses[id]; obj != nil && obj.Pkg() != nil {
					name := obj.Name()
					pkg := obj.Pkg().Path()
					if (name == "NewDiscoveryClientForConfig" && pkg == "k8s.io/client-go/discovery") ||
						(name == "NewDeferredDiscoveryRESTMapper" && pkg == "k8s.io/client-go/restmapper") ||
						(name == "ResetRESTMapper" && pkg == "k8s.io/client-go/restmapper") {
						repeated = true
					}
				}
			}
		}
		return true
	})
	return nil, nil
}
