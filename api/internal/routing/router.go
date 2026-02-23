package routing

import "context"

// RoutingRequest holds the origin and destination coordinates for a route calculation.
type RoutingRequest struct {
	OriginLat      float64
	OriginLon      float64
	DestinationLat float64
	DestinationLon float64
}

// RoutingResponse holds the result of a route calculation.
type RoutingResponse struct {
	// Polyline is the encoded polyline string (Google's Encoded Polyline Algorithm format).
	// Clients are expected to decode this themselves.
	Polyline  string
	DistanceM int
	DurationS int

	// IsFallback is true when the response was produced by the straight-line
	// fallback estimator instead of a live API call. Callers can use this field
	// to surface a warning to the end user or to avoid caching stale estimates.
	IsFallback bool
}

// Router calculates a route between two geographic points.
type Router interface {
	Route(ctx context.Context, req RoutingRequest) (*RoutingResponse, error)
}
