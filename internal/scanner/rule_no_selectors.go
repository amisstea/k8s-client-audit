package scanner

import (
	"context"
	"go/ast"
	"go/token"

	"golang.org/x/tools/go/packages"
)

// Detect List calls without label/field selectors or unscoped controller-runtime List with no options.
type ruleNoSelectors struct{}

func NewRuleNoSelectors() Rule        { return &ruleNoSelectors{} }
func (r *ruleNoSelectors) ID() string { return RuleNoLabelSelectorID }
func (r *ruleNoSelectors) Description() string {
	return "List calls should use label/field selectors where possible"
}

func (r *ruleNoSelectors) Apply(ctx context.Context, fset *token.FileSet, pkg *packages.Package) ([]Issue, error) {
	var issues []Issue
	for _, f := range pkg.Syntax {
		ast.Inspect(f, func(n ast.Node) bool {
			ce, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := ce.Fun.(*ast.SelectorExpr)
			if !ok || sel.Sel == nil || sel.Sel.Name != "List" {
				return true
			}
			// Case 1: controller-runtime client.List(ctx, obj, opts...)
			// Inspect opts for MatchingLabels/Fields and their Selector variants.
			if len(ce.Args) <= 2 {
				pos := fset.Position(sel.Sel.Pos())
				issues = append(issues, Issue{
					RuleID:      r.ID(),
					Title:       "controller-runtime List without options",
					Description: "Consider scoping List with namespace, labels or fields to reduce load",
					Position:    Position{Filename: pos.Filename, Line: pos.Line, Column: pos.Column},
					Suggestion:  "Use client.InNamespace, client.MatchingLabels, or client.MatchingFields",
				})
				return true
			}
			// Recognize opts by call, composite literals, or identifier types via types.Info
			hasLabelOpt := false
			hasFieldOpt := false
			if pkg.TypesInfo != nil {
				hasLabelOpt, hasFieldOpt = findCRMatchOptions(ce.Args[2:], pkg.TypesInfo)
			} else {
				hasLabelOpt, hasFieldOpt = findCRMatchOptions(ce.Args[2:], nil)
			}
			if !(hasLabelOpt || hasFieldOpt) {
				pos := fset.Position(sel.Sel.Pos())
				issues = append(issues, Issue{
					RuleID:      r.ID(),
					Title:       "List without label/field selectors",
					Description: "Unfiltered List may be expensive; add MatchingLabels/MatchingFields where feasible",
					Position:    Position{Filename: pos.Filename, Line: pos.Line, Column: pos.Column},
					Suggestion:  "Use client.MatchingLabels/MatchingFields or their *Selector variants",
				})
				return true
			}
			// Case 2: client-go typed List(ctx, metav1.ListOptions{...}). If a composite literal is provided, check for keys.
			if len(ce.Args) >= 2 {
				if lit, ok := ce.Args[len(ce.Args)-1].(*ast.CompositeLit); ok {
					// Only apply this branch if the type is ListOptions (selector keys live there)
					typeName := ""
					switch t := lit.Type.(type) {
					case *ast.Ident:
						typeName = t.Name
					case *ast.SelectorExpr:
						if t.Sel != nil {
							typeName = t.Sel.Name
						}
					}
					if typeName == "ListOptions" {
						hasLabel := false
						hasField := false
						for _, el := range lit.Elts {
							if kv, ok := el.(*ast.KeyValueExpr); ok {
								if ident, ok := kv.Key.(*ast.Ident); ok {
									if ident.Name == "LabelSelector" {
										hasLabel = true
									} else if ident.Name == "FieldSelector" {
										hasField = true
									}
								}
							}
						}
						if !hasLabel && !hasField {
							pos := fset.Position(sel.Sel.Pos())
							issues = append(issues, Issue{
								RuleID:      r.ID(),
								Title:       "List without label/field selectors",
								Description: "Unfiltered List may be expensive; add label or field selectors when feasible",
								Position:    Position{Filename: pos.Filename, Line: pos.Line, Column: pos.Column},
								Suggestion:  "Set metav1.ListOptions{LabelSelector: ..., FieldSelector: ...}",
							})
						}
					}
				}
			}
			return true
		})
	}
	return issues, nil
}
