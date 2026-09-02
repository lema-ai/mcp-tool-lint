# mcp-tool-lint

`mcptoollint` is a [golangci-lint](https://golangci-lint.run) module plugin that checks every
public HTTP endpoint is also exposed as an MCP tool.

The REST surface and the MCP surface describe the same domain, and nothing else keeps them in
sync: an endpoint can be added and its tool forgotten, leaving the MCP surface silently behind
the REST one. This linter closes that gap at review time.

## What it checks

An endpoint is **covered** when some MCP tool handler reaches the same Go method that serves the
endpoint — directly, or transitively through a helper.

Matching on the handler rather than on the name is deliberate. A tool that composes several
endpoints still counts as covering each of them, so `get_third_party_profile` covers the
`get-third-party` endpoint even though no tool is named `get_third_party`. A name-parity rule
would report that as missing on day one.

It recognises:

- endpoints registered with `huma.Register`, and with the `huma.Get`/`Post`/`Put`/`Patch`/`Delete`
  convenience helpers;
- tools registered with `mcp.AddTool` from the official
  [Go SDK](https://github.com/modelcontextprotocol/go-sdk), including through a local
  registration funnel — a helper that forwards a name and handler parameter to `AddTool` is
  detected, and its own call sites are read for the literal names.

The reverse direction is **not** checked: a tool with no HTTP endpoint is a normal, intentional
thing.

## Example

```
services/public-api/internal/mcp/tools.go:51:6: public-api endpoint "list-audit-logs"
  (GET /audit-logs, handler (*internal.Service).ListAuditLogs, declared at server.go:60)
  has no MCP tool. Register one alongside the other tools in this package, or annotate the
  registration with //mcp:exempt <reason>.
```

The finding is reported on the tool-registration function, not on the endpoint. A diagnostic has
to point into the package being analysed, and the endpoint lives in the package that the tool
package imports — so the endpoint's own location travels in the message instead.

## Exempting an endpoint

Annotate the registration, on its own line or the same line. The reason is required.

```go
//mcp:exempt internal-only, no agent use case
huma.Register(api, huma.Operation{ /* ... */ }, service.Something)
```

`//nolint:mcptoollint` also works, but it has to sit at the report site in the tool package,
which is far from the endpoint that caused it.

## Install

It can run standalone:

```bash
go run github.com/lema-ai/mcp-tool-lint/cmd/mcptoollint ./services/public-api/...
```

## How it works

Endpoints and tools live in different packages, and an `analysis.Analyzer` only ever sees one
package at a time. The two sets meet through an `analysis` **package fact**:

1. Analysing the package that registers endpoints, the linter describes each one — operation id,
   method, path, and the fully-qualified name of its handler method — and exports them as a fact.
2. Facts travel along import edges toward importers. The MCP tool package imports the package
   holding the registrations (never the reverse), so that fact is readable when the tool package
   is analysed.
3. Analysing the tool package, the linter extracts the tools, walks each handler's calls
   transitively within the package, and diffs the methods it reaches against the endpoints in
   the fact.

## Limits

These are deliberate, and each one fails toward silence rather than toward a false positive.

- **Both packages must be in the lint scope.** The linter locates itself by finding
  registrations, so if the tool package is never analysed nothing is reported — deleting the
  whole MCP package would go unflagged.
- **Reachability is intra-package.** Calls are followed through functions declared in the tool
  package. A tool that reached an endpoint's method via a helper in a *third* package would not
  be credited.
- **Registrations must be statically resolvable.** A tool name built at runtime is fine — the
  handler is what decides coverage — but a handler that is not a resolvable function (one pulled
  from a map, say) makes coverage unknowable. The linter says so at that registration and stops
  checking coverage, rather than inventing findings.
- **A diff-filtered run** (`--new-from-rev`, `only-new-issues`) will hide these findings on a PR
  that only touches the endpoint file, because the diagnostic is anchored in the tool package.

## Development

```bash
go test ./...
```

`internal/analyzer/testdata/src` holds a three-package fixture that mirrors the real topology:
one package registering endpoints, one registering tools through a funnel, and one whose handler
cannot be resolved. huma and the MCP SDK are hand-written stubs there, so the linter itself
depends only on `plugin-module-register` and `golang.org/x/tools`.
