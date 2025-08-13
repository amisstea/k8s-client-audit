package analyzers

import (
	"go/ast"
	"go/token"

	"golang.org/x/tools/go/analysis"
)

// AnalyzerNoSelectors flags List calls without label/field selectors or options.
var AnalyzerNoSelectors = &analysis.Analyzer{
	Name: "noselectors",
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
			// Distinguish signatures by arg count first
			if len(ce.Args) == 2 {
				// typed client-go style: List(ctx, metav1.ListOptions{...}) or pointer to it
				// Accept CompositeLit or address-of CompositeLit
				checkLit := func(cl *ast.CompositeLit) bool {
					typeName := ""
					switch t := cl.Type.(type) {
					case *ast.Ident:
						typeName = t.Name
					case *ast.SelectorExpr:
						if t.Sel != nil {
							typeName = t.Sel.Name
						}
					}
					if typeName != "ListOptions" {
						return false
					}
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
				if cl, ok := ce.Args[1].(*ast.CompositeLit); ok {
					if checkLit(cl) {
						return true
					}
				}
				if ue, ok := ce.Args[1].(*ast.UnaryExpr); ok && ue.Op == token.AND {
					if cl, ok := ue.X.(*ast.CompositeLit); ok {
						if checkLit(cl) {
							return true
						}
					}
				}
				// Two-arg controller-runtime style (missing opts): flag
				pass.Reportf(sel.Sel.Pos(), "List without options; provide MatchingLabels/Fields or scope namespace")
				return true
			}
			// controller-runtime style opts: look for MatchingLabels/Fields and *Selector variants
			hasLabel, hasField := false, false
			hasOpts := false
			if len(ce.Args) >= 3 {
				// If opts are passed via a variadic slice (opts...), we conservatively skip to avoid false positives
				if ce.Ellipsis != 0 {
					if _, isIdent := ce.Args[len(ce.Args)-1].(*ast.Ident); isIdent {
						return true
					}
				}
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
						default:
							// Any identifier in opts position is considered options; avoid false positives
							hasOpts = true
						}
					case *ast.CompositeLit:
						// client-go or controller-runtime ListOptions {...}
						cl := x
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
						} else if typeName == "MatchingLabels" {
							hasLabel = true
						} else if typeName == "MatchingFields" {
							hasField = true
						}
					case *ast.UnaryExpr:
						// support &ListOptions{...}
						if x.Op == token.AND {
							if cl, ok := x.X.(*ast.CompositeLit); ok {
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
								} else if typeName == "MatchingLabels" {
									hasLabel = true
								} else if typeName == "MatchingFields" {
									hasField = true
								}
							}
						}
					}
				}
			}
			if len(ce.Args) >= 3 && !(hasLabel || hasField || hasOpts) {
				pass.Reportf(sel.Sel.Pos(), "List without label/field selectors; add MatchingLabels/Fields or set ListOptions selectors")
			}
			return true
		})
	}
	return nil, nil
}
