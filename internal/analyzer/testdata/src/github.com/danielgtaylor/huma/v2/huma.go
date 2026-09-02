// Package huma is a hand-written stub of github.com/danielgtaylor/huma/v2,
// carrying only what mcptoollint matches on. Stubbing the dependency keeps the
// linter's own go.mod down to two modules.
package huma

import "context"

// API is a stub of huma.API.
type API interface {
	stub()
}

// Operation is a stub of huma.Operation.
type Operation struct {
	Method      string
	Path        string
	Summary     string
	Description string
	OperationID string
	Tags        []string
}

// Register is a stub of huma.Register.
func Register[I, O any](api API, op Operation, handler func(context.Context, *I) (*O, error)) {}

// Get is a stub of huma's convenience registration helper.
func Get[I, O any](api API, path string, handler func(context.Context, *I) (*O, error), operationHandlers ...func(o *Operation)) {
}
