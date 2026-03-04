package container

import (
	"context"

	oapigen "github.com/dcm-project/k8s-container-service-provider/internal/api/server"
)

func (h *Handler) ListContainers(ctx context.Context, req oapigen.ListContainersRequestObject) (oapigen.ListContainersResponseObject, error) {
	var maxPageSize int32
	if req.Params.MaxPageSize != nil {
		maxPageSize = *req.Params.MaxPageSize
	}

	var pageToken string
	if req.Params.PageToken != nil {
		pageToken = *req.Params.PageToken
	}

	result, err := h.store.List(ctx, maxPageSize, pageToken)
	if err != nil {
		return h.mapListError(err), nil
	}
	return oapigen.ListContainers200JSONResponse(*result), nil
}
