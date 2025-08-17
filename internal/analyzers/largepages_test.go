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

func runLargePagesAnalyzerOnSrc(t *testing.T, src string, spoofKubernetesTypes bool) []analysis.Diagnostic {
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

	// Optionally spoof type info to mark types as coming from Kubernetes packages
	if spoofKubernetesTypes {
		pkgMeta := types.NewPackage("k8s.io/apimachinery/pkg/apis/meta/v1", "v1")
		pkgClient := types.NewPackage("sigs.k8s.io/controller-runtime/pkg/client", "client")

		// Find type declarations and mark them as Kubernetes types
		ast.Inspect(f, func(n ast.Node) bool {
			if ts, ok := n.(*ast.TypeSpec); ok && ts.Name.Name == "ListOptions" {
				// Create a named type for Kubernetes ListOptions
				listOptionsType := types.NewNamed(types.NewTypeName(token.NoPos, pkgMeta, "ListOptions", nil), types.NewStruct(nil, nil), nil)
				info.Defs[ts.Name] = listOptionsType.Obj()
			}
			return true
		})

		// Find composite literals and method calls
		ast.Inspect(f, func(n ast.Node) bool {
			if cl, ok := n.(*ast.CompositeLit); ok {
				if id, ok := cl.Type.(*ast.Ident); ok && id.Name == "ListOptions" {
					// Create the Kubernetes ListOptions type
					listOptionsType := types.NewNamed(types.NewTypeName(token.NoPos, pkgMeta, "ListOptions", nil), types.NewStruct(nil, nil), nil)
					info.Types[cl] = types.TypeAndValue{Type: listOptionsType}
				}
			} else if se, ok := n.(*ast.SelectorExpr); ok && se.Sel != nil && se.Sel.Name == "List" {
				// Mark List calls as coming from Kubernetes client package
				sig := types.NewSignatureType(nil, nil, nil, types.NewTuple(), types.NewTuple(), false)
				info.Uses[se.Sel] = types.NewFunc(token.NoPos, pkgClient, "List", sig)
			}
			return true
		})
	}

	var diags []analysis.Diagnostic
	pass := &analysis.Pass{Analyzer: AnalyzerLargePageSizes, Fset: fset, Files: files, TypesInfo: info, TypesSizes: types.SizesFor("gc", "amd64"), Report: func(d analysis.Diagnostic) { diags = append(diags, d) }, ResultOf: map[*analysis.Analyzer]interface{}{insppass.Analyzer: inspector.New(files)}}
	_, _ = AnalyzerLargePageSizes.Run(pass)
	return diags
}

func TestLargePages_LimitLarge_Flagged(t *testing.T) {
	src := `package a
type ListOptions struct{ Limit int64 }
type IFace interface{ List(x any, opts ListOptions) error }
func f(c IFace){ _ = c.List(nil, ListOptions{Limit:2000}) }`
	diags := runLargePagesAnalyzerOnSrc(t, src, true) // spoof as Kubernetes types
	if len(diags) == 0 {
		t.Fatalf("expected diagnostic for large Kubernetes page size")
	}
}

func TestLargePages_LimitReasonable_NoDiag(t *testing.T) {
	src := `package a
type ListOptions struct{ Limit int64 }
type IFace interface{ List(x any, opts ListOptions) error }
func f(c IFace){ _ = c.List(nil, ListOptions{Limit:100}) }`
	diags := runLargePagesAnalyzerOnSrc(t, src, true) // spoof as Kubernetes types
	if len(diags) != 0 {
		t.Fatalf("did not expect diagnostic for reasonable page size")
	}
}

func TestLargePages_NonKubernetesListOptions_NoDiag(t *testing.T) {
	src := `package a
type ListOptions struct{ Limit int64 }
type DatabaseClient interface{ List(opts ListOptions) error }
func f(c DatabaseClient){ _ = c.List(ListOptions{Limit:2000}) }`
	diags := runLargePagesAnalyzerOnSrc(t, src, false) // don't spoof - should be treated as non-Kubernetes
	if len(diags) != 0 {
		t.Fatalf("expected 0 diagnostics for non-Kubernetes ListOptions, got %d", len(diags))
	}
}

func TestLargePages_NonKubernetesList_NoDiag(t *testing.T) {
	src := `package a
type ListOptions struct{ Limit int64 }
type APIClient interface{ List(opts ListOptions) error }
func f(c APIClient){ _ = c.List(ListOptions{Limit:2000}) }`
	diags := runLargePagesAnalyzerOnSrc(t, src, false) // don't spoof - should be treated as non-Kubernetes
	if len(diags) != 0 {
		t.Fatalf("expected 0 diagnostics for non-Kubernetes List calls, got %d", len(diags))
	}
}
