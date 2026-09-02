package mcp

import (
	"context"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"example.com/api"
)

// toolServer mirrors lemmata's shape: the tool handlers are methods on an
// unexported type that holds the HTTP service and calls it in-process.
type toolServer struct {
	service *api.Service
}

// addTool is a local registration funnel. The tool name reaches mcpsdk.AddTool
// through a parameter, so the literal names only exist at addTool's own call
// sites — the linter has to unwrap one level to find them.
func addTool[In, Out any](server *mcpsdk.Server, registered map[string]bool, name, title string, handler mcpsdk.ToolHandlerFor[In, Out]) {
	registered[name] = true
	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        name,
		Annotations: &mcpsdk.ToolAnnotations{ReadOnlyHint: true, Title: title},
	}, handler)
}

func registerTools(server *mcpsdk.Server, service *api.Service) map[string]bool { // want `endpoint "list-events" .* has no MCP tool`
	ts := &toolServer{service: service}
	registered := map[string]bool{}

	addTool(server, registered, "list_things", "List things", ts.listThings)
	addTool(server, registered, "get_thing_profile", "Thing profile", ts.getThingProfile)
	addTool(server, registered, "list_pings", "List pings", ts.listPings)
	addTool(server, registered, "list_only_tool", "Tool with no endpoint", ts.listOnlyTool)

	// A tool whose name is not a constant: unreadable as a name, but its
	// handler still covers an endpoint, so coverage must still be credited.
	suffix := "dynamic"
	addTool(server, registered, "prefix_"+suffix, "Dynamic", ts.listDynamic)

	return registered
}

// listThings calls the endpoint's own method directly.
func (this *toolServer) listThings(ctx context.Context, req *mcpsdk.CallToolRequest, in api.ListThingsRequest) (*mcpsdk.CallToolResult, *api.ListThingsResponse, error) {
	resp, err := this.service.ListThings(ctx, &in)
	return nil, resp, err
}

// getThingProfile reaches GetThing only through a helper, two hops down. This
// is what a depth-1 walk would miss.
func (this *toolServer) getThingProfile(ctx context.Context, req *mcpsdk.CallToolRequest, in api.GetThingRequest) (*mcpsdk.CallToolResult, *api.GetThingResponse, error) {
	resp, err := this.loadThing(ctx, in.ID)
	return nil, resp, err
}

func (this *toolServer) loadThing(ctx context.Context, id string) (*api.GetThingResponse, error) {
	return this.describe(ctx, id)
}

func (this *toolServer) describe(ctx context.Context, id string) (*api.GetThingResponse, error) {
	return this.service.GetThing(ctx, &api.GetThingRequest{ID: id})
}

func (this *toolServer) listPings(ctx context.Context, req *mcpsdk.CallToolRequest, in api.ListPingsRequest) (*mcpsdk.CallToolResult, *api.ListPingsResponse, error) {
	resp, err := this.service.ListPings(ctx, &in)
	return nil, resp, err
}

func (this *toolServer) listOnlyTool(ctx context.Context, req *mcpsdk.CallToolRequest, in api.ListOnlyToolRequest) (*mcpsdk.CallToolResult, *api.ListOnlyToolResponse, error) {
	resp, err := this.service.ListOnlyTool(ctx, &in)
	return nil, resp, err
}

func (this *toolServer) listDynamic(ctx context.Context, req *mcpsdk.CallToolRequest, in api.ListDynamicRequest) (*mcpsdk.CallToolResult, *api.ListDynamicResponse, error) {
	resp, err := this.service.ListDynamic(ctx, &in)
	return nil, resp, err
}
