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

func runNoRetryTransientAnalyzerOnSrc(t *testing.T, src string) []analysis.Diagnostic {
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
	pass := &analysis.Pass{Analyzer: AnalyzerNoRetryTransient, Fset: fset, Files: files, TypesInfo: info, TypesSizes: types.SizesFor("gc", "amd64"), Report: func(d analysis.Diagnostic) { diags = append(diags, d) }, ResultOf: map[*analysis.Analyzer]interface{}{insppass.Analyzer: inspector.New(files)}}
	_, _ = AnalyzerNoRetryTransient.Run(pass)
	return diags
}

func TestNoRetryTransient_TransientWithoutRetry_Flagged(t *testing.T) {
	src := `package a
import "k8s.io/client-go/kubernetes"
func f(err any){ if Timeout { /* no retry */ } }`
	diags := runNoRetryTransientAnalyzerOnSrc(t, src)
	if len(diags) == 0 {
		t.Fatalf("expected diagnostic for Kubernetes-related transient error without retry")
	}
}

func TestNoRetryTransient_WithRetry_NoDiag(t *testing.T) {
	src := `package a
import "sigs.k8s.io/controller-runtime/pkg/client"
func Backoff(){}
func f(err any){ if Temporary { Backoff() } }`
	diags := runNoRetryTransientAnalyzerOnSrc(t, src)
	if len(diags) != 0 {
		t.Fatalf("did not expect diagnostic when retry present")
	}
}

func TestNoRetryTransient_NonKubernetesCode_NoDiag(t *testing.T) {
	src := `package a
func f(err any){ if Timeout { /* no retry */ } }`
	diags := runNoRetryTransientAnalyzerOnSrc(t, src)
	if len(diags) != 0 {
		t.Fatalf("expected 0 diagnostics for non-Kubernetes code, got %d", len(diags))
	}
}

func TestNoRetryTransient_NoTransientError_NoDiag(t *testing.T) {
	src := `package a
import "k8s.io/client-go/kubernetes"
func f(err any){ if SomeOtherError { /* no retry needed */ } }`
	diags := runNoRetryTransientAnalyzerOnSrc(t, src)
	if len(diags) != 0 {
		t.Fatalf("expected 0 diagnostics when no transient error mentioned, got %d", len(diags))
	}
}
