package app

import (
	"strings"

	"cursor-experiment/internal/analyzers"
	arunner "cursor-experiment/internal/analyzers/runner"
)

func splitAndTrim(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		t := strings.TrimSpace(p)
		out = append(out, t)
	}
	return out
}

// buildAnalyzerSpecs builds the list of analyzers to run based on include/disable flags.
// If includeCSV is non-empty, only those rules are enabled. Otherwise, all known analyzers
// are enabled except those explicitly disabled via disableCSV.
func buildAnalyzerSpecs(includeCSV, disableCSV string) []arunner.Spec {
	// known analyzers
	catalog := map[string]arunner.Spec{
		"K8S002": {RuleID: "K8S002", Title: "rest.Config QPS/Burst missing or unrealistic", Suggestion: "Set reasonable QPS/Burst (e.g., QPS=20, Burst=50)", Analyzer: analyzers.AnalyzerQPSBurst},
		"K8S032": {RuleID: "K8S032", Title: "Tight error loop without backoff around Kubernetes API calls", Suggestion: "Insert backoff or sleep when retrying on errors", Analyzer: analyzers.AnalyzerTightErrorLoops},
		"K8S011": {RuleID: "K8S011", Title: "List/Watch call inside loop", Suggestion: "Prefer informers/cache or move calls outside loops", Analyzer: analyzers.AnalyzerListInLoop},
	}
	var out []arunner.Spec
	if strings.TrimSpace(includeCSV) != "" {
		for _, id := range splitAndTrim(includeCSV) {
			if spec, ok := catalog[id]; ok {
				out = append(out, spec)
			}
		}
		return out
	}
	disabled := map[string]struct{}{}
	for _, id := range splitAndTrim(disableCSV) {
		if id != "" {
			disabled[id] = struct{}{}
		}
	}
	for id, spec := range catalog {
		if _, off := disabled[id]; off {
			continue
		}
		out = append(out, spec)
	}
	return out
}
