package analyzers

import (
	"go/ast"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
	insppass "golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// AnalyzerTightErrorLoops flags tight retry loops on errors that call the Kubernetes API without any backoff/sleep.
var AnalyzerTightErrorLoops = &analysis.Analyzer{
	Name:     "tighterrorloops",
	Doc:      "flags tight loops retrying on errors around Kubernetes API calls without backoff",
	Run:      runTightErrorLoops,
	Requires: []*analysis.Analyzer{insppass.Analyzer},
}

func runTightErrorLoops(pass *analysis.Pass) (any, error) {
	insp := pass.ResultOf[insppass.Analyzer].(*inspector.Inspector)

	// Check if a method call is a Kubernetes API operation
	isKubernetesAPICall := func(obj types.Object) bool {
		if obj == nil || obj.Pkg() == nil {
			return false
		}
		name := obj.Name()
		if !(name == "Get" || name == "List" || name == "Create" || name == "Update" || name == "Patch" || name == "Delete" || name == "Watch") {
			return false
		}
		pkg := obj.Pkg().Path()

		// Check for Kubernetes-related packages
		switch {
		case pkg == "sigs.k8s.io/controller-runtime/pkg/client":
			return true
		case pkg == "k8s.io/client-go/dynamic":
			return true
		default:
			// Check for any k8s.io or sigs.k8s.io packages
			if strings.HasPrefix(pkg, "k8s.io/") || strings.HasPrefix(pkg, "sigs.k8s.io/") {
				return true
			}
			// Check for packages containing client-go, controller-runtime, or apimachinery
			if strings.Contains(pkg, "client-go") || strings.Contains(pkg, "controller-runtime") || strings.Contains(pkg, "apimachinery") {
				return true
			}
		}
		return false
	}

	// Check if a method call is a sleep operation
	isSleepCall := func(obj types.Object) bool {
		if obj == nil || obj.Pkg() == nil {
			return false
		}
		name := obj.Name()
		pkg := obj.Pkg().Path()

		// time.Sleep
		return name == "Sleep" && pkg == "time"
	}

	// Track loops and their contents
	loopDepth := 0
	var currentLoop *ast.ForStmt
	var hasErrorCheck bool
	var hasKubeAPICall bool
	var hasSleep bool

	nodes := []ast.Node{(*ast.ForStmt)(nil), (*ast.RangeStmt)(nil), (*ast.CallExpr)(nil), (*ast.IfStmt)(nil)}
	insp.Nodes(nodes, func(n ast.Node, push bool) bool {
		switch x := n.(type) {
		case *ast.ForStmt:
			if push {
				loopDepth++
				if loopDepth == 1 {
					currentLoop = x
					hasErrorCheck = false
					hasKubeAPICall = false
					hasSleep = false
				}
			} else {
				if loopDepth == 1 && currentLoop != nil {
					// Check for tight error loops
					if hasErrorCheck && hasKubeAPICall && !hasSleep {
						pass.Reportf(currentLoop.For, "tight loop on errors without backoff around Kubernetes API calls")
					}
					currentLoop = nil
				}
				loopDepth--
			}
		case *ast.RangeStmt:
			if push {
				loopDepth++
			} else {
				loopDepth--
			}
		case *ast.CallExpr:
			if !push || loopDepth != 1 || currentLoop == nil {
				return true
			}

			// Check if this is a method call
			sel, ok := x.Fun.(*ast.SelectorExpr)
			if !ok || sel.Sel == nil {
				return true
			}

			// Use type information to determine what kind of call this is
			if obj := pass.TypesInfo.Uses[sel.Sel]; obj != nil {
				if isKubernetesAPICall(obj) {
					hasKubeAPICall = true
				} else if isSleepCall(obj) {
					hasSleep = true
				}
			}
		case *ast.IfStmt:
			if !push || loopDepth != 1 || currentLoop == nil {
				return true
			}

			// Check for error checking patterns: if err != nil
			if be, ok := x.Cond.(*ast.BinaryExpr); ok {
				if be.Op.String() == "!=" {
					if _, ok := be.X.(*ast.Ident); ok {
						if id, ok := be.Y.(*ast.Ident); ok && id.Name == "nil" {
							hasErrorCheck = true
						}
					}
				}
			}
		}
		return true
	})

	return nil, nil
}
