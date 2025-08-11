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
			// If there are fewer than 3 args, there are no options (ctx, obj).
			if len(ce.Args) <= 2 {
				pos := fset.Position(sel.Sel.Pos())
				issues = append(issues, Issue{
					RuleID:      r.ID(),
					Title:       "controller-runtime List without options",
					Description: "Consider scoping List with namespace, labels or fields to reduce load",
					PackagePath: pkg.PkgPath,
					Position:    Position{Filename: pos.Filename, Line: pos.Line, Column: pos.Column},
					Suggestion:  "Use client.InNamespace, client.MatchingLabels, or client.MatchingFields",
				})
				return true
			}
			// Case 2: client-go typed List(ctx, metav1.ListOptions{...}). If a composite literal is provided, check for keys.
			if len(ce.Args) >= 2 {
				if lit, ok := ce.Args[len(ce.Args)-1].(*ast.CompositeLit); ok {
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
							PackagePath: pkg.PkgPath,
							Position:    Position{Filename: pos.Filename, Line: pos.Line, Column: pos.Column},
							Suggestion:  "Set metav1.ListOptions{LabelSelector: ..., FieldSelector: ...}",
						})
					}
				}
			}
			return true
		})
	}
	return issues, nil
}
