package analyzer

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
)

// funcDecl pairs a function declared in this package with its syntax.
type funcDecl struct {
	fn   *types.Func
	decl *ast.FuncDecl
}

// packageFuncs lists every function and method declared in the package, in
// source order so that analysis is deterministic.
func packageFuncs(pass *analysis.Pass) []funcDecl {
	var funcs []funcDecl
	for _, file := range pass.Files {
		for _, decl := range file.Decls {
			fd, ok := decl.(*ast.FuncDecl)
			if !ok || fd.Body == nil {
				continue
			}
			fn, ok := pass.TypesInfo.ObjectOf(fd.Name).(*types.Func)
			if !ok {
				continue
			}
			funcs = append(funcs, funcDecl{fn: origin(fn), decl: fd})
		}
	}
	return funcs
}

func origin(fn *types.Func) *types.Func {
	if fn == nil {
		return nil
	}
	if o := fn.Origin(); o != nil {
		return o
	}
	return fn
}

// reachedFuncs returns the fully-qualified names of every function outside
// this package that the given tool handlers reach, following calls
// transitively through functions declared in this package.
//
// Transitivity is what makes handler-identity matching work in practice: a
// tool may reach the endpoint's method through a helper rather than calling it
// in its own body.
func reachedFuncs(pass *analysis.Pass, tools []tool, byFunc map[*types.Func]*ast.FuncDecl) map[string]bool {
	reached := map[string]bool{}
	visited := map[*types.Func]bool{}
	var queue []ast.Node

	enqueue := func(fn *types.Func) {
		fn = origin(fn)
		if fn == nil {
			return
		}
		decl, local := byFunc[fn]
		if !local {
			// Declared elsewhere: we cannot walk its body, but the call itself
			// is the coverage. A handler registered directly as the endpoint's
			// own method lands here.
			reached[funcKey(fn)] = true
			return
		}
		if visited[fn] {
			return
		}
		visited[fn] = true
		queue = append(queue, decl)
	}

	for _, t := range tools {
		switch {
		case t.handlerLit != nil:
			queue = append(queue, t.handlerLit)
		case t.handlerFn != nil:
			enqueue(t.handlerFn)
		}
	}

	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]

		ast.Inspect(node, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			if callee := resolveFunc(pass.TypesInfo, call.Fun); callee != nil {
				enqueue(callee)
			}
			return true
		})
	}
	return reached
}
