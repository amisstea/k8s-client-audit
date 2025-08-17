package analyzers

import (
	"go/ast"
	"go/types"
	"strings"

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

	// Check if a method call is a Kubernetes InNamespace operation
	isKubernetesInNamespace := func(obj types.Object) bool {
		if obj == nil || obj.Pkg() == nil {
			return false
		}
		name := obj.Name()
		if name != "InNamespace" {
			return false
		}
		pkg := obj.Pkg().Path()

		// Check for Kubernetes-related packages
		switch {
		case pkg == "sigs.k8s.io/controller-runtime/pkg/client":
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

	// Check if a method call is a Kubernetes List operation (for client-go pattern detection)
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
				if obj := pass.TypesInfo.Uses[fun.Sel]; isKubernetesInNamespace(obj) {
					if isEmptyString(ce.Args[0]) {
						pass.Reportf(fun.Sel.Pos(), "all-namespaces list; scope to a namespace if possible")
					}
				}
			}

			// Check for client-go style List calls with empty namespace: client.Pods("").List()
			if fun.Sel.Name == "List" {
				if obj := pass.TypesInfo.Uses[fun.Sel]; isKubernetesListCall(obj) {
					if hasEmptyStringNamespaceArg(fun.X) {
						pass.Reportf(fun.Sel.Pos(), "all-namespaces list; scope to a namespace if possible")
					}
				}
			}

		case *ast.Ident:
			// Check for standalone InNamespace("") function calls
			if fun.Name == "InNamespace" && len(ce.Args) == 1 {
				if obj := pass.TypesInfo.Uses[fun]; isKubernetesInNamespace(obj) {
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
