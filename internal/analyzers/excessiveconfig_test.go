package analyzers

import (
	"go/ast"
	"go/token"
	"go/types"
	"testing"

	"cursor-experiment/internal/analyzers/testutil"

	"golang.org/x/tools/go/analysis"
)

func runExcessiveConfigAnalyzerOnSrc(t *testing.T, src string) []analysis.Diagnostic {
	t.Helper()
	diags, err := testutil.RunAnalyzerOnSrc(AnalyzerExcessiveConfig, src, func(f *ast.File, info *types.Info) {
		// Spoof client constructor funcs to expected package paths for type-based detection
		pkgKube := types.NewPackage("k8s.io/client-go/kubernetes", "kubernetes")
		ast.Inspect(f, func(n ast.Node) bool {
			ce, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			switch fun := ce.Fun.(type) {
			case *ast.Ident:
				if fun.Name == "NewForConfig" {
					info.Uses[fun] = types.NewFunc(token.NoPos, pkgKube, "NewForConfig", nil)
				}
			case *ast.SelectorExpr:
				if fun.Sel != nil && fun.Sel.Name == "NewForConfig" {
					info.Uses[fun.Sel] = types.NewFunc(token.NoPos, pkgKube, "NewForConfig", nil)
				}
			}
			return true
		})
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
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
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
