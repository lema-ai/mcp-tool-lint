// Command mcptoollint runs the endpoint/MCP-tool parity check standalone,
// outside golangci-lint. Both the package registering endpoints and the
// package registering tools must be in the analysed set, e.g.
//
//	mcptoollint ./services/public-api/...
package main

import (
	"github.com/lema-ai/mcp-tool-lint/internal/analyzer"
	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() {
	singlechecker.Main(analyzer.Analyzer)
}
