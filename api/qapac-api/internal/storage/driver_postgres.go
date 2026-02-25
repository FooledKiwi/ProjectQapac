package storage

import (
	"context"
	"errors"
	"fmt"

	"github.com/FooledKiwi/ProjectQapac/api/qapac-api/internal/generated/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ---------------------------------------------------------------------------
// DriverRepository
// ---------------------------------------------------------------------------

type pgDriverRepository struct {
	q *db.Queries
}

// NewDriverRepository creates a DriverRepository backed by the given pool.
func NewDriverRepository(pool *pgxpool.Pool) DriverRepository {
	return &pgDriverRepository{q: db.New(pool)}
}

func (r *pgDriverRepository) GetAssignmentByDriver(ctx context.Context, driverID int32) (*DriverAssignment, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	row, err := r.q.GetAssignmentByDriver(ctx, driverID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("storage: GetAssignmentByDriver: %w", err)
	}

	return &DriverAssignment{
		VehicleID:     row.VehicleID,
		PlateNumber:   row.PlateNumber,
		RouteName:     row.RouteName,
		CollectorName: row.CollectorName,
		AssignedAt:    row.AssignedAt.Time,
	}, nil
}

func (r *pgDriverRepository) UpsertPosition(ctx context.Context, vehicleID int32, lat, lon float64, heading, speed *float64) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	err := r.q.UpsertPosition(ctx, db.UpsertPositionParams{
		VehicleID: vehicleID,
		Lon:       lon,
		Lat:       lat,
		Heading:   pgfloat8ptr(heading),
		Speed:     pgfloat8ptr(speed),
	})
	if err != nil {
		return fmt.Errorf("storage: UpsertPosition: %w", err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// TripsRepository
// ---------------------------------------------------------------------------

type pgTripsRepository struct {
	q *db.Queries
}

// NewTripsRepository creates a TripsRepository backed by the given pool.
func NewTripsRepository(pool *pgxpool.Pool) TripsRepository {
	return &pgTripsRepository{q: db.New(pool)}
}

func (r *pgTripsRepository) StartTrip(ctx context.Context, t *Trip) (*Trip, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	row, err := r.q.StartTrip(ctx, db.StartTripParams{
		VehicleID: t.VehicleID,
		RouteID:   t.RouteID,
		DriverID:  t.DriverID,
	})
	if err != nil {
		return nil, fmt.Errorf("storage: StartTrip: %w", err)
	}

	t.ID = row.ID
	t.StartedAt = row.StartedAt.Time
	t.Status = row.Status.String
	return t, nil
}

func (r *pgTripsRepository) StartTripFromAssignment(ctx context.Context, driverID int32, vehicleID int32) (*Trip, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	// Look up the vehicle's route_id.
	routeIDPg, err := r.q.GetVehicleRouteID(ctx, vehicleID)
	if err != nil {
		return nil, fmt.Errorf("storage: StartTripFromAssignment: vehicle lookup: %w", err)
	}
	if !routeIDPg.Valid {
		return nil, fmt.Errorf("storage: StartTripFromAssignment: no route assigned")
	}

	row, err := r.q.StartTrip(ctx, db.StartTripParams{
		VehicleID: vehicleID,
		RouteID:   routeIDPg.Int32,
		DriverID:  driverID,
	})
	if err != nil {
		return nil, fmt.Errorf("storage: StartTripFromAssignment: insert: %w", err)
	}

	return &Trip{
		ID:        row.ID,
		VehicleID: vehicleID,
		RouteID:   routeIDPg.Int32,
		DriverID:  driverID,
		StartedAt: row.StartedAt.Time,
		Status:    row.Status.String,
	}, nil
}

func (r *pgTripsRepository) GetActiveTrip(ctx context.Context, driverID int32) (*Trip, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	row, err := r.q.GetActiveTrip(ctx, driverID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("storage: GetActiveTrip: %w", err)
	}

	t := &Trip{
		ID:        row.ID,
		VehicleID: row.VehicleID,
		RouteID:   row.RouteID,
		DriverID:  row.DriverID,
		StartedAt: row.StartedAt.Time,
		Status:    row.Status.String,
	}
	if row.EndedAt.Valid {
		endedAt := row.EndedAt.Time
		t.EndedAt = &endedAt
	}
	return t, nil
}

func (r *pgTripsRepository) EndTrip(ctx context.Context, driverID int32) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	rowsAffected, err := r.q.EndTrip(ctx, driverID)
	if err != nil {
		return fmt.Errorf("storage: EndTrip: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("storage: EndTrip: no active trip found")
	}
	return nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// pgfloat8ptr converts a *float64 to pgtype.Float8 (NULL if nil).
func pgfloat8ptr(p *float64) pgtype.Float8 {
	if p == nil {
		return pgtype.Float8{}
	}
	return pgtype.Float8{Float64: *p, Valid: true}
}
