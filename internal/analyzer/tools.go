package analyzer

import (
	"go/ast"
	"go/token"
	"go/types"
	"sort"

	"golang.org/x/tools/go/analysis"
)

// mcpAddToolFuncs are the fully-qualified names of the MCP SDK functions that
// register a tool. Matched by name so the plugin needs no MCP SDK dependency.
var mcpAddToolFuncs = map[string]bool{
	"github.com/modelcontextprotocol/go-sdk/mcp.AddTool": true,
}

// tool is one registered MCP tool. Exactly one of handlerFn and handlerLit is
// set when the handler was resolvable; both are nil when it was not.
type tool struct {
	name       string
	handlerFn  *types.Func
	handlerLit *ast.FuncLit
	pos        token.Pos
	// enclosing is the function the registration call sits in, used as the
	// diagnostic position: the finding is about an endpoint in another package,
	// so it has to be pinned to something in this one.
	enclosing *ast.FuncDecl
}

// reader knows how to pull the tool name and handler out of one registration
// call. The SDK shape carries the name inside a &mcp.Tool{Name: ...} literal;
// a local funnel carries both in its own parameters.
type reader struct {
	sdk     bool
	name    int
	handler int
}

func (r reader) read(call *ast.CallExpr) (nameExpr, handlerExpr ast.Expr) {
	if r.sdk {
		if len(call.Args) < 3 {
			return nil, nil
		}
		return litField(compositeLit(call.Args[1]), "Name"), call.Args[2]
	}
	if r.name >= len(call.Args) || r.handler >= len(call.Args) {
		return nil, nil
	}
	return call.Args[r.name], call.Args[r.handler]
}

// collectTools finds every MCP tool the package registers, following local
// registration funnels back to the call sites that carry the literal name.
//
// It returns degraded=true when a registration was found whose handler could
// not be resolved. Coverage is then unknowable, so the caller must not report
// missing tools — silently passing is wrong, but so is inventing findings.
func collectTools(pass *analysis.Pass, funcs []funcDecl) (found []tool, degraded bool) {
	readers := map[string]reader{}
	for key := range mcpAddToolFuncs {
		readers[key] = reader{sdk: true}
	}

	byPos := map[token.Pos]tool{}

	// Iterate to a fixpoint: discovering that addTool is a funnel makes its own
	// call sites readable, and a funnel may wrap another funnel.
	for {
		changed := false

		for _, fd := range funcs {
			ast.Inspect(fd.decl, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				callee := resolveFunc(pass.TypesInfo, call.Fun)
				if callee == nil {
					return true
				}
				r, known := readers[funcKey(callee)]
				if !known {
					return true
				}
				nameExpr, handlerExpr := r.read(call)
				if nameExpr == nil || handlerExpr == nil {
					return true
				}

				name, named := constString(pass.TypesInfo, nameExpr)

				// A non-constant name arriving through a parameter means the
				// enclosing function is itself a registration funnel, and the
				// literals live at its own call sites.
				if !named {
					nameIdx := paramIndex(pass.TypesInfo, fd.decl, nameExpr)
					handlerIdx := paramIndex(pass.TypesInfo, fd.decl, handlerExpr)
					if nameIdx >= 0 && handlerIdx >= 0 {
						key := funcKey(fd.fn)
						if _, already := readers[key]; !already {
							readers[key] = reader{name: nameIdx, handler: handlerIdx}
							changed = true
						}
						return true
					}
				}

				// Record the registration even when the name could not be
				// read. Coverage is decided by the handler, so an unreadable
				// name must not cost the endpoint its tool — that would be a
				// false positive on the endpoint, not on the tool.
				if _, dup := byPos[call.Pos()]; dup {
					return true
				}
				t := tool{name: name, pos: call.Pos(), enclosing: fd.decl}
				if lit, ok := ast.Unparen(handlerExpr).(*ast.FuncLit); ok {
					t.handlerLit = lit
				} else {
					t.handlerFn = resolveFunc(pass.TypesInfo, handlerExpr)
				}
				byPos[call.Pos()] = t
				changed = true
				return true
			})
		}

		if !changed {
			break
		}
	}

	for _, t := range byPos {
		if t.handlerFn == nil && t.handlerLit == nil {
			degraded = true
			pass.Report(analysis.Diagnostic{
				Pos: t.pos,
				Message: "mcptoollint cannot determine which endpoints the MCP tool " + describeTool(t) +
					" covers: its handler is not a statically resolvable function. " +
					"Endpoint coverage is not verified while this is the case.",
			})
		}
		found = append(found, t)
	}
	sort.Slice(found, func(i, j int) bool { return found[i].pos < found[j].pos })
	return found, degraded
}

// paramIndex returns the flat parameter index an identifier refers to, or -1.
// Parameters are flattened because one field can name several: in
// `name, title, description string` each gets its own index.
func paramIndex(info *types.Info, decl *ast.FuncDecl, expr ast.Expr) int {
	ident, ok := ast.Unparen(expr).(*ast.Ident)
	if !ok {
		return -1
	}
	obj := info.ObjectOf(ident)
	if obj == nil || decl.Type.Params == nil {
		return -1
	}
	idx := 0
	for _, field := range decl.Type.Params.List {
		if len(field.Names) == 0 {
			idx++
			continue
		}
		for _, name := range field.Names {
			if info.ObjectOf(name) == obj {
				return idx
			}
			idx++
		}
	}
	return -1
}

// describeTool names a tool for a diagnostic, tolerating a name that could
// not be read from source.
func describeTool(t tool) string {
	if t.name == "" {
		return "registered here"
	}
	return `"` + t.name + `"`
}
