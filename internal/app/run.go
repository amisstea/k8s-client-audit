package app

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

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

	ghToken := os.Getenv("GITHUB_TOKEN")
	gh := githubclient.New(ghToken)

	repos, err := gh.ListOrgRepos(ctx, opts.Org)
	if err != nil {
		return fmt.Errorf("list org repos: %w", err)
	}

	log.Printf("🔍 Found %d repositories under %s", len(repos), opts.Org)

	if opts.SkipClone {
		log.Printf("⏭️  Skipping clone/update. Assuming sources already exist in %q", opts.DestDir)
		return nil
	}

	var cloned, updated, failedClone, failedUpdate, skipped int
	for _, r := range repos {
		repoDir := filepath.Join(opts.DestDir, r.Name)
		url := r.SSHURL
		if url == "" {
			url = r.CloneURL
		}
		if url == "" {
			log.Printf("⚠️  %s: no clone URL available; skipping", r.Name)
			skipped++
			continue
		}

		if _, err := os.Stat(repoDir); err == nil {
			log.Printf("🔄 Updating %s → branch %q", r.Name, r.DefaultBranch)
			started := time.Now()
			if err := gitutil.FetchAndCheckoutLatest(repoDir, r.DefaultBranch, 1, 30*time.Second); err != nil {
				log.Printf("❌ Update failed for %s: %v", r.Name, err)
				failedUpdate++
			} else {
				log.Printf("✅ Updated %s in %s", r.Name, time.Since(started).Truncate(time.Millisecond))
				updated++
			}
			continue
		}

		log.Printf("⬇️  Cloning %s from %s (branch %q)", r.Name, url, r.DefaultBranch)
		started := time.Now()
		if err := gitutil.ShallowClone(url, repoDir, r.DefaultBranch, 1, 60*time.Second); err != nil {
			log.Printf("❌ Clone failed for %s: %v", r.Name, err)
			failedClone++
		} else {
			log.Printf("✅ Cloned %s in %s", r.Name, time.Since(started).Truncate(time.Millisecond))
			cloned++
		}
	}

	log.Printf("📦 Summary: ✅ cloned=%d, ✅ updated=%d, ❌ clone_failures=%d, ❌ update_failures=%d, ⚠️ skipped=%d", cloned, updated, failedClone, failedUpdate, skipped)

	return nil
}
