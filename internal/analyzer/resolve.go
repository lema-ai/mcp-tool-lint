package analyzer

import (
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
)

// funcKey is the stable fully-qualified name of a function or method, and the
// join key between endpoints and tools. Generic instantiations fold back to
// their origin, so Register[A,B] and Register[C,D] share one key.
func funcKey(fn *types.Func) string {
	if fn == nil {
		return ""
	}
	if origin := fn.Origin(); origin != nil {
		fn = origin
	}
	return fn.FullName()
}

// resolveFunc returns the function or method an expression denotes: a plain
// identifier (handler), a method value (service.ListThirdParties), or a
// package-qualified function (pkg.Handler). It returns nil for anything not
// statically resolvable — a func literal, a value pulled out of a map — which
// the caller must treat as "unknown", never as "absent".
func resolveFunc(info *types.Info, expr ast.Expr) *types.Func {
	switch e := ast.Unparen(expr).(type) {
	case *ast.Ident:
		fn, _ := info.ObjectOf(e).(*types.Func)
		return fn
	case *ast.SelectorExpr:
		// A method value (x.M) is recorded in Selections. A package-qualified
		// identifier (pkg.F) is not, and resolves through the selected ident.
		if sel, ok := info.Selections[e]; ok {
			fn, _ := sel.Obj().(*types.Func)
			return fn
		}
		fn, _ := info.ObjectOf(e.Sel).(*types.Func)
		return fn
	case *ast.IndexExpr:
		// Explicitly instantiated generic function: F[T].
		return resolveFunc(info, e.X)
	case *ast.IndexListExpr:
		// Explicitly instantiated with several type arguments: F[T, U].
		return resolveFunc(info, e.X)
	}
	return nil
}

// constString reads a constant string expression, so that a named constant
// such as http.MethodGet yields "GET" just as a bare literal would.
func constString(info *types.Info, expr ast.Expr) (string, bool) {
	if tv, ok := info.Types[expr]; ok && tv.Value != nil && tv.Value.Kind() == constant.String {
		return constant.StringVal(tv.Value), true
	}
	// Identifiers are not always present in Types; fall back to the object.
	switch e := ast.Unparen(expr).(type) {
	case *ast.Ident:
		if c, ok := info.ObjectOf(e).(*types.Const); ok && c.Val().Kind() == constant.String {
			return constant.StringVal(c.Val()), true
		}
	case *ast.SelectorExpr:
		if c, ok := info.ObjectOf(e.Sel).(*types.Const); ok && c.Val().Kind() == constant.String {
			return constant.StringVal(c.Val()), true
		}
	}
	return "", false
}

// compositeLit unwraps &T{...} / T{...} to the composite literal underneath.
func compositeLit(expr ast.Expr) *ast.CompositeLit {
	switch e := ast.Unparen(expr).(type) {
	case *ast.CompositeLit:
		return e
	case *ast.UnaryExpr:
		if e.Op == token.AND {
			return compositeLit(e.X)
		}
	}
	return nil
}

// litField returns the value assigned to a named field in a composite literal.
func litField(lit *ast.CompositeLit, name string) ast.Expr {
	if lit == nil {
		return nil
	}
	for _, elt := range lit.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		if key, ok := kv.Key.(*ast.Ident); ok && key.Name == name {
			return kv.Value
		}
	}
	return nil
}
