package scanner

import (
	"context"
	"go/token"

	"golang.org/x/tools/go/packages"
)

// Registry holds a set of rules to run.
type Registry struct {
	rules []Rule
}

func NewRegistry() *Registry      { return &Registry{} }
func (r *Registry) Add(rule Rule) { r.rules = append(r.rules, rule) }
func (r *Registry) Rules() []Rule { return append([]Rule{}, r.rules...) }

// BuildDefaultRegistry returns a registry with all planned rules registered (stubs for now).
func BuildDefaultRegistry() *Registry {
	reg := NewRegistry()
	// Concrete early rules
	reg.Add(NewRuleExcessiveConfig())
	reg.Add(NewRuleClientReuse())
	reg.Add(NewRuleListInLoop())
	reg.Add(NewRuleMissingContext())
	reg.Add(NewRuleNoSelectors())
	reg.Add(NewRuleNoFieldSelector())
	reg.Add(NewRuleWideNamespace())
	reg.Add(NewRuleTightErrorLoops())

	// Remaining rules as stubs to wire taxonomy
	reg.Add(newStubRule(RuleClientReuseID, "Clients should be reused; avoid creating per-request clients"))
	reg.Add(newStubRule(RuleQPSBurstConfigID, "Ensure rest.Config QPS/Burst are tuned and not unlimited"))
	reg.Add(newStubRule(RuleMissingSharedInformerID, "Prefer SharedInformerFactory over manual polling"))
	reg.Add(newStubRule(RuleManualPollingInsteadOfWatchID, "Avoid manual polling where watch/informers suffice"))
	reg.Add(newStubRule(RuleUnboundedWorkQueueID, "Use rate-limited and bounded work queues in controllers"))
	reg.Add(newStubRule(RuleNoBackoffOnRequeueID, "Requeues should use rate limiting/backoff"))
	reg.Add(newStubRule(RuleLargePageSizesID, "Avoid unbounded page sizes in list calls"))
	reg.Add(newStubRule(RuleIgnoring429AndBackoffID, "Honor 429s and implement backoff"))
	reg.Add(newStubRule(RuleNoRetryForTransientErrorsID, "Retry transient errors with backoff"))
	reg.Add(newStubRule(RuleNoResyncPeriodID, "Set appropriate resync periods for informers if needed"))
	reg.Add(newStubRule(RuleLeakyWatchChannelsID, "Ensure watches are stopped and channels drained"))
	reg.Add(newStubRule(RuleRestConfigDefaultsID, "Verify rest.Config has sane timeouts and UserAgent"))
	reg.Add(newStubRule(RuleDynamicClientOveruseID, "Avoid overuse of dynamic/unstructured when typed clients exist"))
	reg.Add(newStubRule(RuleUnstructuredEverywhereID, "Prefer typed clients for performance and safety"))
	reg.Add(newStubRule(RuleWebhookTimeoutsID, "Webhook clients must have strict timeouts"))
	reg.Add(newStubRule(RuleWebhookNoContextID, "Webhook handlers must honor request context"))
	reg.Add(newStubRule(RuleDiscoveryFloodID, "Avoid frequent discovery or RESTMapper re-builds"))
	reg.Add(newStubRule(RuleRESTMapperNotCachedID, "Use a cached RESTMapper or controller-runtime mapper"))
	reg.Add(newStubRule(RuleExcessiveClusterScopeID, "Use namespace-scoped RBAC when possible"))
	reg.Add(newStubRule(RuleWildcardVerbsID, "Avoid wildcard verbs in RBAC"))
	return reg
}

// Stub rule implementation to wire engine; returns no issues for now.
type stubRule struct {
	id          string
	description string
}

func newStubRule(id, description string) *stubRule {
	return &stubRule{id: id, description: description}
}
func (s *stubRule) ID() string          { return s.id }
func (s *stubRule) Description() string { return s.description }
func (s *stubRule) Apply(ctx context.Context, _ *token.FileSet, _ *packages.Package) ([]Issue, error) {
	return nil, nil
}
