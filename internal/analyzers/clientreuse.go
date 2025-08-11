package analyzers

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
)

// AnalyzerClientReuse flags creating Kubernetes clients in hot paths or inside loops.
var AnalyzerClientReuse = &analysis.Analyzer{
	Name: "k8s001_clientreuse",
	Doc:  "flags client construction inside loops or hot paths; clients should be reused",
	Run:  runClientReuse,
}

func runClientReuse(pass *analysis.Pass) (any, error) {
	isCtor := func(call *ast.CallExpr) bool {
		switch fun := call.Fun.(type) {
		case *ast.SelectorExpr:
			if fun.Sel == nil {
				return false
			}
			switch fun.Sel.Name {
			case "NewForConfig", "NewForConfigOrDie", "RESTClientFor", "New":
				return true
			}
		case *ast.Ident:
			switch fun.Name {
			case "NewForConfig", "RESTClientFor", "New":
				return true
			}
		}
		return false
	}

	for _, f := range pass.Files {
		ast.Inspect(f, func(n ast.Node) bool {
			fd, ok := n.(*ast.FuncDecl)
			if !ok || fd.Body == nil {
				return true
			}
			// loops
			ast.Inspect(fd.Body, func(n2 ast.Node) bool {
				switch x := n2.(type) {
				case *ast.ForStmt, *ast.RangeStmt:
					ast.Inspect(n2, func(nn ast.Node) bool {
						if call, ok := nn.(*ast.CallExpr); ok && isCtor(call) {
							pass.Reportf(call.Pos(), "client constructed inside loop; reuse a singleton client")
							return false
						}
						return true
					})
				case *ast.CallExpr:
					if isCtor(x) && isHotPath(fd) {
						pass.Reportf(x.Pos(), "client constructed in hot path; reuse a singleton client")
					}
				}
				return true
			})
			return true
		})
	}
	return nil, nil
}
