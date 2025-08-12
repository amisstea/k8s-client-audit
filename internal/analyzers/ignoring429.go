package analyzers

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
	insppass "golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// AnalyzerIgnoring429 (K8S030) flags code that checks for HTTP 429 or throttling
// but does not back off (e.g., immediately retries with no sleep/backoff).
var AnalyzerIgnoring429 = &analysis.Analyzer{
	Name:     "k8s030_ignoring429",
	Doc:      "flags handling of 429 without backoff",
	Run:      runIgnoring429,
	Requires: []*analysis.Analyzer{insppass.Analyzer},
}

func runIgnoring429(pass *analysis.Pass) (any, error) {
	insp := pass.ResultOf[insppass.Analyzer].(*inspector.Inspector)
	mentions429 := func(cond ast.Node) bool {
		seen := false
		ast.Inspect(cond, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.SelectorExpr:
				if x.Sel != nil {
					if obj := pass.TypesInfo.Uses[x.Sel]; obj != nil && obj.Pkg() != nil {
						if obj.Pkg().Path() == "net/http" && obj.Name() == "StatusTooManyRequests" {
							seen = true
						}
					} else if id, ok := x.X.(*ast.Ident); ok && id.Name == "http" && x.Sel.Name == "StatusTooManyRequests" {
						// Fallback when type info is not fully populated
						seen = true
					}
				}
			case *ast.BasicLit:
				if x.Value == "429" {
					seen = true
				}
			case *ast.Ident:
				if obj := pass.TypesInfo.Uses[x]; obj != nil && obj.Pkg() != nil {
					if obj.Pkg().Path() == "net/http" && obj.Name() == "StatusTooManyRequests" {
						seen = true
					}
				}
			}
			return true
		})
		return seen
	}
	hasBackoff := func(body *ast.BlockStmt) bool {
		found := false
		ast.Inspect(body, func(n ast.Node) bool {
			ce, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			if sel, ok := ce.Fun.(*ast.SelectorExpr); ok && sel.Sel != nil {
				if obj := pass.TypesInfo.Uses[sel.Sel]; obj != nil && obj.Pkg() != nil {
					if obj.Pkg().Path() == "time" && obj.Name() == "Sleep" {
						found = true
						return false
					}
				} else if id, ok := sel.X.(*ast.Ident); ok && id.Name == "time" && sel.Sel.Name == "Sleep" {
					found = true
					return false
				}
			}
			if id, ok := ce.Fun.(*ast.Ident); ok {
				if obj := pass.TypesInfo.Uses[id]; obj != nil {
					if obj.Name() == "Backoff" || obj.Name() == "Wait" {
						found = true
						return false
					}
				}
			}
			return true
		})
		return found
	}
	insp.Preorder([]ast.Node{(*ast.IfStmt)(nil)}, func(n ast.Node) {
		ifs := n.(*ast.IfStmt)
		if mentions429(ifs.Cond) && !hasBackoff(ifs.Body) {
			pass.Reportf(ifs.If, "Handling 429 without backoff; add sleep/backoff before retrying")
		}
	})
	return nil, nil
}
