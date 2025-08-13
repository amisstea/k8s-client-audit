package analyzers

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// AnalyzerRestConfigDefaults flags rest.Config creations missing
// timeouts or user-agent.
var AnalyzerRestConfigDefaults = &analysis.Analyzer{
	Name:     "restconfigdefaults",
	Doc:      "flags rest.Config initialization without timeouts or UserAgent",
	Run:      runRestConfigDefaults,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
}

func runRestConfigDefaults(pass *analysis.Pass) (any, error) {
	// Get the inspector from the analysis pass.
	// The inspector is configured to visit all nodes in the AST.
	inspector := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	// Define the types of AST nodes we want to inspect.
	// We are interested in CompositeLiterals, which represent struct instantiations.
	// e.g., rest.Config{...} or &rest.Config{...}
	nodeFilter := []ast.Node{
		(*ast.CompositeLit)(nil),
	}

	// The inspector's Preorder function traverses the AST.
	// It calls the provided function for each node that matches the filter.
	inspector.Preorder(nodeFilter, func(n ast.Node) {
		// Assert that the node is a CompositeLit.
		compLit, ok := n.(*ast.CompositeLit)
		if !ok {
			return // Should not happen due to the node filter
		}

		// Get the type information for the composite literal.
		// pass.TypesInfo.TypeOf returns the type of an expression.
		typ := pass.TypesInfo.TypeOf(compLit)
		if typ == nil {
			return
		}

		// We need to handle both `rest.Config{}` and `&rest.Config{}`.
		// If it's a pointer, we get the underlying element type.
		if ptr, ok := typ.(*types.Pointer); ok {
			typ = ptr.Elem()
		}

		// Check if the type is a named type (like a struct).
		named, ok := typ.(*types.Named)
		if !ok {
			return
		}

		// Check if the type object and its package match what we're looking for.
		// named.Obj() gives us the TypeName object.
		// named.Obj().Pkg() gives us the package of the type.
		// named.Obj().Name() gives us the name of the type.
		if named.Obj() != nil && named.Obj().Pkg() != nil &&
			named.Obj().Pkg().Path() == "k8s.io/client-go/rest" &&
			named.Obj().Name() == "Config" {

			// Track whether Timeout and UserAgent fields are set in the rest.Config composite literal.
			var hasTimeout, hasUserAgent bool

			// Iterate over each element in the composite literal (i.e., each field assignment).
			for _, el := range compLit.Elts {
				// We are only interested in key-value pairs (e.g., Timeout: ..., UserAgent: ...).
				kv, ok := el.(*ast.KeyValueExpr)
				if !ok {
					continue
				}
				// Check if the key is an identifier (the field name).
				switch k := kv.Key.(type) {
				case *ast.Ident:
					switch k.Name {
					case "Timeout":
						hasTimeout = true
						// Check if the Timeout value is a basic literal and is set to zero.
						if bl, ok := kv.Value.(*ast.BasicLit); ok && bl.Value == "0" {
							pass.Reportf(k.Pos(), "rest.Config Timeout is zero; set a reasonable timeout")
						}
					case "UserAgent":
						hasUserAgent = true
					}
				}
			}
			if !hasTimeout || !hasUserAgent {
				pass.Reportf(n.Pos(), "rest.Config missing Timeout and/or UserAgent; set sane defaults")
			}
		}
	})

	return nil, nil
}
