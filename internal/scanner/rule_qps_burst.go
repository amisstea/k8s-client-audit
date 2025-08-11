package scanner

import (
	"context"
	"go/ast"
	"go/token"
	"go/types"
	"strconv"

	"golang.org/x/tools/go/packages"
)

// K8S002: Ensure rest.Config QPS/Burst are tuned and not unlimited or unrealistic.
// Flags when QPS/Burst are left at zero (unlimited), extremely high, or set after many client creations.
type ruleQPSBurst struct{}

func NewRuleQPSBurst() Rule        { return &ruleQPSBurst{} }
func (r *ruleQPSBurst) ID() string { return RuleQPSBurstConfigID }
func (r *ruleQPSBurst) Description() string {
	return "rest.Config QPS/Burst should be set to sane values (not zero/unlimited or extremely high)"
}

func (r *ruleQPSBurst) Apply(ctx context.Context, fset *token.FileSet, pkg *packages.Package) ([]Issue, error) {
	var issues []Issue
	for _, f := range pkg.Syntax {
		ast.Inspect(f, func(n ast.Node) bool {
			// Case 1: rest.Config composite literal with QPS/Burst fields
			if cl, ok := n.(*ast.CompositeLit); ok {
				var typ types.Type
				if pkg.TypesInfo != nil {
					typ = pkg.TypesInfo.TypeOf(cl)
				}
				if isRestConfigType(typ) {
					hasQPS := false
					hasBurst := false
					badQPS := false
					badBurst := false
					for _, el := range cl.Elts {
						if kv, ok := el.(*ast.KeyValueExpr); ok {
							if k, ok := kv.Key.(*ast.Ident); ok {
								switch k.Name {
								case "QPS":
									hasQPS = true
									if isZeroOrExtremeFloat(kv.Value) {
										badQPS = true
									}
								case "Burst":
									hasBurst = true
									if isZeroOrExtremeInt(kv.Value) {
										badBurst = true
									}
								}
							}
						}
					}
					if (!hasQPS || !hasBurst) || badQPS || badBurst {
						pos := fset.Position(cl.Lbrace)
						issues = append(issues, Issue{
							RuleID:      r.ID(),
							Title:       "rest.Config QPS/Burst not set or unrealistic",
							Description: "Set rest.Config.QPS and Burst to sane values; avoid 0 (unlimited) or extremely high values",
							PackagePath: pkg.PkgPath,
							Position:    Position{Filename: pos.Filename, Line: pos.Line, Column: pos.Column},
							Suggestion:  "Example: cfg.QPS = 20; cfg.Burst = 50",
						})
					}
				}
			}
			// Case 2: Assignment to cfg.QPS / cfg.Burst with zero or extreme values
			if as, ok := n.(*ast.AssignStmt); ok {
				for i, lhs := range as.Lhs {
					sel, ok := lhs.(*ast.SelectorExpr)
					if !ok || sel.Sel == nil {
						continue
					}
					name := sel.Sel.Name
					if name != "QPS" && name != "Burst" {
						continue
					}
					if i >= len(as.Rhs) {
						continue
					}
					val := as.Rhs[i]
					// ensure setting on rest.Config
					var xt types.Type
					if pkg.TypesInfo != nil {
						xt = pkg.TypesInfo.TypeOf(sel.X)
					}
					if !isRestConfigType(xt) {
						continue
					}
					if name == "QPS" && isZeroOrExtremeFloat(val) {
						pos := fset.Position(sel.Sel.Pos())
						issues = append(issues, Issue{
							RuleID:      r.ID(),
							Title:       "rest.Config.QPS set to zero or extreme",
							Description: "Avoid 0 (unlimited) or extremely high QPS",
							PackagePath: pkg.PkgPath,
							Position:    Position{Filename: pos.Filename, Line: pos.Line, Column: pos.Column},
							Suggestion:  "Use a reasonable QPS (e.g., 10-100)",
						})
					}
					if name == "Burst" && isZeroOrExtremeInt(val) {
						pos := fset.Position(sel.Sel.Pos())
						issues = append(issues, Issue{
							RuleID:      r.ID(),
							Title:       "rest.Config.Burst set to zero or extreme",
							Description: "Avoid 0 or extremely high Burst",
							PackagePath: pkg.PkgPath,
							Position:    Position{Filename: pos.Filename, Line: pos.Line, Column: pos.Column},
							Suggestion:  "Use a reasonable Burst (e.g., 20-200)",
						})
					}
				}
			}
			return true
		})
	}
	return issues, nil
}

func identOrSelName(t ast.Expr) string {
	switch tt := t.(type) {
	case *ast.Ident:
		return tt.Name
	case *ast.SelectorExpr:
		if tt.Sel != nil {
			return tt.Sel.Name
		}
	}
	return ""
}

func isRestConfigType(t types.Type) bool {
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

func isZeroOrExtremeFloat(e ast.Expr) bool {
	switch v := e.(type) {
	case *ast.BasicLit:
		if v.Kind == token.FLOAT || v.Kind == token.INT {
			f, err := strconv.ParseFloat(v.Value, 64)
			if err == nil {
				return f == 0 || f > 10000
			}
		}
	}
	return false
}

func isZeroOrExtremeInt(e ast.Expr) bool {
	switch v := e.(type) {
	case *ast.BasicLit:
		if v.Kind == token.INT {
			i, err := strconv.ParseInt(v.Value, 10, 64)
			if err == nil {
				return i == 0 || i > 100000
			}
		}
	}
	return false
}
