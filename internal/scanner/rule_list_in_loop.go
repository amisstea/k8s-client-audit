package scanner

import (
	"context"
	"go/ast"
	"go/token"

	"golang.org/x/tools/go/packages"
)

// Detect List/Watch calls inside loops (for/range) which can hammer API server.
type ruleListInLoop struct{}

func NewRuleListInLoop() Rule        { return &ruleListInLoop{} }
func (r *ruleListInLoop) ID() string { return RuleDirectListWatchInLoopsID }
func (r *ruleListInLoop) Description() string {
	return "Avoid List/Watch calls inside loops; use informers or cache"
}

func (r *ruleListInLoop) Apply(ctx context.Context, fset *token.FileSet, pkg *packages.Package) ([]Issue, error) {
	var issues []Issue
	for _, f := range pkg.Syntax {
		ast.Inspect(f, func(n ast.Node) bool {
			switch loop := n.(type) {
			case *ast.ForStmt:
				issues = append(issues, r.checkLoopBody(fset, pkg, loop.Body)...)
				return true
			case *ast.RangeStmt:
				issues = append(issues, r.checkLoopBody(fset, pkg, loop.Body)...)
				return true
			}
			return true
		})
	}
	return issues, nil
}

func (r *ruleListInLoop) checkLoopBody(fset *token.FileSet, pkg *packages.Package, body *ast.BlockStmt) []Issue {
	var issues []Issue
	if body == nil {
		return nil
	}
	ast.Inspect(body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
			if sel.Sel != nil {
				if sel.Sel.Name == "List" || sel.Sel.Name == "Watch" {
					pos := fset.Position(sel.Sel.Pos())
					issues = append(issues, Issue{
						RuleID:      r.ID(),
						Title:       "List/Watch call inside loop",
						Description: "List/Watch inside loops can overload the API server; prefer informers/cache or move calls outside loops",
						PackagePath: pkg.PkgPath,
						Position:    Position{Filename: pos.Filename, Line: pos.Line, Column: pos.Column},
						Suggestion:  "Refactor to use SharedInformers or cached client, or collect identities and perform batched queries",
					})
				}
			}
		}
		return true
	})
	return issues
}
