package analyzers

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
)

// AnalyzerNoResync (K8S040) flags informer creations with resync period set to 0
// where a positive resync might be desirable. Heuristic only.
var AnalyzerNoResync = &analysis.Analyzer{
	Name: "k8s040_noresync",
	Doc:  "flags informer creation with zero resync period",
	Run:  runNoResync,
}

func runNoResync(pass *analysis.Pass) (any, error) {
	for _, f := range pass.Files {
		ast.Inspect(f, func(n ast.Node) bool {
			ce, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			if sel, ok := ce.Fun.(*ast.SelectorExpr); ok && sel.Sel != nil {
				if sel.Sel.Name == "NewSharedIndexInformer" || sel.Sel.Name == "NewSharedInformer" {
					// Usually takes a resync period argument; flag literal zero duration or 0
					for _, a := range ce.Args {
						if bl, ok := a.(*ast.BasicLit); ok && bl.Value == "0" {
							pass.Reportf(bl.Pos(), "Informer resync period is zero; ensure this is intentional")
						}
					}
				}
			}
			if id, ok := ce.Fun.(*ast.Ident); ok {
				if id.Name == "NewSharedIndexInformer" || id.Name == "NewSharedInformer" {
					for _, a := range ce.Args {
						if bl, ok := a.(*ast.BasicLit); ok && bl.Value == "0" {
							pass.Reportf(bl.Pos(), "Informer resync period is zero; ensure this is intentional")
						}
					}
				}
			}
			return true
		})
	}
	return nil, nil
}
