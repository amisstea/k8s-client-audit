package scanner

import "context"

// Scanner ties together the registry and the engine.
type Scanner struct {
	engine   *Engine
	registry *Registry
}

func New() *Scanner {
	reg := BuildDefaultRegistry()
	eng := NewEngine(reg.Rules()...)
	return &Scanner{engine: eng, registry: reg}
}

// ScanDirectory runs all rules against packages under root and returns issues.
func (s *Scanner) ScanDirectory(ctx context.Context, root string) ([]Issue, error) {
	return s.engine.Run(ctx, root)
}

// SetDisabledRules filters out rules by ID and rebuilds the engine with the remainder.
func (s *Scanner) SetDisabledRules(ids []string) {
	disabled := map[string]struct{}{}
	for _, id := range ids {
		if id == "" {
			continue
		}
		disabled[id] = struct{}{}
	}
	var filtered []Rule
	for _, r := range s.registry.Rules() {
		if _, found := disabled[r.ID()]; found {
			continue
		}
		filtered = append(filtered, r)
	}
	s.engine = NewEngine(filtered...)
}
