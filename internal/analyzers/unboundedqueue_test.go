package analyzers

import (
	"testing"

	"github.com/amisstea/k8s-client-audit/internal/analyzers/testutil"

	"golang.org/x/tools/go/analysis"
)

func runUnboundedQueueAnalyzerOnSrc(t *testing.T, src string) []analysis.Diagnostic {
	t.Helper()
	diags, err := testutil.RunAnalyzerOnSrc(AnalyzerUnboundedQueue, src, testutil.SpoofWorkqueue)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	return diags
}

func TestUnboundedQueue_WorkqueueNew_Flagged(t *testing.T) {
	src := `package a
import wq "k8s.io/client-go/util/workqueue"
func f(){ _ = wq.New() }`
	diags := runUnboundedQueueAnalyzerOnSrc(t, src)
	if len(diags) == 0 {
		t.Fatalf("expected diagnostic for workqueue.New without rate limiter")
	}
}

func TestUnboundedQueue_RateLimitingQueue_NoDiag(t *testing.T) {
	src := `package a
import wq "k8s.io/client-go/util/workqueue"
func NewItemExponentialFailureRateLimiter() any { return nil }
func f(){ rl := NewItemExponentialFailureRateLimiter(); _ = wq.NewRateLimitingQueue(rl) }`
	diags := runUnboundedQueueAnalyzerOnSrc(t, src)
	if len(diags) != 0 {
		t.Fatalf("did not expect diagnostic for NewRateLimitingQueue")
	}
}

func TestUnboundedQueue_NonWorkqueue_New_NoDiag(t *testing.T) {
	src := `package a
func New() any { return nil }
func f(){ _ = New() }`
	diags := runUnboundedQueueAnalyzerOnSrc(t, src)
	if len(diags) != 0 {
		t.Fatalf("did not expect diagnostic for non-workqueue New")
	}
}
