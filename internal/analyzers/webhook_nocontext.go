package analyzers

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
)

// AnalyzerWebhookNoContext (K8S061) flags HTTP handlers that ignore request
// context or call Background/TODO for outgoing calls.
var AnalyzerWebhookNoContext = &analysis.Analyzer{
	Name: "k8s061_webhook_nocontext",
	Doc:  "flags webhook handlers that don't use request context",
	Run:  runWebhookNoContext,
}

func runWebhookNoContext(pass *analysis.Pass) (any, error) {
	for _, f := range pass.Files {
		ast.Inspect(f, func(n ast.Node) bool {
			ce, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			if sel, ok := ce.Fun.(*ast.SelectorExpr); ok && sel.Sel != nil {
				if sel.Sel.Name == "Background" || sel.Sel.Name == "TODO" {
					pass.Reportf(sel.Sel.Pos(), "Webhook code using context.Background/TODO; propagate request context instead")
				}
			}
			return true
		})
	}
	return nil, nil
}
