package analyzers

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
	insppass "golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// AnalyzerWebhookNoContext flags HTTP handlers that ignore request
// context or call Background/TODO for outgoing calls.
var AnalyzerWebhookNoContext = &analysis.Analyzer{
	Name:     "webhook_nocontext",
	Doc:      "flags webhook handlers that don't use request context",
	Run:      runWebhookNoContext,
	Requires: []*analysis.Analyzer{insppass.Analyzer},
}

func runWebhookNoContext(pass *analysis.Pass) (any, error) {
	insp := pass.ResultOf[insppass.Analyzer].(*inspector.Inspector)
	isContextCtor := func(obj types.Object) bool {
		if obj == nil || obj.Pkg() == nil {
			return false
		}
		if obj.Pkg().Path() != "context" {
			return false
		}
		name := obj.Name()
		return name == "Background" || name == "TODO"
	}
	insp.Preorder([]ast.Node{(*ast.CallExpr)(nil)}, func(n ast.Node) {
		ce := n.(*ast.CallExpr)
		if id := calleeIdent(ce.Fun); id != nil {
			if isContextCtor(pass.TypesInfo.Uses[id]) {
				pass.Reportf(id.Pos(), "Webhook code using context.Background/TODO; propagate request context instead")
			}
		}
	})
	return nil, nil
}
