package analyzers

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
)

// AnalyzerUnstructuredEverywhere (K8S052) flags heavy use of unstructured.Unstructured
// in functions that could use typed objects. Heuristic: many composite literals or
// declarations of Unstructured within a file.
var AnalyzerUnstructuredEverywhere = &analysis.Analyzer{
	Name: "k8s052_unstructuredeverywhere",
	Doc:  "flags pervasive use of unstructured objects instead of typed",
	Run:  runUnstructuredEverywhere,
}

func runUnstructuredEverywhere(pass *analysis.Pass) (any, error) {
	count := 0
	for _, f := range pass.Files {
		ast.Inspect(f, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.CompositeLit:
				if id, ok := x.Type.(*ast.Ident); ok && id.Name == "Unstructured" {
					count++
				}
				if se, ok := x.Type.(*ast.SelectorExpr); ok && se.Sel != nil && se.Sel.Name == "Unstructured" {
					count++
				}
			case *ast.CallExpr:
				if se, ok := x.Fun.(*ast.SelectorExpr); ok && se.Sel != nil && se.Sel.Name == "Unstructured" {
					count++
				}
			}
			return true
		})
	}
	if count >= 3 { // threshold heuristic
		// Report once per package
		for _, f := range pass.Files {
			pass.Reportf(f.Package, "Heavy use of unstructured.Unstructured; prefer typed clients/objects when possible")
			break
		}
	}
	return nil, nil
}
