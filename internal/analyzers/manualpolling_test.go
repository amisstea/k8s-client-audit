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

func runManualPollingAnalyzerOnSrc(t *testing.T, src string, spoofKubernetesTypes bool) []analysis.Diagnostic {
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

	// Optionally spoof type info to mark method calls as coming from Kubernetes/time packages
	if spoofKubernetesTypes {
		pkgCR := types.NewPackage("sigs.k8s.io/controller-runtime/pkg/client", "client")
		pkgTime := types.NewPackage("time", "time")
		ast.Inspect(f, func(n ast.Node) bool {
			if se, ok := n.(*ast.SelectorExpr); ok && se.Sel != nil {
				name := se.Sel.Name
				if name == "List" {
					// Mark this method as coming from a Kubernetes client package
					sig := types.NewSignatureType(nil, nil, nil, types.NewTuple(), types.NewTuple(), false)
					info.Uses[se.Sel] = types.NewFunc(token.NoPos, pkgCR, name, sig)
				} else if name == "Sleep" {
					// Mark this method as coming from time package
					sig := types.NewSignatureType(nil, nil, nil, types.NewTuple(), types.NewTuple(), false)
					info.Uses[se.Sel] = types.NewFunc(token.NoPos, pkgTime, name, sig)
				}
			}
			return true
		})
	}

	var diags []analysis.Diagnostic
	pass := &analysis.Pass{Analyzer: AnalyzerManualPolling, Fset: fset, Files: files, TypesInfo: info, TypesSizes: types.SizesFor("gc", "amd64"), Report: func(d analysis.Diagnostic) { diags = append(diags, d) }, ResultOf: map[*analysis.Analyzer]interface{}{insppass.Analyzer: inspector.New(files)}}
	_, _ = AnalyzerManualPolling.Run(pass)
	return diags
}

func TestManualPolling_ListWithSleep_Flagged(t *testing.T) {
	src := `package a
import "time"
type Client interface{ List(ctx any, obj any) error }
func f(c Client){ for { var o struct{}; _ = c.List(nil, &o); time.Sleep(100) } }`
	diags := runManualPollingAnalyzerOnSrc(t, src, true) // spoof as Kubernetes/time types
	if len(diags) == 0 {
		t.Fatalf("expected diagnostic for manual polling with List + Sleep")
	}
}

func TestManualPolling_Watch_NoDiag(t *testing.T) {
	src := `package a
import "time"
type IFace interface{ Watch(x any) error }
func f(c IFace){ for { _ = c.Watch(nil); time.Sleep(100) } }`
	diags := runManualPollingAnalyzerOnSrc(t, src, false) // don't spoof - Watch shouldn't be flagged anyway
	if len(diags) != 0 {
		t.Fatalf("did not expect diagnostic when using Watch")
	}
}

func TestManualPolling_NonKubernetesClient_NoDiag(t *testing.T) {
	src := `package a
import "time"
type DatabaseClient interface{ List() []string }
func f(c DatabaseClient){ for { _ = c.List(); time.Sleep(100) } }`
	diags := runManualPollingAnalyzerOnSrc(t, src, false) // don't spoof - should be treated as non-Kubernetes
	if len(diags) != 0 {
		t.Fatalf("expected 0 diagnostics for non-Kubernetes client calls, got %d", len(diags))
	}
}
