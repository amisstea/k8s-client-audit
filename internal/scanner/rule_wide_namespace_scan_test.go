package scanner

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestRuleWideNamespace_AllNamespaces(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module tmp\n\ngo 1.21\n"), 0o644)
	src := `package a
type Opts struct{}
type NSOpt func() Opts
func InNamespace(ns string) Opts { return Opts{} }
type Client interface{ List(ctx any, obj any, opts ...Opts) error }
// controller-runtime style: List with InNamespace("") option
func f1(c Client){ var o struct{}; _ = c.List(nil, &o, InNamespace("")) }
// typed client style: chain call Pods("").List
type PodsIFace interface{ List(ctx any) error }
type CoreV1 interface{ Pods(ns string) PodsIFace }
func g(c CoreV1){ _ = c.Pods("").List(nil) }`
	_ = os.WriteFile(filepath.Join(dir, "a.go"), []byte(src), 0o644)

	fset, pkgs, err := LoadPackages(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(pkgs) == 0 {
		t.Fatalf("no pkgs")
	}
	r := NewRuleWideNamespace()
	issues, err := r.Apply(context.Background(), fset, pkgs[0])
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if len(issues) < 2 {
		t.Fatalf("expected issues for both controller-runtime and typed client patterns, got %d", len(issues))
	}
}
