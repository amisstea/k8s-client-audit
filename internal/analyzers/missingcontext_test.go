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

func runMissingCtxAnalyzerOnSrc(t *testing.T, src string, spoof bool) []analysis.Diagnostic {
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
				if name == "Get" || name == "List" || name == "Create" || name == "Update" || name == "Patch" || name == "Delete" {
					// Mark this method as coming from a Kubernetes client package
					sig := types.NewSignatureType(nil, nil, nil, types.NewTuple(), types.NewTuple(), false)
					info.Uses[se.Sel] = types.NewFunc(token.NoPos, pkgCR, name, sig)
				}
			}
			return true
		})
	}

	var diags []analysis.Diagnostic
	pass := &analysis.Pass{Analyzer: AnalyzerMissingContext, Fset: fset, Files: files, TypesInfo: info, TypesSizes: types.SizesFor("gc", "amd64"), Report: func(d analysis.Diagnostic) { diags = append(diags, d) }, ResultOf: map[*analysis.Analyzer]interface{}{insppass.Analyzer: inspector.New(files)}}
	_, _ = AnalyzerMissingContext.Run(pass)
	return diags
}

func TestMissingContext_BackgroundFlagged(t *testing.T) {
	src := `package a
import "context"
type C interface{ Get(ctx any) }
func f(c C){ c.Get(context.Background()) }`
	diags := runMissingCtxAnalyzerOnSrc(t, src, true) // spoof as Kubernetes types
	if len(diags) == 0 {
		t.Fatalf("expected diagnostic for Background")
	}
}

func TestMissingContext_Propagated_NoDiag(t *testing.T) {
	src := `package a
type C interface{ Get(ctx any) }
func f(c C, ctx any){ c.Get(ctx) }`
	diags := runMissingCtxAnalyzerOnSrc(t, src, true) // spoof as Kubernetes types
	if len(diags) != 0 {
		t.Fatalf("expected 0 diagnostics, got %d", len(diags))
	}
}

func TestMissingContext_GitHubClient_NoDiag(t *testing.T) {
	src := `package a
import "context"
type GitHubAppsService struct{}
func (g *GitHubAppsService) Get(ctx context.Context, slug string) (interface{}, interface{}, error) { return nil, nil, nil }
type GitHubClient struct{ Apps *GitHubAppsService }
func f(client *GitHubClient){ client.Apps.Get(context.Background(), "") }`
	diags := runMissingCtxAnalyzerOnSrc(t, src, false) // don't spoof - should be treated as non-Kubernetes
	if len(diags) != 0 {
		t.Fatalf("expected 0 diagnostics for GitHub client calls, got %d", len(diags))
	}
}

func TestMissingContext_GenericClient_NoDiag(t *testing.T) {
	src := `package a
import "context"
type HTTPClient interface{ Get(ctx context.Context, url string) error }
func f(client HTTPClient){ client.Get(context.Background(), "https://api.github.com") }`
	diags := runMissingCtxAnalyzerOnSrc(t, src, false) // don't spoof - should be treated as non-Kubernetes
	if len(diags) != 0 {
		t.Fatalf("expected 0 diagnostics for non-Kubernetes client calls, got %d", len(diags))
	}
}

func TestMissingContext_KubernetesClient_Flagged(t *testing.T) {
	src := `package a
import "context"
type KubernetesClient interface{ Get(ctx context.Context, name string) error }
func f(client KubernetesClient){ client.Get(context.Background(), "my-pod") }`
	diags := runMissingCtxAnalyzerOnSrc(t, src, true) // spoof as Kubernetes types
	if len(diags) == 0 {
		t.Fatalf("expected diagnostic for Kubernetes client with Background context")
	}
}
