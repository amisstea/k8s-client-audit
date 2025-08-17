package analyzers

import (
	"testing"

	"github.com/amisstea/k8s-client-audit/internal/analyzers/testutil"

	"golang.org/x/tools/go/analysis"
)

func runClientReuseAnalyzerOnSrc(t *testing.T, src string) []analysis.Diagnostic {
	t.Helper()
	diags, err := testutil.RunAnalyzerOnSrc(AnalyzerClientReuse, src, testutil.SpoofCommonK8s)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	return diags
}

func TestClientReuse_InLoop_Flagged(t *testing.T) {
	src := `package a
func NewForConfig(x any) any { return nil }
func f(){ for i:=0;i<3;i++{ _ = NewForConfig(nil) } }`
	diags := runClientReuseAnalyzerOnSrc(t, src)
	if len(diags) == 0 {
		t.Fatalf("expected diagnostic for client creation in loop")
	}
}

func TestClientReuse_InHotPath_Flagged(t *testing.T) {
	src := `package a
func NewForConfig(x any) any { return nil }
type Reconciler struct{}
func (r *Reconciler) Reconcile(){ _ = NewForConfig(nil) }`
	diags := runClientReuseAnalyzerOnSrc(t, src)
	if len(diags) == 0 {
		t.Fatalf("expected diagnostic for client creation in hot path")
	}
}

func TestClientReuse_Init_NoDiag(t *testing.T) {
	src := `package a
func NewForConfig(x any) any { return nil }
func init(){ _ = NewForConfig(nil) }`
	diags := runClientReuseAnalyzerOnSrc(t, src)
	if len(diags) != 0 {
		t.Fatalf("did not expect diagnostic for init-time creation, got %d", len(diags))
	}
}
