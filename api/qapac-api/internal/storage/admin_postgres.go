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
// Vehicles
// ---------------------------------------------------------------------------

// pgVehiclesRepository is the pgx-backed implementation of VehiclesRepository.
type pgVehiclesRepository struct {
	q    *db.Queries
	pool *pgxpool.Pool
}

// NewVehiclesRepository creates a VehiclesRepository backed by the given pool.
func NewVehiclesRepository(pool *pgxpool.Pool) VehiclesRepository {
	return &pgVehiclesRepository{q: db.New(pool), pool: pool}
}

func (r *pgVehiclesRepository) CreateVehicle(ctx context.Context, v *Vehicle) (*Vehicle, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	row, err := r.q.CreateVehicle(ctx, db.CreateVehicleParams{
		PlateNumber: v.PlateNumber,
		RouteID:     pgint4ptr(v.RouteID),
		Status:      pgtype.Text{String: v.Status, Valid: v.Status != ""},
	})
	if err != nil {
		return nil, fmt.Errorf("storage: CreateVehicle: %w", err)
	}

	v.ID = row.ID
	v.RouteID = int4ptr(row.RouteID)
	v.Status = row.Status.String
	v.CreatedAt = row.CreatedAt.Time
	return v, nil
}

func (r *pgVehiclesRepository) GetVehicleByID(ctx context.Context, id int32) (*Vehicle, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	row, err := r.q.GetVehicleByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("storage: GetVehicleByID: %w", err)
	}

	return vehicleFromRow(row), nil
}

func (r *pgVehiclesRepository) ListVehicles(ctx context.Context, routeID *int32, status string) ([]Vehicle, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	rows, err := r.q.ListVehicles(ctx, db.ListVehiclesParams{
		RouteID: pgint4ptr(routeID),
		Status:  pgtextOrNull(status),
	})
	if err != nil {
		return nil, fmt.Errorf("storage: ListVehicles: %w", err)
	}

	vehicles := make([]Vehicle, 0, len(rows))
	for _, row := range rows {
		vehicles = append(vehicles, *vehicleFromRow(row))
	}
	return vehicles, nil
}

func (r *pgVehiclesRepository) UpdateVehicle(ctx context.Context, v *Vehicle) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	err := r.q.UpdateVehicle(ctx, db.UpdateVehicleParams{
		PlateNumber: v.PlateNumber,
		RouteID:     pgint4ptr(v.RouteID),
		Status:      pgtype.Text{String: v.Status, Valid: v.Status != ""},
		ID:          v.ID,
	})
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
	defer func() { _ = tx.Rollback(ctx) }() //nolint:errcheck // rollback after commit is harmless

	qtx := r.q.WithTx(tx)

	// Deactivate any existing active assignment for this vehicle.
	if deactErr := qtx.DeactivateOldAssignment(ctx, a.VehicleID); deactErr != nil {
		return nil, fmt.Errorf("storage: AssignVehicle: deactivate old: %w", deactErr)
	}

	// Insert new assignment.
	row, err := qtx.InsertAssignment(ctx, db.InsertAssignmentParams{
		VehicleID:   a.VehicleID,
		DriverID:    a.DriverID,
		CollectorID: pgint4ptr(a.CollectorID),
	})
	if err != nil {
		return nil, fmt.Errorf("storage: AssignVehicle: insert: %w", err)
	}

	if commitErr := tx.Commit(ctx); commitErr != nil {
		return nil, fmt.Errorf("storage: AssignVehicle: commit: %w", commitErr)
	}

	a.ID = row.ID
	a.AssignedAt = row.AssignedAt.Time
	a.Active = true
	return a, nil
}

func (r *pgVehiclesRepository) GetActiveAssignment(ctx context.Context, vehicleID int32) (*VehicleAssignment, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	row, err := r.q.GetActiveAssignment(ctx, vehicleID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("storage: GetActiveAssignment: %w", err)
	}

	return &VehicleAssignment{
		ID:          row.ID,
		VehicleID:   row.VehicleID,
		DriverID:    row.DriverID,
		CollectorID: int4ptr(row.CollectorID),
		AssignedAt:  row.AssignedAt.Time,
		Active:      row.Active.Bool,
	}, nil
}

// ---------------------------------------------------------------------------
// Alerts
// ---------------------------------------------------------------------------

// pgAlertsRepository is the pgx-backed implementation of AlertsRepository.
type pgAlertsRepository struct {
	q *db.Queries
}

// NewAlertsRepository creates an AlertsRepository backed by the given pool.
func NewAlertsRepository(pool *pgxpool.Pool) AlertsRepository {
	return &pgAlertsRepository{q: db.New(pool)}
}

func (r *pgAlertsRepository) CreateAlert(ctx context.Context, a *Alert) (*Alert, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	row, err := r.q.CreateAlert(ctx, db.CreateAlertParams{
		Title:        a.Title,
		Description:  pgtext(a.Description),
		RouteID:      pgint4ptr(a.RouteID),
		VehiclePlate: pgtext(a.VehiclePlate),
		ImagePath:    pgtext(a.ImagePath),
		CreatedBy:    pgint4ptr(a.CreatedBy),
	})
	if err != nil {
		return nil, fmt.Errorf("storage: CreateAlert: %w", err)
	}

	a.ID = row.ID
	a.Description = row.Description
	a.RouteID = int4ptr(row.RouteID)
	a.VehiclePlate = row.VehiclePlate
	a.ImagePath = row.ImagePath
	a.CreatedBy = int4ptr(row.CreatedBy)
	a.CreatedAt = row.CreatedAt.Time
	return a, nil
}

func (r *pgAlertsRepository) GetAlertByID(ctx context.Context, id int32) (*Alert, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	row, err := r.q.GetAlertByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("storage: GetAlertByID: %w", err)
	}

	return &Alert{
		ID:           row.ID,
		Title:        row.Title,
		Description:  row.Description,
		RouteID:      int4ptr(row.RouteID),
		VehiclePlate: row.VehiclePlate,
		ImagePath:    row.ImagePath,
		CreatedBy:    int4ptr(row.CreatedBy),
		CreatedAt:    row.CreatedAt.Time,
	}, nil
}

func (r *pgAlertsRepository) ListAlerts(ctx context.Context, routeID *int32) ([]Alert, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	rows, err := r.q.ListAlerts(ctx, pgint4ptr(routeID))
	if err != nil {
		return nil, fmt.Errorf("storage: ListAlerts: %w", err)
	}

	alerts := make([]Alert, 0, len(rows))
	for _, row := range rows {
		alerts = append(alerts, Alert{
			ID:           row.ID,
			Title:        row.Title,
			Description:  row.Description,
			RouteID:      int4ptr(row.RouteID),
			VehiclePlate: row.VehiclePlate,
			ImagePath:    row.ImagePath,
			CreatedBy:    int4ptr(row.CreatedBy),
			CreatedAt:    row.CreatedAt.Time,
		})
	}
	return alerts, nil
}

func (r *pgAlertsRepository) DeleteAlert(ctx context.Context, id int32) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	err := r.q.DeleteAlert(ctx, id)
	if err != nil {
		return fmt.Errorf("storage: DeleteAlert: %w", err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Stops admin CRUD
// ---------------------------------------------------------------------------

// pgStopsAdminRepository is the pgx-backed implementation of StopsAdminRepository.
type pgStopsAdminRepository struct {
	q *db.Queries
}

// NewStopsAdminRepository creates a StopsAdminRepository backed by the given pool.
func NewStopsAdminRepository(pool *pgxpool.Pool) StopsAdminRepository {
	return &pgStopsAdminRepository{q: db.New(pool)}
}

func (r *pgStopsAdminRepository) CreateStop(ctx context.Context, s *AdminStop) (*AdminStop, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	row, err := r.q.AdminCreateStop(ctx, db.AdminCreateStopParams{
		Name: s.Name,
		Lon:  s.Lon,
		Lat:  s.Lat,
	})
	if err != nil {
		return nil, fmt.Errorf("storage: CreateStop: %w", err)
	}

	s.ID = row.ID
	s.Lon = toFloat64(row.Lon)
	s.Lat = toFloat64(row.Lat)
	s.Active = row.Active.Bool
	s.CreatedAt = row.CreatedAt.Time
	return s, nil
}

func (r *pgStopsAdminRepository) GetStopByID(ctx context.Context, id int32) (*AdminStop, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	row, err := r.q.AdminGetStopByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("storage: GetStopByID: %w", err)
	}

	return &AdminStop{
		ID:        row.ID,
		Name:      row.Name,
		Lon:       toFloat64(row.Lon),
		Lat:       toFloat64(row.Lat),
		Active:    row.Active.Bool,
		CreatedAt: row.CreatedAt.Time,
	}, nil
}

func (r *pgStopsAdminRepository) ListStops(ctx context.Context, activeOnly bool) ([]AdminStop, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	rows, err := r.q.AdminListStops(ctx, activeOnly)
	if err != nil {
		return nil, fmt.Errorf("storage: ListStops: %w", err)
	}

	stops := make([]AdminStop, 0, len(rows))
	for _, row := range rows {
		stops = append(stops, AdminStop{
			ID:        row.ID,
			Name:      row.Name,
			Lon:       toFloat64(row.Lon),
			Lat:       toFloat64(row.Lat),
			Active:    row.Active.Bool,
			CreatedAt: row.CreatedAt.Time,
		})
	}
	return stops, nil
}

func (r *pgStopsAdminRepository) UpdateStop(ctx context.Context, s *AdminStop) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	err := r.q.AdminUpdateStop(ctx, db.AdminUpdateStopParams{
		Name:   s.Name,
		Lon:    s.Lon,
		Lat:    s.Lat,
		Active: pgbool(s.Active),
		ID:     s.ID,
	})
	if err != nil {
		return fmt.Errorf("storage: UpdateStop: %w", err)
	}
	return nil
}

func (r *pgStopsAdminRepository) DeactivateStop(ctx context.Context, id int32) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	err := r.q.AdminDeactivateStop(ctx, id)
	if err != nil {
		return fmt.Errorf("storage: DeactivateStop: %w", err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Routes admin CRUD
// ---------------------------------------------------------------------------

// pgRoutesAdminRepository is the pgx-backed implementation of RoutesAdminRepository.
type pgRoutesAdminRepository struct {
	q    *db.Queries
	pool *pgxpool.Pool
}

// NewRoutesAdminRepository creates a RoutesAdminRepository backed by the given pool.
func NewRoutesAdminRepository(pool *pgxpool.Pool) RoutesAdminRepository {
	return &pgRoutesAdminRepository{q: db.New(pool), pool: pool}
}

func (r *pgRoutesAdminRepository) CreateRoute(ctx context.Context, rt *AdminRoute) (*AdminRoute, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	row, err := r.q.AdminCreateRoute(ctx, rt.Name)
	if err != nil {
		return nil, fmt.Errorf("storage: CreateRoute: %w", err)
	}

	rt.ID = row.ID
	rt.Name = row.Name
	rt.Active = row.Active.Bool
	return rt, nil
}

func (r *pgRoutesAdminRepository) GetRouteByID(ctx context.Context, id int32) (*AdminRouteDetail, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	// Fetch route.
	route, err := r.q.AdminGetRouteByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("storage: GetRouteByID: %w", err)
	}

	detail := &AdminRouteDetail{
		ID:     route.ID,
		Name:   route.Name,
		Active: route.Active.Bool,
	}

	// Fetch ordered stops.
	stopRows, err := r.q.AdminGetRouteStops(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("storage: GetRouteByID: stops: %w", err)
	}
	for _, sr := range stopRows {
		detail.Stops = append(detail.Stops, RouteStopEntry{
			StopID:   sr.StopID,
			Sequence: int(sr.Sequence),
		})
	}

	// Fetch shape (optional).
	geomWKT, err := r.q.AdminGetRouteShapeWKT(ctx, id)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("storage: GetRouteByID: shape: %w", err)
	}
	if wkt, ok := geomWKT.(string); ok {
		detail.ShapeGeomWKT = wkt
	}

	return detail, nil
}

func (r *pgRoutesAdminRepository) ListRoutes(ctx context.Context, activeOnly bool) ([]AdminRoute, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	rows, err := r.q.AdminListRoutes(ctx, activeOnly)
	if err != nil {
		return nil, fmt.Errorf("storage: ListRoutes: %w", err)
	}

	routes := make([]AdminRoute, 0, len(rows))
	for _, row := range rows {
		routes = append(routes, AdminRoute{
			ID:     row.ID,
			Name:   row.Name,
			Active: row.Active.Bool,
		})
	}
	return routes, nil
}

func (r *pgRoutesAdminRepository) UpdateRoute(ctx context.Context, rt *AdminRoute) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	err := r.q.AdminUpdateRoute(ctx, db.AdminUpdateRouteParams{
		Name:   rt.Name,
		Active: pgbool(rt.Active),
		ID:     rt.ID,
	})
	if err != nil {
		return fmt.Errorf("storage: UpdateRoute: %w", err)
	}
	return nil
}

func (r *pgRoutesAdminRepository) DeactivateRoute(ctx context.Context, id int32) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	err := r.q.AdminDeactivateRoute(ctx, id)
	if err != nil {
		return fmt.Errorf("storage: DeactivateRoute: %w", err)
	}
	return nil
}

func (r *pgRoutesAdminRepository) ReplaceRouteStops(ctx context.Context, routeID int32, stops []RouteStopEntry) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("storage: ReplaceRouteStops: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }() //nolint:errcheck

	qtx := r.q.WithTx(tx)

	// Delete existing stop associations.
	if delErr := qtx.AdminDeleteRouteStops(ctx, routeID); delErr != nil {
		return fmt.Errorf("storage: ReplaceRouteStops: delete: %w", delErr)
	}

	// Insert new stop associations.
	for _, s := range stops {
		if insErr := qtx.AdminInsertRouteStop(ctx, db.AdminInsertRouteStopParams{
			RouteID:  routeID,
			StopID:   s.StopID,
			Sequence: int32(s.Sequence),
		}); insErr != nil {
			return fmt.Errorf("storage: ReplaceRouteStops: insert stop_id=%d: %w", s.StopID, insErr)
		}
	}

	if commitErr := tx.Commit(ctx); commitErr != nil {
		return fmt.Errorf("storage: ReplaceRouteStops: commit: %w", commitErr)
	}
	return nil
}

func (r *pgRoutesAdminRepository) UpsertRouteShape(ctx context.Context, routeID int32, geomWKT string) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	err := r.q.AdminUpsertRouteShape(ctx, db.AdminUpsertRouteShapeParams{
		RouteID: routeID,
		GeomWkt: geomWKT,
	})
	if err != nil {
		return fmt.Errorf("storage: UpsertRouteShape: %w", err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// vehicleFromRow converts a sqlc Vehicle model to the domain Vehicle struct.
func vehicleFromRow(row db.Vehicle) *Vehicle {
	return &Vehicle{
		ID:          row.ID,
		PlateNumber: row.PlateNumber,
		RouteID:     int4ptr(row.RouteID),
		Status:      row.Status.String,
		CreatedAt:   row.CreatedAt.Time,
	}
}

// pgint4ptr converts a *int32 to pgtype.Int4 (NULL if nil).
func pgint4ptr(p *int32) pgtype.Int4 {
	if p == nil {
		return pgtype.Int4{}
	}
	return pgtype.Int4{Int32: *p, Valid: true}
}

// int4ptr converts a pgtype.Int4 to *int32 (nil if not valid).
func int4ptr(v pgtype.Int4) *int32 {
	if !v.Valid {
		return nil
	}
	val := v.Int32
	return &val
}

// pgtextOrNull builds a pgtype.Text that is NULL when the string is empty.
func pgtextOrNull(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: s, Valid: true}
}

// toFloat64 converts an any (from PostGIS ST_X/ST_Y) to float64.
func toFloat64(v any) float64 {
	if v == nil {
		return 0
	}
	switch val := v.(type) {
	case float64:
		return val
	case float32:
		return float64(val)
	default:
		return 0
	}
}
