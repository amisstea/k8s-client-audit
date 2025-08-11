package analyzers

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"golang.org/x/tools/go/analysis"
)

func runTightAnalyzerOnSrc(t *testing.T, src string) []analysis.Diagnostic {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "a.go", src, 0)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	files := []*ast.File{f}
	info := &types.Info{Types: map[ast.Expr]types.TypeAndValue{}, Defs: map[*ast.Ident]types.Object{}, Uses: map[*ast.Ident]types.Object{}, Selections: map[*ast.SelectorExpr]*types.Selection{}}
	var conf types.Config
	_, _ = conf.Check("p", fset, files, info)
	var diags []analysis.Diagnostic
	pass := &analysis.Pass{Analyzer: AnalyzerTightErrorLoops, Fset: fset, Files: files, TypesInfo: info, TypesSizes: types.SizesFor("gc", "amd64"), Report: func(d analysis.Diagnostic) { diags = append(diags, d) }, ResultOf: map[*analysis.Analyzer]interface{}{}}
	_, _ = AnalyzerTightErrorLoops.Run(pass)
	return diags
}

func TestTightErrorLoops_WithAPICall_NoSleep(t *testing.T) {
	src := `package a
type PodsIFace interface{ List(ctx any) error }
type CoreV1 interface{ Pods(ns string) PodsIFace }
func f(c CoreV1){ var err error; for { if err != nil { _ = c.Pods("").List(nil) } else { break } } }`
	diags := runTightAnalyzerOnSrc(t, src)
	if len(diags) == 0 {
		t.Fatalf("expected diagnostic for tight loop without sleep around API call")
	}
}

func TestTightErrorLoops_NoAPICall_NoDiag(t *testing.T) {
	src := `package a
func f(){ var err error; for { if err != nil { _ = 1+1 } else { break } } }`
	diags := runTightAnalyzerOnSrc(t, src)
	if len(diags) != 0 {
		t.Fatalf("did not expect diagnostic without API call, got %d", len(diags))
	}
}

func TestTightErrorLoops_WithSleep_NoDiag(t *testing.T) {
	src := `package a
import "time"
type PodsIFace interface{ List(ctx any) error }
type CoreV1 interface{ Pods(ns string) PodsIFace }
func f(c CoreV1){ var err error; for { if err != nil { _ = c.Pods("").List(nil); time.Sleep(100) } else { break } } }`
	diags := runTightAnalyzerOnSrc(t, src)
	if len(diags) != 0 {
		t.Fatalf("did not expect diagnostic when sleep present, got %d", len(diags))
	}
}
