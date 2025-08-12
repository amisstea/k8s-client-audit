package analyzers

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
)

// AnalyzerDynamicOveruse (K8S051) flags creation of dynamic/unstructured clients
// when typed clients appear to be available in the same package (heuristic).
var AnalyzerDynamicOveruse = &analysis.Analyzer{
	Name: "k8s051_dynamicoveruse",
	Doc:  "flags overuse of dynamic/unstructured when typed clients exist",
	Run:  runDynamicOveruse,
}

func runDynamicOveruse(pass *analysis.Pass) (any, error) {
	hasTyped := false
	for _, f := range pass.Files {
		ast.Inspect(f, func(n ast.Node) bool {
			if fn, ok := n.(*ast.FuncDecl); ok {
				if fn.Name != nil && (fn.Name.Name == "NewForConfig" || fn.Name.Name == "NewForConfigOrDie") {
					hasTyped = true
				}
			}
			return true
		})
	}
	for _, f := range pass.Files {
		ast.Inspect(f, func(n ast.Node) bool {
			ce, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			if sel, ok := ce.Fun.(*ast.SelectorExpr); ok && sel.Sel != nil {
				if sel.Sel.Name == "NewDynamicClientForConfig" {
					if hasTyped {
						pass.Reportf(sel.Sel.Pos(), "Prefer typed client over dynamic/unstructured when available")
					}
				}
			}
			if id, ok := ce.Fun.(*ast.Ident); ok {
				if id.Name == "NewDynamicClientForConfig" {
					if hasTyped {
						pass.Reportf(id.Pos(), "Prefer typed client over dynamic/unstructured when available")
					}
				}
			}
			return true
		})
	}
	return nil, nil
}
