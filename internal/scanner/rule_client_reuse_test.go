package scanner

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestRuleClientReuse_InLoop(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module tmp\n\ngo 1.21\n"), 0o644)
	src := `package a
import (
  clientset "k8s.io/client-go/kubernetes"
  "k8s.io/client-go/rest"
)
func f(){ cfg, _ := rest.InClusterConfig(); for i:=0;i<3;i++{ _, _ = clientset.NewForConfig(cfg) } }`
	_ = os.WriteFile(filepath.Join(dir, "a.go"), []byte(src), 0o644)

	fset, pkgs, err := LoadPackages(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(pkgs) == 0 {
		t.Fatalf("no pkgs")
	}
	r := NewRuleClientReuse()
	issues, err := r.Apply(context.Background(), fset, pkgs[0])
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if len(issues) == 0 {
		t.Fatalf("expected an issue for client creation inside loop")
	}
}

func TestRuleClientReuse_InReconcile(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module tmp\n\ngo 1.21\n"), 0o644)
	src := `package a
import (
  clientset "k8s.io/client-go/kubernetes"
  "k8s.io/client-go/rest"
)
type Reconciler struct{}
func (r *Reconciler) Reconcile(){ cfg, _ := rest.InClusterConfig(); _, _ = clientset.NewForConfig(cfg) }`
	_ = os.WriteFile(filepath.Join(dir, "a.go"), []byte(src), 0o644)

	fset, pkgs, err := LoadPackages(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(pkgs) == 0 {
		t.Fatalf("no pkgs")
	}
	r := NewRuleClientReuse()
	issues, err := r.Apply(context.Background(), fset, pkgs[0])
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if len(issues) == 0 {
		t.Fatalf("expected an issue for client creation in Reconcile")
	}
}

func TestRuleClientReuse_NotTriggered_OutsideHotPath(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module tmp\n\ngo 1.21\n"), 0o644)
	src := `package a
import (
  clientset "k8s.io/client-go/kubernetes"
  "k8s.io/client-go/rest"
)
func init(){ cfg, _ := rest.InClusterConfig(); _, _ = clientset.NewForConfig(cfg) }`
	_ = os.WriteFile(filepath.Join(dir, "a.go"), []byte(src), 0o644)

	fset, pkgs, err := LoadPackages(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(pkgs) == 0 {
		t.Fatalf("no pkgs")
	}
	r := NewRuleClientReuse()
	issues, err := r.Apply(context.Background(), fset, pkgs[0])
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if len(issues) != 0 {
		t.Fatalf("did not expect issues for init-time client creation, got %d", len(issues))
	}
}
