package analyzers

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
	insppass "golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// AnalyzerDynamicOveruse flags creation of dynamic/unstructured clients
// when typed clients appear to be available in the same package (heuristic).
var AnalyzerDynamicOveruse = &analysis.Analyzer{
	Name:     "dynamicoveruse",
	Doc:      "flags overuse of dynamic/unstructured when typed clients exist",
	Run:      runDynamicOveruse,
	Requires: []*analysis.Analyzer{insppass.Analyzer},
}

func runDynamicOveruse(pass *analysis.Pass) (any, error) {
	insp := pass.ResultOf[insppass.Analyzer].(*inspector.Inspector)
	hasTyped := false
	insp.Preorder([]ast.Node{(*ast.CallExpr)(nil)}, func(n ast.Node) {
		ce := n.(*ast.CallExpr)
		if id := calleeIdent(ce.Fun); id != nil {
			if obj := pass.TypesInfo.Uses[id]; obj != nil && obj.Pkg() != nil {
				if (obj.Name() == "NewForConfig" || obj.Name() == "NewForConfigOrDie") && obj.Pkg().Path() == "k8s.io/client-go/kubernetes" {
					hasTyped = true
				}
			}
		}
	})
	insp.Preorder([]ast.Node{(*ast.CallExpr)(nil)}, func(n ast.Node) {
		ce := n.(*ast.CallExpr)
		if id := calleeIdent(ce.Fun); id != nil {
			if obj := pass.TypesInfo.Uses[id]; obj != nil && obj.Pkg() != nil {
				if obj.Pkg().Path() == "k8s.io/client-go/dynamic" {
					if obj.Name() == "NewForConfig" || obj.Name() == "NewDynamicClientForConfig" || obj.Name() == "NewForConfigOrDie" {
						if hasTyped {
							pass.Reportf(id.Pos(), "Prefer typed client over dynamic/unstructured when available")
						}
					}
				}
			}
		}
	})
	return nil, nil
}
