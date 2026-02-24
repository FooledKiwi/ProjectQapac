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
