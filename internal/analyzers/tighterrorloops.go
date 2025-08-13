package analyzers

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
)

// AnalyzerTightErrorLoops flags tight retry loops on errors that call the Kubernetes API without any backoff/sleep.
var AnalyzerTightErrorLoops = &analysis.Analyzer{
	Name: "tighterrorloops",
	Doc:  "flags tight loops retrying on errors around Kubernetes API calls without backoff",
	Run:  runTightErrorLoops,
}

func runTightErrorLoops(pass *analysis.Pass) (any, error) {
	for _, f := range pass.Files {
		ast.Inspect(f, func(n ast.Node) bool {
			loop, ok := n.(*ast.ForStmt)
			if !ok || loop.Body == nil {
				return true
			}
			hasErrCheck := false
			hasSleep := false
			hasKubeAPICall := false
			ast.Inspect(loop.Body, func(n2 ast.Node) bool {
				switch x := n2.(type) {
				case *ast.CallExpr:
					if sel, ok := x.Fun.(*ast.SelectorExpr); ok && sel.Sel != nil && sel.Sel.Name == "Sleep" {
						hasSleep = true
					}
					if isLikelyKubeAPICall(x) {
						hasKubeAPICall = true
					}
				case *ast.IfStmt:
					if be, ok := x.Cond.(*ast.BinaryExpr); ok {
						if be.Op.String() == "!=" {
							if _, ok := be.X.(*ast.Ident); ok {
								if id, ok := be.Y.(*ast.Ident); ok && id.Name == "nil" {
									hasErrCheck = true
								}
							}
						}
					}
				}
				return true
			})
			if hasErrCheck && hasKubeAPICall && !hasSleep {
				pass.Reportf(loop.For, "tight loop on errors without backoff around Kubernetes API calls")
			}
			return true
		})
	}
	return nil, nil
}

func isLikelyKubeAPICall(call *ast.CallExpr) bool {
	// Common API methods
	method := ""
	if sel, ok := call.Fun.(*ast.SelectorExpr); ok && sel.Sel != nil {
		method = sel.Sel.Name
		if (method == "Get" || method == "List" || method == "Create" || method == "Update" || method == "Patch" || method == "Delete" || method == "Watch") &&
			(chainHasResourceName(sel.X) || argsContainKubeOptions(call.Args) || looksLikeContextFirstArg(call)) {
			return true
		}
	}
	return false
}

func looksLikeContextFirstArg(call *ast.CallExpr) bool {
	if len(call.Args) == 0 {
		return false
	}
	switch a := call.Args[0].(type) {
	case *ast.CallExpr:
		if sel, ok := a.Fun.(*ast.SelectorExpr); ok {
			if id, ok := sel.X.(*ast.Ident); ok {
				return id.Name == "context" && (sel.Sel.Name == "Background" || sel.Sel.Name == "TODO")
			}
		}
	case *ast.Ident:
		return a.Name == "ctx" || a.Name == "context"
	}
	return false
}

func chainHasResourceName(expr ast.Expr) bool {
	switch e := expr.(type) {
	case *ast.SelectorExpr:
		if e.Sel != nil {
			if isResourceSelector(e.Sel.Name) {
				return true
			}
		}
		return chainHasResourceName(e.X)
	case *ast.CallExpr:
		return chainHasResourceName(e.Fun)
	}
	return false
}

func isResourceSelector(name string) bool {
	switch name {
	case "Pods", "Deployments", "Services", "StatefulSets", "ConfigMaps", "Secrets", "Nodes", "Namespaces", "Events", "Jobs", "CronJobs", "PersistentVolumes", "PersistentVolumeClaims", "DaemonSets", "ReplicaSets", "TaskRuns", "Tasks", "Pipelines", "PipelineRuns":
		return true
	default:
		return false
	}
}

func argsContainKubeOptions(args []ast.Expr) bool {
	for _, a := range args {
		switch x := a.(type) {
		case *ast.CallExpr:
			if sel, ok := x.Fun.(*ast.SelectorExpr); ok && sel.Sel != nil {
				switch sel.Sel.Name {
				case "InNamespace", "MatchingLabels", "MatchingFields", "MatchingFieldsSelector":
					return true
				}
			}
		case *ast.CompositeLit:
			for _, el := range x.Elts {
				if kv, ok := el.(*ast.KeyValueExpr); ok {
					if ident, ok := kv.Key.(*ast.Ident); ok {
						if ident.Name == "LabelSelector" || ident.Name == "FieldSelector" {
							return true
						}
					}
				}
			}
		}
	}
	return false
}
