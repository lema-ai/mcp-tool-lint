// Package mcp is a hand-written stub of
// github.com/modelcontextprotocol/go-sdk/mcp, carrying only what mcptoollint
// matches on.
package mcp

import "context"

// Server is a stub of mcp.Server.
type Server struct{}

// Tool is a stub of mcp.Tool.
type Tool struct {
	Name        string
	Description string
	Annotations *ToolAnnotations
}

// ToolAnnotations is a stub of mcp.ToolAnnotations.
type ToolAnnotations struct {
	ReadOnlyHint bool
	Title        string
}

// CallToolRequest is a stub of mcp.CallToolRequest.
type CallToolRequest struct{}

// CallToolResult is a stub of mcp.CallToolResult.
type CallToolResult struct{}

// ToolHandlerFor is a stub of mcp.ToolHandlerFor.
type ToolHandlerFor[In, Out any] func(ctx context.Context, req *CallToolRequest, in In) (*CallToolResult, Out, error)

// AddTool is a stub of mcp.AddTool.
func AddTool[In, Out any](server *Server, tool *Tool, handler ToolHandlerFor[In, Out]) {}
