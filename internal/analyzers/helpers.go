package analyzers

import (
	"go/ast"
	"go/types"
	"strings"
)

// calleeIdent returns the identifier for a call expression's callee, handling
// both simple identifiers and selector expressions. Returns nil if unresolved.
func calleeIdent(expr ast.Expr) *ast.Ident {
	switch x := expr.(type) {
	case *ast.Ident:
		return x
	case *ast.SelectorExpr:
		if x.Sel != nil {
			return x.Sel
		}
	}
	return nil
}

// deref returns the non-pointer type for a given type.
func deref(t types.Type) types.Type {
	if p, ok := t.(*types.Pointer); ok {
		return p.Elem()
	}
	return t
}

// isNamed returns true if t is a named type whose package path and name match.
func isNamed(t types.Type, pkgPath, name string) bool {
	if n, ok := t.(*types.Named); ok {
		if n.Obj() != nil && n.Obj().Pkg() != nil {
			return n.Obj().Pkg().Path() == pkgPath && n.Obj().Name() == name
		}
	}
	return false
}

// Common Kubernetes package paths
const (
	PkgControllerRuntimeClient    = "sigs.k8s.io/controller-runtime/pkg/client"
	PkgControllerRuntimeReconcile = "sigs.k8s.io/controller-runtime/pkg/reconcile"
	PkgClientGoDynamic            = "k8s.io/client-go/dynamic"
	PkgClientGoKubernetes         = "k8s.io/client-go/kubernetes"
	PkgClientGoRest               = "k8s.io/client-go/rest"
	PkgMetaV1                     = "k8s.io/apimachinery/pkg/apis/meta/v1"
	PkgClientGoDiscovery          = "k8s.io/client-go/discovery"
	PkgClientGoRestMapper         = "k8s.io/client-go/restmapper"
)

// isKubernetesPackage returns true if the package path is a known Kubernetes package.
func isKubernetesClientPackage(pkgPath string) bool {
	switch pkgPath {
	case PkgControllerRuntimeClient, PkgClientGoDynamic, PkgClientGoKubernetes, PkgClientGoRest:
		return true
	default:
		// Handle typed client packages dynamically
		if strings.HasPrefix(pkgPath, "k8s.io/client-go/kubernetes/typed/") {
			return true
		}
		return false
	}
}

// isKubernetesClientConstructor returns true if the object is a known Kubernetes client constructor.
func isKubernetesClientConstructor(obj types.Object) bool {
	if obj == nil || obj.Pkg() == nil {
		return false
	}
	pkg := obj.Pkg().Path()
	name := obj.Name()
	switch pkg {
	case PkgClientGoKubernetes:
		return name == "NewForConfig" || name == "NewForConfigOrDie"
	case PkgClientGoDynamic:
		return name == "NewForConfig"
	case PkgClientGoRest:
		return name == "RESTClientFor"
	case PkgControllerRuntimeClient:
		return name == "New"
	default:
		return false
	}
}

// isKubernetesMethodCall returns true if the object represents a method call from a Kubernetes client
// with the specified method name(s).
func isKubernetesMethodCall(obj types.Object, methodNames ...string) bool {
	if obj == nil || obj.Pkg() == nil {
		return false
	}

	name := obj.Name()
	found := false
	for _, methodName := range methodNames {
		if name == methodName {
			found = true
			break
		}
	}

	if !found {
		return false
	}

	pkg := obj.Pkg().Path()
	return isKubernetesClientPackage(pkg)
}

// isKubernetesType returns true if the type is a known Kubernetes type with the specified name(s).
func isKubernetesType(t types.Type, typeNames ...string) bool {
	named, ok := t.(*types.Named)
	if !ok || named.Obj() == nil || named.Obj().Pkg() == nil {
		return false
	}

	pkg := named.Obj().Pkg().Path()
	name := named.Obj().Name()

	if isKubernetesClientPackage(pkg) {
		for _, typeName := range typeNames {
			if name == typeName {
				return true
			}
		}
	}

	return false
}

// isKubernetesListOptions returns true if the type is a Kubernetes ListOptions type.
func isKubernetesListOptions(t types.Type) bool {
	return isKubernetesType(t, "ListOptions") || isNamed(t, PkgMetaV1, "ListOptions")
}
