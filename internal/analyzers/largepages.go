package analyzers

import (
	"go/ast"
	"go/constant"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
	insppass "golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// AnalyzerLargePageSizes flags ListOptions with very large Limit values.
var AnalyzerLargePageSizes = &analysis.Analyzer{
	Name:     "largepages",
	Doc:      "flags excessively large page sizes in list calls",
	Run:      runLargePages,
	Requires: []*analysis.Analyzer{insppass.Analyzer},
}

const defaultLargePageThreshold = 1000

func runLargePages(pass *analysis.Pass) (any, error) {
	insp := pass.ResultOf[insppass.Analyzer].(*inspector.Inspector)

	// Check if a method call is a Kubernetes List operation
	isKubernetesListCall := func(obj types.Object) bool {
		if obj == nil || obj.Pkg() == nil {
			return false
		}
		name := obj.Name()
		if name != "List" {
			return false
		}
		pkg := obj.Pkg().Path()

		// Check for Kubernetes-related packages
		switch {
		case pkg == "sigs.k8s.io/controller-runtime/pkg/client":
			return true
		case pkg == "k8s.io/client-go/dynamic":
			return true
		default:
			// Check for any k8s.io or sigs.k8s.io packages
			if strings.HasPrefix(pkg, "k8s.io/") || strings.HasPrefix(pkg, "sigs.k8s.io/") {
				return true
			}
			// Check for packages containing client-go, controller-runtime, or apimachinery
			if strings.Contains(pkg, "client-go") || strings.Contains(pkg, "controller-runtime") || strings.Contains(pkg, "apimachinery") {
				return true
			}
		}
		return false
	}

	// Check if a type is a Kubernetes ListOptions
	isKubernetesListOptions := func(t types.Type) bool {
		if named, ok := t.(*types.Named); ok {
			if named.Obj() != nil && named.Obj().Pkg() != nil {
				pkg := named.Obj().Pkg().Path()
				name := named.Obj().Name()

				// Check for Kubernetes ListOptions types
				if name == "ListOptions" {
					switch {
					case pkg == "k8s.io/apimachinery/pkg/apis/meta/v1":
						return true
					case pkg == "sigs.k8s.io/controller-runtime/pkg/client":
						return true
					case strings.HasPrefix(pkg, "k8s.io/") && strings.Contains(pkg, "meta"):
						return true
					case strings.HasPrefix(pkg, "k8s.io/") && strings.Contains(pkg, "apimachinery"):
						return true
					}
				}
			}
		}
		return false
	}

	nodes := []ast.Node{(*ast.CallExpr)(nil)}
	insp.Nodes(nodes, func(n ast.Node, push bool) bool {
		if !push {
			return true
		}

		ce := n.(*ast.CallExpr)

		// Check if this is a Kubernetes List call
		sel, ok := ce.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel == nil {
			return true
		}

		// Use type information to verify this is a Kubernetes List call
		if obj := pass.TypesInfo.Uses[sel.Sel]; obj != nil && isKubernetesListCall(obj) {
			// Look for ListOptions in the arguments
			for _, arg := range ce.Args {
				cl, ok := arg.(*ast.CompositeLit)
				if !ok {
					continue
				}

				// Use type information to verify this is a Kubernetes ListOptions
				if t := pass.TypesInfo.TypeOf(cl); t != nil && isKubernetesListOptions(t) {
					// Look for Limit field with large values
					for _, el := range cl.Elts {
						if kv, ok := el.(*ast.KeyValueExpr); ok {
							if id, ok := kv.Key.(*ast.Ident); ok && id.Name == "Limit" {
								if tv := pass.TypesInfo.Types[kv.Value]; tv.Value != nil {
									if v, ok := constant.Int64Val(tv.Value); ok {
										if v >= defaultLargePageThreshold {
											pass.Reportf(id.Pos(), "Kubernetes ListOptions.Limit is very large (%d); use reasonable page sizes", v)
										}
									}
								}
							}
						}
					}
				}
			}
		}

		return true
	})

	return nil, nil
}
