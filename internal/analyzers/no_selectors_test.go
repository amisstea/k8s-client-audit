package analyzers

import (
	"testing"

	"github.com/amisstea/k8s-client-audit/internal/analyzers/testutil"

	"golang.org/x/tools/go/analysis"
)

func runNoSelectorsAnalyzerOnSrc(t *testing.T, src string) []analysis.Diagnostic {
	t.Helper()
	diags, err := testutil.RunAnalyzerOnSrc(AnalyzerNoSelectors, src, nil)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
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

func TestNoSelectors_ControllerRuntime_WithListOptions_NoDiag(t *testing.T) {
	src := `package a
type Opts interface{}
type ListOptions struct{ LabelSelector, FieldSelector any }
type Client interface{ List(ctx any, obj any, opts ...Opts) error }
func f(c Client){ var o struct{}; _ = c.List(nil, &o, &ListOptions{ LabelSelector: 1 }) }`
	diags := runNoSelectorsAnalyzerOnSrc(t, src)
	if len(diags) != 0 {
		t.Fatalf("did not expect diagnostic when *ListOptions with selectors is provided")
	}
}

func TestNoSelectors_ControllerRuntime_VariadicSlice_NoDiag(t *testing.T) {
	src := `package a
type Opts interface{}
type Client interface{ List(ctx any, obj any, opts ...Opts) error }
type MatchingFields map[string]string
func f(c Client){ var o struct{}; opts := []Opts{ MatchingFields{"a":"b"} }; _ = c.List(nil, &o, opts...) }`
	diags := runNoSelectorsAnalyzerOnSrc(t, src)
	if len(diags) != 0 {
		t.Fatalf("did not expect diagnostic when opts provided via variadic slice")
	}
}

func TestNoSelectors_ControllerRuntime_IdentListOptions_NoDiag(t *testing.T) {
	src := `package a
type Opts interface{}
type Client interface{ List(ctx any, obj any, opts ...Opts) error }
type ListOptions struct{ Namespace string; FieldSelector any }
type fieldsType struct{}
func (f fieldsType) OneTermEqualSelector(a string, b any) any { return nil }
var fields fieldsType
func f(c Client){ var o struct{}; opts := &ListOptions{ Namespace: "ns", FieldSelector: fields.OneTermEqualSelector("k","v") }; _ = c.List(nil, &o, opts) }`
	diags := runNoSelectorsAnalyzerOnSrc(t, src)
	if len(diags) != 0 {
		t.Fatalf("did not expect diagnostic when options are provided via ident with selectors")
	}
}

func TestNoSelectors_ControllerRuntime_IdentListOptions_NoSelectors_Flagged(t *testing.T) {
	src := `package a
type Opts interface{}
type Client interface{ List(ctx any, obj any, opts ...Opts) error }
type ListOptions struct{ Namespace string }
func f(c Client){ var o struct{}; opts := &ListOptions{ Namespace: "ns" }; _ = c.List(nil, &o, opts) }`
	diags := runNoSelectorsAnalyzerOnSrc(t, src)
	if len(diags) == 0 {
		t.Fatalf("expected diagnostic when ident options lack selectors")
	}
}
