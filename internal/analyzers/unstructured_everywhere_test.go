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

func runUnstructuredEverywhereAnalyzerOnSrc(t *testing.T, src string, spoof bool) []analysis.Diagnostic {
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

	// Optionally spoof type info to mark types as coming from Kubernetes unstructured packages
	if spoof {
		pkgUnstructured := types.NewPackage("k8s.io/apimachinery/pkg/apis/meta/v1/unstructured", "unstructured")

		// Find type declarations and mark them as Kubernetes Unstructured types
		ast.Inspect(f, func(n ast.Node) bool {
			if ts, ok := n.(*ast.TypeSpec); ok && ts.Name.Name == "Unstructured" {
				// Create a named type for Kubernetes Unstructured
				unstructuredType := types.NewNamed(types.NewTypeName(token.NoPos, pkgUnstructured, "Unstructured", nil), types.NewStruct(nil, nil), nil)
				info.Defs[ts.Name] = unstructuredType.Obj()
			}
			return true
		})

		// Find composite literals and function calls
		ast.Inspect(f, func(n ast.Node) bool {
			if cl, ok := n.(*ast.CompositeLit); ok {
				if id, ok := cl.Type.(*ast.Ident); ok && id.Name == "Unstructured" {
					// Create the Kubernetes Unstructured type
					unstructuredType := types.NewNamed(types.NewTypeName(token.NoPos, pkgUnstructured, "Unstructured", nil), types.NewStruct(nil, nil), nil)
					info.Types[cl] = types.TypeAndValue{Type: unstructuredType}
				}
			} else if se, ok := n.(*ast.SelectorExpr); ok && se.Sel != nil && se.Sel.Name == "Unstructured" {
				// Mark Unstructured calls as coming from Kubernetes package
				sig := types.NewSignatureType(nil, nil, nil, types.NewTuple(), types.NewTuple(), false)
				info.Uses[se.Sel] = types.NewFunc(token.NoPos, pkgUnstructured, "Unstructured", sig)
			}
			return true
		})
	}

	var diags []analysis.Diagnostic
	pass := &analysis.Pass{Analyzer: AnalyzerUnstructuredEverywhere, Fset: fset, Files: files, TypesInfo: info, TypesSizes: types.SizesFor("gc", "amd64"), Report: func(d analysis.Diagnostic) { diags = append(diags, d) }, ResultOf: map[*analysis.Analyzer]interface{}{insppass.Analyzer: inspector.New(files)}}
	_, _ = AnalyzerUnstructuredEverywhere.Run(pass)
	return diags
}

func TestUnstructuredEverywhere_ManyUsages_Flagged(t *testing.T) {
	src := `package a
type Unstructured struct{}
var _ = Unstructured{}
func a1(){ _ = Unstructured{} }
func a2(){ _ = Unstructured{} }`
	diags := runUnstructuredEverywhereAnalyzerOnSrc(t, src, true) // spoof as Kubernetes Unstructured types
	if len(diags) == 0 {
		t.Fatalf("expected diagnostic for heavy Kubernetes unstructured usage")
	}
}

func TestUnstructuredEverywhere_SparseUsage_NoDiag(t *testing.T) {
	src := `package a
type Unstructured struct{}
func a1(){ _ = Unstructured{} }`
	diags := runUnstructuredEverywhereAnalyzerOnSrc(t, src, true) // spoof as Kubernetes Unstructured types
	if len(diags) != 0 {
		t.Fatalf("did not expect diagnostic for light Kubernetes unstructured usage")
	}
}

func TestUnstructuredEverywhere_NonKubernetesUnstructured_NoDiag(t *testing.T) {
	src := `package a
type Unstructured struct{}
var _ = Unstructured{}
func a1(){ _ = Unstructured{} }
func a2(){ _ = Unstructured{} }
func a3(){ _ = Unstructured{} }`
	diags := runUnstructuredEverywhereAnalyzerOnSrc(t, src, false) // don't spoof - should be treated as non-Kubernetes
	if len(diags) != 0 {
		t.Fatalf("expected 0 diagnostics for non-Kubernetes Unstructured usage, got %d", len(diags))
	}
}

func TestUnstructuredEverywhere_MixedUsage_OnlyKubernetesUsage_Flagged(t *testing.T) {
	src := `package a
type Unstructured struct{}
type SomeOtherType struct{}
var _ = Unstructured{}
var _ = SomeOtherType{}
func a1(){ _ = Unstructured{} }
func a2(){ _ = Unstructured{} }`
	diags := runUnstructuredEverywhereAnalyzerOnSrc(t, src, true) // spoof only Unstructured as Kubernetes
	if len(diags) == 0 {
		t.Fatalf("expected diagnostic for heavy Kubernetes unstructured usage in mixed code")
	}
}
