package scanner

import (
	"context"
	"go/ast"
	"go/token"

	"golang.org/x/tools/go/packages"
)

// Detect List calls that do not specify any field selector option.
// This is more specific than K8S021 (no selectors) and fires when label selectors
// are present but field selectors are not.
type ruleNoFieldSelector struct{}

func NewRuleNoFieldSelector() Rule        { return &ruleNoFieldSelector{} }
func (r *ruleNoFieldSelector) ID() string { return RuleNoFieldSelectorID }
func (r *ruleNoFieldSelector) Description() string {
	return "List calls should consider using field selectors when feasible"
}

func (r *ruleNoFieldSelector) Apply(ctx context.Context, fset *token.FileSet, pkg *packages.Package) ([]Issue, error) {
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
			// Two families:
			// 1) controller-runtime: List(ctx, obj, opts...) where opts may include MatchingFields/MatchingFieldsSelector
			// 2) client-go typed: List(ctx, metav1.ListOptions{ FieldSelector: ... })

			// controller-runtime style: any of the variadic opts is a call to MatchingFields or MatchingFieldsSelector
			hasFieldOpt := false
			for _, arg := range ce.Args {
				switch a := arg.(type) {
				case *ast.CallExpr:
					switch fun := a.Fun.(type) {
					case *ast.SelectorExpr:
						if fun.Sel != nil && (fun.Sel.Name == "MatchingFields" || fun.Sel.Name == "MatchingFieldsSelector") {
							hasFieldOpt = true
						}
					case *ast.Ident:
						if fun.Name == "MatchingFields" || fun.Name == "MatchingFieldsSelector" {
							hasFieldOpt = true
						}
					}
				}
			}

			// client-go typed style: last arg could be a composite literal with FieldSelector
			if !hasFieldOpt {
				if len(ce.Args) >= 2 {
					if lit, ok := ce.Args[len(ce.Args)-1].(*ast.CompositeLit); ok {
						for _, el := range lit.Elts {
							if kv, ok := el.(*ast.KeyValueExpr); ok {
								if ident, ok := kv.Key.(*ast.Ident); ok && ident.Name == "FieldSelector" {
									// FieldSelector key present (regardless of empty/non-empty). Consider present.
									hasFieldOpt = true
									break
								}
							}
						}
					}
				}
			}

			if !hasFieldOpt {
				pos := fset.Position(sel.Sel.Pos())
				issues = append(issues, Issue{
					RuleID:      r.ID(),
					Title:       "List without field selector",
					Description: "Consider using field selectors (e.g., name or namespace) to scope List calls when feasible",
					Severity:    SeverityWarning,
					PackagePath: pkg.PkgPath,
					Position:    Position{Filename: pos.Filename, Line: pos.Line, Column: pos.Column},
					Suggestion:  "Use client.MatchingFields/MatchingFieldsSelector or set ListOptions.FieldSelector",
				})
			}
			return true
		})
	}
	return issues, nil
}
