package analyzer

import (
	"go/ast"
	"go/token"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// humaRegisterFuncs maps the fully-qualified name of each huma registration
// function to the HTTP method it implies. An empty value means the method is
// carried in a huma.Operation literal instead (the generic Register form).
//
// Matching by name rather than by type keeps this linter free of any
// dependency on huma itself, so the plugin builds from two modules.
var humaRegisterFuncs = map[string]string{
	"github.com/danielgtaylor/huma/v2.Register": "",
	"github.com/danielgtaylor/huma/v2.Get":      "GET",
	"github.com/danielgtaylor/huma/v2.Post":     "POST",
	"github.com/danielgtaylor/huma/v2.Put":      "PUT",
	"github.com/danielgtaylor/huma/v2.Patch":    "PATCH",
	"github.com/danielgtaylor/huma/v2.Delete":   "DELETE",
}

const exemptDirective = "//mcp:exempt"

// collectEndpoints describes every endpoint the package registers.
func collectEndpoints(pass *analysis.Pass) []Endpoint {
	var endpoints []Endpoint

	for _, file := range pass.Files {
		// Directives are read from the file currently being walked, so a
		// comment can only ever suppress something in its own file.
		exempt := exemptLines(pass, file)

		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			endpoint, ok := describeRegistration(pass, call)
			if !ok {
				return true
			}

			line := pass.Fset.Position(call.Pos()).Line
			reason, marked := exempt[line]
			endpoint.Exempt = marked
			if marked && reason == "" {
				pass.Report(analysis.Diagnostic{
					Pos: call.Pos(),
					End: call.End(),
					Message: "//mcp:exempt needs a reason — write //mcp:exempt <why this endpoint " +
						"is deliberately not an MCP tool>",
				})
			}

			endpoints = append(endpoints, endpoint)
			return true
		})
	}

	// Deterministic order: the fact is cached and compared across runs.
	sort.Slice(endpoints, func(i, j int) bool {
		if endpoints[i].OperationID != endpoints[j].OperationID {
			return endpoints[i].OperationID < endpoints[j].OperationID
		}
		return endpoints[i].Handler < endpoints[j].Handler
	})
	return endpoints
}

// describeRegistration reads one huma registration call. It reports false for
// any call that is not a registration, and for a registration whose handler
// cannot be resolved statically — unknown, which must never be mistaken for
// uncovered.
func describeRegistration(pass *analysis.Pass, call *ast.CallExpr) (Endpoint, bool) {
	fn := resolveFunc(pass.TypesInfo, call.Fun)
	if fn == nil {
		return Endpoint{}, false
	}
	method, ok := humaRegisterFuncs[funcKey(fn)]
	if !ok {
		return Endpoint{}, false
	}
	if len(call.Args) < 3 {
		return Endpoint{}, false
	}
	handler := resolveFunc(pass.TypesInfo, call.Args[2])
	if handler == nil {
		return Endpoint{}, false
	}

	endpoint := Endpoint{
		Handler:  funcKey(handler),
		Position: shortPosition(pass, call.Pos()),
	}
	if method == "" {
		// huma.Register(api, huma.Operation{...}, handler)
		lit := compositeLit(call.Args[1])
		endpoint.OperationID, _ = constString(pass.TypesInfo, litField(lit, "OperationID"))
		endpoint.Method, _ = constString(pass.TypesInfo, litField(lit, "Method"))
		endpoint.Path, _ = constString(pass.TypesInfo, litField(lit, "Path"))
	} else {
		// huma.Get(api, path, handler) and friends.
		endpoint.Method = method
		endpoint.Path, _ = constString(pass.TypesInfo, call.Args[1])
	}
	return endpoint, true
}

// exemptLines maps every line covered by a //mcp:exempt directive to its
// reason. A directive covers its own line and the line below, the same
// same-line-or-next semantics lemmata's //nolint:preloadorgid escape hatch
// already uses.
func exemptLines(pass *analysis.Pass, file *ast.File) map[int]string {
	out := map[int]string{}
	for _, group := range file.Comments {
		for _, comment := range group.List {
			text := strings.TrimSpace(comment.Text)
			if !strings.HasPrefix(text, exemptDirective) {
				continue
			}
			reason := strings.TrimSpace(strings.TrimPrefix(text, exemptDirective))
			reason = strings.TrimSpace(strings.TrimLeft(reason, ":"))

			line := pass.Fset.Position(comment.Slash).Line
			for _, covered := range []int{line, line + 1} {
				// Never let a bare directive erase a reason already recorded
				// for the same line by an adjacent one.
				if existing, seen := out[covered]; !seen || existing == "" {
					out[covered] = reason
				}
			}
		}
	}
	return out
}

// shortPosition renders a position as file:line, without the absolute path
// that would otherwise dominate the diagnostic message.
func shortPosition(pass *analysis.Pass, pos token.Pos) string {
	p := pass.Fset.Position(pos)
	return filepath.Base(p.Filename) + ":" + strconv.Itoa(p.Line)
}
