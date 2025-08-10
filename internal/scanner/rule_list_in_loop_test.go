package scanner

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestRuleListInLoop(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module tmp\n\ngo 1.21\n"), 0o644)
	src := `package a
import (
  corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)
func f(c corev1.PodInterface){
  for i:=0;i<3;i++{ c.List(nil) }
}`
	_ = os.WriteFile(filepath.Join(dir, "a.go"), []byte(src), 0o644)

	fset, pkgs, err := LoadPackages(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(pkgs) == 0 {
		t.Fatalf("no pkgs")
	}
	r := NewRuleListInLoop()
	issues, err := r.Apply(context.Background(), fset, pkgs[0])
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if len(issues) == 0 {
		t.Fatalf("expected at least one issue, got 0")
	}
}
