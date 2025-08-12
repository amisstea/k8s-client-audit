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

func runDiscoveryFloodAnalyzerOnSrc(t *testing.T, src string) []analysis.Diagnostic {
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
	// Spoof discovery and restmapper functions to expected packages for type-based detection
	pkgDisc := types.NewPackage("k8s.io/client-go/discovery", "discovery")
	pkgRM := types.NewPackage("k8s.io/client-go/restmapper", "restmapper")
	ast.Inspect(f, func(n ast.Node) bool {
		if ce, ok := n.(*ast.CallExpr); ok {
			switch fun := ce.Fun.(type) {
			case *ast.Ident:
				switch fun.Name {
				case "NewDiscoveryClientForConfig":
					info.Uses[fun] = types.NewFunc(token.NoPos, pkgDisc, fun.Name, nil)
				case "NewDeferredDiscoveryRESTMapper", "ResetRESTMapper":
					info.Uses[fun] = types.NewFunc(token.NoPos, pkgRM, fun.Name, nil)
				}
			case *ast.SelectorExpr:
				if fun.Sel != nil {
					switch fun.Sel.Name {
					case "NewDiscoveryClientForConfig":
						info.Uses[fun.Sel] = types.NewFunc(token.NoPos, pkgDisc, fun.Sel.Name, nil)
					case "NewDeferredDiscoveryRESTMapper", "ResetRESTMapper":
						info.Uses[fun.Sel] = types.NewFunc(token.NoPos, pkgRM, fun.Sel.Name, nil)
					}
				}
			}
		}
		return true
	})
	var diags []analysis.Diagnostic
	pass := &analysis.Pass{Analyzer: AnalyzerDiscoveryFlood, Fset: fset, Files: files, TypesInfo: info, TypesSizes: types.SizesFor("gc", "amd64"), Report: func(d analysis.Diagnostic) { diags = append(diags, d) }, ResultOf: map[*analysis.Analyzer]interface{}{insppass.Analyzer: inspector.New(files)}}
	_, _ = AnalyzerDiscoveryFlood.Run(pass)
	return diags
}

func TestDiscoveryFlood_RepeatedInLoop_Flagged(t *testing.T) {
	src := `package a
func NewDiscoveryClientForConfig(x any) any { return nil }
func f(){ for { _ = NewDiscoveryClientForConfig(nil) } }`
	diags := runDiscoveryFloodAnalyzerOnSrc(t, src)
	if len(diags) == 0 {
		t.Fatalf("expected diagnostic for discovery in loop")
	}
}

func TestDiscoveryFlood_OutsideLoop_NoDiag(t *testing.T) {
	src := `package a
func NewDiscoveryClientForConfig(x any) any { return nil }
func f(){ _ = NewDiscoveryClientForConfig(nil) }`
	diags := runDiscoveryFloodAnalyzerOnSrc(t, src)
	if len(diags) != 0 {
		t.Fatalf("did not expect diagnostic outside loop")
	}
}
