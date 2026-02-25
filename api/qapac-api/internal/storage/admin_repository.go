package storage

import (
	"context"
	"time"
)

// Vehicle represents a bus in the fleet.
type Vehicle struct {
	ID          int32
	PlateNumber string
	RouteID     *int32 // nullable
	Status      string // "active", "inactive", "maintenance"
	CreatedAt   time.Time
}

// VehicleAssignment represents a driver/collector assignment to a vehicle.
type VehicleAssignment struct {
	ID          int32
	VehicleID   int32
	DriverID    int32
	CollectorID *int32 // nullable
	AssignedAt  time.Time
	Active      bool
}

// Alert represents a route change notification or incident report.
type Alert struct {
	ID           int32
	Title        string
	Description  string
	RouteID      *int32 // nullable
	VehiclePlate string
	ImagePath    string
	CreatedBy    *int32 // nullable
	CreatedAt    time.Time
}

// VehiclesRepository defines operations on the vehicles table.
type VehiclesRepository interface {
	// CreateVehicle inserts a new vehicle.
	CreateVehicle(ctx context.Context, v *Vehicle) (*Vehicle, error)

	// GetVehicleByID returns a vehicle by ID, or (nil, nil) if not found.
	GetVehicleByID(ctx context.Context, id int32) (*Vehicle, error)

	// ListVehicles returns vehicles filtered by optional route and status.
	ListVehicles(ctx context.Context, routeID *int32, status string) ([]Vehicle, error)

	// UpdateVehicle updates mutable fields on a vehicle.
	UpdateVehicle(ctx context.Context, v *Vehicle) error

	// AssignVehicle creates or replaces the active assignment for a vehicle.
	AssignVehicle(ctx context.Context, a *VehicleAssignment) (*VehicleAssignment, error)

	// GetActiveAssignment returns the current active assignment for a vehicle, or (nil, nil).
	GetActiveAssignment(ctx context.Context, vehicleID int32) (*VehicleAssignment, error)
}

// ---------------------------------------------------------------------------
// Stops (admin CRUD)
// ---------------------------------------------------------------------------

// AdminStop represents a stop with all fields visible to administrators.
type AdminStop struct {
	ID        int32
	Name      string
	Lat       float64
	Lon       float64
	Active    bool
	CreatedAt time.Time
}

// StopsAdminRepository defines admin CRUD operations on the stops table.
type StopsAdminRepository interface {
	// CreateStop inserts a new stop.
	CreateStop(ctx context.Context, s *AdminStop) (*AdminStop, error)

	// GetStopByID returns a stop by ID (including inactive), or (nil, nil) if not found.
	GetStopByID(ctx context.Context, id int32) (*AdminStop, error)

	// ListStops returns all stops, optionally filtered by active flag.
	ListStops(ctx context.Context, activeOnly bool) ([]AdminStop, error)

	// UpdateStop updates mutable fields on a stop.
	UpdateStop(ctx context.Context, s *AdminStop) error

	// DeactivateStop soft-deletes a stop by setting active = false.
	DeactivateStop(ctx context.Context, id int32) error
}

// ---------------------------------------------------------------------------
// Routes (admin CRUD)
// ---------------------------------------------------------------------------

// AdminRoute represents a route with all fields visible to administrators.
type AdminRoute struct {
	ID     int32
	Name   string
	Active bool
}

// RouteStopEntry represents one stop in a route's ordered stop list.
type RouteStopEntry struct {
	StopID   int32
	Sequence int
}

// AdminRouteDetail is the full representation returned by GetRouteByID.
type AdminRouteDetail struct {
	ID           int32
	Name         string
	Active       bool
	Stops        []RouteStopEntry
	ShapeGeomWKT string // WKT LINESTRING or empty if no shape
}

// RoutesAdminRepository defines admin CRUD operations on routes, route_stops, and route_shapes.
type RoutesAdminRepository interface {
	// CreateRoute inserts a new route.
	CreateRoute(ctx context.Context, r *AdminRoute) (*AdminRoute, error)

	// GetRouteByID returns a route with its stops and shape, or (nil, nil) if not found.
	GetRouteByID(ctx context.Context, id int32) (*AdminRouteDetail, error)

	// ListRoutes returns all routes, optionally filtered by active flag.
	ListRoutes(ctx context.Context, activeOnly bool) ([]AdminRoute, error)

	// UpdateRoute updates mutable fields (name, active) on a route.
	UpdateRoute(ctx context.Context, r *AdminRoute) error

	// DeactivateRoute soft-deletes a route by setting active = false.
	DeactivateRoute(ctx context.Context, id int32) error

	// ReplaceRouteStops replaces the ordered stop list for a route in a transaction.
	ReplaceRouteStops(ctx context.Context, routeID int32, stops []RouteStopEntry) error

	// UpsertRouteShape inserts or updates the geometry for a route.
	// geomWKT must be a valid WKT LINESTRING (e.g. "LINESTRING(lon1 lat1, lon2 lat2, ...)").
	UpsertRouteShape(ctx context.Context, routeID int32, geomWKT string) error
}

// ---------------------------------------------------------------------------
// Alerts
// ---------------------------------------------------------------------------

// AlertsRepository defines operations on the alerts table.
type AlertsRepository interface {
	// CreateAlert inserts a new alert.
	CreateAlert(ctx context.Context, a *Alert) (*Alert, error)

	// GetAlertByID returns an alert by ID, or (nil, nil) if not found.
	GetAlertByID(ctx context.Context, id int32) (*Alert, error)

	// ListAlerts returns recent alerts, optionally filtered by route.
	ListAlerts(ctx context.Context, routeID *int32) ([]Alert, error)

	// DeleteAlert removes an alert by ID.
	DeleteAlert(ctx context.Context, id int32) error
}
