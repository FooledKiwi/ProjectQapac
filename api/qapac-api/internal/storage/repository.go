// Package storage provides PostgreSQL-backed repository implementations.
package storage

import "context"

// Stop represents a public transport stop with its geographic location.
type Stop struct {
	ID   int32
	Name string
	Lat  float64
	Lon  float64
}

// RouteShape represents the path geometry of a route.
type RouteShape struct {
	ID      int32
	RouteID int32
	// WKT representation of the linestring geometry (e.g. "LINESTRING(...)").
	GeomWKT string
}

// StopsRepository defines read operations on the stops table.
type StopsRepository interface {
	// FindStopsNear returns all active stops within radiusMeters of (lat, lon),
	// ordered by distance ascending.
	FindStopsNear(ctx context.Context, lat, lon, radiusMeters float64) ([]Stop, error)

	// GetStop returns a single active stop by ID.
	// Returns (nil, nil) when the stop does not exist.
	GetStop(ctx context.Context, id int32) (*Stop, error)
}

// RoutesRepository defines read operations on route geometry.
type RoutesRepository interface {
	// GetRouteShape returns the shape of the route identified by routeID.
	// Returns (nil, nil) when the route has no shape recorded.
	GetRouteShape(ctx context.Context, routeID int32) (*RouteShape, error)
}
