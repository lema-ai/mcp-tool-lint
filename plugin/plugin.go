// Package plugin registers mcptoollint as a golangci-lint module plugin.
package plugin

import (
	"github.com/golangci/plugin-module-register/register"
	"github.com/lema-ai/mcp-tool-lint/internal/analyzer"
	"golang.org/x/tools/go/analysis"
)

func init() {
	register.Plugin("mcptoollint", New)
}

// New creates a new instance of the mcptoollint plugin. It may receive the
// config object from linters.settings.custom.mcptoollint.settings, but the
// linter has nothing to configure: it locates both API surfaces itself, and
// the escape hatch is a //mcp:exempt comment at the endpoint.
func New(settings any) (register.LinterPlugin, error) {
	return &MCPToolLint{}, nil
}

// MCPToolLint implements the LinterPlugin interface.
type MCPToolLint struct{}

// BuildAnalyzers returns the analyzers provided by this linter.
func (l *MCPToolLint) BuildAnalyzers() ([]*analysis.Analyzer, error) {
	return []*analysis.Analyzer{analyzer.Analyzer}, nil
}

// GetLoadMode returns the load mode required by this linter. Type information
// is required: endpoints and tools are matched by resolved method identity,
// not by name.
func (l *MCPToolLint) GetLoadMode() string {
	return register.LoadModeTypesInfo
}
