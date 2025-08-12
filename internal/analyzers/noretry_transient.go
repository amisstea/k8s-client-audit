package analyzers

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
)

// AnalyzerNoRetryTransient (K8S031) flags error handling that detects transient
// network issues but returns immediately without any retry/backoff.
var AnalyzerNoRetryTransient = &analysis.Analyzer{
	Name: "k8s031_noretrytransient",
	Doc:  "flags transient errors handled without retry",
	Run:  runNoRetryTransient,
}

func runNoRetryTransient(pass *analysis.Pass) (any, error) {
	for _, f := range pass.Files {
		ast.Inspect(f, func(n ast.Node) bool {
			ifs, ok := n.(*ast.IfStmt)
			if !ok {
				return true
			}
			mentionsTransient := false
			ast.Inspect(ifs.Cond, func(m ast.Node) bool {
				if id, ok := m.(*ast.Ident); ok {
					if id.Name == "Timeout" || id.Name == "Temporary" || id.Name == "NetError" {
						mentionsTransient = true
					}
				}
				return true
			})
			if !mentionsTransient {
				return true
			}
			hasRetry := false
			ast.Inspect(ifs.Body, func(m ast.Node) bool {
				ce, ok := m.(*ast.CallExpr)
				if !ok {
					return true
				}
				if id, ok := ce.Fun.(*ast.Ident); ok {
					if id.Name == "Retry" || id.Name == "Backoff" || id.Name == "Sleep" {
						hasRetry = true
					}
				}
				return true
			})
			if mentionsTransient && !hasRetry {
				pass.Reportf(ifs.If, "Transient error handled without retry/backoff")
			}
			return true
		})
	}
	return nil, nil
}
