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
