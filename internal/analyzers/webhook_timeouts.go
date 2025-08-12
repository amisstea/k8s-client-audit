package analyzers

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
)

// AnalyzerWebhookTimeouts (K8S060) flags HTTP clients/servers in webhook packages
// missing reasonable timeouts. Heuristic: http.Client{Timeout: 0} or missing; http.Server{Read/WriteTimeout: 0}.
var AnalyzerWebhookTimeouts = &analysis.Analyzer{
	Name: "k8s060_webhook_timeouts",
	Doc:  "flags webhook HTTP client/server without timeouts",
	Run:  runWebhookTimeouts,
}

func runWebhookTimeouts(pass *analysis.Pass) (any, error) {
	for _, f := range pass.Files {
		ast.Inspect(f, func(n ast.Node) bool {
			cl, ok := n.(*ast.CompositeLit)
			if !ok {
				return true
			}
			// Identify http.Client or http.Server by type name heuristic
			isHTTPClient := false
			isHTTPServer := false
			switch t := cl.Type.(type) {
			case *ast.Ident:
				if t.Name == "Client" {
					isHTTPClient = true
				}
				if t.Name == "Server" {
					isHTTPServer = true
				}
			case *ast.SelectorExpr:
				if t.Sel != nil && t.Sel.Name == "Client" {
					isHTTPClient = true
				}
				if t.Sel != nil && t.Sel.Name == "Server" {
					isHTTPServer = true
				}
			}
			if !(isHTTPClient || isHTTPServer) {
				return true
			}
			hasTimeout := false
			zeroTimeout := false
			for _, el := range cl.Elts {
				kv, ok := el.(*ast.KeyValueExpr)
				if !ok {
					continue
				}
				if k, ok := kv.Key.(*ast.Ident); ok {
					if isHTTPClient && k.Name == "Timeout" {
						hasTimeout = true
						if bl, ok := kv.Value.(*ast.BasicLit); ok && bl.Value == "0" {
							zeroTimeout = true
						}
					}
					if isHTTPServer && (k.Name == "ReadTimeout" || k.Name == "WriteTimeout" || k.Name == "IdleTimeout") {
						hasTimeout = true
						if bl, ok := kv.Value.(*ast.BasicLit); ok && bl.Value == "0" {
							zeroTimeout = true
						}
					}
				}
			}
			if !hasTimeout || zeroTimeout {
				pass.Reportf(cl.Lbrace, "Webhook HTTP client/server missing or having zero timeouts; set conservative timeouts")
			}
			return true
		})
	}
	return nil, nil
}
