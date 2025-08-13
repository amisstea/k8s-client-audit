package analyzers

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
	insppass "golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// AnalyzerUnstructuredEverywhere flags heavy use of unstructured.Unstructured
// in functions that could use typed objects. Heuristic: many composite literals or
// declarations of Unstructured within a file.
var AnalyzerUnstructuredEverywhere = &analysis.Analyzer{
	Name:     "unstructuredeverywhere",
	Doc:      "flags pervasive use of unstructured objects instead of typed",
	Run:      runUnstructuredEverywhere,
	Requires: []*analysis.Analyzer{insppass.Analyzer},
}

func runUnstructuredEverywhere(pass *analysis.Pass) (any, error) {
	insp := pass.ResultOf[insppass.Analyzer].(*inspector.Inspector)
	count := 0
	insp.Preorder([]ast.Node{(*ast.CompositeLit)(nil), (*ast.CallExpr)(nil)}, func(n ast.Node) {
		switch x := n.(type) {
		case *ast.CompositeLit:
			if se, ok := x.Type.(*ast.SelectorExpr); ok && se.Sel != nil {
				if se.Sel.Name == "Unstructured" {
					count++
				}
			}
			if id, ok := x.Type.(*ast.Ident); ok {
				if id.Name == "Unstructured" {
					count++
				}
			}
		case *ast.CallExpr:
			if se, ok := x.Fun.(*ast.SelectorExpr); ok && se.Sel != nil {
				if se.Sel.Name == "Unstructured" {
					count++
				}
			}
		}
	})
	if count >= 3 {
		// Report once per package
		for _, f := range pass.Files {
			pass.Reportf(f.Package, "Heavy use of unstructured.Unstructured; prefer typed clients/objects when possible")
			break
		}
	}
	return nil, nil
}
