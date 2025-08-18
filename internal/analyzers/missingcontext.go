package analyzers

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
	insppass "golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// AnalyzerMissingContext flags client calls that pass context.Background/TODO instead of a propagated context.
var AnalyzerMissingContext = &analysis.Analyzer{
	Name:     "missingcontext",
	Doc:      "flags client calls using context.Background/TODO instead of propagated context",
	Run:      runMissingContext,
	Requires: []*analysis.Analyzer{insppass.Analyzer},
}

func runMissingContext(pass *analysis.Pass) (any, error) {
	insp := pass.ResultOf[insppass.Analyzer].(*inspector.Inspector)

	// Check if a method call is from a Kubernetes-related package
	isKubernetesClientMethod := func(obj types.Object) bool {
		if obj == nil || obj.Pkg() == nil {
			return false
		}
		pkg := obj.Pkg().Path()
		name := obj.Name()

		// Only flag specific method names
		if !(name == "Get" || name == "List" || name == "Create" || name == "Update" || name == "Patch" || name == "Delete") {
			return false
		}

		// Check for Kubernetes-related packages
		switch {
		case pkg == "k8s.io/client-go/kubernetes/typed/apps/v1" ||
			pkg == "k8s.io/client-go/kubernetes/typed/core/v1" ||
			pkg == "k8s.io/client-go/kubernetes/typed/batch/v1" ||
			pkg == "k8s.io/client-go/kubernetes/typed/networking/v1" ||
			pkg == "k8s.io/client-go/kubernetes/typed/rbac/v1" ||
			pkg == "sigs.k8s.io/controller-runtime/pkg/client":
			return true
		case pkg == "k8s.io/client-go/dynamic":
			// Dynamic client methods
			return name == "Get" || name == "List" || name == "Create" || name == "Update" || name == "Patch" || name == "Delete"
		}
		return false
	}

	nodes := []ast.Node{(*ast.CallExpr)(nil)}
	insp.Nodes(nodes, func(n ast.Node, push bool) bool {
		if !push {
			return true
		}

		call := n.(*ast.CallExpr)
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel == nil {
			return true
		}

		if len(call.Args) == 0 {
			return true
		}

		// Check if this is a Kubernetes client method using type information
		if obj := pass.TypesInfo.Uses[sel.Sel]; isKubernetesClientMethod(obj) {
			if isContextBackgroundOrTODO(call.Args[0]) {
				pass.Reportf(sel.Sel.Pos(), "client call uses context.Background/TODO; propagate a request context instead")
			}
		}

		return true
	})

	return nil, nil
}

func isContextBackgroundOrTODO(arg ast.Expr) bool {
	if sub, ok := arg.(*ast.CallExpr); ok {
		if s2, ok := sub.Fun.(*ast.SelectorExpr); ok {
			if ident, ok := s2.X.(*ast.Ident); ok && ident.Name == "context" {
				return s2.Sel.Name == "Background" || s2.Sel.Name == "TODO"
			}
		}
	}
	return false
}
