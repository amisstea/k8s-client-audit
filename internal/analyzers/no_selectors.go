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
		// Analyze per-function to enable tracking of identifier initializers within scope
		for _, d := range f.Decls {
			fd, ok := d.(*ast.FuncDecl)
			if !ok || fd.Body == nil {
				continue
			}
			varInits := map[string]ast.Expr{}
			ast.Inspect(fd.Body, func(n ast.Node) bool {
				switch n := n.(type) {
				case *ast.AssignStmt:
					if len(n.Lhs) == 1 && len(n.Rhs) == 1 {
						if id, ok := n.Lhs[0].(*ast.Ident); ok {
							varInits[id.Name] = n.Rhs[0]
						}
					}
				case *ast.DeclStmt:
					if gd, ok := n.Decl.(*ast.GenDecl); ok && gd.Tok == token.VAR {
						for _, sp := range gd.Specs {
							if vs, ok := sp.(*ast.ValueSpec); ok {
								for i, name := range vs.Names {
									if i < len(vs.Values) {
										varInits[name.Name] = vs.Values[i]
									}
								}
							}
						}
					}
				case *ast.CallExpr:
					ce := n
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
								// Attempt to resolve identifier initializer in current function
								if init, ok := varInits[x.Name]; ok {
									// Handle function calls that might return selector options
									if ce, ok := init.(*ast.CallExpr); ok {
										// Check if it's a function call that might return selectors
										if sel, ok := ce.Fun.(*ast.SelectorExpr); ok && sel.Sel != nil {
											if sel.Sel.Name == "ListOption" {
												// This is likely client.ListOption(...) - conservatively assume it has selectors
												hasOpts = true
											}
										}
									}

									// Unwrap address-of
									if ue, ok := init.(*ast.UnaryExpr); ok && ue.Op == token.AND {
										init = ue.X
									}
									if cl, ok := init.(*ast.CompositeLit); ok {
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
											// Do not set hasOpts here; we want to require selectors
											break
										} else if typeName == "MatchingLabels" {
											hasLabel = true
										} else if typeName == "MatchingFields" {
											hasField = true
										}
									}
								} else {
									// Unknown identifier: conservatively treat as options to avoid false positives
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
								// support &ListOptions{...} and &identifier
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
									} else if id, ok := x.X.(*ast.Ident); ok {
										// Handle &identifier case - resolve the identifier
										if init, ok := varInits[id.Name]; ok {
											if cl, ok := init.(*ast.CompositeLit); ok {
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
													// Do not set hasOpts here; we want to require selectors
												} else if typeName == "MatchingLabels" {
													hasLabel = true
												} else if typeName == "MatchingFields" {
													hasField = true
												}
											}
										} else {
											// Unknown identifier: conservatively treat as options to avoid false positives
											hasOpts = true
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
				}
				return true
			})
		}
	}
	return nil, nil
}
