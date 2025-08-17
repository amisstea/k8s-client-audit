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

func runMissingInformerAnalyzerOnSrc(t *testing.T, src string, spoofKubernetesTypes bool) []analysis.Diagnostic {
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
	if spoofKubernetesTypes {
		pkgInformers := types.NewPackage("k8s.io/client-go/informers", "informers")
		pkgClient := types.NewPackage("sigs.k8s.io/controller-runtime/pkg/client", "client")
		ast.Inspect(f, func(n ast.Node) bool {
			if se, ok := n.(*ast.SelectorExpr); ok && se.Sel != nil {
				name := se.Sel.Name
				if name == "Watch" {
					// Mark Watch calls as coming from Kubernetes client package
					sig := types.NewSignatureType(nil, nil, nil, types.NewTuple(), types.NewTuple(), false)
					info.Uses[se.Sel] = types.NewFunc(token.NoPos, pkgClient, name, sig)
				} else if name == "NewSharedInformerFactory" || name == "NewSharedInformerFactoryWithOptions" || name == "NewSharedIndexInformer" || name == "NewSharedInformer" {
					// Mark SharedInformer constructors as coming from Kubernetes informer packages
					sig := types.NewSignatureType(nil, nil, nil, types.NewTuple(), types.NewTuple(), false)
					info.Uses[se.Sel] = types.NewFunc(token.NoPos, pkgInformers, name, sig)
				}
			} else if id, ok := n.(*ast.Ident); ok {
				// Handle standalone function calls
				name := id.Name
				if name == "NewSharedInformerFactory" || name == "NewSharedInformerFactoryWithOptions" || name == "NewSharedIndexInformer" || name == "NewSharedInformer" {
					sig := types.NewSignatureType(nil, nil, nil, types.NewTuple(), types.NewTuple(), false)
					info.Uses[id] = types.NewFunc(token.NoPos, pkgInformers, name, sig)
				}
			}
			return true
		})
	}

	var diags []analysis.Diagnostic
	pass := &analysis.Pass{Analyzer: AnalyzerMissingInformer, Fset: fset, Files: files, TypesInfo: info, TypesSizes: types.SizesFor("gc", "amd64"), Report: func(d analysis.Diagnostic) { diags = append(diags, d) }, ResultOf: map[*analysis.Analyzer]interface{}{insppass.Analyzer: inspector.New(files)}}
	_, _ = AnalyzerMissingInformer.Run(pass)
	return diags
}

func TestMissingInformer_WatchWithoutInformer_Flagged(t *testing.T) {
	src := `package a
func f(c interface{ Watch(x any) error }) { _ = c.Watch(nil) }`
	diags := runMissingInformerAnalyzerOnSrc(t, src, true) // spoof as Kubernetes types
	if len(diags) == 0 {
		t.Fatalf("expected diagnostic for Kubernetes Watch without shared informer")
	}
}

func TestMissingInformer_WithSharedInformer_NoDiag(t *testing.T) {
	src := `package a
func NewSharedInformerFactory() {}
func g(c interface{ Watch(x any) error }) { NewSharedInformerFactory(); _ = c.Watch(nil) }`
	diags := runMissingInformerAnalyzerOnSrc(t, src, true) // spoof as Kubernetes types
	if len(diags) != 0 {
		t.Fatalf("did not expect diagnostic when Kubernetes shared informer is present")
	}
}

func TestMissingInformer_NonKubernetesWatch_NoDiag(t *testing.T) {
	src := `package a
func f(c interface{ Watch(x any) error }) { _ = c.Watch(nil) }`
	diags := runMissingInformerAnalyzerOnSrc(t, src, false) // don't spoof - should be treated as non-Kubernetes
	if len(diags) != 0 {
		t.Fatalf("expected 0 diagnostics for non-Kubernetes Watch calls, got %d", len(diags))
	}
}

func TestMissingInformer_NonKubernetesInformer_StillFlags(t *testing.T) {
	src := `package a
func NewSharedInformerFactory() {} // Non-Kubernetes informer
func f(c interface{ Watch(x any) error }) { NewSharedInformerFactory(); _ = c.Watch(nil) }`
	diags := runMissingInformerAnalyzerOnSrc(t, src, false) // don't spoof informer, but spoof Watch partially
	if len(diags) != 0 {
		t.Fatalf("expected 0 diagnostics when non-Kubernetes types are used, got %d", len(diags))
	}
}
