package container

import (
	"context"

	oapigen "github.com/dcm-project/k8s-container-service-provider/internal/api/server"
)

// GetHealth is a minimal placeholder required by StrictServerInterface.
// The composite handler (handlers.Handler) overrides this by explicitly
// delegating GetHealth to the enriched health.Handler, which returns
// uptime, version, and type information. This method is never reached
// in production.
func (h *Handler) GetHealth(_ context.Context, _ oapigen.GetHealthRequestObject) (oapigen.GetHealthResponseObject, error) {
	return oapigen.GetHealth200JSONResponse{Status: "healthy"}, nil
}
