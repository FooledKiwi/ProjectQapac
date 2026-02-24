package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/FooledKiwi/ProjectQapac/api/qapac-api/internal/routing"
	"github.com/FooledKiwi/ProjectQapac/api/qapac-api/internal/storage"
)

// ErrStopNotFound is returned by GetRouteTo when the requested stop does not
// exist in the database. Callers should use errors.Is to distinguish this from
// other errors.
var ErrStopNotFound = errors.New("stop not found")

// RoutingService orchestrates route lookups from a user's location to a transit stop.
// It uses a CachedRouter to minimise Google API calls.
type RoutingService struct {
	router    routing.Router
	stopsRepo storage.StopsRepository
}

// NewRoutingService creates a RoutingService.
//
//   - router should be a *routing.CachedRouter wrapping a *routing.GoogleRouter for
//     production use, or any Router implementation for testing.
//   - stopsRepo is used to look up the stop's geographic coordinates by ID.
func NewRoutingService(router routing.Router, stopsRepo storage.StopsRepository) *RoutingService {
	return &RoutingService{
		router:    router,
		stopsRepo: stopsRepo,
	}
}

// GetRouteTo calculates the route from (userLat, userLon) to the stop identified by
// stopID and returns the routing result.
//
// Errors:
//   - Returns ErrStopNotFound (wrapped) if the stop does not exist.
//   - Returns a descriptive error if the underlying Router fails.
func (s *RoutingService) GetRouteTo(ctx context.Context, userLat, userLon float64, stopID int32) (*routing.RoutingResponse, error) {
	stop, err := s.stopsRepo.GetStop(ctx, stopID)
	if err != nil {
		return nil, fmt.Errorf("service: GetRouteTo: fetch stop %d: %w", stopID, err)
	}
	if stop == nil {
		return nil, fmt.Errorf("service: GetRouteTo: stop %d: %w", stopID, ErrStopNotFound)
	}

	// Embed the stop ID in the context so that CachedRouter can build a
	// precise cache key (geohash + stop_id) without altering the Router interface.
	ctx = routing.WithStopID(ctx, stopID)

	resp, err := s.router.Route(ctx, routing.RoutingRequest{
		OriginLat:      userLat,
		OriginLon:      userLon,
		DestinationLat: stop.Lat,
		DestinationLon: stop.Lon,
	})
	if err != nil {
		return nil, fmt.Errorf("service: GetRouteTo: route to stop %d: %w", stopID, err)
	}

	return resp, nil
}
