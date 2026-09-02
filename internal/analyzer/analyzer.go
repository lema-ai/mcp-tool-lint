// Package analyzer implements mcptoollint, which checks that every public HTTP
// endpoint is also reachable as an MCP tool.
package analyzer

import (
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/analysis"
)

const Doc = `mcptoollint checks that every public-api HTTP endpoint is exposed as an MCP tool.

The REST surface and the MCP surface describe the same domain, and nothing else keeps
them in sync: an endpoint can be added and its tool forgotten, leaving the MCP surface
silently behind the REST one.

An endpoint is considered covered when some MCP tool handler reaches the same Go method
that serves the endpoint, directly or through a helper. Matching on the handler rather
than on the name means a tool that composes several endpoints still counts as covering
each of them.

To exempt an endpoint that deliberately has no tool, annotate its registration:

	//mcp:exempt internal-only, no agent use case
	huma.Register(api, huma.Operation{ ... }, service.Something)

The reason is required.
`

var Analyzer = &analysis.Analyzer{
	Name:      "mcptoollint",
	Doc:       Doc,
	Run:       run,
	FactTypes: []analysis.Fact{(*endpointsFact)(nil)},
}

func run(pass *analysis.Pass) (any, error) {
	// Producer role: describe the endpoints this package registers and publish
	// them to importers. The MCP tool package imports the package holding the
	// registrations, and facts travel toward importers, so this is the only
	// direction in which the two sets can meet.
	local := collectEndpoints(pass)
	if len(local) > 0 {
		pass.ExportPackageFact(&endpointsFact{Endpoints: local})
	}

	// Consumer role: only packages that actually register tools go further.
	funcs := packageFuncs(pass)
	tools, degraded := collectTools(pass, funcs)
	if len(tools) == 0 {
		return nil, nil
	}
	if degraded {
		// collectTools already explained why, at the offending registration.
		return nil, nil
	}

	endpoints := knownEndpoints(pass, local)
	if len(endpoints) == 0 {
		return nil, nil
	}

	byFunc := make(map[*types.Func]*ast.FuncDecl, len(funcs))
	for _, fd := range funcs {
		byFunc[fd.fn] = fd.decl
	}
	reached := reachedFuncs(pass, tools, byFunc)

	pos := reportPos(pass, tools)
	for _, endpoint := range endpoints {
		if endpoint.Exempt || reached[endpoint.Handler] {
			continue
		}
		pass.Report(analysis.Diagnostic{
			Pos: pos,
			Message: "public-api endpoint " + endpoint.describe() + " has no MCP tool. " +
				"Register one alongside the other tools in this package, or annotate the " +
				"registration with //mcp:exempt <reason>.",
		})
	}
	return nil, nil
}

// knownEndpoints gathers the endpoints this package registers together with
// those published by the packages it imports, de-duplicated by handler.
func knownEndpoints(pass *analysis.Pass, local []Endpoint) []Endpoint {
	var all []Endpoint
	all = append(all, local...)
	for _, imported := range pass.Pkg.Imports() {
		var fact endpointsFact
		if pass.ImportPackageFact(imported, &fact) {
			all = append(all, fact.Endpoints...)
		}
	}

	seen := map[string]bool{}
	out := make([]Endpoint, 0, len(all))
	for _, endpoint := range all {
		key := endpoint.Handler + "\x00" + endpoint.OperationID
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, endpoint)
	}
	return out
}

// reportPos picks the position to hang findings on. A diagnostic must point
// into the package being analyzed, but the endpoint it describes lives in
// another one, so the registration function is the closest honest anchor.
func reportPos(pass *analysis.Pass, tools []tool) token.Pos {
	for _, t := range tools {
		if t.enclosing != nil {
			return t.enclosing.Name.Pos()
		}
	}
	if len(tools) > 0 && tools[0].pos.IsValid() {
		return tools[0].pos
	}
	if len(pass.Files) > 0 {
		return pass.Files[0].Package
	}
	return token.NoPos
}
