package analyzers

import (
	"go/ast"
	"go/types"
)

// calleeIdent returns the identifier for a call expression's callee, handling
// both simple identifiers and selector expressions. Returns nil if unresolved.
func calleeIdent(expr ast.Expr) *ast.Ident {
	switch x := expr.(type) {
	case *ast.Ident:
		return x
	case *ast.SelectorExpr:
		if x.Sel != nil {
			return x.Sel
		}
	}
	return nil
}

// deref returns the non-pointer type for a given type.
func deref(t types.Type) types.Type {
	if p, ok := t.(*types.Pointer); ok {
		return p.Elem()
	}
	return t
}

// isNamed returns true if t is a named type whose package path and name match.
func isNamed(t types.Type, pkgPath, name string) bool {
	if n, ok := t.(*types.Named); ok {
		if n.Obj() != nil && n.Obj().Pkg() != nil {
			return n.Obj().Pkg().Path() == pkgPath && n.Obj().Name() == name
		}
	}
	return false
}
