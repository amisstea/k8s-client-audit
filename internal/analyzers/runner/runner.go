package runner

import (
	"context"
	"go/build"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/packages"
)

type Spec struct {
	RuleID     string
	Title      string
	Suggestion string
	Analyzer   *analysis.Analyzer
}

type Finding struct {
	RuleID     string
	Title      string
	Suggestion string
	Filename   string
	Line       int
	Column     int
	Message    string
}

// RunSpecs loads packages under dir once and runs all analyzers, returning aggregated findings.
func RunSpecs(ctx context.Context, dir string, specs []Spec) ([]Finding, error) {
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles | packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedImports,
		Dir:  dir,
	}
	fset := token.NewFileSet()
	cfg.Fset = fset
	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		return nil, err
	}
	sizes := types.SizesFor("gc", build.Default.GOARCH)
	var out []Finding
	for _, p := range pkgs {
		if len(p.Syntax) == 0 || p.TypesInfo == nil {
			continue
		}
		for _, spec := range specs {
			spec := spec
			var diags []analysis.Diagnostic
			pass := &analysis.Pass{
				Analyzer:   spec.Analyzer,
				Fset:       fset,
				Files:      p.Syntax,
				Pkg:        p.Types,
				TypesInfo:  p.TypesInfo,
				TypesSizes: sizes,
				Report:     func(d analysis.Diagnostic) { diags = append(diags, d) },
				ResultOf:   map[*analysis.Analyzer]interface{}{},
			}
			if _, err := spec.Analyzer.Run(pass); err != nil {
				continue
			}
			for _, d := range diags {
				pos := fset.Position(d.Pos)
				out = append(out, Finding{
					RuleID:     spec.RuleID,
					Title:      spec.Title,
					Suggestion: spec.Suggestion,
					Filename:   pos.Filename,
					Line:       pos.Line,
					Column:     pos.Column,
					Message:    d.Message,
				})
			}
		}
	}
	return out, nil
}
