package container

import (
	"context"
	"time"

	oapigen "github.com/dcm-project/k8s-container-service-provider/internal/api/server"
	"github.com/dcm-project/k8s-container-service-provider/internal/util"
)

// GetHealth returns the service health status including uptime and version.
func (h *Handler) GetHealth(_ context.Context, _ oapigen.GetHealthRequestObject) (oapigen.GetHealthResponseObject, error) {
	uptime := max(0, int(time.Since(h.startTime).Seconds()))
	return oapigen.GetHealth200JSONResponse{
		Status:  "healthy",
		Type:    util.Ptr("k8s-container-service-provider.dcm.io/health"),
		Path:    util.Ptr("health"),
		Uptime:  &uptime,
		Version: util.Ptr(h.version),
	}, nil
}
