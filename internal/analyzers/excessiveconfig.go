package analyzers

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
)

// AnalyzerExcessiveConfig flags repeated creation of rest.Config/clients in hot paths.
var AnalyzerExcessiveConfig = &analysis.Analyzer{
	Name: "k8s003_excessiveconfig",
	Doc:  "flags repeated rest.Config or client creation in loops or hot paths",
	Run:  runExcessiveConfig,
}

func runExcessiveConfig(pass *analysis.Pass) (any, error) {
	isClientCtor := func(call *ast.CallExpr) bool {
		switch fun := call.Fun.(type) {
		case *ast.SelectorExpr:
			if fun.Sel == nil {
				return false
			}
			method := fun.Sel.Name
			if method == "NewForConfig" || method == "NewForConfigOrDie" || method == "RESTClientFor" || method == "New" {
				return true
			}
		case *ast.Ident:
			// allow direct calls to NewForConfig in tests
			if fun.Name == "NewForConfig" || fun.Name == "RESTClientFor" || fun.Name == "New" {
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
			ast.Inspect(fd.Body, func(n2 ast.Node) bool {
				switch x := n2.(type) {
				case *ast.ForStmt, *ast.RangeStmt:
					ast.Inspect(n2, func(nn ast.Node) bool {
						if call, ok := nn.(*ast.CallExpr); ok && isClientCtor(call) {
							pass.Reportf(call.Pos(), "client constructed inside loop; create once and reuse")
							return false
						}
						return true
					})
				case *ast.CallExpr:
					if isClientCtor(x) && isHotPath(pass, fd) {
						pass.Reportf(x.Pos(), "client constructed in hot path; create once and reuse")
					}
				}
				return true
			})
			return true
		})
	}
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
			if isNamed(deref(results.At(0).Type()), "sigs.k8s.io/controller-runtime/pkg/reconcile", "Result") {
				return true
			}
		}
	}
	return false
}

func deref(t types.Type) types.Type {
	if p, ok := t.(*types.Pointer); ok {
		return p.Elem()
	}
	return t
}

func isNamed(t types.Type, pkgPath, name string) bool {
	if n, ok := t.(*types.Named); ok {
		if n.Obj() != nil && n.Obj().Pkg() != nil {
			return n.Obj().Pkg().Path() == pkgPath && n.Obj().Name() == name
		}
	}
	return false
}
