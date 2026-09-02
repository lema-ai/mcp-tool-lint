package api // want package:`endpoints: =>.*ListPings, get-thing=>.*GetThing, list-bare=>.*ListBare \(exempt\), list-dynamic=>.*ListDynamic, list-events=>.*ListEvents, list-exempt=>.*ListExempt \(exempt\), list-things=>.*ListThings\}$`

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
)

// Service serves the HTTP endpoints. Its methods are the join key between the
// REST surface and the MCP surface.
type Service struct{}

type ListThingsRequest struct{}
type ListThingsResponse struct{}
type GetThingRequest struct{ ID string }
type GetThingResponse struct{}
type ListEventsRequest struct{}
type ListEventsResponse struct{}
type ListExemptRequest struct{}
type ListExemptResponse struct{}
type ListBareRequest struct{}
type ListBareResponse struct{}
type ListDynamicRequest struct{}
type ListDynamicResponse struct{}
type ListPingsRequest struct{}
type ListPingsResponse struct{}
type ListOnlyToolRequest struct{}
type ListOnlyToolResponse struct{}
type ListSecretRequest struct{}
type ListSecretResponse struct{}

func (this *Service) ListThings(ctx context.Context, req *ListThingsRequest) (*ListThingsResponse, error) {
	return nil, nil
}

func (this *Service) GetThing(ctx context.Context, req *GetThingRequest) (*GetThingResponse, error) {
	return nil, nil
}

func (this *Service) ListEvents(ctx context.Context, req *ListEventsRequest) (*ListEventsResponse, error) {
	return nil, nil
}

func (this *Service) ListExempt(ctx context.Context, req *ListExemptRequest) (*ListExemptResponse, error) {
	return nil, nil
}

func (this *Service) ListBare(ctx context.Context, req *ListBareRequest) (*ListBareResponse, error) {
	return nil, nil
}

func (this *Service) ListPings(ctx context.Context, req *ListPingsRequest) (*ListPingsResponse, error) {
	return nil, nil
}

// ListDynamic is covered only by a tool whose name is not a constant. The
// name is unreadable, but the handler still proves coverage.
func (this *Service) ListDynamic(ctx context.Context, req *ListDynamicRequest) (*ListDynamicResponse, error) {
	return nil, nil
}

// ListOnlyTool is reached by an MCP tool but never registered as an endpoint.
// The reverse direction is deliberately not checked, so this must stay silent.
func (this *Service) ListOnlyTool(ctx context.Context, req *ListOnlyToolRequest) (*ListOnlyToolResponse, error) {
	return nil, nil
}

// ListSecret is only ever passed to a non-huma Register, so it must not be
// treated as an endpoint at all.
func (this *Service) ListSecret(ctx context.Context, req *ListSecretRequest) (*ListSecretResponse, error) {
	return nil, nil
}

// fakeRouter has a Register method that is not huma's. Matching on the method
// name alone would turn the call below into a phantom endpoint.
type fakeRouter struct{}

func (fakeRouter) Register(api huma.API, op huma.Operation, handler any) {}

func NewServer(api huma.API, service *Service) {
	huma.Register(api, huma.Operation{
		Method:      http.MethodGet,
		Path:        "/things",
		OperationID: "list-things",
	}, service.ListThings)

	huma.Register(api, huma.Operation{
		Method:      http.MethodGet,
		Path:        "/things/{id}",
		OperationID: "get-thing",
	}, service.GetThing)

	huma.Register(api, huma.Operation{
		Method:      http.MethodGet,
		Path:        "/events",
		OperationID: "list-events",
	}, service.ListEvents)

	//mcp:exempt internal-only, no agent use case
	huma.Register(api, huma.Operation{
		Method:      http.MethodGet,
		Path:        "/exempt",
		OperationID: "list-exempt",
	}, service.ListExempt)

	//mcp:exempt
	huma.Register(api, huma.Operation{ // want `//mcp:exempt needs a reason`
		Method:      http.MethodGet,
		Path:        "/bare",
		OperationID: "list-bare",
	}, service.ListBare)

	huma.Register(api, huma.Operation{
		Method:      http.MethodGet,
		Path:        "/dynamic",
		OperationID: "list-dynamic",
	}, service.ListDynamic)

	// The convenience helper carries the method in its own name and the path
	// as a plain argument, with no operation id at all.
	huma.Get(api, "/pings", service.ListPings)

	fakeRouter{}.Register(api, huma.Operation{
		Method:      http.MethodGet,
		Path:        "/secret",
		OperationID: "not-an-endpoint",
	}, service.ListSecret)
}
