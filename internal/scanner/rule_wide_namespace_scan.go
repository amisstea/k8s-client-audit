package scanner

import (
	"context"
	"go/ast"
	"go/token"

	"golang.org/x/tools/go/packages"
)

// Detect cluster-wide scans when namespace-scoped is likely sufficient.
// Heuristic: usage of AllNamespaces() in controller-runtime or empty namespace in typed clients.
type ruleWideNamespace struct{}

func NewRuleWideNamespace() Rule        { return &ruleWideNamespace{} }
func (r *ruleWideNamespace) ID() string { return RuleWideNamespaceScansID }
func (r *ruleWideNamespace) Description() string {
	return "Avoid cluster-wide scans when namespace-scoped suffices"
}

func (r *ruleWideNamespace) Apply(ctx context.Context, fset *token.FileSet, pkg *packages.Package) ([]Issue, error) {
	var issues []Issue
	for _, f := range pkg.Syntax {
		ast.Inspect(f, func(n ast.Node) bool {
			ce, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			// Match something like client.List(ctx, obj, client.InNamespace("") )
			// or direct InNamespace("") when unqualified in local package (test harness)
			switch fun := ce.Fun.(type) {
			case *ast.SelectorExpr:
				if fun.Sel != nil && fun.Sel.Name == "InNamespace" {
					if len(ce.Args) == 1 {
						if bl, ok := ce.Args[0].(*ast.BasicLit); ok {
							if bl.Value == "\"\"" {
								pos := fset.Position(fun.Sel.Pos())
								issues = append(issues, Issue{
									RuleID:      r.ID(),
									Title:       "All-namespaces list",
									Description: "Listing across all namespaces is expensive; scope to a namespace if possible",
									PackagePath: pkg.PkgPath,
									Position:    Position{Filename: pos.Filename, Line: pos.Line, Column: pos.Column},
									Suggestion:  "Specify a concrete namespace with client.InNamespace(\"ns\")",
								})
							}
						}
					}
				}
				// If this is a List call, inspect receiver chain for typed client namespace argument of ""
				if fun.Sel != nil && fun.Sel.Name == "List" {
					if hasEmptyStringNamespaceArg(fun.X) {
						pos := fset.Position(fun.Sel.Pos())
						issues = append(issues, Issue{
							RuleID:      r.ID(),
							Title:       "All-namespaces list",
							Description: "Listing across all namespaces is expensive; scope to a namespace if possible",
							PackagePath: pkg.PkgPath,
							Position:    Position{Filename: pos.Filename, Line: pos.Line, Column: pos.Column},
							Suggestion:  "Provide a concrete namespace in typed client calls (e.g., Pods(\"ns\").List) or use namespaced clients",
						})
					}
				}
			case *ast.Ident:
				if fun.Name == "InNamespace" {
					if len(ce.Args) == 1 {
						if bl, ok := ce.Args[0].(*ast.BasicLit); ok {
							if bl.Value == "\"\"" {
								pos := fset.Position(fun.Pos())
								issues = append(issues, Issue{
									RuleID:      r.ID(),
									Title:       "All-namespaces list",
									Description: "Listing across all namespaces is expensive; scope to a namespace if possible",
									PackagePath: pkg.PkgPath,
									Position:    Position{Filename: pos.Filename, Line: pos.Line, Column: pos.Column},
									Suggestion:  "Specify a concrete namespace with client.InNamespace(\"ns\")",
								})
							}
						}
					}
				}
			}
			return true
		})
	}
	return issues, nil
}
