package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	githubclient "github.com/amisstea/k8s-client-audit/internal/githubclient"
	gitutil "github.com/amisstea/k8s-client-audit/internal/gitutil"
)

type Options struct {
	Org       string
	DestDir   string
	SkipClone bool
}

func run(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("clone-github-org", flag.ExitOnError)
	org := fs.String("org", "konflux-ci", "GitHub organization to clone")
	dest := fs.String("dest", "sources", "Destination directory for repositories")
	debug := fs.Bool("debug", false, "Enable debug logging across the app")
	if err := fs.Parse(args); err != nil {
		return err
	}

	opts := Options{Org: *org, DestDir: *dest}
	if opts.Org == "" {
		return errors.New("org must not be empty")
	}
	if opts.DestDir == "" {
		return errors.New("dest must not be empty")
	}

	if err := os.MkdirAll(opts.DestDir, 0o755); err != nil {
		return fmt.Errorf("create dest dir: %w", err)
	}

	// Configure application-wide logger via slog
	level := slog.LevelInfo
	if *debug {
		level = slog.LevelDebug
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level}))
	slog.SetDefault(logger)

	ghToken := os.Getenv("GITHUB_TOKEN")
	gh := githubclient.New(ghToken)

	repos, err := gh.ListOrgRepos(ctx, opts.Org)
	if err != nil {
		return fmt.Errorf("list org repos: %w", err)
	}

	slog.Info("🔍 Found repositories", "count", len(repos), "org", opts.Org)

	if opts.SkipClone {
		slog.Info("⏭️  Skipping clone/update; assuming sources exist", "dest", opts.DestDir)
	} else {
		var cloned, updated, failedClone, failedUpdate, skipped int
		for _, r := range repos {
			repoDir := filepath.Join(opts.DestDir, r.Name)
			url := r.SSHURL
			if url == "" {
				url = r.CloneURL
			}
			if url == "" {
				slog.Warn("⚠️  No clone URL available; skipping", "repo", r.Name)
				skipped++
				continue
			}

			if _, err := os.Stat(repoDir); err == nil {
				slog.Info("🔄 Updating repo", "repo", r.Name, "branch", r.DefaultBranch)
				started := time.Now()
				if err := gitutil.FetchAndCheckoutLatest(repoDir, r.DefaultBranch, 1, 30*time.Second); err != nil {
					slog.Error("❌ Update failed", "repo", r.Name, "error", err)
					failedUpdate++
				} else {
					slog.Info("✅ Updated repo", "repo", r.Name, "elapsed", time.Since(started).Truncate(time.Millisecond).String())
					updated++
				}
				continue
			}

			slog.Info("⬇️  Cloning repo", "repo", r.Name, "url", url, "branch", r.DefaultBranch)
			started := time.Now()
			if err := gitutil.ShallowClone(url, repoDir, r.DefaultBranch, 1, 60*time.Second); err != nil {
				slog.Error("❌ Clone failed", "repo", r.Name, "error", err)
				failedClone++
			} else {
				slog.Info("✅ Cloned repo", "repo", r.Name, "elapsed", time.Since(started).Truncate(time.Millisecond).String())
				cloned++
			}
		}

		slog.Info("📦 Summary", "cloned", cloned, "updated", updated, "clone_failures", failedClone, "update_failures", failedUpdate, "skipped", skipped)
	}

	return nil
}

func main() {
	if err := run(context.Background(), os.Args[1:]); err != nil {
		log.Fatalf("error: %v", err)
	}
}
