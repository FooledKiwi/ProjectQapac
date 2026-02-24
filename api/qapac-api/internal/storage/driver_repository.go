package storage

import (
	"context"
	"time"
)

// Trip represents a bus trip (a driver running a route).
type Trip struct {
	ID        int32
	VehicleID int32
	RouteID   int32
	DriverID  int32
	StartedAt time.Time
	EndedAt   *time.Time
	Status    string // "active", "completed", "cancelled"
}

// DriverAssignment is the driver-facing view of their current vehicle assignment.
type DriverAssignment struct {
	VehicleID     int32
	PlateNumber   string
	RouteName     string
	CollectorName string
	AssignedAt    time.Time
}

// DriverRepository defines driver-specific data operations.
type DriverRepository interface {
	// GetAssignmentByDriver returns the active assignment for a driver,
	// or (nil, nil) if none exists.
	GetAssignmentByDriver(ctx context.Context, driverID int32) (*DriverAssignment, error)

	// UpsertPosition inserts or updates the latest GPS position for a vehicle.
	UpsertPosition(ctx context.Context, vehicleID int32, lat, lon float64, heading, speed *float64) error
}

// TripsRepository defines operations on the trips table.
type TripsRepository interface {
	// StartTrip creates a new active trip.
	StartTrip(ctx context.Context, t *Trip) (*Trip, error)

	// StartTripFromAssignment creates a new active trip by looking up the
	// vehicle's route_id from the vehicles table. Returns an error containing
	// "no route assigned" if the vehicle has no route.
	StartTripFromAssignment(ctx context.Context, driverID int32, vehicleID int32) (*Trip, error)

	// GetActiveTrip returns the active trip for a driver, or (nil, nil).
	GetActiveTrip(ctx context.Context, driverID int32) (*Trip, error)

	// EndTrip marks the active trip for a driver as completed.
	EndTrip(ctx context.Context, driverID int32) error
}
