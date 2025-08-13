package analyzers

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
)

// AnalyzerLeakyWatch flags Watch calls whose ResultChan is obtained but
// not stopped/drained. Heuristic: if a call to Stop/Cancel is not found.
var AnalyzerLeakyWatch = &analysis.Analyzer{
	Name: "leakywatch",
	Doc:  "flags potential leaky watch channels without stop",
	Run:  runLeakyWatch,
}

func runLeakyWatch(pass *analysis.Pass) (any, error) {
	for _, f := range pass.Files {
		ast.Inspect(f, func(n ast.Node) bool {
			as, ok := n.(*ast.AssignStmt)
			if !ok || len(as.Rhs) == 0 {
				return true
			}
			// x := w.ResultChan()
			if ce, ok := as.Rhs[0].(*ast.CallExpr); ok {
				if sel, ok := ce.Fun.(*ast.SelectorExpr); ok && sel.Sel != nil && sel.Sel.Name == "ResultChan" {
					// Conservatively scan function body for Stop/Cancel
					foundStop := false
					ast.Inspect(f, func(m ast.Node) bool {
						ce2, ok := m.(*ast.CallExpr)
						if !ok {
							return true
						}
						if s2, ok := ce2.Fun.(*ast.SelectorExpr); ok && s2.Sel != nil {
							if s2.Sel.Name == "Stop" || s2.Sel.Name == "StopWatching" || s2.Sel.Name == "Cancel" {
								foundStop = true
								return false
							}
						}
						return true
					})
					if !foundStop {
						pass.Reportf(sel.Sel.Pos(), "Watch channel may not be stopped; ensure Stop()/Cancel() is called")
					}
				}
			}
			return true
		})
	}
	return nil, nil
}
