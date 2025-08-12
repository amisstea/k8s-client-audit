package analyzers

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
)

// AnalyzerIgnoring429 (K8S030) flags code that checks for HTTP 429 or throttling
// but does not back off (e.g., immediately retries with no sleep/backoff).
var AnalyzerIgnoring429 = &analysis.Analyzer{
	Name: "k8s030_ignoring429",
	Doc:  "flags handling of 429 without backoff",
	Run:  runIgnoring429,
}

func runIgnoring429(pass *analysis.Pass) (any, error) {
	for _, f := range pass.Files {
		ast.Inspect(f, func(n ast.Node) bool {
			// Look for if statements that mention 429 or TooManyRequests and within the body call immediately again
			ifs, ok := n.(*ast.IfStmt)
			if !ok {
				return true
			}
			condMentions429 := false
			ast.Inspect(ifs.Cond, func(m ast.Node) bool {
				if id, ok := m.(*ast.Ident); ok {
					if id.Name == "TooManyRequests" || id.Name == "StatusTooManyRequests" || id.Name == "429" {
						condMentions429 = true
					}
				}
				return true
			})
			if !condMentions429 {
				return true
			}
			hasSleep := false
			hasBackoff := false
			ast.Inspect(ifs.Body, func(m ast.Node) bool {
				ce, ok := m.(*ast.CallExpr)
				if !ok {
					return true
				}
				if sel, ok := ce.Fun.(*ast.SelectorExpr); ok && sel.Sel != nil {
					if sel.Sel.Name == "Sleep" {
						hasSleep = true
					}
				}
				if id, ok := ce.Fun.(*ast.Ident); ok {
					// any function named Backoff/Wait is considered a backoff
					if id.Name == "Backoff" || id.Name == "Wait" {
						hasBackoff = true
					}
				}
				return true
			})
			if condMentions429 && !(hasSleep || hasBackoff) {
				pass.Reportf(ifs.If, "Handling 429 without backoff; add sleep/backoff before retrying")
			}
			return true
		})
	}
	return nil, nil
}
