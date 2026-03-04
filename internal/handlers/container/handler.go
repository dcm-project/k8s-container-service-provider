package container

import (
	"context"
	"log/slog"

	oapigen "github.com/dcm-project/k8s-container-service-provider/internal/api/server"
	"github.com/dcm-project/k8s-container-service-provider/internal/store"
	"github.com/google/uuid"
)

// Handler implements oapigen.StrictServerInterface for container CRUD
// operations. It delegates persistence to a store.ContainerRepository and maps
// store errors to typed OpenAPI responses.
type Handler struct {
	store     store.ContainerRepository
	namespace string
	logger    *slog.Logger
}

// NewHandler creates a Handler backed by the given repository.
func NewHandler(repo store.ContainerRepository, namespace string, logger *slog.Logger) *Handler {
	return &Handler{
		store:     repo,
		namespace: namespace,
		logger:    logger,
	}
}

func (h *Handler) CreateContainer(ctx context.Context, req oapigen.CreateContainerRequestObject) (oapigen.CreateContainerResponseObject, error) {
	var id string
	if req.Params.Id != nil {
		if err := validateContainerID(*req.Params.Id); err != nil {
			return newCreateError400(err.Error()), nil
		}
		id = *req.Params.Id
	} else {
		id = uuid.New().String()
	}

	if err := validateResources(req.Body.Resources); err != nil {
		return newCreateError400(err.Error()), nil
	}

	if err := validateUserLabels(req.Body.Metadata.Labels); err != nil {
		return newCreateError400(err.Error()), nil
	}

	result, err := h.store.Create(ctx, *req.Body, id)
	if err != nil {
		return h.mapCreateError(err), nil
	}
	return oapigen.CreateContainer201JSONResponse(*result), nil
}
