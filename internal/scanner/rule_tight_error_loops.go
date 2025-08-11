package scanner

import (
	"context"
	"go/ast"
	"go/token"

	"golang.org/x/tools/go/packages"
)

// Detect tight loops on errors where there is no sleep/backoff.
type ruleTightErrorLoops struct{}

func NewRuleTightErrorLoops() Rule        { return &ruleTightErrorLoops{} }
func (r *ruleTightErrorLoops) ID() string { return RuleTightLoopsOnErrorsID }
func (r *ruleTightErrorLoops) Description() string {
	return "Avoid tight loops retrying on errors without backoff"
}

func (r *ruleTightErrorLoops) Apply(ctx context.Context, fset *token.FileSet, pkg *packages.Package) ([]Issue, error) {
	var issues []Issue
	for _, f := range pkg.Syntax {
		ast.Inspect(f, func(n ast.Node) bool {
			// Look for for-loops containing an if err != nil block without any sleep calls
			loop, ok := n.(*ast.ForStmt)
			if !ok || loop.Body == nil {
				return true
			}
			hasErrorCheck := false
			hasSleep := false
			hasKubeAPICall := false
			ast.Inspect(loop.Body, func(n2 ast.Node) bool {
				// detect time.Sleep or backoff.Sleep calls
				if call, ok := n2.(*ast.CallExpr); ok {
					if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
						if sel.Sel != nil && sel.Sel.Name == "Sleep" {
							hasSleep = true
						}
					}
					if isLikelyKubeAPICall(call) {
						hasKubeAPICall = true
					}
				}
				if ifs, ok := n2.(*ast.IfStmt); ok {
					// quick heuristic: if err != nil { ... }
					if be, ok := ifs.Cond.(*ast.BinaryExpr); ok {
						if be.Op.String() == "!=" {
							if _, ok := be.X.(*ast.Ident); ok { // err
								if bl, ok := be.Y.(*ast.Ident); ok && bl.Name == "nil" {
									hasErrorCheck = true
								}
							}
						}
					}
				}
				return true
			})
			if hasErrorCheck && hasKubeAPICall && !hasSleep {
				pos := fset.Position(loop.For)
				issues = append(issues, Issue{
					RuleID:      r.ID(),
					Title:       "Tight loop on errors without backoff",
					Description: "Add backoff or sleep when retrying on errors in a loop",
					PackagePath: pkg.PkgPath,
					Position:    Position{Filename: pos.Filename, Line: pos.Line, Column: pos.Column},
					Suggestion:  "Use time.Sleep or a rate-limiter/backoff strategy",
				})
			}
			return true
		})
	}
	return issues, nil
}
