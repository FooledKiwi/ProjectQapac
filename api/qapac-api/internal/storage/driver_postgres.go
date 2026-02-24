package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ---------------------------------------------------------------------------
// DriverRepository
// ---------------------------------------------------------------------------

type pgDriverRepository struct {
	pool *pgxpool.Pool
}

// NewDriverRepository creates a DriverRepository backed by the given pool.
func NewDriverRepository(pool *pgxpool.Pool) DriverRepository {
	return &pgDriverRepository{pool: pool}
}

func (r *pgDriverRepository) GetAssignmentByDriver(ctx context.Context, driverID int32) (*DriverAssignment, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	da := &DriverAssignment{}
	err := r.pool.QueryRow(ctx, `
		SELECT va.vehicle_id, v.plate_number,
		       COALESCE(rt.name, '') AS route_name,
		       COALESCE(col.full_name, '') AS collector_name,
		       va.assigned_at
		FROM vehicle_assignments va
		JOIN vehicles v ON v.id = va.vehicle_id
		LEFT JOIN routes rt ON rt.id = v.route_id
		LEFT JOIN users col ON col.id = va.collector_id
		WHERE va.driver_id = $1 AND va.active = true`,
		driverID,
	).Scan(&da.VehicleID, &da.PlateNumber, &da.RouteName, &da.CollectorName, &da.AssignedAt)

	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("storage: GetAssignmentByDriver: %w", err)
	}
	return da, nil
}

func (r *pgDriverRepository) UpsertPosition(ctx context.Context, vehicleID int32, lat, lon float64, heading, speed *float64) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	_, err := r.pool.Exec(ctx, `
		INSERT INTO vehicle_positions (vehicle_id, geom, heading, speed, recorded_at)
		VALUES ($1, ST_SetSRID(ST_MakePoint($2, $3), 4326), $4, $5, NOW())
		ON CONFLICT (vehicle_id)
		DO UPDATE SET
			geom = ST_SetSRID(ST_MakePoint($2, $3), 4326),
			heading = $4,
			speed = $5,
			recorded_at = NOW()`,
		vehicleID, lon, lat, heading, speed)
	if err != nil {
		return fmt.Errorf("storage: UpsertPosition: %w", err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// TripsRepository
// ---------------------------------------------------------------------------

type pgTripsRepository struct {
	pool *pgxpool.Pool
}

// NewTripsRepository creates a TripsRepository backed by the given pool.
func NewTripsRepository(pool *pgxpool.Pool) TripsRepository {
	return &pgTripsRepository{pool: pool}
}

func (r *pgTripsRepository) StartTrip(ctx context.Context, t *Trip) (*Trip, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	var id int32
	var startedAt time.Time
	err := r.pool.QueryRow(ctx, `
		INSERT INTO trips (vehicle_id, route_id, driver_id, status)
		VALUES ($1, $2, $3, 'active')
		RETURNING id, started_at`,
		t.VehicleID, t.RouteID, t.DriverID,
	).Scan(&id, &startedAt)
	if err != nil {
		return nil, fmt.Errorf("storage: StartTrip: %w", err)
	}

	t.ID = id
	t.StartedAt = startedAt
	t.Status = "active"
	return t, nil
}

func (r *pgTripsRepository) StartTripFromAssignment(ctx context.Context, driverID int32, vehicleID int32) (*Trip, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	// Look up the vehicle's route_id.
	var routeID *int32
	err := r.pool.QueryRow(ctx,
		`SELECT route_id FROM vehicles WHERE id = $1`, vehicleID,
	).Scan(&routeID)
	if err != nil {
		return nil, fmt.Errorf("storage: StartTripFromAssignment: vehicle lookup: %w", err)
	}
	if routeID == nil {
		return nil, fmt.Errorf("storage: StartTripFromAssignment: no route assigned")
	}

	var id int32
	var startedAt time.Time
	err = r.pool.QueryRow(ctx, `
		INSERT INTO trips (vehicle_id, route_id, driver_id, status)
		VALUES ($1, $2, $3, 'active')
		RETURNING id, started_at`,
		vehicleID, *routeID, driverID,
	).Scan(&id, &startedAt)
	if err != nil {
		return nil, fmt.Errorf("storage: StartTripFromAssignment: insert: %w", err)
	}

	return &Trip{
		ID:        id,
		VehicleID: vehicleID,
		RouteID:   *routeID,
		DriverID:  driverID,
		StartedAt: startedAt,
		Status:    "active",
	}, nil
}

func (r *pgTripsRepository) GetActiveTrip(ctx context.Context, driverID int32) (*Trip, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	t := &Trip{}
	err := r.pool.QueryRow(ctx, `
		SELECT id, vehicle_id, route_id, driver_id, started_at, ended_at, status
		FROM trips
		WHERE driver_id = $1 AND status = 'active'
		ORDER BY started_at DESC
		LIMIT 1`,
		driverID,
	).Scan(&t.ID, &t.VehicleID, &t.RouteID, &t.DriverID, &t.StartedAt, &t.EndedAt, &t.Status)

	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("storage: GetActiveTrip: %w", err)
	}
	return t, nil
}

func (r *pgTripsRepository) EndTrip(ctx context.Context, driverID int32) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	tag, err := r.pool.Exec(ctx, `
		UPDATE trips
		SET status = 'completed', ended_at = NOW()
		WHERE driver_id = $1 AND status = 'active'`,
		driverID)
	if err != nil {
		return fmt.Errorf("storage: EndTrip: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("storage: EndTrip: no active trip found")
	}
	return nil
}
