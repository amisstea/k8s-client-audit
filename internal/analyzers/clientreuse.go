package analyzers

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
	insppass "golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// AnalyzerClientReuse flags creating Kubernetes clients in hot paths or inside loops.
var AnalyzerClientReuse = &analysis.Analyzer{
	Name:     "k8s001_clientreuse",
	Doc:      "flags client construction inside loops or hot paths; clients should be reused",
	Run:      runClientReuse,
	Requires: []*analysis.Analyzer{insppass.Analyzer},
}

func runClientReuse(pass *analysis.Pass) (any, error) {
	insp := pass.ResultOf[insppass.Analyzer].(*inspector.Inspector)

	// Determine if a call expression constructs a K8s client by checking the
	// fully-qualified package path and function name via type information.
	isCtor := func(call *ast.CallExpr) bool {
		// Helper to check a function object
		isKnownCtor := func(obj types.Object) bool {
			if obj == nil || obj.Pkg() == nil {
				return false
			}
			pkg := obj.Pkg().Path()
			name := obj.Name()
			switch pkg {
			case "k8s.io/client-go/kubernetes":
				return name == "NewForConfig" || name == "NewForConfigOrDie"
			case "k8s.io/client-go/dynamic":
				return name == "NewForConfig"
			case "k8s.io/client-go/rest":
				return name == "RESTClientFor"
			case "sigs.k8s.io/controller-runtime/pkg/client":
				return name == "New"
			default:
				return false
			}
		}
		switch fun := call.Fun.(type) {
		case *ast.SelectorExpr:
			if fun.Sel != nil {
				if obj := pass.TypesInfo.Uses[fun.Sel]; isKnownCtor(obj) {
					return true
				}
			}
		case *ast.Ident:
			if obj := pass.TypesInfo.Uses[fun]; isKnownCtor(obj) {
				return true
			}
		}
		return false
	}

	var currentFunc *ast.FuncDecl
	loopDepth := 0
	nodes := []ast.Node{(*ast.FuncDecl)(nil), (*ast.ForStmt)(nil), (*ast.RangeStmt)(nil), (*ast.CallExpr)(nil)}
	insp.Nodes(nodes, func(n ast.Node, push bool) bool {
		switch node := n.(type) {
		case *ast.FuncDecl:
			if push {
				currentFunc = node
			} else {
				currentFunc = nil
			}
		case *ast.ForStmt, *ast.RangeStmt:
			if push {
				loopDepth++
			} else {
				loopDepth--
			}
		case *ast.CallExpr:
			if !push {
				return true
			}
			if isCtor(node) {
				if loopDepth > 0 {
					pass.Reportf(node.Pos(), "client constructed inside loop; reuse a singleton client")
				} else if currentFunc != nil && isHotPath(pass, currentFunc) {
					pass.Reportf(node.Pos(), "client constructed in hot path; reuse a singleton client")
				}
			}
		}
		return true
	})
	return nil, nil
}
