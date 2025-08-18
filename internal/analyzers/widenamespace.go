package analyzers

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
	insppass "golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// AnalyzerWideNamespace flags all-namespaces listing heuristics like InNamespace("") or typed Pods("").List.
var AnalyzerWideNamespace = &analysis.Analyzer{
	Name:     "widenamespace",
	Doc:      "flags cluster-wide scans when namespace-scoped suffices",
	Run:      runWideNamespace,
	Requires: []*analysis.Analyzer{insppass.Analyzer},
}

func runWideNamespace(pass *analysis.Pass) (any, error) {
	insp := pass.ResultOf[insppass.Analyzer].(*inspector.Inspector)

	nodes := []ast.Node{(*ast.CallExpr)(nil)}
	insp.Nodes(nodes, func(n ast.Node, push bool) bool {
		if !push {
			return true
		}

		ce := n.(*ast.CallExpr)

		switch fun := ce.Fun.(type) {
		case *ast.SelectorExpr:
			if fun.Sel == nil {
				return true
			}

			// Check for InNamespace("") calls from Kubernetes packages
			if fun.Sel.Name == "InNamespace" && len(ce.Args) == 1 {
				if obj := pass.TypesInfo.Uses[fun.Sel]; isKubernetesMethodCall(obj, "InNamespace") {
					if isEmptyString(ce.Args[0]) {
						pass.Reportf(fun.Sel.Pos(), "all-namespaces list; scope to a namespace if possible")
					}
				}
			}

			// Check for client-go style List calls with empty namespace: client.Pods("").List()
			if fun.Sel.Name == "List" {
				if obj := pass.TypesInfo.Uses[fun.Sel]; isKubernetesMethodCall(obj, "List") {
					if hasEmptyStringNamespaceArg(fun.X) {
						pass.Reportf(fun.Sel.Pos(), "all-namespaces list; scope to a namespace if possible")
					}
				}
			}

		case *ast.Ident:
			// Check for standalone InNamespace("") function calls
			if fun.Name == "InNamespace" && len(ce.Args) == 1 {
				if obj := pass.TypesInfo.Uses[fun]; isKubernetesMethodCall(obj, "InNamespace") {
					if isEmptyString(ce.Args[0]) {
						pass.Reportf(fun.Pos(), "all-namespaces list; scope to a namespace if possible")
					}
				}
			}
		}

		return true
	})

	return nil, nil
}

func isEmptyString(e ast.Expr) bool {
	if bl, ok := e.(*ast.BasicLit); ok {
		return bl.Value == "\"\""
	}
	return false
}

func hasEmptyStringNamespaceArg(expr ast.Expr) bool {
	switch e := expr.(type) {
	case *ast.CallExpr:
		if len(e.Args) > 0 {
			if isEmptyString(e.Args[0]) {
				return true
			}
		}
		return hasEmptyStringNamespaceArg(e.Fun)
	case *ast.SelectorExpr:
		return hasEmptyStringNamespaceArg(e.X)
	}
	return false
}
