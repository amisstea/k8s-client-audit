package analyzers

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
	insppass "golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// AnalyzerMissingContext flags client calls that pass context.Background/TODO instead of a propagated context.
var AnalyzerMissingContext = &analysis.Analyzer{
	Name:     "missingcontext",
	Doc:      "flags client calls using context.Background/TODO instead of propagated context",
	Run:      runMissingContext,
	Requires: []*analysis.Analyzer{insppass.Analyzer},
}

func runMissingContext(pass *analysis.Pass) (any, error) {
	insp := pass.ResultOf[insppass.Analyzer].(*inspector.Inspector)

	nodes := []ast.Node{(*ast.CallExpr)(nil)}
	insp.Nodes(nodes, func(n ast.Node, push bool) bool {
		if !push {
			return true
		}

		call := n.(*ast.CallExpr)
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel == nil {
			return true
		}

		if len(call.Args) == 0 {
			return true
		}

		// Check if this is a Kubernetes client method using type information
		if obj := pass.TypesInfo.Uses[sel.Sel]; isKubernetesMethodCall(obj, "Get", "List", "Create", "Update", "Patch", "Delete", "Watch") {
			if isContextBackgroundOrTODO(call.Args[0]) {
				pass.Reportf(sel.Sel.Pos(), "client call uses context.Background/TODO; propagate a request context instead")
			}
		}

		return true
	})

	return nil, nil
}

func isContextBackgroundOrTODO(arg ast.Expr) bool {
	if sub, ok := arg.(*ast.CallExpr); ok {
		if s2, ok := sub.Fun.(*ast.SelectorExpr); ok {
			if ident, ok := s2.X.(*ast.Ident); ok && ident.Name == "context" {
				return s2.Sel.Name == "Background" || s2.Sel.Name == "TODO"
			}
		}
	}
	return false
}
