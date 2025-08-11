package analyzers

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"golang.org/x/tools/go/analysis"
)

func runNoSelectorsAnalyzerOnSrc(t *testing.T, src string) []analysis.Diagnostic {
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
	var diags []analysis.Diagnostic
	pass := &analysis.Pass{Analyzer: AnalyzerNoSelectors, Fset: fset, Files: files, TypesInfo: info, TypesSizes: types.SizesFor("gc", "amd64"), Report: func(d analysis.Diagnostic) { diags = append(diags, d) }, ResultOf: map[*analysis.Analyzer]interface{}{}}
	_, _ = AnalyzerNoSelectors.Run(pass)
	return diags
}

func TestNoSelectors_ControllerRuntime_NoOpts_Flagged(t *testing.T) {
	src := `package a
type Opts interface{}
type Client interface{ List(ctx any, obj any, opts ...Opts) error }
func f(c Client){ var o struct{}; _ = c.List(nil, &o) }`
	diags := runNoSelectorsAnalyzerOnSrc(t, src)
	if len(diags) == 0 {
		t.Fatalf("expected diagnostic for controller-runtime List without opts")
	}
}

func TestNoSelectors_ControllerRuntime_MatchingLabels_NoDiag(t *testing.T) {
	src := `package a
type Opts interface{}
func MatchingLabels(m map[string]string) Opts { return nil }
type Client interface{ List(ctx any, obj any, opts ...Opts) error }
func f(c Client){ var o struct{}; _ = c.List(nil, &o, MatchingLabels(map[string]string{"k":"v"})) }`
	diags := runNoSelectorsAnalyzerOnSrc(t, src)
	if len(diags) != 0 {
		t.Fatalf("did not expect diagnostic when MatchingLabels provided")
	}
}

func TestNoSelectors_ClientGo_ListOptions_NoSelectors_Flagged(t *testing.T) {
	src := `package a
type ListOptions struct{ LabelSelector, FieldSelector string }
type IFace interface{ List(ctx any, opts ListOptions) error }
func f(c IFace){ _ = c.List(nil, ListOptions{}) }`
	diags := runNoSelectorsAnalyzerOnSrc(t, src)
	if len(diags) == 0 {
		t.Fatalf("expected diagnostic for ListOptions without selectors")
	}
}
