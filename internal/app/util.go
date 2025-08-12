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
		"K8S001": {RuleID: "K8S001", Title: "Client constructed in loop or hot path", Suggestion: "Reuse a singleton client", Analyzer: analyzers.AnalyzerClientReuse},
		"K8S002": {RuleID: "K8S002", Title: "rest.Config QPS/Burst missing or unrealistic", Suggestion: "Set reasonable QPS/Burst (e.g., QPS=20, Burst=50)", Analyzer: analyzers.AnalyzerQPSBurst},
		"K8S010": {RuleID: "K8S010", Title: "Direct Watch without shared informer", Suggestion: "Use shared informers/cache to avoid expensive Watches", Analyzer: analyzers.AnalyzerMissingInformer},
		"K8S011": {RuleID: "K8S011", Title: "List/Watch call inside loop", Suggestion: "Prefer informers/cache or move calls outside loops", Analyzer: analyzers.AnalyzerListInLoop},
		"K8S012": {RuleID: "K8S012", Title: "Manual polling with List + sleep", Suggestion: "Use Watch or informers instead of polling", Analyzer: analyzers.AnalyzerManualPolling},
		"K8S013": {RuleID: "K8S013", Title: "Unbounded workqueue without rate limiter", Suggestion: "Use RateLimitingInterface and backoff", Analyzer: analyzers.AnalyzerUnboundedQueue},
		"K8S014": {RuleID: "K8S014", Title: "Requeue without backoff", Suggestion: "Use RequeueAfter or rate-limited queues", Analyzer: analyzers.AnalyzerRequeueBackoff},
		"K8S021": {RuleID: "K8S021", Title: "List without label/field selectors", Suggestion: "Use MatchingLabels/Fields or ListOptions selectors", Analyzer: analyzers.AnalyzerNoSelectors},
		"K8S022": {RuleID: "K8S022", Title: "All-namespaces list", Suggestion: "Scope to a specific namespace", Analyzer: analyzers.AnalyzerWideNamespace},
		"K8S023": {RuleID: "K8S023", Title: "Excessively large page sizes", Suggestion: "Use reasonable ListOptions.Limit", Analyzer: analyzers.AnalyzerLargePageSizes},
		"K8S032": {RuleID: "K8S032", Title: "Tight error loop without backoff around Kubernetes API calls", Suggestion: "Insert backoff or sleep when retrying on errors", Analyzer: analyzers.AnalyzerTightErrorLoops},
		"K8S041": {RuleID: "K8S041", Title: "Client call uses context.Background/TODO", Suggestion: "Propagate a request context", Analyzer: analyzers.AnalyzerMissingContext},
		"K8S042": {RuleID: "K8S042", Title: "Leaky watch channels", Suggestion: "Ensure Stop()/Cancel() is called and channels drained", Analyzer: analyzers.AnalyzerLeakyWatch},
		"K8S050": {RuleID: "K8S050", Title: "rest.Config missing sane defaults", Suggestion: "Set Timeout and UserAgent", Analyzer: analyzers.AnalyzerRestConfigDefaults},
		"K8S051": {RuleID: "K8S051", Title: "Overuse of dynamic/unstructured clients", Suggestion: "Prefer typed clients when available", Analyzer: analyzers.AnalyzerDynamicOveruse},
		"K8S052": {RuleID: "K8S052", Title: "Unstructured everywhere", Suggestion: "Prefer typed objects for performance and safety", Analyzer: analyzers.AnalyzerUnstructuredEverywhere},
		"K8S060": {RuleID: "K8S060", Title: "Webhook HTTP timeouts missing/zero", Suggestion: "Set client/server timeouts", Analyzer: analyzers.AnalyzerWebhookTimeouts},
		"K8S061": {RuleID: "K8S061", Title: "Webhook uses Background/TODO", Suggestion: "Propagate request context", Analyzer: analyzers.AnalyzerWebhookNoContext},
		"K8S070": {RuleID: "K8S070", Title: "Discovery/RESTMapper flood", Suggestion: "Cache and reuse discovery/RESTMapper", Analyzer: analyzers.AnalyzerDiscoveryFlood},
		"K8S071": {RuleID: "K8S071", Title: "RESTMapper not cached", Suggestion: "Use cached/deferred RESTMapper", Analyzer: analyzers.AnalyzerRESTMapperNotCached},
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
