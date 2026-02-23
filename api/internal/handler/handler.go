package handler

import (
	"github.com/dom1nux/qapac-api/internal/service"
	"github.com/dom1nux/qapac-api/internal/storage"
)

// Handler holds the domain dependencies for all HTTP handlers.
// A single Handler is shared across all route groups; individual methods are
// registered as gin handler functions.
type Handler struct {
	stopsRepo      storage.StopsRepository
	etaService     *service.ETAService
	routingService *service.RoutingService
}

// New creates a Handler with the given dependencies.
func New(
	stopsRepo storage.StopsRepository,
	etaService *service.ETAService,
	routingService *service.RoutingService,
) *Handler {
	return &Handler{
		stopsRepo:      stopsRepo,
		etaService:     etaService,
		routingService: routingService,
	}
}
