package analyzers

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"golang.org/x/tools/go/analysis"
)

func runManualPollingAnalyzerOnSrc(t *testing.T, src string) []analysis.Diagnostic {
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
	pass := &analysis.Pass{Analyzer: AnalyzerManualPolling, Fset: fset, Files: files, TypesInfo: info, TypesSizes: types.SizesFor("gc", "amd64"), Report: func(d analysis.Diagnostic) { diags = append(diags, d) }, ResultOf: map[*analysis.Analyzer]interface{}{}}
	_, _ = AnalyzerManualPolling.Run(pass)
	return diags
}

func TestManualPolling_ListWithSleep_Flagged(t *testing.T) {
	src := `package a
import "time"
type Client interface{ List(ctx any, obj any) error }
func f(c Client){ for { var o struct{}; _ = c.List(nil, &o); time.Sleep(100) } }`
	diags := runManualPollingAnalyzerOnSrc(t, src)
	if len(diags) == 0 {
		t.Fatalf("expected diagnostic for manual polling with List + Sleep")
	}
}

func TestManualPolling_Watch_NoDiag(t *testing.T) {
	src := `package a
import "time"
type IFace interface{ Watch(x any) error }
func f(c IFace){ for { _ = c.Watch(nil); time.Sleep(100) } }`
	diags := runManualPollingAnalyzerOnSrc(t, src)
	if len(diags) != 0 {
		t.Fatalf("did not expect diagnostic when using Watch")
	}
}
