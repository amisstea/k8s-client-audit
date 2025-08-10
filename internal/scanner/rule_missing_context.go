package scanner

import (
	"context"
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/packages"
)

// Detect client calls without passing context where APIs expect a context.Context.
type ruleMissingContext struct{}

func NewRuleMissingContext() Rule        { return &ruleMissingContext{} }
func (r *ruleMissingContext) ID() string { return RuleNoContextCancellationID }
func (r *ruleMissingContext) Description() string {
	return "Client calls should accept and propagate context.Context"
}

func (r *ruleMissingContext) Apply(ctx context.Context, fset *token.FileSet, pkg *packages.Package) ([]Issue, error) {
	var issues []Issue
	for _, f := range pkg.Syntax {
		ast.Inspect(f, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			// Heuristic: known client method names with first argument being context.Background/TODO
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			name := sel.Sel.Name
			if !(name == "Get" || name == "List" || name == "Create" || name == "Update" || name == "Patch" || name == "Delete") {
				return true
			}
			if len(call.Args) == 0 {
				return true
			}
			// First arg is a call?
			if sub, ok := call.Args[0].(*ast.CallExpr); ok {
				if s2, ok := sub.Fun.(*ast.SelectorExpr); ok {
					if ident, ok := s2.X.(*ast.Ident); ok {
						if obj := pkg.TypesInfo.Uses[ident]; obj != nil {
							if p, ok := obj.(*types.PkgName); ok && p.Imported().Path() == "context" {
								if s2.Sel.Name == "Background" || s2.Sel.Name == "TODO" {
									pos := fset.Position(sel.Sel.Pos())
									issues = append(issues, Issue{
										RuleID:      r.ID(),
										Title:       "Client call uses context.Background/TODO",
										Description: "Propagate caller's context instead of using context.Background/TODO to honor deadlines and cancellation",
										Severity:    SeverityWarning,
										PackagePath: pkg.PkgPath,
										Position:    Position{Filename: pos.Filename, Line: pos.Line, Column: pos.Column},
										Suggestion:  "Accept a context in the surrounding function and pass it through",
									})
								}
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
