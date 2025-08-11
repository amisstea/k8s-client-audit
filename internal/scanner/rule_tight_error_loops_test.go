package scanner

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestRuleTightErrorLoops_NoSleep_WithAPICall(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module tmp\n\ngo 1.21\n"), 0o644)
	src := `package a
// simulate typed client chain Pods("").List
type PodsIFace interface{ List(ctx any) error }
type CoreV1 interface{ Pods(ns string) PodsIFace }
func f(c CoreV1){ var err error; for { if err != nil { _ = c.Pods("").List(nil) } else { break } } }`
	_ = os.WriteFile(filepath.Join(dir, "a.go"), []byte(src), 0o644)

	fset, pkgs, err := LoadPackages(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(pkgs) == 0 {
		t.Fatalf("no pkgs")
	}
	r := NewRuleTightErrorLoops()
	issues, err := r.Apply(context.Background(), fset, pkgs[0])
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if len(issues) == 0 {
		t.Fatalf("expected an issue for tight error loop without sleep and with kube API call")
	}
}

func TestRuleTightErrorLoops_NoSleep_NoAPICall_NoIssue(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module tmp\n\ngo 1.21\n"), 0o644)
	src := `package a
func f(){ var err error; for { if err != nil { _ = 1+1 } else { break } } }`
	_ = os.WriteFile(filepath.Join(dir, "a.go"), []byte(src), 0o644)

	fset, pkgs, err := LoadPackages(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(pkgs) == 0 {
		t.Fatalf("no pkgs")
	}
	r := NewRuleTightErrorLoops()
	issues, err := r.Apply(context.Background(), fset, pkgs[0])
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if len(issues) != 0 {
		t.Fatalf("did not expect issue when no kube API call is present, got %d", len(issues))
	}
}
