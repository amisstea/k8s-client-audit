package scanner

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestRuleNoSelectors_ControllerRuntime_NoOptions(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module tmp\n\ngo 1.21\n"), 0o644)
	src := `package a
type Obj struct{}
type Client interface{ List(ctx any, obj any, opts ...any) error }
func f(c Client){ var o Obj; _ = c.List(nil, &o) }`
	_ = os.WriteFile(filepath.Join(dir, "a.go"), []byte(src), 0o644)

	fset, pkgs, err := LoadPackages(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(pkgs) == 0 {
		t.Fatalf("no pkgs")
	}
	r := NewRuleNoSelectors()
	issues, err := r.Apply(context.Background(), fset, pkgs[0])
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if len(issues) == 0 {
		t.Fatalf("expected issues for List without options")
	}
}

func TestRuleNoSelectors_ClientGo_ListOptions_NoSelectors(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module tmp\n\ngo 1.21\n"), 0o644)
	src := `package a
type Opts struct{ LabelSelector string; FieldSelector string }
type IFace interface{ List(ctx any, opts Opts) error }
func f(c IFace){ _ = c.List(nil, Opts{}) }`
	_ = os.WriteFile(filepath.Join(dir, "a.go"), []byte(src), 0o644)

	fset, pkgs, err := LoadPackages(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(pkgs) == 0 {
		t.Fatalf("no pkgs")
	}
	r := NewRuleNoSelectors()
	issues, err := r.Apply(context.Background(), fset, pkgs[0])
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if len(issues) == 0 {
		t.Fatalf("expected issues for empty selectors")
	}
}

func TestRuleNoSelectors_ControllerRuntime_WithMatchingLabels(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module tmp\n\ngo 1.21\n"), 0o644)
	src := `package a
type Opts interface{}
func MatchingLabels(m map[string]string) Opts { return nil }
type Client interface{ List(ctx any, obj any, opts ...Opts) error }
func f(c Client){ var o struct{}; _ = c.List(nil, &o, MatchingLabels(map[string]string{"k":"v"})) }`
	_ = os.WriteFile(filepath.Join(dir, "a.go"), []byte(src), 0o644)

	fset, pkgs, err := LoadPackages(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(pkgs) == 0 {
		t.Fatalf("no pkgs")
	}
	r := NewRuleNoSelectors()
	issues, err := r.Apply(context.Background(), fset, pkgs[0])
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if len(issues) != 0 {
		t.Fatalf("did not expect issues when MatchingLabels is provided, got %d", len(issues))
	}
}

func TestRuleNoSelectors_ControllerRuntime_WithMatchingFieldsSelector(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module tmp\n\ngo 1.21\n"), 0o644)
	src := `package a
type Opts interface{}
type Selector struct{}
func MatchingFieldsSelector(_ Selector) Opts { return nil }
type Client interface{ List(ctx any, obj any, opts ...Opts) error }
func f(c Client){ var o struct{}; _ = c.List(nil, &o, MatchingFieldsSelector(Selector{})) }`
	_ = os.WriteFile(filepath.Join(dir, "a.go"), []byte(src), 0o644)

	fset, pkgs, err := LoadPackages(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(pkgs) == 0 {
		t.Fatalf("no pkgs")
	}
	r := NewRuleNoSelectors()
	issues, err := r.Apply(context.Background(), fset, pkgs[0])
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if len(issues) != 0 {
		t.Fatalf("did not expect issues when MatchingFieldsSelector is provided, got %d", len(issues))
	}
}

func TestRuleNoSelectors_ControllerRuntime_WithCompositeMatchingLabels(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module tmp\n\ngo 1.21\n"), 0o644)
	src := `package a
type Opts interface{}
type MatchingLabels struct{ M map[string]string }
type Client interface{ List(ctx any, obj any, opts ...Opts) error }
func f(c Client){ var o struct{}; _ = c.List(nil, &o, MatchingLabels{M: map[string]string{"k":"v"}}) }`
	_ = os.WriteFile(filepath.Join(dir, "a.go"), []byte(src), 0o644)

	fset, pkgs, err := LoadPackages(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(pkgs) == 0 {
		t.Fatalf("no pkgs")
	}
	r := NewRuleNoSelectors()
	issues, err := r.Apply(context.Background(), fset, pkgs[0])
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if len(issues) != 0 {
		t.Fatalf("did not expect issues when composite MatchingLabels is provided, got %d", len(issues))
	}
}
