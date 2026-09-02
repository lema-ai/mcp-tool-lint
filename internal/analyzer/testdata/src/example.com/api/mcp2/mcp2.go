// Package mcp2 registers a tool whose handler cannot be resolved statically.
// Coverage is then unknowable, so the linter must say so rather than invent
// findings for every endpoint in the imported package.
package mcp2

import (
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"example.com/api"
)

func pick() mcpsdk.ToolHandlerFor[api.ListThingsRequest, *api.ListThingsResponse] {
	return nil
}

func registerTools(server *mcpsdk.Server) {
	mcpsdk.AddTool(server, &mcpsdk.Tool{Name: "opaque"}, pick()) // want `cannot determine which endpoints the MCP tool "opaque" covers`
}
