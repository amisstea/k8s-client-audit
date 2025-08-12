package main

import (
	"cursor-experiment/internal/analyzers"

	"golang.org/x/tools/go/analysis/multichecker"
)

func main() {
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
}
