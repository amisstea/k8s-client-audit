package gitutil

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

func run(ctx context.Context, dir string, timeout time.Duration, name string, args ...string) (string, error) {
	ctx2, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx2, name, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	outStr := stdout.String()
	if err != nil {
		if outStr == "" {
			outStr = stderr.String()
		} else {
			outStr = outStr + "\n" + stderr.String()
		}
		return outStr, fmt.Errorf("%s %s: %w", name, strings.Join(args, " "), err)
	}
	return outStr, nil
}

// ShallowClone clones repository at the given branch with limited history depth.
func ShallowClone(repoURL, destDir, branch string, depth int, timeout time.Duration) error {
	if branch == "" {
		branch = "main"
	}
	args := []string{"clone", "--depth", fmt.Sprintf("%d", depth), "--single-branch", "--branch", branch, repoURL, destDir}
	_, err := run(context.Background(), "", timeout, "git", args...)
	return err
}

// FetchAndCheckoutLatest updates an existing repository to the latest commit on branch.
func FetchAndCheckoutLatest(repoDir, branch string, depth int, timeout time.Duration) error {
	if branch == "" {
		branch = "main"
	}
	// Ensure we have the branch locally
	_, _ = run(context.Background(), repoDir, timeout, "git", "fetch", "--depth", fmt.Sprintf("%d", depth), "origin", branch)
	// Create or switch branch to track origin/branch
	// Try to checkout the branch (create if doesn't exist)
	if _, err := run(context.Background(), repoDir, timeout, "git", "checkout", branch); err != nil {
		// Create local branch pointing to origin/branch
		_, _ = run(context.Background(), repoDir, timeout, "git", "checkout", "-B", branch, "origin/"+branch)
	}
	// Hard reset to origin/branch to ensure latest
	_, err := run(context.Background(), repoDir, timeout, "git", "reset", "--hard", "origin/"+branch)
	return err
}
