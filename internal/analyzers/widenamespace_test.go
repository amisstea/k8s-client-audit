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

func runWideNamespaceAnalyzerOnSrc(t *testing.T, src string, spoof bool) []analysis.Diagnostic {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", src, 0)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	files := []*ast.File{f}
	info := &types.Info{Types: map[ast.Expr]types.TypeAndValue{}, Defs: map[*ast.Ident]types.Object{}, Uses: map[*ast.Ident]types.Object{}, Selections: map[*ast.SelectorExpr]*types.Selection{}}
	var conf types.Config
	_, _ = conf.Check("p", fset, files, info)

	// Optionally spoof type info to mark method calls as coming from Kubernetes packages
	if spoof {
		pkgCR := types.NewPackage("sigs.k8s.io/controller-runtime/pkg/client", "client")
		ast.Inspect(f, func(n ast.Node) bool {
			if se, ok := n.(*ast.SelectorExpr); ok && se.Sel != nil {
				name := se.Sel.Name
				if name == "InNamespace" || name == "List" {
					// Mark this method as coming from a Kubernetes package
					sig := types.NewSignatureType(nil, nil, nil, types.NewTuple(), types.NewTuple(), false)
					info.Uses[se.Sel] = types.NewFunc(token.NoPos, pkgCR, name, sig)
				}
			} else if id, ok := n.(*ast.Ident); ok {
				// Handle standalone InNamespace function calls
				if id.Name == "InNamespace" {
					sig := types.NewSignatureType(nil, nil, nil, types.NewTuple(), types.NewTuple(), false)
					info.Uses[id] = types.NewFunc(token.NoPos, pkgCR, id.Name, sig)
				}
			}
			return true
		})
	}

	var diags []analysis.Diagnostic
	pass := &analysis.Pass{Analyzer: AnalyzerWideNamespace, Fset: fset, Files: files, TypesInfo: info, TypesSizes: types.SizesFor("gc", "amd64"), Report: func(d analysis.Diagnostic) { diags = append(diags, d) }, ResultOf: map[*analysis.Analyzer]interface{}{insppass.Analyzer: inspector.New(files)}}
	_, _ = AnalyzerWideNamespace.Run(pass)
	return diags
}

func TestWideNamespace_InNamespaceEmpty_Flagged(t *testing.T) {
	src := `package a
type Opts struct{}
func InNamespace(ns string) Opts { return Opts{} }
type Client interface{ List(ctx any, obj any, opts ...Opts) error }
func f(c Client){ var o struct{}; _ = c.List(nil, &o, InNamespace("")) }`
	diags := runWideNamespaceAnalyzerOnSrc(t, src, true) // spoof as Kubernetes types
	if len(diags) == 0 {
		t.Fatalf("expected diagnostic for InNamespace(\"\")")
	}
}

func TestWideNamespace_TypedChain_PodsEmpty_Flagged(t *testing.T) {
	src := `package a
type PodsIFace interface{ List(ctx any) error }
type CoreV1 interface{ Pods(ns string) PodsIFace }
func f(c CoreV1){ _ = c.Pods("").List(nil) }`
	diags := runWideNamespaceAnalyzerOnSrc(t, src, true) // spoof as Kubernetes types
	if len(diags) == 0 {
		t.Fatalf("expected diagnostic for typed Pods(\"\").List")
	}
}

func TestWideNamespace_NonKubernetesClient_NoDiag(t *testing.T) {
	src := `package a
type DatabaseOpts struct{}
func InNamespace(ns string) DatabaseOpts { return DatabaseOpts{} }
type DatabaseClient interface{ List(opts ...DatabaseOpts) error }
func f(c DatabaseClient){ _ = c.List(InNamespace("")) }`
	diags := runWideNamespaceAnalyzerOnSrc(t, src, false) // don't spoof - should be treated as non-Kubernetes
	if len(diags) != 0 {
		t.Fatalf("expected 0 diagnostics for non-Kubernetes InNamespace calls, got %d", len(diags))
	}
}

func TestWideNamespace_InNamespaceWithValue_NoDiag(t *testing.T) {
	src := `package a
type Opts struct{}
func InNamespace(ns string) Opts { return Opts{} }
type Client interface{ List(ctx any, obj any, opts ...Opts) error }
func f(c Client){ var o struct{}; _ = c.List(nil, &o, InNamespace("default")) }`
	diags := runWideNamespaceAnalyzerOnSrc(t, src, true) // spoof as Kubernetes types
	if len(diags) != 0 {
		t.Fatalf("expected 0 diagnostics when InNamespace has a value, got %d", len(diags))
	}
}
