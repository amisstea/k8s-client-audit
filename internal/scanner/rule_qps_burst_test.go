package scanner

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestRuleQPSBurst_ConfigLiteralMissingOrBad(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module tmp\n\ngo 1.21\n"), 0o644)
	src := `package a
type Config struct{ QPS float32; Burst int }
var _ = Config{}
var _ = Config{QPS: 0}
var _ = Config{Burst: 0}
var _ = Config{QPS: 200000.0, Burst: 1}
`
	_ = os.WriteFile(filepath.Join(dir, "a.go"), []byte(src), 0o644)
	fset, pkgs, err := LoadPackages(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(pkgs) == 0 {
		t.Fatalf("no pkgs")
	}
	r := NewRuleQPSBurst()
	issues, err := r.Apply(context.Background(), fset, pkgs[0])
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if len(issues) < 3 {
		t.Fatalf("expected issues for missing/bad QPS/Burst, got %d", len(issues))
	}
}

func TestRuleQPSBurst_AssignmentsBad(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module tmp\n\ngo 1.21\n"), 0o644)
	src := `package a
type Config struct{ QPS float32; Burst int }
func f(){ var cfg Config; cfg.QPS = 0; cfg.Burst = 0; cfg.QPS = 200000.0; cfg.Burst = 1000000 }
`
	_ = os.WriteFile(filepath.Join(dir, "a.go"), []byte(src), 0o644)
	fset, pkgs, err := LoadPackages(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(pkgs) == 0 {
		t.Fatalf("no pkgs")
	}
	r := NewRuleQPSBurst()
	issues, err := r.Apply(context.Background(), fset, pkgs[0])
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if len(issues) < 3 {
		t.Fatalf("expected multiple issues for bad assignments, got %d", len(issues))
	}
}

func TestRuleQPSBurst_GoodValues_NoIssue(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module tmp\n\ngo 1.21\n"), 0o644)
	src := `package a
type Config struct{ QPS float32; Burst int }
func f(){ var cfg Config; cfg.QPS = 30; cfg.Burst = 100 }
`
	_ = os.WriteFile(filepath.Join(dir, "a.go"), []byte(src), 0o644)
	fset, pkgs, err := LoadPackages(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(pkgs) == 0 {
		t.Fatalf("no pkgs")
	}
	r := NewRuleQPSBurst()
	issues, err := r.Apply(context.Background(), fset, pkgs[0])
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if len(issues) != 0 {
		t.Fatalf("did not expect issues for reasonable values, got %d", len(issues))
	}
}
