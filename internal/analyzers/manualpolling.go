package analyzers

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
	insppass "golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// AnalyzerManualPolling flags loops that poll with List + sleep/ticker
// instead of using watches/informers.
var AnalyzerManualPolling = &analysis.Analyzer{
	Name:     "manualpolling",
	Doc:      "flags manual polling loops using List with sleep/ticker",
	Run:      runManualPolling,
	Requires: []*analysis.Analyzer{insppass.Analyzer},
}

func runManualPolling(pass *analysis.Pass) (any, error) {
	insp := pass.ResultOf[insppass.Analyzer].(*inspector.Inspector)

	// Check if a method call is a sleep operation
	isSleepCall := func(obj types.Object) bool {
		if obj == nil || obj.Pkg() == nil {
			return false
		}
		name := obj.Name()
		pkg := obj.Pkg().Path()

		// time.Sleep
		return name == "Sleep" && pkg == "time"
	}

	loopDepth := 0
	var foundKubernetesLists []ast.Node
	var foundSleeps []ast.Node

	nodes := []ast.Node{(*ast.ForStmt)(nil), (*ast.RangeStmt)(nil), (*ast.CallExpr)(nil)}
	insp.Nodes(nodes, func(n ast.Node, push bool) bool {
		switch x := n.(type) {
		case *ast.ForStmt, *ast.RangeStmt:
			if push {
				loopDepth++
				// Reset tracking for this loop level
				foundKubernetesLists = foundKubernetesLists[:0]
				foundSleeps = foundSleeps[:0]
			} else {
				// Check if we found both K8s List calls and Sleep calls in this loop
				if loopDepth == 1 && len(foundKubernetesLists) > 0 && len(foundSleeps) > 0 {
					var pos ast.Node
					if fl, ok := x.(*ast.ForStmt); ok {
						pos = fl
					} else if rl, ok := x.(*ast.RangeStmt); ok {
						pos = rl
					}
					if pos != nil {
						pass.Reportf(pos.Pos(), "Manual polling with Kubernetes List and sleep/ticker; prefer Watch or shared informers")
					}
				}
				loopDepth--
			}
		case *ast.CallExpr:
			if !push || loopDepth == 0 {
				return true
			}

			// Check if this is a method call
			sel, ok := x.Fun.(*ast.SelectorExpr)
			if !ok || sel.Sel == nil {
				return true
			}

			// Use type information to determine what kind of call this is
			if obj := pass.TypesInfo.Uses[sel.Sel]; obj != nil {
				if isKubernetesMethodCall(obj, "List") {
					foundKubernetesLists = append(foundKubernetesLists, x)
				} else if isSleepCall(obj) {
					foundSleeps = append(foundSleeps, x)
				}
			}
		}
		return true
	})

	return nil, nil
}
