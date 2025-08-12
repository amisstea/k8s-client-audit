package analyzers

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"golang.org/x/tools/go/analysis"
)

func runExcessiveConfigAnalyzerOnSrc(t *testing.T, src string) []analysis.Diagnostic {
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
	// Spoof Reconcile signature for hot-path detection
	ast.Inspect(f, func(n ast.Node) bool {
		fd, ok := n.(*ast.FuncDecl)
		if !ok || fd.Name == nil || fd.Name.Name != "Reconcile" {
			return true
		}
		pkgRec := types.NewPackage("sigs.k8s.io/controller-runtime/pkg/reconcile", "reconcile")
		resNamed := types.NewNamed(types.NewTypeName(token.NoPos, pkgRec, "Result", nil), types.NewStruct(nil, nil), nil)
		resTuple := types.NewTuple(
			types.NewVar(token.NoPos, nil, "", resNamed),
			types.NewVar(token.NoPos, nil, "", types.Universe.Lookup("error").Type()),
		)
		sig := types.NewSignatureType(nil, nil, nil, types.NewTuple(), resTuple, false)
		info.Defs[fd.Name] = types.NewFunc(token.NoPos, nil, fd.Name.Name, sig)
		return false
	})
	var diags []analysis.Diagnostic
	pass := &analysis.Pass{Analyzer: AnalyzerExcessiveConfig, Fset: fset, Files: files, TypesInfo: info, TypesSizes: types.SizesFor("gc", "amd64"), Report: func(d analysis.Diagnostic) { diags = append(diags, d) }, ResultOf: map[*analysis.Analyzer]interface{}{}}
	_, _ = AnalyzerExcessiveConfig.Run(pass)
	return diags
}

func TestExcessiveConfig_InLoop_Flagged(t *testing.T) {
	src := `package a
type Clientset interface{}
func NewForConfig(x any) Clientset { return nil }
func f(){ for i:=0;i<3;i++{ _ = NewForConfig(nil) } }`
	diags := runExcessiveConfigAnalyzerOnSrc(t, src)
	if len(diags) == 0 {
		t.Fatalf("expected diagnostic for client creation in loop")
	}
}

func TestExcessiveConfig_InHotPath_Flagged(t *testing.T) {
	src := `package a
type Clientset interface{}
func NewForConfig(x any) Clientset { return nil }
type Reconciler struct{}
func (r *Reconciler) Reconcile(){ _ = NewForConfig(nil) }`
	diags := runExcessiveConfigAnalyzerOnSrc(t, src)
	if len(diags) == 0 {
		t.Fatalf("expected diagnostic for client creation in hot path")
	}
}

func TestExcessiveConfig_Init_NoDiag(t *testing.T) {
	src := `package a
type Clientset interface{}
func NewForConfig(x any) Clientset { return nil }
func init(){ _ = NewForConfig(nil) }`
	diags := runExcessiveConfigAnalyzerOnSrc(t, src)
	if len(diags) != 0 {
		t.Fatalf("did not expect diagnostic for init-time creation, got %d", len(diags))
	}
}
