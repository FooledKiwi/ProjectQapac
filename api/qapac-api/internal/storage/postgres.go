package storage

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/FooledKiwi/ProjectQapac/api/qapac-api/internal/generated/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// queryTimeout is applied to every database query.
const queryTimeout = 5 * time.Second

// pgStopsRepository is the pgx-backed implementation of StopsRepository.
type pgStopsRepository struct {
	q *db.Queries
}

// NewStopsRepository creates a StopsRepository backed by the given connection pool.
func NewStopsRepository(pool *pgxpool.Pool) StopsRepository {
	return &pgStopsRepository{q: db.New(pool)}
}

// FindStopsNear returns active stops within radiusMeters of (lat, lon).
func (r *pgStopsRepository) FindStopsNear(ctx context.Context, lat, lon, radiusMeters float64) ([]Stop, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	rows, err := r.q.FindStopsNear(ctx, db.FindStopsNearParams{
		Lat:     lat,
		Lon:     lon,
		RadiusM: radiusMeters,
	})
	if err != nil {
		return nil, fmt.Errorf("storage: FindStopsNear: %w", err)
	}

	stops := make([]Stop, 0, len(rows))
	for _, row := range rows {
		s, err := rowToStop(row.ID, row.Name, row.Geom)
		if err != nil {
			return nil, fmt.Errorf("storage: FindStopsNear: parse geometry: %w", err)
		}
		stops = append(stops, s)
	}

	return stops, nil
}

// GetStop returns a single active stop by ID, or (nil, nil) if not found.
func (r *pgStopsRepository) GetStop(ctx context.Context, id int32) (*Stop, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	row, err := r.q.GetStop(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("storage: GetStop: %w", err)
	}

	s, err := rowToStop(row.ID, row.Name, row.Geom)
	if err != nil {
		return nil, fmt.Errorf("storage: GetStop: parse geometry: %w", err)
	}

	return &s, nil
}

// pgRoutesRepository is the pgx-backed implementation of RoutesRepository.
type pgRoutesRepository struct {
	q *db.Queries
}

// NewRoutesRepository creates a RoutesRepository backed by the given connection pool.
func NewRoutesRepository(pool *pgxpool.Pool) RoutesRepository {
	return &pgRoutesRepository{q: db.New(pool)}
}

// GetRouteShape returns the shape for routeID, or (nil, nil) if not found.
func (r *pgRoutesRepository) GetRouteShape(ctx context.Context, routeID int32) (*RouteShape, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	row, err := r.q.GetRouteShape(ctx, routeID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("storage: GetRouteShape: %w", err)
	}

	geomWKT, ok := row.Geom.(string)
	if row.Geom == nil {
		return nil, fmt.Errorf("storage: GetRouteShape: route_id=%d has NULL geometry (data integrity issue)", routeID)
	}
	if !ok {
		return nil, fmt.Errorf("storage: GetRouteShape: unexpected geom type %T", row.Geom)
	}

	return &RouteShape{
		ID:      row.ID,
		RouteID: row.RouteID,
		GeomWKT: geomWKT,
	}, nil
}

// rowToStop converts a raw query row into a Stop domain object.
// geom must be a WKT POINT string produced by ST_AsText, e.g. "POINT(lon lat)".
func rowToStop(id int32, name string, geom interface{}) (Stop, error) {
	if geom == nil {
		return Stop{}, fmt.Errorf("stop id=%d has NULL geometry (data integrity issue)", id)
	}
	wkt, ok := geom.(string)
	if !ok {
		return Stop{}, fmt.Errorf("unexpected geom type %T", geom)
	}

	lat, lon, err := parsePointWKT(wkt)
	if err != nil {
		return Stop{}, err
	}

	return Stop{
		ID:   id,
		Name: name,
		Lat:  lat,
		Lon:  lon,
	}, nil
}

// parsePointWKT parses a WKT POINT string into (lat, lon).
// PostGIS ST_AsText(GEOMETRY(POINT, 4326)) returns "POINT(lon lat)".
func parsePointWKT(wkt string) (lat, lon float64, err error) {
	// Expected format: "POINT(lon lat)"
	wkt = strings.TrimSpace(wkt)
	if !strings.HasPrefix(wkt, "POINT(") || !strings.HasSuffix(wkt, ")") {
		return 0, 0, fmt.Errorf("unexpected WKT format: %q", wkt)
	}

	inner := wkt[len("POINT(") : len(wkt)-1]
	parts := strings.Fields(inner)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("unexpected WKT coordinates: %q", inner)
	}

	lon, err = strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return 0, 0, fmt.Errorf("parse lon %q: %w", parts[0], err)
	}

	lat, err = strconv.ParseFloat(parts[1], 64)
	if err != nil {
		return 0, 0, fmt.Errorf("parse lat %q: %w", parts[1], err)
	}

	return lat, lon, nil
}
