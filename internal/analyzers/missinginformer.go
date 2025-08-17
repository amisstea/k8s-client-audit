package analyzers

import (
	"go/ast"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
	insppass "golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// AnalyzerMissingInformer flags direct client-go Watch calls in packages
// that do not appear to use shared informers/caches. Prefer shared informers to
// reduce API server load and improve efficiency.
var AnalyzerMissingInformer = &analysis.Analyzer{
	Name:     "missinginformer",
	Doc:      "flags direct Watch calls when no SharedInformer is used",
	Run:      runMissingInformer,
	Requires: []*analysis.Analyzer{insppass.Analyzer},
}

func runMissingInformer(pass *analysis.Pass) (any, error) {
	insp := pass.ResultOf[insppass.Analyzer].(*inspector.Inspector)

	// Check if a method call is a Kubernetes SharedInformer constructor
	isKubernetesInformerConstructor := func(obj types.Object) bool {
		if obj == nil || obj.Pkg() == nil {
			return false
		}
		name := obj.Name()
		if !(name == "NewSharedInformerFactory" || name == "NewSharedInformerFactoryWithOptions" || name == "NewSharedIndexInformer" || name == "NewSharedInformer") {
			return false
		}
		pkg := obj.Pkg().Path()

		// Check for Kubernetes informer packages
		switch {
		case strings.HasPrefix(pkg, "k8s.io/client-go/informers"):
			return true
		case strings.HasPrefix(pkg, "k8s.io/client-go/tools/cache"):
			return true
		case strings.HasPrefix(pkg, "sigs.k8s.io/controller-runtime/pkg/cache"):
			return true
		default:
			// Check for any k8s.io or sigs.k8s.io packages
			if strings.HasPrefix(pkg, "k8s.io/") || strings.HasPrefix(pkg, "sigs.k8s.io/") {
				return true
			}
			// Check for packages containing client-go or controller-runtime
			if strings.Contains(pkg, "client-go") || strings.Contains(pkg, "controller-runtime") {
				return true
			}
		}
		return false
	}

	// Check if a method call is a Kubernetes Watch operation
	isKubernetesWatchCall := func(obj types.Object) bool {
		if obj == nil || obj.Pkg() == nil {
			return false
		}
		name := obj.Name()
		if name != "Watch" {
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

	// First pass: detect if package appears to use Kubernetes SharedInformers
	hasInformer := false
	var watchCalls []ast.Node

	nodes := []ast.Node{(*ast.CallExpr)(nil)}
	insp.Nodes(nodes, func(n ast.Node, push bool) bool {
		if !push {
			return true
		}

		ce := n.(*ast.CallExpr)

		// Check both selector expressions and identifiers for function calls
		var obj types.Object
		switch fun := ce.Fun.(type) {
		case *ast.SelectorExpr:
			if fun.Sel != nil {
				obj = pass.TypesInfo.Uses[fun.Sel]
			}
		case *ast.Ident:
			obj = pass.TypesInfo.Uses[fun]
		}

		if obj != nil {
			if isKubernetesInformerConstructor(obj) {
				hasInformer = true
			} else if isKubernetesWatchCall(obj) {
				watchCalls = append(watchCalls, ce)
			}
		}

		return true
	})

	// Second pass: if no Kubernetes SharedInformer usage detected, report direct Watch calls
	if !hasInformer {
		for _, call := range watchCalls {
			pass.Reportf(call.Pos(), "Direct Kubernetes Watch call detected with no SharedInformer in package; prefer shared informers (client-go informers/cache)")
		}
	}

	return nil, nil
}
