package analyzers

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
)

// AnalyzerExcessiveClusterScope (K8S080) flags ClusterRole/ClusterRoleBinding
// composite literals when Role/RoleBinding would suffice (heuristic).
var AnalyzerExcessiveClusterScope = &analysis.Analyzer{
	Name: "k8s080_excessiveclusterscope",
	Doc:  "flags cluster-scoped RBAC where namespace scope may suffice",
	Run:  runExcessiveClusterScope,
}

func runExcessiveClusterScope(pass *analysis.Pass) (any, error) {
	for _, f := range pass.Files {
		ast.Inspect(f, func(n ast.Node) bool {
			cl, ok := n.(*ast.CompositeLit)
			if !ok {
				return true
			}
			if id, ok := cl.Type.(*ast.Ident); ok {
				if id.Name == "ClusterRole" || id.Name == "ClusterRoleBinding" {
					pass.Reportf(cl.Lbrace, "Cluster-scoped RBAC detected; use namespace-scoped RBAC when possible")
				}
			}
			if se, ok := cl.Type.(*ast.SelectorExpr); ok && se.Sel != nil {
				if se.Sel.Name == "ClusterRole" || se.Sel.Name == "ClusterRoleBinding" {
					pass.Reportf(cl.Lbrace, "Cluster-scoped RBAC detected; use namespace-scoped RBAC when possible")
				}
			}
			return true
		})
	}
	return nil, nil
}

// AnalyzerWildcardVerbs (K8S081) flags RBAC policy rules with verbs ["*"]
var AnalyzerWildcardVerbs = &analysis.Analyzer{
	Name: "k8s081_wildcardverbs",
	Doc:  "flags wildcard verbs in RBAC rules",
	Run:  runWildcardVerbs,
}

func runWildcardVerbs(pass *analysis.Pass) (any, error) {
	for _, f := range pass.Files {
		ast.Inspect(f, func(n ast.Node) bool {
			cl, ok := n.(*ast.CompositeLit)
			if !ok {
				return true
			}
			// Look for a field named Verbs: []string{"*"}
			for _, el := range cl.Elts {
				kv, ok := el.(*ast.KeyValueExpr)
				if !ok {
					continue
				}
				if id, ok := kv.Key.(*ast.Ident); ok && id.Name == "Verbs" {
					if arr, ok := kv.Value.(*ast.CompositeLit); ok {
						for _, v := range arr.Elts {
							if bl, ok := v.(*ast.BasicLit); ok && bl.Value == "\"*\"" {
								pass.Reportf(id.Pos(), "RBAC rule uses wildcard verbs; restrict to specific verbs")
							}
						}
					}
				}
			}
			return true
		})
	}
	return nil, nil
}
