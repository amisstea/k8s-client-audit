package analyzers

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
	insppass "golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// AnalyzerLeakyWatch flags Watch calls whose ResultChan is obtained but
// not stopped/drained. Heuristic: if a call to Stop/Cancel is not found.
var AnalyzerLeakyWatch = &analysis.Analyzer{
	Name:     "leakywatch",
	Doc:      "flags potential leaky watch channels without stop",
	Run:      runLeakyWatch,
	Requires: []*analysis.Analyzer{insppass.Analyzer},
}

func runLeakyWatch(pass *analysis.Pass) (any, error) {
	insp := pass.ResultOf[insppass.Analyzer].(*inspector.Inspector)

	// Check if a method call is a stop/cancel operation
	isStopCall := func(obj types.Object) bool {
		if obj == nil {
			return false
		}
		name := obj.Name()
		return name == "Stop" || name == "StopWatching" || name == "Cancel"
	}

	// Track ResultChan calls per function and their corresponding stop calls
	var currentFunc *ast.FuncDecl
	var resultChanCalls []ast.Node
	var stopCalls []ast.Node

	nodes := []ast.Node{(*ast.FuncDecl)(nil), (*ast.CallExpr)(nil)}
	insp.Nodes(nodes, func(n ast.Node, push bool) bool {
		switch x := n.(type) {
		case *ast.FuncDecl:
			if push {
				currentFunc = x
				// Reset tracking for this function
				resultChanCalls = resultChanCalls[:0]
				stopCalls = stopCalls[:0]
			} else {
				// Check for leaky watches in this function
				if currentFunc != nil && len(resultChanCalls) > 0 && len(stopCalls) == 0 {
					// Found ResultChan calls but no Stop calls
					for _, call := range resultChanCalls {
						pass.Reportf(call.Pos(), "Kubernetes Watch channel may not be stopped; ensure Stop()/Cancel() is called")
					}
				}
				currentFunc = nil
			}
		case *ast.CallExpr:
			if !push || currentFunc == nil {
				return true
			}

			// Check if this is a method call
			sel, ok := x.Fun.(*ast.SelectorExpr)
			if !ok || sel.Sel == nil {
				return true
			}

			// Use type information to determine what kind of call this is
			if obj := pass.TypesInfo.Uses[sel.Sel]; obj != nil {
				if isKubernetesMethodCall(obj, "ResultChan") {
					resultChanCalls = append(resultChanCalls, x)
				} else if isStopCall(obj) {
					stopCalls = append(stopCalls, x)
				}
			}
		}
		return true
	})

	return nil, nil
}
