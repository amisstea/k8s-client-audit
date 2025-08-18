package analyzers

import (
	"go/ast"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
	insppass "golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// AnalyzerExcessiveClusterScope flags ClusterRole/ClusterRoleBinding
// composite literals when Role/RoleBinding would suffice (heuristic).
var AnalyzerExcessiveClusterScope = &analysis.Analyzer{
	Name:     "excessiveclusterscope",
	Doc:      "flags cluster-scoped RBAC where namespace scope may suffice",
	Run:      runExcessiveClusterScope,
	Requires: []*analysis.Analyzer{insppass.Analyzer},
}

func runExcessiveClusterScope(pass *analysis.Pass) (any, error) {
	insp := pass.ResultOf[insppass.Analyzer].(*inspector.Inspector)

	// Check if a type is a Kubernetes RBAC ClusterRole or ClusterRoleBinding
	isKubernetesClusterRBAC := func(t types.Type) bool {
		if named, ok := t.(*types.Named); ok {
			if named.Obj() != nil && named.Obj().Pkg() != nil {
				pkg := named.Obj().Pkg().Path()
				name := named.Obj().Name()

				// Check for Kubernetes RBAC cluster-scoped types
				if name == "ClusterRole" || name == "ClusterRoleBinding" {
					switch {
					case pkg == "k8s.io/api/rbac/v1":
						return true
					case pkg == "k8s.io/api/rbac/v1beta1":
						return true
					}
				}
			}
		}
		return false
	}

	nodes := []ast.Node{(*ast.CompositeLit)(nil)}
	insp.Nodes(nodes, func(n ast.Node, push bool) bool {
		if !push {
			return true
		}

		cl := n.(*ast.CompositeLit)

		// Use type information to verify this is a Kubernetes ClusterRole/ClusterRoleBinding
		if t := pass.TypesInfo.TypeOf(cl); t != nil {
			if isKubernetesClusterRBAC(t) {
				pass.Reportf(cl.Lbrace, "Kubernetes cluster-scoped RBAC detected; use namespace-scoped RBAC when possible")
			}
		}

		return true
	})

	return nil, nil
}

// AnalyzerWildcardVerbs flags RBAC policy rules with verbs ["*"]
var AnalyzerWildcardVerbs = &analysis.Analyzer{
	Name:     "wildcardverbs",
	Doc:      "flags wildcard verbs in RBAC rules",
	Run:      runWildcardVerbs,
	Requires: []*analysis.Analyzer{insppass.Analyzer},
}

func runWildcardVerbs(pass *analysis.Pass) (any, error) {
	insp := pass.ResultOf[insppass.Analyzer].(*inspector.Inspector)

	// Check if a type is a Kubernetes RBAC PolicyRule or similar structure with Verbs field
	isKubernetesRBACRule := func(t types.Type) bool {
		if named, ok := t.(*types.Named); ok {
			if named.Obj() != nil && named.Obj().Pkg() != nil {
				pkg := named.Obj().Pkg().Path()
				name := named.Obj().Name()

				// Check for Kubernetes RBAC rule types
				if name == "PolicyRule" || name == "Rule" {
					switch {
					case pkg == "k8s.io/api/rbac/v1":
						return true
					case pkg == "k8s.io/api/rbac/v1beta1":
						return true
					}
				}
			}
		}

		// Also check if this might be an RBAC structure by looking at the file context
		return false
	}

	// Check if the context suggests this is Kubernetes RBAC
	isKubernetesRBACContext := func(f *ast.File) bool {
		// Look for imports that suggest this is Kubernetes RBAC-related
		for _, imp := range f.Imports {
			if imp.Path == nil {
				continue
			}
			path := strings.Trim(imp.Path.Value, `"`)
			if strings.Contains(path, "rbac") && (strings.HasPrefix(path, "k8s.io/") || strings.HasPrefix(path, "sigs.k8s.io/")) {
				return true
			}
		}
		return false
	}

	nodes := []ast.Node{(*ast.CompositeLit)(nil)}
	insp.Nodes(nodes, func(n ast.Node, push bool) bool {
		if !push {
			return true
		}

		cl := n.(*ast.CompositeLit)

		// Check if this composite literal is in a Kubernetes RBAC context
		var currentFile *ast.File
		for _, f := range pass.Files {
			if f.Pos() <= cl.Pos() && cl.End() <= f.End() {
				currentFile = f
				break
			}
		}

		// Use type information or context to determine if this is Kubernetes RBAC
		isRBACRelated := false
		if t := pass.TypesInfo.TypeOf(cl); t != nil {
			isRBACRelated = isKubernetesRBACRule(t)
		}
		if !isRBACRelated && currentFile != nil {
			isRBACRelated = isKubernetesRBACContext(currentFile)
		}

		if !isRBACRelated {
			return true
		}

		// Look for a field named Verbs: []string{"*"} in RBAC-related structures
		for _, el := range cl.Elts {
			kv, ok := el.(*ast.KeyValueExpr)
			if !ok {
				continue
			}
			if id, ok := kv.Key.(*ast.Ident); ok && id.Name == "Verbs" {
				if arr, ok := kv.Value.(*ast.CompositeLit); ok {
					for _, v := range arr.Elts {
						if bl, ok := v.(*ast.BasicLit); ok && bl.Value == "\"*\"" {
							pass.Reportf(id.Pos(), "Kubernetes RBAC rule uses wildcard verbs; restrict to specific verbs")
						}
					}
				}
			}
		}

		return true
	})

	return nil, nil
}
