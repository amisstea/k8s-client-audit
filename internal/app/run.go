package app

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	arunner "cursor-experiment/internal/analyzers/runner"
	"cursor-experiment/internal/githubclient"
	"cursor-experiment/internal/gitutil"
)

type Options struct {
	Org       string
	DestDir   string
	SkipClone bool
}

func Run(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("cursor-experiment", flag.ContinueOnError)
	org := fs.String("org", "konflux-ci", "GitHub organization to clone")
	dest := fs.String("dest", "sources", "Destination directory for repositories")
	skipClone := fs.Bool("skip-clone", false, "Skip cloning/updating sources; assume they exist")
	debug := fs.Bool("debug", false, "Enable debug logging across the app")
	disableRules := fs.String("disable-rules", "K8S003,K8S021", "Comma-separated rule IDs to disable (applied only if --rules not set)")
	includeRules := fs.String("rules", "", "Comma-separated rule IDs to include exclusively (disables all others)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	opts := Options{Org: *org, DestDir: *dest, SkipClone: *skipClone}
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

	// Run scanner per-repository directory under DestDir
	slog.Info("🔎 Scanning repositories for Kubernetes API usage anti-patterns", "root", opts.DestDir)

	entries, err := os.ReadDir(opts.DestDir)
	if err != nil {
		slog.Error("❌ Failed to read destination directory", "error", err, "dir", opts.DestDir)
		return err
	}
	totalIssues := 0
	scanned := 0
	ruleCounts := map[string]int{}
	repoCounts := map[string]int{}
	for _, ent := range entries {
		if !ent.IsDir() {
			continue
		}
		repoDir := filepath.Join(opts.DestDir, ent.Name())
		// Prefer scanning only go modules
		if _, err := os.Stat(filepath.Join(repoDir, "go.mod")); err != nil {
			// Not a Go module
			slog.Info("⚪ Not a go module; skipping", "repo", ent.Name())
			continue
		}
		slog.Info("📂 Scanning repo", "repo", ent.Name())

		specs := buildAnalyzerSpecs(*includeRules, *disableRules)
		issues, err := arunner.RunSpecs(ctx, repoDir, specs)
		if err != nil {
			slog.Error("❌ Analyzer run failed", "repo", ent.Name(), "error", err)
		}
		scanned++
		if len(issues) == 0 {
			slog.Info("✅ No issues", "repo", ent.Name())
			continue
		}
		totalIssues += len(issues)
		slog.Warn("⚠️  Issues found", "repo", ent.Name(), "count", len(issues))
		for _, is := range issues {
			slog.Log(ctx, slog.LevelWarn, "⚠️  Issue",
				"repo", ent.Name(),
				"rule", is.RuleID,
				"title", is.Title,
				"message", is.Message,
				"file", is.Filename,
				"line", is.Line,
				"column", is.Column,
				"suggestion", is.Suggestion,
			)
			ruleCounts[is.RuleID]++
			repoCounts[ent.Name()]++
		}
	}
	slog.Info("📊 Scan summary", "repos_scanned", scanned, "total_issues", totalIssues, "issues_by_rule", ruleCounts, "issues_by_repo", repoCounts)

	return nil
}
