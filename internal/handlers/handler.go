package handlers

import (
	"log/slog"
	"net/http"
	"time"

	v1alpha1 "github.com/dcm-project/k8s-container-service-provider/api/v1alpha1"
	oapigen "github.com/dcm-project/k8s-container-service-provider/internal/api/server"
	"github.com/dcm-project/k8s-container-service-provider/internal/handlers/health"
)

// TC-U008: Compile-time assertion that Handler satisfies oapigen.ServerInterface.
var _ oapigen.ServerInterface = (*Handler)(nil)

// Handler is the composite API handler that implements oapigen.ServerInterface.
// It delegates container CRUD to a strict handler wrapper and health to the
// enriched health sub-handler. The apiserver package is responsible only for
// HTTP transport concerns (router, middleware, lifecycle).
type Handler struct {
	oapigen.Unimplemented
	health    *health.Handler
	container oapigen.ServerInterface
}

// New creates a composite Handler with all sub-handlers initialised.
// The container parameter is a ServerInterface obtained from wrapping a
// StrictServerInterface via oapigen.NewStrictHandlerWithOptions.
func New(logger *slog.Logger, startTime time.Time, version string, container oapigen.ServerInterface) *Handler {
	return &Handler{
		health:    health.NewHandler(startTime, version, logger),
		container: container,
	}
}

// GetHealth delegates to the enriched health sub-handler (not the strict
// handler's placeholder), preserving uptime, version, and type information.
func (h *Handler) GetHealth(w http.ResponseWriter, r *http.Request) {
	h.health.GetHealth(w, r)
}

// CreateContainer delegates to the strict container handler.
func (h *Handler) CreateContainer(w http.ResponseWriter, r *http.Request, params v1alpha1.CreateContainerParams) {
	h.container.CreateContainer(w, r, params)
}

// ListContainers delegates to the strict container handler.
func (h *Handler) ListContainers(w http.ResponseWriter, r *http.Request, params v1alpha1.ListContainersParams) {
	h.container.ListContainers(w, r, params)
}

// GetContainer delegates to the strict container handler.
func (h *Handler) GetContainer(w http.ResponseWriter, r *http.Request, containerId v1alpha1.ContainerIdPath) {
	h.container.GetContainer(w, r, containerId)
}

// DeleteContainer delegates to the strict container handler.
func (h *Handler) DeleteContainer(w http.ResponseWriter, r *http.Request, containerId v1alpha1.ContainerIdPath) {
	h.container.DeleteContainer(w, r, containerId)
}
