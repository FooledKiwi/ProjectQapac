package storage

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ---------------------------------------------------------------------------
// PublicRoutesRepository
// ---------------------------------------------------------------------------

type pgPublicRoutesRepository struct {
	pool *pgxpool.Pool
}

// NewPublicRoutesRepository creates a PublicRoutesRepository backed by the given pool.
func NewPublicRoutesRepository(pool *pgxpool.Pool) PublicRoutesRepository {
	return &pgPublicRoutesRepository{pool: pool}
}

func (r *pgPublicRoutesRepository) ListRoutes(ctx context.Context) ([]Route, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	rows, err := r.pool.Query(ctx, `
		SELECT r.id, r.name, r.active,
		       COUNT(v.id) FILTER (WHERE v.status = 'active') AS vehicle_count
		FROM routes r
		LEFT JOIN vehicles v ON v.route_id = r.id
		WHERE r.active = true
		GROUP BY r.id
		ORDER BY r.name`)
	if err != nil {
		return nil, fmt.Errorf("storage: ListRoutes: %w", err)
	}
	defer rows.Close()

	var routes []Route
	for rows.Next() {
		var rt Route
		if err := rows.Scan(&rt.ID, &rt.Name, &rt.Active, &rt.VehicleCount); err != nil {
			return nil, fmt.Errorf("storage: ListRoutes: scan: %w", err)
		}
		routes = append(routes, rt)
	}
	return routes, rows.Err()
}

func (r *pgPublicRoutesRepository) GetRouteDetail(ctx context.Context, id int32) (*RouteDetail, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	// Fetch route.
	var rd RouteDetail
	var active bool
	err := r.pool.QueryRow(ctx,
		`SELECT id, name, active FROM routes WHERE id = $1`, id,
	).Scan(&rd.ID, &rd.Name, &active)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("storage: GetRouteDetail: route: %w", err)
	}
	rd.Active = active

	// Fetch stops in order.
	stopRows, err := r.pool.Query(ctx, `
		SELECT s.id, s.name, ST_AsText(s.geom) AS geom, rs.sequence
		FROM route_stops rs
		JOIN stops s ON s.id = rs.stop_id
		WHERE rs.route_id = $1
		ORDER BY rs.sequence`, id)
	if err != nil {
		return nil, fmt.Errorf("storage: GetRouteDetail: stops: %w", err)
	}
	defer stopRows.Close()

	for stopRows.Next() {
		var rs RouteStop
		var geomWKT string
		if err = stopRows.Scan(&rs.ID, &rs.Name, &geomWKT, &rs.Sequence); err != nil {
			return nil, fmt.Errorf("storage: GetRouteDetail: stops scan: %w", err)
		}
		var lat, lon float64
		lat, lon, err = parsePointWKT(geomWKT)
		if err != nil {
			return nil, fmt.Errorf("storage: GetRouteDetail: parse stop geom: %w", err)
		}
		rs.Lat = lat
		rs.Lon = lon
		rd.Stops = append(rd.Stops, rs)
	}
	if err = stopRows.Err(); err != nil {
		return nil, fmt.Errorf("storage: GetRouteDetail: stops rows: %w", err)
	}

	// Fetch assigned vehicles with driver/collector names.
	vehRows, err := r.pool.Query(ctx, `
		SELECT v.id, v.plate_number, v.status,
		       COALESCE(d.full_name, '') AS driver_name,
		       COALESCE(col.full_name, '') AS collector_name
		FROM vehicles v
		LEFT JOIN vehicle_assignments va ON va.vehicle_id = v.id AND va.active = true
		LEFT JOIN users d ON d.id = va.driver_id
		LEFT JOIN users col ON col.id = va.collector_id
		WHERE v.route_id = $1
		ORDER BY v.plate_number`, id)
	if err != nil {
		return nil, fmt.Errorf("storage: GetRouteDetail: vehicles: %w", err)
	}
	defer vehRows.Close()

	for vehRows.Next() {
		var rv RouteVehicle
		if err = vehRows.Scan(&rv.ID, &rv.PlateNumber, &rv.Status, &rv.DriverName, &rv.CollectorName); err != nil {
			return nil, fmt.Errorf("storage: GetRouteDetail: vehicles scan: %w", err)
		}
		rd.Vehicles = append(rd.Vehicles, rv)
	}
	if err = vehRows.Err(); err != nil {
		return nil, fmt.Errorf("storage: GetRouteDetail: vehicles rows: %w", err)
	}

	// Fetch route shape polyline.
	var geom any
	err = r.pool.QueryRow(ctx,
		`SELECT ST_AsText(geom) FROM route_shapes WHERE route_id = $1`, id,
	).Scan(&geom)
	if err == nil {
		if wkt, ok := geom.(string); ok {
			rd.ShapePolyline = wkt
		}
	}
	// If no shape found, leave ShapePolyline empty — not an error.

	return &rd, nil
}

func (r *pgPublicRoutesRepository) GetRouteVehiclesWithPositions(ctx context.Context, routeID int32) ([]RouteVehicleWithPosition, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	rows, err := r.pool.Query(ctx, `
		SELECT v.id, v.plate_number, v.status,
		       COALESCE(d.full_name, '') AS driver_name,
		       COALESCE(col.full_name, '') AS collector_name,
		       ST_AsText(vp.geom) AS pos_geom,
		       vp.heading, vp.speed, vp.recorded_at
		FROM vehicles v
		LEFT JOIN vehicle_assignments va ON va.vehicle_id = v.id AND va.active = true
		LEFT JOIN users d ON d.id = va.driver_id
		LEFT JOIN users col ON col.id = va.collector_id
		LEFT JOIN vehicle_positions vp ON vp.vehicle_id = v.id
		WHERE v.route_id = $1 AND v.status = 'active'
		ORDER BY v.plate_number`, routeID)
	if err != nil {
		return nil, fmt.Errorf("storage: GetRouteVehiclesWithPositions: %w", err)
	}
	defer rows.Close()

	var results []RouteVehicleWithPosition
	for rows.Next() {
		var rv RouteVehicleWithPosition
		var posGeom *string
		var heading, speed *float64
		var recordedAt *time.Time

		if err := rows.Scan(
			&rv.ID, &rv.PlateNumber, &rv.Status,
			&rv.DriverName, &rv.CollectorName,
			&posGeom, &heading, &speed, &recordedAt,
		); err != nil {
			return nil, fmt.Errorf("storage: GetRouteVehiclesWithPositions: scan: %w", err)
		}

		if posGeom != nil {
			lat, lon, err := parsePointWKT(*posGeom)
			if err == nil {
				rv.Position = &VehiclePosition{
					VehicleID:  rv.ID,
					Lat:        lat,
					Lon:        lon,
					Heading:    heading,
					Speed:      speed,
					RecordedAt: *recordedAt,
				}
			}
		}

		results = append(results, rv)
	}
	return results, rows.Err()
}

// ---------------------------------------------------------------------------
// VehiclePositionsRepository
// ---------------------------------------------------------------------------

type pgVehiclePositionsRepository struct {
	pool *pgxpool.Pool
}

// NewVehiclePositionsRepository creates a VehiclePositionsRepository backed by the given pool.
func NewVehiclePositionsRepository(pool *pgxpool.Pool) VehiclePositionsRepository {
	return &pgVehiclePositionsRepository{pool: pool}
}

func (r *pgVehiclePositionsRepository) GetPosition(ctx context.Context, vehicleID int32) (*VehiclePosition, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	var geomWKT string
	vp := &VehiclePosition{VehicleID: vehicleID}
	err := r.pool.QueryRow(ctx, `
		SELECT ST_AsText(geom), heading, speed, recorded_at
		FROM vehicle_positions
		WHERE vehicle_id = $1`, vehicleID,
	).Scan(&geomWKT, &vp.Heading, &vp.Speed, &vp.RecordedAt)

	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("storage: GetPosition: %w", err)
	}

	lat, lon, err := parsePointWKT(geomWKT)
	if err != nil {
		return nil, fmt.Errorf("storage: GetPosition: parse geom: %w", err)
	}
	vp.Lat = lat
	vp.Lon = lon
	return vp, nil
}

func (r *pgVehiclePositionsRepository) FindNearby(ctx context.Context, lat, lon, radiusMeters float64) ([]NearbyVehicle, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	rows, err := r.pool.Query(ctx, `
		SELECT v.id, v.plate_number, COALESCE(rt.name, '') AS route_name,
		       ST_Y(vp.geom) AS lat, ST_X(vp.geom) AS lon
		FROM vehicle_positions vp
		JOIN vehicles v ON v.id = vp.vehicle_id
		LEFT JOIN routes rt ON rt.id = v.route_id
		WHERE v.status = 'active'
		  AND ST_DWithin(vp.geom::geography, ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography, $3)
		ORDER BY vp.geom::geography <-> ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography`,
		lon, lat, radiusMeters)
	if err != nil {
		return nil, fmt.Errorf("storage: FindNearby: %w", err)
	}
	defer rows.Close()

	var vehicles []NearbyVehicle
	for rows.Next() {
		var nv NearbyVehicle
		if err := rows.Scan(&nv.ID, &nv.PlateNumber, &nv.RouteName, &nv.Lat, &nv.Lon); err != nil {
			return nil, fmt.Errorf("storage: FindNearby: scan: %w", err)
		}
		vehicles = append(vehicles, nv)
	}
	return vehicles, rows.Err()
}

// ---------------------------------------------------------------------------
// RatingsRepository
// ---------------------------------------------------------------------------

type pgRatingsRepository struct {
	pool *pgxpool.Pool
}

// NewRatingsRepository creates a RatingsRepository backed by the given pool.
func NewRatingsRepository(pool *pgxpool.Pool) RatingsRepository {
	return &pgRatingsRepository{pool: pool}
}

func (r *pgRatingsRepository) CreateRating(ctx context.Context, rt *Rating) (*Rating, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	var id int32
	var createdAt time.Time
	err := r.pool.QueryRow(ctx, `
		INSERT INTO ratings (trip_id, rating, device_id)
		VALUES ($1, $2, $3)
		RETURNING id, created_at`,
		rt.TripID, rt.Rating, rt.DeviceID,
	).Scan(&id, &createdAt)
	if err != nil {
		// Check for unique constraint violation (device already rated this trip).
		if strings.Contains(err.Error(), "duplicate key") ||
			strings.Contains(err.Error(), "unique constraint") {
			return nil, fmt.Errorf("storage: CreateRating: already rated: %w", err)
		}
		return nil, fmt.Errorf("storage: CreateRating: %w", err)
	}

	rt.ID = id
	rt.CreatedAt = createdAt
	return rt, nil
}

// ---------------------------------------------------------------------------
// FavoritesRepository
// ---------------------------------------------------------------------------

type pgFavoritesRepository struct {
	pool *pgxpool.Pool
}

// NewFavoritesRepository creates a FavoritesRepository backed by the given pool.
func NewFavoritesRepository(pool *pgxpool.Pool) FavoritesRepository {
	return &pgFavoritesRepository{pool: pool}
}

func (r *pgFavoritesRepository) ListByDevice(ctx context.Context, deviceID string) ([]Favorite, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	rows, err := r.pool.Query(ctx, `
		SELECT f.id, f.device_id, f.route_id, COALESCE(rt.name, '') AS route_name, f.created_at
		FROM favorites f
		LEFT JOIN routes rt ON rt.id = f.route_id
		WHERE f.device_id = $1
		ORDER BY f.created_at DESC`, deviceID)
	if err != nil {
		return nil, fmt.Errorf("storage: ListByDevice: %w", err)
	}
	defer rows.Close()

	var favs []Favorite
	for rows.Next() {
		var f Favorite
		if err := rows.Scan(&f.ID, &f.DeviceID, &f.RouteID, &f.RouteName, &f.CreatedAt); err != nil {
			return nil, fmt.Errorf("storage: ListByDevice: scan: %w", err)
		}
		favs = append(favs, f)
	}
	return favs, rows.Err()
}

func (r *pgFavoritesRepository) Add(ctx context.Context, deviceID string, routeID int32) (*Favorite, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	var f Favorite
	err := r.pool.QueryRow(ctx, `
		INSERT INTO favorites (device_id, route_id)
		VALUES ($1, $2)
		ON CONFLICT (device_id, route_id) DO UPDATE SET device_id = EXCLUDED.device_id
		RETURNING id, device_id, route_id, created_at`,
		deviceID, routeID,
	).Scan(&f.ID, &f.DeviceID, &f.RouteID, &f.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("storage: AddFavorite: %w", err)
	}
	return &f, nil
}

func (r *pgFavoritesRepository) Remove(ctx context.Context, deviceID string, routeID int32) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	_, err := r.pool.Exec(ctx,
		`DELETE FROM favorites WHERE device_id = $1 AND route_id = $2`,
		deviceID, routeID)
	if err != nil {
		return fmt.Errorf("storage: RemoveFavorite: %w", err)
	}
	return nil
}
