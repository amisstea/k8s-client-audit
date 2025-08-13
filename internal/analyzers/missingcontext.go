package analyzers

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
)

// AnalyzerMissingContext flags client calls that pass context.Background/TODO instead of a propagated context.
var AnalyzerMissingContext = &analysis.Analyzer{
	Name: "missingcontext",
	Doc:  "flags client calls using context.Background/TODO instead of propagated context",
	Run:  runMissingContext,
}

func runMissingContext(pass *analysis.Pass) (any, error) {
	for _, f := range pass.Files {
		ast.Inspect(f, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || sel.Sel == nil {
				return true
			}
			name := sel.Sel.Name
			if !(name == "Get" || name == "List" || name == "Create" || name == "Update" || name == "Patch" || name == "Delete") {
				return true
			}
			if len(call.Args) == 0 {
				return true
			}
			if isContextBackgroundOrTODO(call.Args[0]) {
				pass.Reportf(sel.Sel.Pos(), "client call uses context.Background/TODO; propagate a request context instead")
			}
			return true
		})
	}
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
