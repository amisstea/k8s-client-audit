package scanner

import (
	"context"
	"go/ast"
	"go/token"

	"golang.org/x/tools/go/packages"
)

// Detect repeated creation of rest.Config and clients (per-call or inside functions likely invoked often).
type ruleExcessiveConfig struct{}

func NewRuleExcessiveConfig() Rule        { return &ruleExcessiveConfig{} }
func (r *ruleExcessiveConfig) ID() string { return RuleExcessiveRestConfigCreationID }
func (r *ruleExcessiveConfig) Description() string {
	return "Avoid repeated creation of rest.Config/client; reuse singletons"
}

func (r *ruleExcessiveConfig) Apply(ctx context.Context, fset *token.FileSet, pkg *packages.Package) ([]Issue, error) {
	var issues []Issue
	for _, f := range pkg.Syntax {
		ast.Inspect(f, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
				// clientcmd.BuildConfigFromFlags, rest.InClusterConfig, kubernetes.NewForConfig, client.New
				name := sel.Sel.Name
				if name == "BuildConfigFromFlags" || name == "InClusterConfig" || name == "NewForConfig" || name == "New" {
					pos := fset.Position(sel.Sel.Pos())
					issues = append(issues, Issue{
						RuleID:      r.ID(),
						Title:       "Potential repeated client/config creation",
						Description: "Constructing rest.Config or clients frequently can be expensive; prefer app-level singletons and DI",
						Severity:    SeverityWarning,
						PackagePath: pkg.PkgPath,
						Position:    Position{Filename: pos.Filename, Line: pos.Line, Column: pos.Column},
						Suggestion:  "Create rest.Config and clients once and pass them into components",
					})
				}
			}
			return true
		})
	}
	return issues, nil
}
