package scanner

import (
	"context"
	"go/ast"
	"go/token"
	"log/slog"

	"golang.org/x/tools/go/packages"
)

// Rule defines a static analysis rule over Go packages/AST/types.
type Rule interface {
	ID() string
	Description() string
	Apply(ctx context.Context, fset *token.FileSet, pkg *packages.Package) ([]Issue, error)
}

// Engine coordinates loading packages and executing rules.
type Engine struct {
	rules []Rule
}

func NewEngine(rules ...Rule) *Engine {
	return &Engine{rules: append([]Rule{}, rules...)}
}

// LoadPackages loads Go packages under the given dir using go/packages with syntax and types.
func LoadPackages(dir string) (*token.FileSet, []*packages.Package, error) {
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles | packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedImports,
		Dir:  dir,
	}
	fset := token.NewFileSet()
	cfg.Fset = fset
	// Recursively load all packages within this module/repo
	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		return nil, nil, err
	}
	return fset, pkgs, nil
}

func (e *Engine) Run(ctx context.Context, dir string) ([]Issue, error) {
	fset, pkgs, err := LoadPackages(dir)
	if err != nil {
		return nil, err
	}
	var out []Issue
	for _, p := range pkgs {
		// Skip packages without syntax
		if len(p.Syntax) == 0 {
			continue
		}
		slog.Info("🧩 Scanning package", "pkg", p.PkgPath, "files", len(p.Syntax))
		// Ensure AST is available
		for range p.Syntax {
			_ = &ast.File{}
		}
		for _, r := range e.rules {
			slog.Debug("▶️  Applying rule", "id", r.ID(), "desc", r.Description(), "pkg", p.PkgPath)
			issues, err := r.Apply(ctx, fset, p)
			if err != nil {
				// continue collecting from other rules/packages
				continue
			}
			out = append(out, issues...)
		}
	}
	return out, nil
}
