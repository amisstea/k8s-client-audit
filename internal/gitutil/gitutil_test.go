package gitutil

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=test@example.com",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, string(out))
	}
	return string(out)
}

func TestShallowCloneAndFetchUpdate(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "src")
	os.MkdirAll(src, 0o755)
	git(t, src, "init")
	// ensure main branch
	git(t, src, "checkout", "-b", "main")
	os.WriteFile(filepath.Join(src, "file.txt"), []byte("v1"), 0o644)
	git(t, src, "add", ".")
	git(t, src, "commit", "-m", "initial")

	dest := filepath.Join(tmp, "clone")
	if err := ShallowClone(src, dest, "main", 1, 30*time.Second); err != nil {
		t.Fatalf("shallow clone: %v", err)
	}
	// confirm limited history by checking commit count is 1
	out := git(t, dest, "rev-list", "--count", "HEAD")
	if out != "1\n" {
		t.Fatalf("expected shallow history with 1 commit, got %q", out)
	}
	// Record current HEAD of src
	head1 := git(t, src, "rev-parse", "HEAD")

	// Make another commit in src
	os.WriteFile(filepath.Join(src, "file.txt"), []byte("v2"), 0o644)
	git(t, src, "add", ".")
	git(t, src, "commit", "-m", "update")
	head2 := git(t, src, "rev-parse", "HEAD")
	if head1 == head2 {
		t.Fatalf("expected new commit")
	}

	if err := FetchAndCheckoutLatest(dest, "main", 1, 30*time.Second); err != nil {
		t.Fatalf("fetch update: %v", err)
	}
	// Verify clone HEAD matches src HEAD
	cloneHead := git(t, dest, "rev-parse", "HEAD")
	if cloneHead != head2 {
		t.Fatalf("expected clone head %q to equal src head %q", cloneHead, head2)
	}
}
