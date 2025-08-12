package analyzers

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
	insppass "golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// AnalyzerListInLoop flags List/Watch calls inside loops.
var AnalyzerListInLoop = &analysis.Analyzer{
	Name:     "k8s011_listinloop",
	Doc:      "flags List/Watch calls inside loops (prefer informers/cache)",
	Run:      runListInLoop,
	Requires: []*analysis.Analyzer{insppass.Analyzer},
}

func runListInLoop(pass *analysis.Pass) (any, error) {
	insp := pass.ResultOf[insppass.Analyzer].(*inspector.Inspector)
	loopDepth := 0
	nodes := []ast.Node{(*ast.ForStmt)(nil), (*ast.RangeStmt)(nil), (*ast.CallExpr)(nil)}
	isKubeListOrWatch := func(obj types.Object) bool {
		if obj == nil || obj.Pkg() == nil {
			return false
		}
		sel := obj.Name()
		if sel != "List" && sel != "Watch" {
			return false
		}
		pkg := obj.Pkg().Path()
		// client-go typed clients often are methods; we only know the method name. Permit known pkgs.
		// This is conservative: prefer matches on List/Watch regardless of receiver package when inside loop.
		// Still, require that the identifier was resolved (i.e., comes from a real object).
		_ = pkg
		return true
	}
	insp.Nodes(nodes, func(n ast.Node, push bool) bool {
		switch x := n.(type) {
		case *ast.ForStmt, *ast.RangeStmt:
			if push {
				loopDepth++
			} else {
				loopDepth--
			}
		case *ast.CallExpr:
			if !push || loopDepth == 0 {
				return true
			}
			// Only flag if type info resolves the selector ident and it is List/Watch
			if id := calleeIdent(x.Fun); id != nil {
				if isKubeListOrWatch(pass.TypesInfo.Uses[id]) {
					pass.Reportf(id.Pos(), "List/Watch call inside loop; prefer informers/cache or move calls outside loops")
				}
			}
		}
		return true
	})
	return nil, nil
}

// calleeIdent reused from unboundedqueue.go
