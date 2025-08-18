package analyzers

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
	insppass "golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// AnalyzerExcessiveConfig flags repeated creation of rest.Config/clients in hot paths.
var AnalyzerExcessiveConfig = &analysis.Analyzer{
	Name:     "excessiveconfig",
	Doc:      "flags repeated rest.Config or client creation in loops or hot paths",
	Run:      runExcessiveConfig,
	Requires: []*analysis.Analyzer{insppass.Analyzer},
}

func runExcessiveConfig(pass *analysis.Pass) (any, error) {
	insp := pass.ResultOf[insppass.Analyzer].(*inspector.Inspector)
	// Determine if a call expression constructs a K8s client by checking the
	// fully-qualified package path and function name via type information.
	isClientCtor := func(call *ast.CallExpr) bool {
		switch fun := call.Fun.(type) {
		case *ast.SelectorExpr:
			if fun.Sel != nil {
				if obj := pass.TypesInfo.Uses[fun.Sel]; isKubernetesClientConstructor(obj) {
					return true
				}
			}
		case *ast.Ident:
			if obj := pass.TypesInfo.Uses[fun]; isKubernetesClientConstructor(obj) {
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
			if isClientCtor(node) {
				if loopDepth > 0 {
					pass.Reportf(node.Pos(), "client constructed inside loop; create once and reuse")
				} else if currentFunc != nil && isHotPath(pass, currentFunc) {
					pass.Reportf(node.Pos(), "client constructed in hot path; create once and reuse")
				}
			}
		}
		return true
	})
	return nil, nil
}

func isHotPath(pass *analysis.Pass, fd *ast.FuncDecl) bool {
	if fd == nil {
		return false
	}
	obj := pass.TypesInfo.Defs[fd.Name]
	sig, ok := obj.Type().(*types.Signature)
	if !ok {
		return false
	}
	// Detect HTTP handlers: ServeHTTP(http.ResponseWriter, *http.Request)
	if fd.Name.Name == "ServeHTTP" {
		params := sig.Params()
		if params.Len() == 2 {
			if isNamed(params.At(0).Type(), "net/http", "ResponseWriter") && isNamed(deref(params.At(1).Type()), "net/http", "Request") {
				return true
			}
		}
	}
	// Detect controller-runtime reconcilers: Reconcile(...) (reconcile.Result, error)
	if fd.Name.Name == "Reconcile" {
		results := sig.Results()
		if results.Len() >= 1 {
			if isNamed(deref(results.At(0).Type()), PkgControllerRuntimeReconcile, "Result") {
				return true
			}
		}
	}
	return false
}

// moved to helpers.go: deref, isNamed
