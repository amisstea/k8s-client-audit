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

func runLeakyWatchAnalyzerOnSrc(t *testing.T, src string, spoofKubernetesTypes bool) []analysis.Diagnostic {
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
		pkgCR := types.NewPackage("sigs.k8s.io/controller-runtime/pkg/client", "client")
		ast.Inspect(f, func(n ast.Node) bool {
			if se, ok := n.(*ast.SelectorExpr); ok && se.Sel != nil {
				name := se.Sel.Name
				if name == "ResultChan" {
					// Mark this method as coming from a Kubernetes package
					sig := types.NewSignatureType(nil, nil, nil, types.NewTuple(), types.NewTuple(), false)
					info.Uses[se.Sel] = types.NewFunc(token.NoPos, pkgCR, name, sig)
				} else if name == "Stop" || name == "Cancel" || name == "StopWatching" {
					// Mark this method as a stop method
					sig := types.NewSignatureType(nil, nil, nil, types.NewTuple(), types.NewTuple(), false)
					info.Uses[se.Sel] = types.NewFunc(token.NoPos, pkgCR, name, sig)
				}
			}
			return true
		})
	}

	var diags []analysis.Diagnostic
	pass := &analysis.Pass{Analyzer: AnalyzerLeakyWatch, Fset: fset, Files: files, TypesInfo: info, TypesSizes: types.SizesFor("gc", "amd64"), Report: func(d analysis.Diagnostic) { diags = append(diags, d) }, ResultOf: map[*analysis.Analyzer]interface{}{insppass.Analyzer: inspector.New(files)}}
	_, _ = AnalyzerLeakyWatch.Run(pass)
	return diags
}

func TestLeakyWatch_NoStop_Flagged(t *testing.T) {
	src := `package a
type W interface{ ResultChan() chan int; Stop() }
func f(w W){ ch := w.ResultChan(); _ = ch }`
	diags := runLeakyWatchAnalyzerOnSrc(t, src, true) // spoof as Kubernetes types
	if len(diags) == 0 {
		t.Fatalf("expected diagnostic for missing Stop/Cancel on watch")
	}
}

func TestLeakyWatch_WithStop_NoDiag(t *testing.T) {
	src := `package a
type W interface{ ResultChan() chan int; Stop() }
func f(w W){ ch := w.ResultChan(); _ = ch; w.Stop() }`
	diags := runLeakyWatchAnalyzerOnSrc(t, src, true) // spoof as Kubernetes types
	if len(diags) != 0 {
		t.Fatalf("did not expect diagnostic when Stop is called")
	}
}

func TestLeakyWatch_NonKubernetesClient_NoDiag(t *testing.T) {
	src := `package a
type EventChannel interface{ ResultChan() chan string }
func f(ec EventChannel){ ch := ec.ResultChan(); _ = ch }`
	diags := runLeakyWatchAnalyzerOnSrc(t, src, false) // don't spoof - should be treated as non-Kubernetes
	if len(diags) != 0 {
		t.Fatalf("expected 0 diagnostics for non-Kubernetes ResultChan calls, got %d", len(diags))
	}
}
