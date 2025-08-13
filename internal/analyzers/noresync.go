package analyzers

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
	insppass "golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// AnalyzerNoResync flags informer creations with resync period set to 0
// where a positive resync might be desirable. Heuristic only.
var AnalyzerNoResync = &analysis.Analyzer{
	Name:     "noresync",
	Doc:      "flags informer creation with zero resync period",
	Run:      runNoResync,
	Requires: []*analysis.Analyzer{insppass.Analyzer},
}

func runNoResync(pass *analysis.Pass) (any, error) {
	insp := pass.ResultOf[insppass.Analyzer].(*inspector.Inspector)
	isInformerCtor := func(obj types.Object) bool {
		if obj == nil {
			return false
		}
		name := obj.Name()
		return name == "NewSharedIndexInformer" || name == "NewSharedInformer"
	}
	insp.Preorder([]ast.Node{(*ast.CallExpr)(nil)}, func(n ast.Node) {
		ce := n.(*ast.CallExpr)
		if id := calleeIdent(ce.Fun); id != nil {
			if isInformerCtor(pass.TypesInfo.Uses[id]) {
				for _, a := range ce.Args {
					if bl, ok := a.(*ast.BasicLit); ok && bl.Value == "0" {
						pass.Reportf(bl.Pos(), "Informer resync period is zero; ensure this is intentional")
					}
				}
			}
		}
	})
	return nil, nil
}
