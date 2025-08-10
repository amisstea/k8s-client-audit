package scanner

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestRuleNoFieldSelector_ControllerRuntime_NoFieldOpt(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module tmp\n\ngo 1.21\n"), 0o644)
	src := `package a
type Opts interface{}
func MatchingLabels(m map[string]string) Opts { return nil }
func MatchingFields(m map[string]string) Opts { return nil }
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
	r := NewRuleNoFieldSelector()
	issues, err := r.Apply(context.Background(), fset, pkgs[0])
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if len(issues) == 0 {
		t.Fatalf("expected an issue when field selector is missing")
	}
}

func TestRuleNoFieldSelector_ClientGo_ListOptions_NoField(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module tmp\n\ngo 1.21\n"), 0o644)
	src := `package a
type Opts struct{ LabelSelector string /* FieldSelector omitted */ }
type IFace interface{ List(ctx any, opts Opts) error }
func f(c IFace){ _ = c.List(nil, Opts{ LabelSelector: "k=v" }) }`
	_ = os.WriteFile(filepath.Join(dir, "a.go"), []byte(src), 0o644)

	fset, pkgs, err := LoadPackages(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(pkgs) == 0 {
		t.Fatalf("no pkgs")
	}
	r := NewRuleNoFieldSelector()
	issues, err := r.Apply(context.Background(), fset, pkgs[0])
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if len(issues) == 0 {
		t.Fatalf("expected an issue when FieldSelector is omitted")
	}
}

func TestRuleNoFieldSelector_NotTriggered_WhenFieldPresent(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module tmp\n\ngo 1.21\n"), 0o644)
	src := `package a
type Opts struct{ FieldSelector string }
type IFace interface{ List(ctx any, opts Opts) error }
func f(c IFace){ _ = c.List(nil, Opts{ FieldSelector: "metadata.name=my" }) }`
	_ = os.WriteFile(filepath.Join(dir, "a.go"), []byte(src), 0o644)

	fset, pkgs, err := LoadPackages(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(pkgs) == 0 {
		t.Fatalf("no pkgs")
	}
	r := NewRuleNoFieldSelector()
	issues, err := r.Apply(context.Background(), fset, pkgs[0])
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if len(issues) != 0 {
		t.Fatalf("did not expect issues when FieldSelector is present, got %d", len(issues))
	}
}
