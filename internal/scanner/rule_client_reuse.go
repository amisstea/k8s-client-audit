package scanner

import (
	"context"
	"go/ast"
	"go/token"
	"strings"

	"golang.org/x/tools/go/packages"
)

// Detect creation of Kubernetes clients inside hot paths (e.g., Reconcile) or inside loops.
// Clients should be long-lived and reused.
type ruleClientReuse struct{}

func NewRuleClientReuse() Rule        { return &ruleClientReuse{} }
func (r *ruleClientReuse) ID() string { return RuleClientReuseID }
func (r *ruleClientReuse) Description() string {
	return "Avoid creating Kubernetes clients inside hot paths or loops; reuse singletons"
}

func (r *ruleClientReuse) Apply(ctx context.Context, fset *token.FileSet, pkg *packages.Package) ([]Issue, error) {
	var issues []Issue
	for _, f := range pkg.Syntax {
		ast.Inspect(f, func(n ast.Node) bool {
			fd, ok := n.(*ast.FuncDecl)
			if !ok || fd.Body == nil {
				return true
			}
			funcName := fd.Name.Name
			recvName := receiverTypeName(fd)
			inHotPath := looksLikeHotPath(funcName, recvName)

			// Walk function body
			ast.Inspect(fd.Body, func(n2 ast.Node) bool {
				switch x := n2.(type) {
				case *ast.ForStmt, *ast.RangeStmt:
					// Inside loops: flag any client construction
					ast.Inspect(n2, func(nn ast.Node) bool {
						if call, ok := nn.(*ast.CallExpr); ok {
							if isClientConstructor(call) {
								pos := fset.Position(call.Pos())
								issues = append(issues, Issue{
									RuleID:      r.ID(),
									Title:       "Client constructed inside loop",
									Description: "Constructing clients inside loops is expensive; create once and reuse",
									Position:    Position{Filename: pos.Filename, Line: pos.Line, Column: pos.Column},
									Suggestion:  "Initialize clients during setup and pass them into hot paths",
								})
								return false
							}
						}
						return true
					})
				case *ast.CallExpr:
					if isClientConstructor(x) && inHotPath {
						pos := fset.Position(x.Pos())
						issues = append(issues, Issue{
							RuleID:      r.ID(),
							Title:       "Client constructed in hot path",
							Description: "Avoid constructing Kubernetes clients in hot paths like Reconcile/handlers",
							Position:    Position{Filename: pos.Filename, Line: pos.Line, Column: pos.Column},
							Suggestion:  "Create clients once (e.g., at startup) and reuse via dependency injection",
						})
					}
				}
				return true
			})
			return true
		})
	}
	return issues, nil
}

func receiverTypeName(fd *ast.FuncDecl) string {
	if fd.Recv == nil || len(fd.Recv.List) == 0 {
		return ""
	}
	t := fd.Recv.List[0].Type
	switch tt := t.(type) {
	case *ast.Ident:
		return tt.Name
	case *ast.StarExpr:
		if id, ok := tt.X.(*ast.Ident); ok {
			return id.Name
		}
	}
	return ""
}

func looksLikeHotPath(funcName, recvName string) bool {
	name := strings.ToLower(funcName)
	if strings.Contains(strings.ToLower(recvName), "reconcil") || strings.Contains(strings.ToLower(recvName), "controller") {
		return true
	}
	switch name {
	case "reconcile", "servehttp", "handle", "process", "sync", "worker", "run":
		return true
	}
	// also consider names with common prefixes
	if strings.Contains(name, "reconcil") || strings.Contains(name, "handler") || strings.Contains(name, "loop") {
		return true
	}
	return false
}

func isClientConstructor(call *ast.CallExpr) bool {
	// Match selector-based calls to client constructors: kubernetes.NewForConfig, dynamic.NewForConfig,
	// discovery.NewForConfig, client.New (controller-runtime), rest.RESTClientFor, etc.
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel == nil {
		return false
	}
	method := sel.Sel.Name
	base := baseIdentName(sel.X)
	if method == "NewForConfig" || method == "NewForConfigOrDie" || method == "RESTClientFor" {
		// Consider any X.NewForConfig/RESTClientFor as a client constructor
		return true
	}
	if method == "New" && (base == "client" || strings.Contains(strings.ToLower(base), "client")) {
		return true
	}
	return false
}

func baseIdentName(e ast.Expr) string {
	switch v := e.(type) {
	case *ast.Ident:
		return v.Name
	case *ast.SelectorExpr:
		return baseIdentName(v.X)
	}
	return ""
}
