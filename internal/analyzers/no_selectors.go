package analyzers

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
)

// AnalyzerNoSelectors flags List calls without label/field selectors or options.
var AnalyzerNoSelectors = &analysis.Analyzer{
	Name: "k8s021_noselectors",
	Doc:  "flags List calls without label/field selectors",
	Run:  runNoSelectors,
}

func runNoSelectors(pass *analysis.Pass) (any, error) {
	for _, f := range pass.Files {
		ast.Inspect(f, func(n ast.Node) bool {
			ce, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := ce.Fun.(*ast.SelectorExpr)
			if !ok || sel.Sel == nil {
				return true
			}
			if sel.Sel.Name != "List" {
				return true
			}
			// If only ctx and obj are provided in controller-runtime style, flag missing opts
			if len(ce.Args) <= 2 {
				pass.Reportf(sel.Sel.Pos(), "List without options; provide MatchingLabels/Fields or scope namespace")
				return true
			}
			// client-go typed style: List(ctx, ListOptions{...})
			if len(ce.Args) == 2 {
				if cl, ok := ce.Args[1].(*ast.CompositeLit); ok {
					typeName := ""
					switch t := cl.Type.(type) {
					case *ast.Ident:
						typeName = t.Name
					case *ast.SelectorExpr:
						if t.Sel != nil {
							typeName = t.Sel.Name
						}
					}
					if typeName == "ListOptions" {
						hasLabel, hasField := false, false
						for _, el := range cl.Elts {
							if kv, ok := el.(*ast.KeyValueExpr); ok {
								if k, ok := kv.Key.(*ast.Ident); ok {
									if k.Name == "LabelSelector" {
										hasLabel = true
									}
									if k.Name == "FieldSelector" {
										hasField = true
									}
								}
							}
						}
						if !(hasLabel || hasField) {
							pass.Reportf(sel.Sel.Pos(), "ListOptions without LabelSelector/FieldSelector; add selectors to reduce load")
						}
						return true
					}
				}
			}
			// controller-runtime style opts: look for MatchingLabels/Fields and *Selector variants
			hasLabel, hasField := false, false
			if len(ce.Args) >= 3 {
				for _, a := range ce.Args[2:] {
					switch x := a.(type) {
					case *ast.CallExpr:
						if s, ok := x.Fun.(*ast.SelectorExpr); ok && s.Sel != nil {
							switch s.Sel.Name {
							case "MatchingLabels", "MatchingLabelsSelector":
								hasLabel = true
							case "MatchingFields", "MatchingFieldsSelector":
								hasField = true
							}
						}
						if id, ok := x.Fun.(*ast.Ident); ok {
							switch id.Name {
							case "MatchingLabels", "MatchingLabelsSelector":
								hasLabel = true
							case "MatchingFields", "MatchingFieldsSelector":
								hasField = true
							}
						}
					case *ast.Ident:
						// ident form: MatchingLabels without package qualifier
						switch x.Name {
						case "MatchingLabels", "MatchingLabelsSelector":
							hasLabel = true
						case "MatchingFields", "MatchingFieldsSelector":
							hasField = true
						}
					case *ast.CompositeLit:
						// client-go metav1.ListOptions
						if ident, ok := x.Type.(*ast.Ident); ok && ident.Name == "ListOptions" {
							for _, el := range x.Elts {
								if kv, ok := el.(*ast.KeyValueExpr); ok {
									if k, ok := kv.Key.(*ast.Ident); ok {
										if k.Name == "LabelSelector" {
											hasLabel = true
										}
										if k.Name == "FieldSelector" {
											hasField = true
										}
									}
								}
							}
						}
					}
				}
			}
			if len(ce.Args) >= 3 && !(hasLabel || hasField) {
				pass.Reportf(sel.Sel.Pos(), "List without label/field selectors; add MatchingLabels/Fields or set ListOptions selectors")
			}
			return true
		})
	}
	return nil, nil
}
