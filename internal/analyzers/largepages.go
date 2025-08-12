package analyzers

import (
	"go/ast"
	"go/constant"

	"golang.org/x/tools/go/analysis"
)

// AnalyzerLargePageSizes (K8S023) flags ListOptions with very large Limit values.
var AnalyzerLargePageSizes = &analysis.Analyzer{
	Name: "k8s023_largepages",
	Doc:  "flags excessively large page sizes in list calls",
	Run:  runLargePages,
}

const defaultLargePageThreshold = 1000

func runLargePages(pass *analysis.Pass) (any, error) {
	for _, f := range pass.Files {
		ast.Inspect(f, func(n ast.Node) bool {
			ce, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			// Look for List(..., ListOptions{Limit: N})
			if sel, ok := ce.Fun.(*ast.SelectorExpr); ok && sel.Sel != nil && sel.Sel.Name == "List" {
				for _, arg := range ce.Args {
					cl, ok := arg.(*ast.CompositeLit)
					if !ok {
						continue
					}
					// type may be SelectorExpr or Ident
					switch t := cl.Type.(type) {
					case *ast.Ident:
						if t.Name != "ListOptions" {
							continue
						}
					case *ast.SelectorExpr:
						if t.Sel == nil || t.Sel.Name != "ListOptions" {
							continue
						}
					default:
						continue
					}
					// find Limit key value
					for _, el := range cl.Elts {
						if kv, ok := el.(*ast.KeyValueExpr); ok {
							if id, ok := kv.Key.(*ast.Ident); ok && id.Name == "Limit" {
								if tv := pass.TypesInfo.Types[kv.Value]; tv.Value != nil {
									if v, ok := constant.Int64Val(tv.Value); ok {
										if v >= defaultLargePageThreshold {
											pass.Reportf(id.Pos(), "ListOptions.Limit is very large (%d); use reasonable page sizes", v)
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
	}
	return nil, nil
}
