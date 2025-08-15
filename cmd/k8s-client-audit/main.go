package main

import (
	analyzers "github.com/amisstea/k8s-client-audit/internal/analyzers"

	"golang.org/x/tools/go/analysis/multichecker"
)

func main() {
	multichecker.Main(
		analyzers.AnalyzerClientReuse,
		analyzers.AnalyzerDiscoveryFlood,
		analyzers.AnalyzerDynamicOveruse,
		analyzers.AnalyzerLargePageSizes,
		analyzers.AnalyzerLeakyWatch,
		analyzers.AnalyzerListInLoop,
		analyzers.AnalyzerManualPolling,
		analyzers.AnalyzerMissingContext,
		analyzers.AnalyzerMissingInformer,
		analyzers.AnalyzerNoSelectors,
		analyzers.AnalyzerQPSBurst,
		analyzers.AnalyzerRESTMapperNotCached,
		analyzers.AnalyzerRequeueBackoff,
		analyzers.AnalyzerRestConfigDefaults,
		analyzers.AnalyzerTightErrorLoops,
		analyzers.AnalyzerUnboundedQueue,
		analyzers.AnalyzerUnstructuredEverywhere,
		analyzers.AnalyzerWideNamespace,
	)
}
