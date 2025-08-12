package analyzers

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
)

// AnalyzerRESTMapperNotCached (K8S071) flags use of discovery-based RESTMapper
// without caching wrapper. Heuristic: direct NewDiscoveryRESTMapper or NewDeferredDiscoveryRESTMapper
// without surrounding NewShortcutExpander or cached wrapper elsewhere in package.
var AnalyzerRESTMapperNotCached = &analysis.Analyzer{
	Name: "k8s071_restmapper_not_cached",
	Doc:  "flags RESTMapper without caching",
	Run:  runRESTMapperNotCached,
}

func runRESTMapperNotCached(pass *analysis.Pass) (any, error) {
	hasCacheWrapper := false
	for _, f := range pass.Files {
		ast.Inspect(f, func(n ast.Node) bool {
			ce, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			if sel, ok := ce.Fun.(*ast.SelectorExpr); ok && sel.Sel != nil {
				if sel.Sel.Name == "NewShortcutExpander" || sel.Sel.Name == "NewCachedDiscoveryClient" {
					hasCacheWrapper = true
				}
			}
			if id, ok := ce.Fun.(*ast.Ident); ok {
				if id.Name == "NewShortcutExpander" || id.Name == "NewCachedDiscoveryClient" {
					hasCacheWrapper = true
				}
			}
			return true
		})
	}
	for _, f := range pass.Files {
		ast.Inspect(f, func(n ast.Node) bool {
			ce, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			if sel, ok := ce.Fun.(*ast.SelectorExpr); ok && sel.Sel != nil {
				if sel.Sel.Name == "NewDeferredDiscoveryRESTMapper" || sel.Sel.Name == "NewDiscoveryRESTMapper" {
					if !hasCacheWrapper {
						pass.Reportf(sel.Sel.Pos(), "RESTMapper created without caching; prefer deferred/cached RESTMapper wrappers")
					}
				}
			}
			if id, ok := ce.Fun.(*ast.Ident); ok {
				if id.Name == "NewDeferredDiscoveryRESTMapper" || id.Name == "NewDiscoveryRESTMapper" {
					if !hasCacheWrapper {
						pass.Reportf(id.Pos(), "RESTMapper created without caching; prefer deferred/cached RESTMapper wrappers")
					}
				}
			}
			return true
		})
	}
	return nil, nil
}
