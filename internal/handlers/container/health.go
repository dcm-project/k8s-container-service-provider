package container

import (
	"context"

	oapigen "github.com/dcm-project/k8s-container-service-provider/internal/api/server"
)

// GetHealth satisfies the StrictServerInterface requirement. This is a
// minimal placeholder; the production health endpoint is served by
// handlers/health.Handler which returns enriched data (uptime, version).
//
// When this handler is wired via NewStrictHandlerWithOptions, the strict
// handler replaces the entire ServerInterface, so this method would shadow
// the enriched health handler. Resolution requires either delegating to the
// health sub-handler or intercepting health requests via StrictMiddlewareFunc.
func (h *Handler) GetHealth(_ context.Context, _ oapigen.GetHealthRequestObject) (oapigen.GetHealthResponseObject, error) {
	return oapigen.GetHealth200JSONResponse{Status: "healthy"}, nil
}
