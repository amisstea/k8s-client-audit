package analyzers

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
)

// AnalyzerManualPolling (K8S012) flags loops that poll with List + sleep/ticker
// instead of using watches/informers.
var AnalyzerManualPolling = &analysis.Analyzer{
	Name: "k8s012_manualpolling",
	Doc:  "flags manual polling loops using List with sleep/ticker",
	Run:  runManualPolling,
}

func runManualPolling(pass *analysis.Pass) (any, error) {
	for _, f := range pass.Files {
		ast.Inspect(f, func(n ast.Node) bool {
			fl, ok := n.(*ast.ForStmt)
			if !ok {
				return true
			}
			hasSleepOrTicker := false
			hasList := false
			ast.Inspect(fl.Body, func(m ast.Node) bool {
				ce, ok := m.(*ast.CallExpr)
				if !ok {
					return true
				}
				// time.Sleep or ticker.C receive
				if sel, ok := ce.Fun.(*ast.SelectorExpr); ok && sel.Sel != nil {
					if sel.Sel.Name == "Sleep" {
						hasSleepOrTicker = true
					}
					if sel.Sel.Name == "List" {
						hasList = true
					}
				}
				return true
			})
			if hasList && hasSleepOrTicker {
				pass.Reportf(fl.For, "Manual polling with List and sleep/ticker; prefer Watch or shared informers")
			}
			return true
		})
	}
	return nil, nil
}
