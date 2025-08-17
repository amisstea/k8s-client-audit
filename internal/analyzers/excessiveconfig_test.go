package analyzers

import (
	"testing"

	"github.com/amisstea/k8s-client-audit/internal/analyzers/testutil"

	"golang.org/x/tools/go/analysis"
)

func runExcessiveConfigAnalyzerOnSrc(t *testing.T, src string) []analysis.Diagnostic {
	t.Helper()
	diags, err := testutil.RunAnalyzerOnSrc(AnalyzerExcessiveConfig, src, testutil.CommonK8sSpoof)
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
