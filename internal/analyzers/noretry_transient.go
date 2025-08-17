package analyzers

import (
	"go/ast"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
	insppass "golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// AnalyzerNoRetryTransient flags error handling that detects transient
// network issues but returns immediately without any retry/backoff.
var AnalyzerNoRetryTransient = &analysis.Analyzer{
	Name:     "noretrytransient",
	Doc:      "flags transient errors handled without retry",
	Run:      runNoRetryTransient,
	Requires: []*analysis.Analyzer{insppass.Analyzer},
}

func runNoRetryTransient(pass *analysis.Pass) (any, error) {
	insp := pass.ResultOf[insppass.Analyzer].(*inspector.Inspector)

	// Check if a method call suggests this is Kubernetes-related
	isKubernetesContext := func(f *ast.File) bool {
		hasKubernetesAPI := false

		// Look for imports that suggest this is Kubernetes-related
		for _, imp := range f.Imports {
			if imp.Path == nil {
				continue
			}
			path := strings.Trim(imp.Path.Value, `"`)
			if strings.HasPrefix(path, "k8s.io/") ||
				strings.HasPrefix(path, "sigs.k8s.io/") ||
				strings.Contains(path, "client-go") ||
				strings.Contains(path, "controller-runtime") {
				hasKubernetesAPI = true
				break
			}
		}

		// If no Kubernetes imports, look for API method calls in the file
		if !hasKubernetesAPI {
			ast.Inspect(f, func(n ast.Node) bool {
				if ce, ok := n.(*ast.CallExpr); ok {
					if sel, ok := ce.Fun.(*ast.SelectorExpr); ok && sel.Sel != nil {
						name := sel.Sel.Name
						if name == "Get" || name == "List" || name == "Create" || name == "Update" || name == "Patch" || name == "Delete" || name == "Watch" {
							// Check if this call uses types that suggest Kubernetes
							if obj := pass.TypesInfo.Uses[sel.Sel]; obj != nil && obj.Pkg() != nil {
								pkg := obj.Pkg().Path()
								if strings.HasPrefix(pkg, "k8s.io/") || strings.HasPrefix(pkg, "sigs.k8s.io/") ||
									strings.Contains(pkg, "client-go") || strings.Contains(pkg, "controller-runtime") {
									hasKubernetesAPI = true
									return false
								}
							}
						}
					}
				}
				return true
			})
		}

		return hasKubernetesAPI
	}

	// Check if a function call suggests retry logic
	isSleepOrRetryCall := func(obj types.Object) bool {
		if obj == nil {
			return false
		}
		name := obj.Name()
		return name == "Sleep" || name == "Retry" || name == "Backoff"
	}

	nodes := []ast.Node{(*ast.IfStmt)(nil)}
	insp.Nodes(nodes, func(n ast.Node, push bool) bool {
		if !push {
			return true
		}

		ifs := n.(*ast.IfStmt)

		// Only analyze if this file appears to be Kubernetes-related
		var currentFile *ast.File
		for _, f := range pass.Files {
			if f.Pos() <= ifs.Pos() && ifs.End() <= f.End() {
				currentFile = f
				break
			}
		}

		if currentFile == nil || !isKubernetesContext(currentFile) {
			return true
		}

		// Check if the condition mentions transient error patterns
		mentionsTransient := false
		ast.Inspect(ifs.Cond, func(m ast.Node) bool {
			if id, ok := m.(*ast.Ident); ok {
				if id.Name == "Timeout" || id.Name == "Temporary" || id.Name == "NetError" ||
					id.Name == "TooManyRequests" || id.Name == "ServerTimeout" || id.Name == "ConnectionRefused" {
					mentionsTransient = true
				}
			}
			return true
		})

		if !mentionsTransient {
			return true
		}

		// Check if the body contains retry logic
		hasRetry := false
		ast.Inspect(ifs.Body, func(m ast.Node) bool {
			if ce, ok := m.(*ast.CallExpr); ok {
				// Check direct function calls
				if id, ok := ce.Fun.(*ast.Ident); ok {
					if obj := pass.TypesInfo.Uses[id]; isSleepOrRetryCall(obj) {
						hasRetry = true
						return false
					}
					// Also check by name for backward compatibility
					if id.Name == "Retry" || id.Name == "Backoff" || id.Name == "Sleep" {
						hasRetry = true
						return false
					}
				}
				// Check method calls
				if sel, ok := ce.Fun.(*ast.SelectorExpr); ok && sel.Sel != nil {
					if obj := pass.TypesInfo.Uses[sel.Sel]; isSleepOrRetryCall(obj) {
						hasRetry = true
						return false
					}
					if sel.Sel.Name == "Sleep" {
						hasRetry = true
						return false
					}
				}
			}
			return true
		})

		if mentionsTransient && !hasRetry {
			pass.Reportf(ifs.If, "Kubernetes-related transient error handled without retry/backoff")
		}

		return true
	})

	return nil, nil
}
