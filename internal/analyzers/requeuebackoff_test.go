package analyzers

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"golang.org/x/tools/go/analysis"
	insppass "golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

func runRequeueBackoffAnalyzerOnSrc(t *testing.T, src string, spoofControllerRuntimeTypes bool) []analysis.Diagnostic {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", src, 0)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	files := []*ast.File{f}
	info := &types.Info{Types: map[ast.Expr]types.TypeAndValue{}, Defs: map[*ast.Ident]types.Object{}, Uses: map[*ast.Ident]types.Object{}, Selections: map[*ast.SelectorExpr]*types.Selection{}}
	var conf types.Config
	_, err = conf.Check("p", fset, files, info)
	if err != nil {
		// Expected for test files with incomplete type information
	}

	// Optionally spoof type info to mark Result types as coming from controller-runtime packages
	if spoofControllerRuntimeTypes {
		pkgReconcile := types.NewPackage("sigs.k8s.io/controller-runtime/pkg/reconcile", "reconcile")

		// Find Result type declarations and mark them as controller-runtime types
		ast.Inspect(f, func(n ast.Node) bool {
			if ts, ok := n.(*ast.TypeSpec); ok && ts.Name.Name == "Result" {
				// Create a named type for controller-runtime Result
				resultType := types.NewNamed(types.NewTypeName(token.NoPos, pkgReconcile, "Result", nil), types.NewStruct(nil, nil), nil)
				info.Defs[ts.Name] = resultType.Obj()
				return false
			}
			return true
		})

		// Find composite literals of Result type and associate them with the controller-runtime Result type
		ast.Inspect(f, func(n ast.Node) bool {
			if cl, ok := n.(*ast.CompositeLit); ok {
				if id, ok := cl.Type.(*ast.Ident); ok && id.Name == "Result" {
					// Create the controller-runtime Result type
					resultType := types.NewNamed(types.NewTypeName(token.NoPos, pkgReconcile, "Result", nil), types.NewStruct(nil, nil), nil)
					info.Types[cl] = types.TypeAndValue{Type: resultType}
				}
			}
			return true
		})
	}

	var diags []analysis.Diagnostic
	pass := &analysis.Pass{Analyzer: AnalyzerRequeueBackoff, Fset: fset, Files: files, TypesInfo: info, TypesSizes: types.SizesFor("gc", "amd64"), Report: func(d analysis.Diagnostic) { diags = append(diags, d) }, ResultOf: map[*analysis.Analyzer]interface{}{insppass.Analyzer: inspector.New(files)}}
	_, _ = AnalyzerRequeueBackoff.Run(pass)
	return diags
}

func TestRequeueBackoff_RequeueWithoutAfter_Flagged(t *testing.T) {
	src := `package a
type Result struct{ Requeue bool; RequeueAfter int }
func f() (Result, error) { return Result{Requeue:true}, nil }`
	diags := runRequeueBackoffAnalyzerOnSrc(t, src, true) // spoof as controller-runtime types
	if len(diags) == 0 {
		t.Fatalf("expected diagnostic for controller-runtime requeue without backoff")
	}
}

func TestRequeueBackoff_WithRequeueAfter_NoDiag(t *testing.T) {
	src := `package a
type Result struct{ Requeue bool; RequeueAfter int }
func f() (Result, error) { return Result{Requeue:true, RequeueAfter:5}, nil }`
	diags := runRequeueBackoffAnalyzerOnSrc(t, src, true) // spoof as controller-runtime types
	if len(diags) != 0 {
		t.Fatalf("did not expect diagnostic when RequeueAfter is set")
	}
}

func TestRequeueBackoff_NonControllerRuntimeResult_NoDiag(t *testing.T) {
	src := `package a
type Result struct{ Requeue bool; RequeueAfter int }
func f() (Result, error) { return Result{Requeue:true}, nil }`
	diags := runRequeueBackoffAnalyzerOnSrc(t, src, false) // don't spoof - should be treated as non-controller-runtime
	if len(diags) != 0 {
		t.Fatalf("expected 0 diagnostics for non-controller-runtime Result types, got %d", len(diags))
	}
}

func TestRequeueBackoff_NoRequeue_NoDiag(t *testing.T) {
	src := `package a
type Result struct{ Requeue bool; RequeueAfter int }
func f() (Result, error) { return Result{}, nil }`
	diags := runRequeueBackoffAnalyzerOnSrc(t, src, true) // spoof as controller-runtime types
	if len(diags) != 0 {
		t.Fatalf("expected 0 diagnostics when Requeue is not set, got %d", len(diags))
	}
}
