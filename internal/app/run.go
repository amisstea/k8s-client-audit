package app

import (
	"context"
	"cursor-experiment/internal/analyzers"

	"golang.org/x/tools/go/analysis/multichecker"
)

func Run(ctx context.Context, args []string) error {

	// Run scanner per-repository directory under DestDir
	// slog.Info("🔎 Scanning repositories for Kubernetes API usage anti-patterns", "root", opts.DestDir)

	// entries, err := os.ReadDir(opts.DestDir)
	// if err != nil {
	// 	slog.Error("❌ Failed to read destination directory", "error", err, "dir", opts.DestDir)
	// 	return err
	// }

	multichecker.Main(
		analyzers.AnalyzerClientReuse,
		analyzers.AnalyzerQPSBurst,
		analyzers.AnalyzerMissingInformer,
		analyzers.AnalyzerListInLoop,
		analyzers.AnalyzerManualPolling,
		analyzers.AnalyzerUnboundedQueue,
		analyzers.AnalyzerRequeueBackoff,
		analyzers.AnalyzerNoSelectors,
		analyzers.AnalyzerWideNamespace,
		analyzers.AnalyzerLargePageSizes,
		analyzers.AnalyzerTightErrorLoops,
		analyzers.AnalyzerMissingContext,
		analyzers.AnalyzerLeakyWatch,
		analyzers.AnalyzerRestConfigDefaults,
		analyzers.AnalyzerDynamicOveruse,
		analyzers.AnalyzerUnstructuredEverywhere,
		analyzers.AnalyzerWebhookTimeouts,
		analyzers.AnalyzerWebhookNoContext,
		analyzers.AnalyzerDiscoveryFlood,
		analyzers.AnalyzerRESTMapperNotCached,
	)
	// for _, ent := range entries {
	// 	if !ent.IsDir() {
	// 		continue
	// 	}
	// 	repoDir := filepath.Join(opts.DestDir, ent.Name())
	// 	// Prefer scanning only go modules
	// 	if _, err := os.Stat(filepath.Join(repoDir, "go.mod")); err != nil {
	// 		// Not a Go module
	// 		slog.Info("⚪ Not a go module; skipping", "repo", ent.Name())
	// 		continue
	// 	}
	// 	slog.Info("📂 Scanning repo", "repo", ent.Name())

	// issues, err := arunner.RunSpecs(ctx, repoDir, specs)
	// if err != nil {
	// 	slog.Error("❌ Analyzer run failed", "repo", ent.Name(), "error", err)
	// }
	// scanned++
	// if len(issues) == 0 {
	// 	slog.Info("✅ No issues", "repo", ent.Name())
	// 	continue
	// }
	// totalIssues += len(issues)
	// slog.Warn("⚠️  Issues found", "repo", ent.Name(), "count", len(issues))
	// for _, is := range issues {
	// 	slog.Log(ctx, slog.LevelWarn, "⚠️  Issue",
	// 		"repo", ent.Name(),
	// 		"rule", is.RuleID,
	// 		"title", is.Title,
	// 		"message", is.Message,
	// 		"file", is.Filename,
	// 		"line", is.Line,
	// 		"column", is.Column,
	// 		"suggestion", is.Suggestion,
	// 	)
	// 	ruleCounts[is.RuleID]++
	// 	repoCounts[ent.Name()]++
	// }
	// }
	// slog.Info("📊 Scan summary", "repos_scanned", scanned, "total_issues", totalIssues, "issues_by_rule", ruleCounts, "issues_by_repo", repoCounts)

	return nil
}
