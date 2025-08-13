package analyzers

import (
	"go/ast"
	"go/token"
	"go/types"
	"strconv"

	"golang.org/x/tools/go/analysis"
)

// AnalyzerQPSBurst flags rest.Config.QPS/Burst that are zero/unlimited or extreme.
var AnalyzerQPSBurst = &analysis.Analyzer{
	Name: "qpsburst",
	Doc:  "flags rest.Config QPS/Burst zero or extreme values",
	Run:  runQPSBurst,
}

func runQPSBurst(pass *analysis.Pass) (any, error) {
	for _, f := range pass.Files {
		ast.Inspect(f, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.CompositeLit:
				if isRestConfig(pass.TypesInfo.TypeOf(n)) {
					hasQPS, hasBurst := false, false
					badQPS, badBurst := false, false
					for _, el := range n.Elts {
						if kv, ok := el.(*ast.KeyValueExpr); ok {
							if id, ok := kv.Key.(*ast.Ident); ok {
								switch id.Name {
								case "QPS":
									hasQPS = true
									badQPS = isBadFloat(kv.Value)
								case "Burst":
									hasBurst = true
									badBurst = isBadInt(kv.Value)
								}
							}
						}
					}
					if (!hasQPS || !hasBurst) || badQPS || badBurst {
						pass.Reportf(n.Lbrace, "rest.Config QPS/Burst missing or unrealistic")
					}
				}
			case *ast.AssignStmt:
				for i, lhs := range n.Lhs {
					sel, ok := lhs.(*ast.SelectorExpr)
					if !ok || sel.Sel == nil {
						continue
					}
					name := sel.Sel.Name
					if name != "QPS" && name != "Burst" {
						continue
					}
					if i >= len(n.Rhs) {
						continue
					}
					if !isRestConfig(pass.TypesInfo.TypeOf(sel.X)) {
						continue
					}
					if name == "QPS" && isBadFloat(n.Rhs[i]) {
						pass.Reportf(sel.Sel.Pos(), "rest.Config.QPS set to zero or extreme")
					}
					if name == "Burst" && isBadInt(n.Rhs[i]) {
						pass.Reportf(sel.Sel.Pos(), "rest.Config.Burst set to zero or extreme")
					}
				}
			}
			return true
		})
	}
	return nil, nil
}

func isRestConfig(t types.Type) bool {
	if t == nil {
		return false
	}
	if p, ok := t.(*types.Pointer); ok {
		t = p.Elem()
	}
	n, ok := t.(*types.Named)
	if !ok {
		return false
	}
	if n.Obj().Name() != "Config" {
		return false
	}
	if st, ok := n.Underlying().(*types.Struct); ok {
		hasQPS, hasBurst := false, false
		for i := 0; i < st.NumFields(); i++ {
			f := st.Field(i)
			switch f.Name() {
			case "QPS":
				hasQPS = true
			case "Burst":
				hasBurst = true
			}
		}
		return hasQPS && hasBurst
	}
	return false
}

func isBadFloat(e ast.Expr) bool {
	bl, ok := e.(*ast.BasicLit)
	if !ok {
		return false
	}
	if bl.Kind != token.FLOAT && bl.Kind != token.INT {
		return false
	}
	f, err := strconv.ParseFloat(bl.Value, 64)
	if err != nil {
		return false
	}
	return f == 0 || f > 10000
}

func isBadInt(e ast.Expr) bool {
	bl, ok := e.(*ast.BasicLit)
	if !ok || bl.Kind != token.INT {
		return false
	}
	i, err := strconv.ParseInt(bl.Value, 10, 64)
	if err != nil {
		return false
	}
	return i == 0 || i > 100000
}
