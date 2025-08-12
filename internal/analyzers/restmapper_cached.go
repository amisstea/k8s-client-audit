package analyzers

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
	insppass "golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// AnalyzerRESTMapperNotCached (K8S071) flags use of discovery-based RESTMapper
// without caching wrapper. Heuristic: direct NewDiscoveryRESTMapper or NewDeferredDiscoveryRESTMapper
// without surrounding NewShortcutExpander or cached wrapper elsewhere in package.
var AnalyzerRESTMapperNotCached = &analysis.Analyzer{
	Name:     "k8s071_restmapper_not_cached",
	Doc:      "flags RESTMapper without caching",
	Run:      runRESTMapperNotCached,
	Requires: []*analysis.Analyzer{insppass.Analyzer},
}

func runRESTMapperNotCached(pass *analysis.Pass) (any, error) {
	insp := pass.ResultOf[insppass.Analyzer].(*inspector.Inspector)
	hasCacheWrapper := false
	insp.Preorder([]ast.Node{(*ast.CallExpr)(nil)}, func(n ast.Node) {
		ce := n.(*ast.CallExpr)
		if id := calleeIdent(ce.Fun); id != nil {
			if obj := pass.TypesInfo.Uses[id]; obj != nil && obj.Pkg() != nil {
				if obj.Name() == "NewShortcutExpander" && obj.Pkg().Path() == "k8s.io/client-go/restmapper" {
					hasCacheWrapper = true
				}
				if obj.Name() == "NewCachedDiscoveryClient" && obj.Pkg().Path() == "k8s.io/client-go/discovery/cached" {
					hasCacheWrapper = true
				}
			}
		}
	})
	insp.Preorder([]ast.Node{(*ast.CallExpr)(nil)}, func(n ast.Node) {
		ce := n.(*ast.CallExpr)
		if id := calleeIdent(ce.Fun); id != nil {
			if obj := pass.TypesInfo.Uses[id]; obj != nil && obj.Pkg() != nil {
				if (obj.Name() == "NewDeferredDiscoveryRESTMapper" || obj.Name() == "NewDiscoveryRESTMapper") && obj.Pkg().Path() == "k8s.io/client-go/restmapper" {
					if !hasCacheWrapper {
						pass.Reportf(id.Pos(), "RESTMapper created without caching; prefer deferred/cached RESTMapper wrappers")
					}
				}
			}
		}
	})
	return nil, nil
}
