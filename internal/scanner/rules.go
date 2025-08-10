package scanner

// This file declares rule IDs and categories the engine will support.

// Rule categories for K8s API interactions.
const (
	// Client construction and reuse
	RuleClientReuseID                 = "K8S001"
	RuleQPSBurstConfigID              = "K8S002"
	RuleExcessiveRestConfigCreationID = "K8S003"

	// Informers, caches, and controllers
	RuleMissingSharedInformerID       = "K8S010"
	RuleDirectListWatchInLoopsID      = "K8S011"
	RuleManualPollingInsteadOfWatchID = "K8S012"
	RuleUnboundedWorkQueueID          = "K8S013"
	RuleNoBackoffOnRequeueID          = "K8S014"

	// List/Watch usage
	RuleNoFieldSelectorID    = "K8S020"
	RuleNoLabelSelectorID    = "K8S021"
	RuleWideNamespaceScansID = "K8S022"
	RuleLargePageSizesID     = "K8S023"

	// Error handling and rate limiting
	RuleIgnoring429AndBackoffID     = "K8S030"
	RuleNoRetryForTransientErrorsID = "K8S031"
	RuleTightLoopsOnErrorsID        = "K8S032"

	// Watch handling
	RuleNoResyncPeriodID        = "K8S040"
	RuleNoContextCancellationID = "K8S041"
	RuleLeakyWatchChannelsID    = "K8S042"

	// Client-go specifics
	RuleRestConfigDefaultsID     = "K8S050"
	RuleDynamicClientOveruseID   = "K8S051"
	RuleUnstructuredEverywhereID = "K8S052"

	// Admission webhook specifics
	RuleWebhookTimeoutsID  = "K8S060"
	RuleWebhookNoContextID = "K8S061"

	// RESTMapper and discovery
	RuleDiscoveryFloodID      = "K8S070"
	RuleRESTMapperNotCachedID = "K8S071"

	// RBAC and permissions usage patterns
	RuleExcessiveClusterScopeID = "K8S080"
	RuleWildcardVerbsID         = "K8S081"
)
