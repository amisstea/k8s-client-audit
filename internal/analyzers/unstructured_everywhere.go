package analyzers

import (
	"go/ast"
	"go/types"
	"strings"

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

	// Check if a type is a Kubernetes Unstructured object
	isKubernetesUnstructured := func(t types.Type) bool {
		if named, ok := t.(*types.Named); ok {
			if named.Obj() != nil && named.Obj().Pkg() != nil {
				pkg := named.Obj().Pkg().Path()
				name := named.Obj().Name()

				// Check for Kubernetes Unstructured types
				if name == "Unstructured" {
					switch {
					case pkg == "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured":
						return true
					}
				}
			}
		}
		return false
	}

	// Check if a function call creates/uses Kubernetes Unstructured objects
	isKubernetesUnstructuredCall := func(obj types.Object) bool {
		if obj == nil || obj.Pkg() == nil {
			return false
		}
		name := obj.Name()
		pkg := obj.Pkg().Path()

		// Check for Unstructured-related function calls
		if name == "Unstructured" || strings.Contains(name, "Unstructured") {
			switch {
			case strings.HasPrefix(pkg, "k8s.io/apimachinery") && strings.Contains(pkg, "unstructured"):
				return true
			case strings.HasPrefix(pkg, "k8s.io/") && strings.Contains(pkg, "unstructured"):
				return true
			}
		}
		return false
	}

	count := 0
	insp.Preorder([]ast.Node{(*ast.CompositeLit)(nil), (*ast.CallExpr)(nil)}, func(n ast.Node) {
		switch x := n.(type) {
		case *ast.CompositeLit:
			// Use type information to verify this is actually a Kubernetes Unstructured
			if t := pass.TypesInfo.TypeOf(x); t != nil && isKubernetesUnstructured(t) {
				count++
			}
		case *ast.CallExpr:
			// Check if this is a call to Kubernetes Unstructured constructors/methods
			if se, ok := x.Fun.(*ast.SelectorExpr); ok && se.Sel != nil {
				if obj := pass.TypesInfo.Uses[se.Sel]; obj != nil && isKubernetesUnstructuredCall(obj) {
					count++
				}
			} else if id, ok := x.Fun.(*ast.Ident); ok {
				if obj := pass.TypesInfo.Uses[id]; obj != nil && isKubernetesUnstructuredCall(obj) {
					count++
				}
			}
		}
	})

	if count >= 3 {
		// Report once per package
		for _, f := range pass.Files {
			pass.Reportf(f.Package, "Heavy use of Kubernetes unstructured.Unstructured; prefer typed clients/objects when possible")
			break
		}
	}
	return nil, nil
}
