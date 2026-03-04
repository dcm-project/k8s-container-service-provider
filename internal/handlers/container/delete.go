package container

import (
	"context"

	oapigen "github.com/dcm-project/k8s-container-service-provider/internal/api/server"
)

func (h *Handler) DeleteContainer(ctx context.Context, req oapigen.DeleteContainerRequestObject) (oapigen.DeleteContainerResponseObject, error) {
	if err := h.store.Delete(ctx, req.ContainerId); err != nil {
		return h.mapDeleteError(err), nil
	}
	return oapigen.DeleteContainer204Response{}, nil
}
