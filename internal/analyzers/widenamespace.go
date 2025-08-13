package analyzers

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
)

// AnalyzerWideNamespace flags all-namespaces listing heuristics like InNamespace("") or typed Pods("").List.
var AnalyzerWideNamespace = &analysis.Analyzer{
	Name: "widenamespace",
	Doc:  "flags cluster-wide scans when namespace-scoped suffices",
	Run:  runWideNamespace,
}

func runWideNamespace(pass *analysis.Pass) (any, error) {
	for _, f := range pass.Files {
		ast.Inspect(f, func(n ast.Node) bool {
			ce, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			switch fun := ce.Fun.(type) {
			case *ast.SelectorExpr:
				if fun.Sel != nil && fun.Sel.Name == "InNamespace" && len(ce.Args) == 1 {
					if isEmptyString(ce.Args[0]) {
						pass.Reportf(fun.Sel.Pos(), "all-namespaces list; scope to a namespace if possible")
					}
				}
				if fun.Sel != nil && fun.Sel.Name == "List" {
					if hasEmptyStringNamespaceArg(fun.X) {
						pass.Reportf(fun.Sel.Pos(), "all-namespaces list; scope to a namespace if possible")
					}
				}
			case *ast.Ident:
				if fun.Name == "InNamespace" && len(ce.Args) == 1 {
					if isEmptyString(ce.Args[0]) {
						pass.Reportf(fun.Pos(), "all-namespaces list; scope to a namespace if possible")
					}
				}
			}
			return true
		})
	}
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
