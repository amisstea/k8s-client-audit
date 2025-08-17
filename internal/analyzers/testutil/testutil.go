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
// types info, applies optional spoof callbacks, runs the analyzer, and returns
// collected diagnostics.
func RunAnalyzerOnSrc(an *analysis.Analyzer, src string, spoofs ...func(f *ast.File, info *types.Info)) ([]analysis.Diagnostic, error) {
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
	for _, spoof := range spoofs {
		if spoof != nil {
			spoof(f, info)
		}
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

// Common Kubernetes package paths
const (
	PkgKubernetes        = "k8s.io/client-go/kubernetes"
	PkgRest              = "k8s.io/client-go/rest"
	PkgDynamic           = "k8s.io/client-go/dynamic"
	PkgDiscovery         = "k8s.io/client-go/discovery"
	PkgRestMapper        = "k8s.io/client-go/restmapper"
	PkgControllerRuntime = "sigs.k8s.io/controller-runtime/pkg/client"
	PkgReconcile         = "sigs.k8s.io/controller-runtime/pkg/reconcile"
	PkgMetaV1            = "k8s.io/apimachinery/pkg/apis/meta/v1"
	PkgTime              = "time"
)

// CommonK8sSpoofMap returns a SpoofMap with common Kubernetes function names
func CommonK8sSpoofMap() SpoofMap {
	return SpoofMap{
		// Client constructors
		"NewForConfig":                   PkgKubernetes,
		"RESTClientFor":                  PkgRest,
		"New":                            PkgControllerRuntime,
		"NewDynamicClientForConfig":      PkgDynamic,
		"NewDiscoveryClientForConfig":    PkgDiscovery,
		"NewDeferredDiscoveryRESTMapper": PkgRestMapper,
		"ResetRESTMapper":                PkgRestMapper,

		// CRUD operations (controller-runtime client)
		"Get":    PkgControllerRuntime,
		"List":   PkgControllerRuntime,
		"Create": PkgControllerRuntime,
		"Update": PkgControllerRuntime,
		"Patch":  PkgControllerRuntime,
		"Delete": PkgControllerRuntime,
		"Watch":  PkgControllerRuntime,

		// Watch/lifecycle methods
		"ResultChan":   PkgControllerRuntime,
		"Stop":         PkgControllerRuntime,
		"Cancel":       PkgControllerRuntime,
		"StopWatching": PkgControllerRuntime,
	}
}

// TimePackageSpoofMap returns a SpoofMap with Go standard library time package functions
func CommonStdLibSpoofMap() SpoofMap {
	return SpoofMap{
		"Sleep": PkgTime,
	}
}

// SpoofReconcileSignature adds a Reconcile method signature to the types.Info
func SpoofReconcileSignature(f *ast.File, info *types.Info) {
	ast.Inspect(f, func(n ast.Node) bool {
		fd, ok := n.(*ast.FuncDecl)
		if !ok || fd.Name == nil || fd.Name.Name != "Reconcile" {
			return true
		}
		pkgRec := types.NewPackage(PkgReconcile, "reconcile")
		resNamed := types.NewNamed(types.NewTypeName(token.NoPos, pkgRec, "Result", nil), types.NewStruct(nil, nil), nil)
		resTuple := types.NewTuple(
			types.NewVar(token.NoPos, nil, "", resNamed),
			types.NewVar(token.NoPos, nil, "", types.Universe.Lookup("error").Type()),
		)
		sig := types.NewSignatureType(nil, nil, nil, types.NewTuple(), resTuple, false)
		info.Defs[fd.Name] = types.NewFunc(token.NoPos, nil, fd.Name.Name, sig)
		return false
	})
}

// SpoofListOptionsType adds ListOptions type spoofing to types.Info
func SpoofListOptionsType(f *ast.File, info *types.Info) {
	pkgMeta := types.NewPackage(PkgMetaV1, "v1")
	pkgClient := types.NewPackage(PkgControllerRuntime, "client")

	// Find type declarations and mark them as Kubernetes types
	ast.Inspect(f, func(n ast.Node) bool {
		if ts, ok := n.(*ast.TypeSpec); ok && ts.Name.Name == "ListOptions" {
			// Create a named type for Kubernetes ListOptions
			listOptionsType := types.NewNamed(types.NewTypeName(token.NoPos, pkgMeta, "ListOptions", nil), types.NewStruct(nil, nil), nil)
			info.Defs[ts.Name] = listOptionsType.Obj()
		}
		return true
	})

	// Find composite literals and method calls
	ast.Inspect(f, func(n ast.Node) bool {
		if cl, ok := n.(*ast.CompositeLit); ok {
			if id, ok := cl.Type.(*ast.Ident); ok && id.Name == "ListOptions" {
				// Create the Kubernetes ListOptions type
				listOptionsType := types.NewNamed(types.NewTypeName(token.NoPos, pkgMeta, "ListOptions", nil), types.NewStruct(nil, nil), nil)
				info.Types[cl] = types.TypeAndValue{Type: listOptionsType}
			}
		} else if se, ok := n.(*ast.SelectorExpr); ok && se.Sel != nil && se.Sel.Name == "List" {
			// Mark List calls as coming from Kubernetes client package
			sig := types.NewSignatureType(nil, nil, nil, types.NewTuple(), types.NewTuple(), false)
			info.Uses[se.Sel] = types.NewFunc(token.NoPos, pkgClient, "List", sig)
		}
		return true
	})
}

// CommonK8sSpoof applies common Kubernetes type spoofing
func CommonK8sSpoof(f *ast.File, info *types.Info) {
	SpoofUsesFromMap(CommonK8sSpoofMap())(f, info)
	SpoofReconcileSignature(f, info)
}

// CommonStdLibSpoof applies standard library spoofing (e.g., time.Sleep)
func CommonStdLibSpoof(f *ast.File, info *types.Info) {
	SpoofUsesFromMap(CommonStdLibSpoofMap())(f, info)
}
