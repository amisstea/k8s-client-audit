package analyzers

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"golang.org/x/tools/go/analysis"
)

func runWebhookTimeoutsAnalyzerOnSrc(t *testing.T, src string) []analysis.Diagnostic {
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
	pass := &analysis.Pass{Analyzer: AnalyzerWebhookTimeouts, Fset: fset, Files: files, TypesInfo: info, TypesSizes: types.SizesFor("gc", "amd64"), Report: func(d analysis.Diagnostic) { diags = append(diags, d) }, ResultOf: map[*analysis.Analyzer]interface{}{}}
	_, _ = AnalyzerWebhookTimeouts.Run(pass)
	return diags
}

func TestWebhookTimeouts_ClientZero_Flagged(t *testing.T) {
	src := `package a
type Client struct{ Timeout int }
var _ = Client{Timeout:0}`
	diags := runWebhookTimeoutsAnalyzerOnSrc(t, src)
	if len(diags) == 0 {
		t.Fatalf("expected diagnostic for http.Client Timeout zero/missing")
	}
}

func TestWebhookTimeouts_ServerSet_NoDiag(t *testing.T) {
	src := `package a
type Server struct{ ReadTimeout, WriteTimeout, IdleTimeout int }
var _ = Server{ReadTimeout:10, WriteTimeout:10, IdleTimeout:10}`
	diags := runWebhookTimeoutsAnalyzerOnSrc(t, src)
	if len(diags) != 0 {
		t.Fatalf("did not expect diagnostic when server timeouts are set")
	}
}
