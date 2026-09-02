package analyzer

import (
	"fmt"
	"sort"
	"strings"
)

// Endpoint is one HTTP endpoint found at its registration site.
//
// Handler is the join key between the two API surfaces: the fully-qualified
// name of the Go method that serves the endpoint, e.g.
// "(*github.com/lema.ai/lemmata/services/public-api/internal.Service).ListThirdParties".
// An MCP tool covers this endpoint when its handler reaches that same method.
type Endpoint struct {
	OperationID string
	Method      string
	Path        string
	Handler     string
	// Position is the endpoint's own file:line. The diagnostic is reported in
	// the tool package, so the endpoint's location can only travel as text.
	Position string
	// Exempt records a //mcp:exempt directive on the registration.
	Exempt bool
}

// describe renders the endpoint for a diagnostic message.
func (e Endpoint) describe() string {
	id := e.OperationID
	if id == "" {
		id = "(no operation id)"
	}
	route := strings.TrimSpace(e.Method + " " + e.Path)
	if route == "" {
		return fmt.Sprintf("%q (handler %s, declared at %s)", id, e.Handler, e.Position)
	}
	return fmt.Sprintf("%q (%s, handler %s, declared at %s)", id, route, e.Handler, e.Position)
}

// endpointsFact carries the endpoints a package registers to the packages that
// import it. Analysis facts travel along import edges toward importers, which
// is the direction we need: the MCP tool package imports the package holding
// the huma registrations, never the reverse.
type endpointsFact struct {
	Endpoints []Endpoint
}

func (*endpointsFact) AFact() {}

// String is what analysistest asserts against in `// want package:"..."`
// annotations, so it must be deterministic.
func (f *endpointsFact) String() string {
	parts := make([]string, 0, len(f.Endpoints))
	for _, e := range f.Endpoints {
		part := e.OperationID + "=>" + e.Handler
		if e.Exempt {
			part += " (exempt)"
		}
		parts = append(parts, part)
	}
	sort.Strings(parts)
	return "endpoints: " + strings.Join(parts, ", ")
}
