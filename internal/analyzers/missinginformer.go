package analyzers

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
)

// AnalyzerMissingInformer (K8S010) flags direct client-go Watch calls in packages
// that do not appear to use shared informers/caches. Prefer shared informers to
// reduce API server load and improve efficiency.
var AnalyzerMissingInformer = &analysis.Analyzer{
	Name: "k8s010_missinginformer",
	Doc:  "flags direct Watch calls when no SharedInformer is used",
	Run:  runMissingInformer,
}

func runMissingInformer(pass *analysis.Pass) (any, error) {
	// First pass: detect if package appears to use informers/caches.
	hasInformer := false
	for _, f := range pass.Files {
		if hasInformer {
			break
		}
		ast.Inspect(f, func(n ast.Node) bool {
			ce, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			switch fun := ce.Fun.(type) {
			case *ast.SelectorExpr:
				if fun.Sel == nil {
					return true
				}
				name := fun.Sel.Name
				if name == "NewSharedInformerFactory" || name == "NewSharedInformerFactoryWithOptions" || name == "NewSharedIndexInformer" || name == "NewSharedInformer" {
					hasInformer = true
					return false
				}
			case *ast.Ident:
				switch fun.Name {
				case "NewSharedInformerFactory", "NewSharedInformerFactoryWithOptions", "NewSharedIndexInformer", "NewSharedInformer":
					hasInformer = true
					return false
				}
			}
			return true
		})
	}

	// Second pass: if no informer usage detected, report direct typed Watch calls.
	if !hasInformer {
		for _, f := range pass.Files {
			ast.Inspect(f, func(n ast.Node) bool {
				ce, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				if sel, ok := ce.Fun.(*ast.SelectorExpr); ok && sel.Sel != nil {
					if sel.Sel.Name == "Watch" {
						pass.Reportf(sel.Sel.Pos(), "Direct Watch call detected with no SharedInformer in package; prefer shared informers (client-go informers/cache)")
					}
				}
				return true
			})
		}
	}
	return nil, nil
}
