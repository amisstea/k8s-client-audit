package analyzers

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
)

// AnalyzerListInLoop flags List/Watch calls inside loops.
var AnalyzerListInLoop = &analysis.Analyzer{
	Name: "k8s011_listinloop",
	Doc:  "flags List/Watch calls inside loops (prefer informers/cache)",
	Run:  runListInLoop,
}

func runListInLoop(pass *analysis.Pass) (any, error) {
	for _, f := range pass.Files {
		ast.Inspect(f, func(n ast.Node) bool {
			switch loop := n.(type) {
			case *ast.ForStmt:
				checkLoopBody(pass, loop.Body)
			case *ast.RangeStmt:
				checkLoopBody(pass, loop.Body)
			}
			return true
		})
	}
	return nil, nil
}

func checkLoopBody(pass *analysis.Pass, body *ast.BlockStmt) {
	if body == nil {
		return
	}
	ast.Inspect(body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		if sel, ok := call.Fun.(*ast.SelectorExpr); ok && sel.Sel != nil {
			if sel.Sel.Name == "List" || sel.Sel.Name == "Watch" {
				pass.Reportf(sel.Sel.Pos(), "List/Watch call inside loop; prefer informers/cache or move calls outside loops")
			}
		}
		return true
	})
}
