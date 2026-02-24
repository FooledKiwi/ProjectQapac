package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// pgVehiclesRepository is the pgx-backed implementation of VehiclesRepository.
type pgVehiclesRepository struct {
	pool *pgxpool.Pool
}

// NewVehiclesRepository creates a VehiclesRepository backed by the given pool.
func NewVehiclesRepository(pool *pgxpool.Pool) VehiclesRepository {
	return &pgVehiclesRepository{pool: pool}
}

func (r *pgVehiclesRepository) CreateVehicle(ctx context.Context, v *Vehicle) (*Vehicle, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	var id int32
	var createdAt time.Time
	err := r.pool.QueryRow(ctx,
		`INSERT INTO vehicles (plate_number, route_id, status)
		 VALUES ($1, $2, $3)
		 RETURNING id, created_at`,
		v.PlateNumber, v.RouteID, v.Status,
	).Scan(&id, &createdAt)
	if err != nil {
		return nil, fmt.Errorf("storage: CreateVehicle: %w", err)
	}

	v.ID = id
	v.CreatedAt = createdAt
	return v, nil
}

func (r *pgVehiclesRepository) GetVehicleByID(ctx context.Context, id int32) (*Vehicle, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	v := &Vehicle{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, plate_number, route_id, status, created_at
		 FROM vehicles
		 WHERE id = $1`,
		id,
	).Scan(&v.ID, &v.PlateNumber, &v.RouteID, &v.Status, &v.CreatedAt)

	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("storage: GetVehicleByID: %w", err)
	}
	return v, nil
}

func (r *pgVehiclesRepository) ListVehicles(ctx context.Context, routeID *int32, status string) ([]Vehicle, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	query := `SELECT id, plate_number, route_id, status, created_at FROM vehicles WHERE 1=1`
	args := []interface{}{}
	argIdx := 1

	if routeID != nil {
		query += fmt.Sprintf(" AND route_id = $%d", argIdx)
		args = append(args, *routeID)
		argIdx++
	}
	if status != "" {
		query += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, status)
		argIdx++
	}
	query += " ORDER BY created_at DESC"

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("storage: ListVehicles: %w", err)
	}
	defer rows.Close()

	var vehicles []Vehicle
	for rows.Next() {
		var v Vehicle
		if err := rows.Scan(&v.ID, &v.PlateNumber, &v.RouteID, &v.Status, &v.CreatedAt); err != nil {
			return nil, fmt.Errorf("storage: ListVehicles: scan: %w", err)
		}
		vehicles = append(vehicles, v)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("storage: ListVehicles: rows: %w", err)
	}

	return vehicles, nil
}

func (r *pgVehiclesRepository) UpdateVehicle(ctx context.Context, v *Vehicle) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	_, err := r.pool.Exec(ctx,
		`UPDATE vehicles SET plate_number = $1, route_id = $2, status = $3 WHERE id = $4`,
		v.PlateNumber, v.RouteID, v.Status, v.ID,
	)
	if err != nil {
		return fmt.Errorf("storage: UpdateVehicle: %w", err)
	}
	return nil
}

func (r *pgVehiclesRepository) AssignVehicle(ctx context.Context, a *VehicleAssignment) (*VehicleAssignment, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("storage: AssignVehicle: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Deactivate any existing active assignment for this vehicle.
	_, err = tx.Exec(ctx,
		`UPDATE vehicle_assignments SET active = false WHERE vehicle_id = $1 AND active = true`,
		a.VehicleID,
	)
	if err != nil {
		return nil, fmt.Errorf("storage: AssignVehicle: deactivate old: %w", err)
	}

	// Insert new assignment.
	var id int32
	var assignedAt time.Time
	err = tx.QueryRow(ctx,
		`INSERT INTO vehicle_assignments (vehicle_id, driver_id, collector_id, active)
		 VALUES ($1, $2, $3, true)
		 RETURNING id, assigned_at`,
		a.VehicleID, a.DriverID, a.CollectorID,
	).Scan(&id, &assignedAt)
	if err != nil {
		return nil, fmt.Errorf("storage: AssignVehicle: insert: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("storage: AssignVehicle: commit: %w", err)
	}

	a.ID = id
	a.AssignedAt = assignedAt
	a.Active = true
	return a, nil
}

func (r *pgVehiclesRepository) GetActiveAssignment(ctx context.Context, vehicleID int32) (*VehicleAssignment, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	a := &VehicleAssignment{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, vehicle_id, driver_id, collector_id, assigned_at, active
		 FROM vehicle_assignments
		 WHERE vehicle_id = $1 AND active = true`,
		vehicleID,
	).Scan(&a.ID, &a.VehicleID, &a.DriverID, &a.CollectorID, &a.AssignedAt, &a.Active)

	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("storage: GetActiveAssignment: %w", err)
	}
	return a, nil
}

// pgAlertsRepository is the pgx-backed implementation of AlertsRepository.
type pgAlertsRepository struct {
	pool *pgxpool.Pool
}

// NewAlertsRepository creates an AlertsRepository backed by the given pool.
func NewAlertsRepository(pool *pgxpool.Pool) AlertsRepository {
	return &pgAlertsRepository{pool: pool}
}

func (r *pgAlertsRepository) CreateAlert(ctx context.Context, a *Alert) (*Alert, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	var id int32
	var createdAt time.Time
	err := r.pool.QueryRow(ctx,
		`INSERT INTO alerts (title, description, route_id, vehicle_plate, image_path, created_by)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, created_at`,
		a.Title, a.Description, a.RouteID, a.VehiclePlate, a.ImagePath, a.CreatedBy,
	).Scan(&id, &createdAt)
	if err != nil {
		return nil, fmt.Errorf("storage: CreateAlert: %w", err)
	}

	a.ID = id
	a.CreatedAt = createdAt
	return a, nil
}

func (r *pgAlertsRepository) GetAlertByID(ctx context.Context, id int32) (*Alert, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	a := &Alert{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, title, COALESCE(description, ''), route_id, COALESCE(vehicle_plate, ''),
		        COALESCE(image_path, ''), created_by, created_at
		 FROM alerts
		 WHERE id = $1`,
		id,
	).Scan(&a.ID, &a.Title, &a.Description, &a.RouteID, &a.VehiclePlate,
		&a.ImagePath, &a.CreatedBy, &a.CreatedAt)

	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("storage: GetAlertByID: %w", err)
	}
	return a, nil
}

func (r *pgAlertsRepository) ListAlerts(ctx context.Context, routeID *int32) ([]Alert, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	query := `SELECT id, title, COALESCE(description, ''), route_id, COALESCE(vehicle_plate, ''),
	                 COALESCE(image_path, ''), created_by, created_at
	          FROM alerts`
	args := []interface{}{}

	if routeID != nil {
		query += " WHERE route_id = $1"
		args = append(args, *routeID)
	}
	query += " ORDER BY created_at DESC"

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("storage: ListAlerts: %w", err)
	}
	defer rows.Close()

	var alerts []Alert
	for rows.Next() {
		var a Alert
		if err := rows.Scan(&a.ID, &a.Title, &a.Description, &a.RouteID, &a.VehiclePlate,
			&a.ImagePath, &a.CreatedBy, &a.CreatedAt); err != nil {
			return nil, fmt.Errorf("storage: ListAlerts: scan: %w", err)
		}
		alerts = append(alerts, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("storage: ListAlerts: rows: %w", err)
	}

	return alerts, nil
}

func (r *pgAlertsRepository) DeleteAlert(ctx context.Context, id int32) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	_, err := r.pool.Exec(ctx, `DELETE FROM alerts WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("storage: DeleteAlert: %w", err)
	}
	return nil
}
