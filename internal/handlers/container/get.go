package container

import (
	"context"

	oapigen "github.com/dcm-project/k8s-container-service-provider/internal/api/server"
)

func (h *Handler) GetContainer(ctx context.Context, req oapigen.GetContainerRequestObject) (oapigen.GetContainerResponseObject, error) {
	result, err := h.store.Get(ctx, req.ContainerId)
	if err != nil {
		return h.mapGetError(err), nil
	}
	return oapigen.GetContainer200JSONResponse(*result), nil
}
