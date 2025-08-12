package testutil

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
	insppass "golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// RunAnalyzerOnSrc parses src, builds a minimal analysis.Pass with inspector and
// types info, applies an optional spoof callback, runs the analyzer, and returns
// collected diagnostics.
func RunAnalyzerOnSrc(an *analysis.Analyzer, src string, spoof func(f *ast.File, info *types.Info)) ([]analysis.Diagnostic, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", src, 0)
	if err != nil {
		return nil, err
	}
	files := []*ast.File{f}
	info := &types.Info{
		Types:      map[ast.Expr]types.TypeAndValue{},
		Defs:       map[*ast.Ident]types.Object{},
		Uses:       map[*ast.Ident]types.Object{},
		Selections: map[*ast.SelectorExpr]*types.Selection{},
	}
	var conf types.Config
	_, _ = conf.Check("p", fset, files, info)
	if spoof != nil {
		spoof(f, info)
	}
	var diags []analysis.Diagnostic
	pass := &analysis.Pass{
		Analyzer:   an,
		Fset:       fset,
		Files:      files,
		TypesInfo:  info,
		TypesSizes: types.SizesFor("gc", "amd64"),
		Report:     func(d analysis.Diagnostic) { diags = append(diags, d) },
		ResultOf:   map[*analysis.Analyzer]interface{}{insppass.Analyzer: inspector.New(files)},
	}
	_, err = an.Run(pass)
	return diags, err
}

// SpoofMap maps function names to package import paths for creating fake Uses.
type SpoofMap map[string]string

// SpoofUsesFromMap returns a spoof function that assigns types.Func objects for
// callees whose name appears in the provided map, using the map's pkg path.
func SpoofUsesFromMap(m SpoofMap) func(f *ast.File, info *types.Info) {
	return func(f *ast.File, info *types.Info) {
		ast.Inspect(f, func(n ast.Node) bool {
			ce, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			switch fun := ce.Fun.(type) {
			case *ast.Ident:
				if pkgPath, ok := m[fun.Name]; ok {
					info.Uses[fun] = types.NewFunc(token.NoPos, types.NewPackage(pkgPath, lastSegment(pkgPath)), fun.Name, nil)
				}
			case *ast.SelectorExpr:
				if fun.Sel != nil {
					if pkgPath, ok := m[fun.Sel.Name]; ok {
						info.Uses[fun.Sel] = types.NewFunc(token.NoPos, types.NewPackage(pkgPath, lastSegment(pkgPath)), fun.Sel.Name, nil)
					}
				}
			}
			return true
		})
	}
}

func lastSegment(path string) string {
	if i := strings.LastIndex(path, "/"); i >= 0 {
		return path[i+1:]
	}
	return path
}
