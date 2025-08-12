package analyzers

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
)

// AnalyzerRestConfigDefaults (K8S050) flags rest.Config creations missing
// timeouts or user-agent. Heuristic only due to lack of types here.
var AnalyzerRestConfigDefaults = &analysis.Analyzer{
	Name: "k8s050_restconfigdefaults",
	Doc:  "flags rest.Config initialization without timeouts or UserAgent",
	Run:  runRestConfigDefaults,
}

func runRestConfigDefaults(pass *analysis.Pass) (any, error) {
	for _, f := range pass.Files {
		ast.Inspect(f, func(n ast.Node) bool {
			cl, ok := n.(*ast.CompositeLit)
			if !ok {
				return true
			}
			// Look for type named Config (heuristic) and fields of interest
			typeIsConfig := false
			if id, ok := cl.Type.(*ast.Ident); ok && id.Name == "Config" {
				typeIsConfig = true
			}
			if se, ok := cl.Type.(*ast.SelectorExpr); ok && se.Sel != nil && se.Sel.Name == "Config" {
				typeIsConfig = true
			}
			if !typeIsConfig {
				return true
			}
			hasTimeout := false
			hasUA := false
			for _, el := range cl.Elts {
				if kv, ok := el.(*ast.KeyValueExpr); ok {
					if k, ok := kv.Key.(*ast.Ident); ok {
						if k.Name == "Timeout" {
							hasTimeout = true
							// check zero literal
							if bl, ok := kv.Value.(*ast.BasicLit); ok && bl.Value == "0" {
								pass.Reportf(k.Pos(), "rest.Config Timeout is zero; set a reasonable timeout")
							}
						}
						if k.Name == "UserAgent" {
							hasUA = true
						}
					}
				}
			}
			if !hasTimeout || !hasUA {
				pass.Reportf(cl.Lbrace, "rest.Config missing Timeout and/or UserAgent; set sane defaults")
			}
			return true
		})
	}
	return nil, nil
}
