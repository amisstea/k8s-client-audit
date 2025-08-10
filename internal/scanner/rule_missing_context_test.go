package scanner

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestRuleMissingContext(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module tmp\n\ngo 1.21\n"), 0o644)
	// client using context.Background on method call
	src := `package a
type C struct{}
import "context"
func (C) Get(ctx context.Context, a int){}
func f(c C){ c.Get(context.Background(), 1) }`
	_ = os.WriteFile(filepath.Join(dir, "a.go"), []byte(src), 0o644)

	fset, pkgs, err := LoadPackages(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(pkgs) == 0 {
		t.Fatalf("no pkgs")
	}
	r := NewRuleMissingContext()
	issues, err := r.Apply(context.Background(), fset, pkgs[0])
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if len(issues) == 0 {
		t.Fatalf("expected at least one issue, got 0")
	}
}
