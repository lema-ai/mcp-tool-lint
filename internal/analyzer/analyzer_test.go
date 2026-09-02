package analyzer_test

import (
	"testing"

	"github.com/lema-ai/mcp-tool-lint/internal/analyzer"
	"golang.org/x/tools/go/analysis/analysistest"
)

// TestAnalyzer exercises the whole pipeline across three packages: example.com/api
// registers the endpoints and publishes them as a fact, example.com/api/mcp
// registers the tools and does the diffing, and example.com/api/mcp2 covers the
// case where a handler cannot be resolved.
func TestAnalyzer(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, analyzer.Analyzer,
		"example.com/api",
		"example.com/api/mcp",
		"example.com/api/mcp2",
	)
}
