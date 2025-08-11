package analyzers

import (
	"go/ast"

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
					if isClientCtor(x) && isHotPath(fd) {
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

func isHotPath(fd *ast.FuncDecl) bool {
	if fd == nil {
		return false
	}
	name := fd.Name.Name
	lname := lower(name)
	if contains(lower(recvType(fd)), "reconcil") || contains(lower(recvType(fd)), "controller") {
		return true
	}
	switch lname {
	case "reconcile", "servehttp", "handle", "process", "sync", "worker", "run":
		return true
	}
	if contains(lname, "reconcil") || contains(lname, "handler") || contains(lname, "loop") {
		return true
	}
	return false
}

func recvType(fd *ast.FuncDecl) string {
	if fd.Recv == nil || len(fd.Recv.List) == 0 {
		return ""
	}
	switch t := fd.Recv.List[0].Type.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		if id, ok := t.X.(*ast.Ident); ok {
			return id.Name
		}
	}
	return ""
}

func lower(s string) string {
	b := []rune(s)
	for i, r := range b {
		if 'A' <= r && r <= 'Z' {
			b[i] = r + ('a' - 'A')
		}
	}
	return string(b)
}
func contains(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && indexOf(s, sub) >= 0)
}
func indexOf(s, sub string) int {
	n, m := len(s), len(sub)
	if m == 0 {
		return 0
	}
	for i := 0; i+m <= n; i++ {
		if s[i:i+m] == sub {
			return i
		}
	}
	return -1
}
