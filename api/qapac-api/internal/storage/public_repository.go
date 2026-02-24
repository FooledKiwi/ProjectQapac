package storage

import (
	"context"
	"time"
)

// Route represents a bus route with summary info for listing.
type Route struct {
	ID           int32
	Name         string
	Active       bool
	VehicleCount int
}

// RouteDetail is the rich view returned by GET /routes/:id.
type RouteDetail struct {
	ID       int32
	Name     string
	Active   bool
	Stops    []RouteStop
	Vehicles []RouteVehicle
	// ShapePolyline is the WKT linestring of the route geometry (may be empty).
	ShapePolyline string
}

// RouteStop is a stop belonging to a route, ordered by sequence.
type RouteStop struct {
	ID       int32
	Name     string
	Lat      float64
	Lon      float64
	Sequence int32
}

// RouteVehicle is a vehicle currently assigned to a route.
type RouteVehicle struct {
	ID            int32
	PlateNumber   string
	DriverName    string
	CollectorName string
	Status        string
}

// VehiclePosition represents the latest GPS position of a vehicle.
type VehiclePosition struct {
	VehicleID  int32
	Lat        float64
	Lon        float64
	Heading    *float64
	Speed      *float64
	RecordedAt time.Time
}

// NearbyVehicle is the response type for the nearby-vehicles endpoint.
type NearbyVehicle struct {
	ID          int32
	PlateNumber string
	RouteName   string
	Lat         float64
	Lon         float64
}

// Rating represents a trip rating submitted by an anonymous user.
type Rating struct {
	ID        int32
	TripID    int32
	Rating    int16
	DeviceID  string
	CreatedAt time.Time
}

// Favorite represents a user's favorite route (identified by device_id).
type Favorite struct {
	ID        int32
	DeviceID  string
	RouteID   int32
	RouteName string // populated on read via JOIN
	CreatedAt time.Time
}

// PublicRoutesRepository defines read operations on routes for public consumption.
type PublicRoutesRepository interface {
	// ListRoutes returns all active routes with vehicle counts.
	ListRoutes(ctx context.Context) ([]Route, error)

	// GetRouteDetail returns a route with its stops, assigned vehicles, and shape.
	// Returns (nil, nil) if the route is not found.
	GetRouteDetail(ctx context.Context, id int32) (*RouteDetail, error)

	// GetRouteVehiclesWithPositions returns active vehicles on a route,
	// including their latest GPS position.
	GetRouteVehiclesWithPositions(ctx context.Context, routeID int32) ([]RouteVehicleWithPosition, error)
}

// RouteVehicleWithPosition extends RouteVehicle with an optional GPS position.
type RouteVehicleWithPosition struct {
	ID            int32
	PlateNumber   string
	DriverName    string
	CollectorName string
	Status        string
	Position      *VehiclePosition // nil if no position reported yet
}

// VehiclePositionsRepository defines read operations on vehicle GPS positions.
type VehiclePositionsRepository interface {
	// GetPosition returns the latest position for a vehicle, or (nil, nil).
	GetPosition(ctx context.Context, vehicleID int32) (*VehiclePosition, error)

	// FindNearby returns vehicles within radiusMeters of (lat, lon).
	FindNearby(ctx context.Context, lat, lon, radiusMeters float64) ([]NearbyVehicle, error)
}

// RatingsRepository defines operations on trip ratings.
type RatingsRepository interface {
	// CreateRating inserts a new rating. Returns conflict error if the
	// device already rated this trip.
	CreateRating(ctx context.Context, r *Rating) (*Rating, error)
}

// FavoritesRepository defines operations on favorite routes.
type FavoritesRepository interface {
	// ListByDevice returns all favorites for a device, enriched with route name.
	ListByDevice(ctx context.Context, deviceID string) ([]Favorite, error)

	// Add inserts a favorite. No-op if the pair already exists.
	Add(ctx context.Context, deviceID string, routeID int32) (*Favorite, error)

	// Remove deletes a favorite by device + route.
	Remove(ctx context.Context, deviceID string, routeID int32) error
}
