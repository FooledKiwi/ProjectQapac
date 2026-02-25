package storage

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/FooledKiwi/ProjectQapac/api/qapac-api/internal/generated/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ---------------------------------------------------------------------------
// PublicRoutesRepository
// ---------------------------------------------------------------------------

type pgPublicRoutesRepository struct {
	q *db.Queries
}

// NewPublicRoutesRepository creates a PublicRoutesRepository backed by the given pool.
func NewPublicRoutesRepository(pool *pgxpool.Pool) PublicRoutesRepository {
	return &pgPublicRoutesRepository{q: db.New(pool)}
}

func (r *pgPublicRoutesRepository) ListRoutes(ctx context.Context) ([]Route, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	rows, err := r.q.PublicListRoutes(ctx)
	if err != nil {
		return nil, fmt.Errorf("storage: ListRoutes: %w", err)
	}

	routes := make([]Route, 0, len(rows))
	for _, row := range rows {
		routes = append(routes, Route{
			ID:           row.ID,
			Name:         row.Name,
			Active:       row.Active.Bool,
			VehicleCount: int(row.VehicleCount),
		})
	}
	return routes, nil
}

func (r *pgPublicRoutesRepository) GetRouteDetail(ctx context.Context, id int32) (*RouteDetail, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	// Fetch route.
	route, err := r.q.PublicGetRouteByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("storage: GetRouteDetail: route: %w", err)
	}

	rd := &RouteDetail{
		ID:     route.ID,
		Name:   route.Name,
		Active: route.Active.Bool,
	}

	// Fetch stops in order.
	stopRows, err := r.q.PublicGetRouteStopsOrdered(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("storage: GetRouteDetail: stops: %w", err)
	}
	for _, sr := range stopRows {
		geomWKT, ok := sr.Geom.(string)
		if !ok || geomWKT == "" {
			continue
		}
		lat, lon, parseErr := parsePointWKT(geomWKT)
		if parseErr != nil {
			return nil, fmt.Errorf("storage: GetRouteDetail: parse stop geom: %w", parseErr)
		}
		rd.Stops = append(rd.Stops, RouteStop{
			ID:       sr.ID,
			Name:     sr.Name,
			Lat:      lat,
			Lon:      lon,
			Sequence: sr.Sequence,
		})
	}

	// Fetch assigned vehicles with driver/collector names.
	routeIDParam := pgtype.Int4{Int32: id, Valid: true}
	vehRows, err := r.q.PublicGetRouteVehicles(ctx, routeIDParam)
	if err != nil {
		return nil, fmt.Errorf("storage: GetRouteDetail: vehicles: %w", err)
	}
	for _, vr := range vehRows {
		rd.Vehicles = append(rd.Vehicles, RouteVehicle{
			ID:            vr.ID,
			PlateNumber:   vr.PlateNumber,
			Status:        vr.Status.String,
			DriverName:    vr.DriverName,
			CollectorName: vr.CollectorName,
		})
	}

	// Fetch route shape polyline.
	geomWKT, err := r.q.PublicGetRouteShapeWKT(ctx, id)
	if err == nil {
		if wkt, ok := geomWKT.(string); ok {
			rd.ShapePolyline = wkt
		}
	}
	// If no shape found, leave ShapePolyline empty — not an error.

	return rd, nil
}

func (r *pgPublicRoutesRepository) GetRouteVehiclesWithPositions(ctx context.Context, routeID int32) ([]RouteVehicleWithPosition, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	routeIDParam := pgtype.Int4{Int32: routeID, Valid: true}
	rows, err := r.q.PublicGetRouteVehiclesWithPositions(ctx, routeIDParam)
	if err != nil {
		return nil, fmt.Errorf("storage: GetRouteVehiclesWithPositions: %w", err)
	}

	results := make([]RouteVehicleWithPosition, 0, len(rows))
	for _, row := range rows {
		rv := RouteVehicleWithPosition{
			ID:            row.ID,
			PlateNumber:   row.PlateNumber,
			Status:        row.Status.String,
			DriverName:    row.DriverName,
			CollectorName: row.CollectorName,
		}

		if posGeom, ok := row.PosGeom.(string); ok && posGeom != "" {
			lat, lon, parseErr := parsePointWKT(posGeom)
			if parseErr == nil {
				var heading, speed *float64
				if row.Heading.Valid {
					heading = &row.Heading.Float64
				}
				if row.Speed.Valid {
					speed = &row.Speed.Float64
				}
				rv.Position = &VehiclePosition{
					VehicleID:  rv.ID,
					Lat:        lat,
					Lon:        lon,
					Heading:    heading,
					Speed:      speed,
					RecordedAt: row.RecordedAt.Time,
				}
			}
		}

		results = append(results, rv)
	}
	return results, nil
}

// ---------------------------------------------------------------------------
// VehiclePositionsRepository
// ---------------------------------------------------------------------------

type pgVehiclePositionsRepository struct {
	q *db.Queries
}

// NewVehiclePositionsRepository creates a VehiclePositionsRepository backed by the given pool.
func NewVehiclePositionsRepository(pool *pgxpool.Pool) VehiclePositionsRepository {
	return &pgVehiclePositionsRepository{q: db.New(pool)}
}

func (r *pgVehiclePositionsRepository) GetPosition(ctx context.Context, vehicleID int32) (*VehiclePosition, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	row, err := r.q.GetVehiclePosition(ctx, vehicleID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("storage: GetPosition: %w", err)
	}

	geomWKT, ok := row.GeomWkt.(string)
	if !ok || geomWKT == "" {
		return nil, fmt.Errorf("storage: GetPosition: unexpected geom type")
	}

	lat, lon, parseErr := parsePointWKT(geomWKT)
	if parseErr != nil {
		return nil, fmt.Errorf("storage: GetPosition: parse geom: %w", parseErr)
	}

	var heading, speed *float64
	if row.Heading.Valid {
		heading = &row.Heading.Float64
	}
	if row.Speed.Valid {
		speed = &row.Speed.Float64
	}

	return &VehiclePosition{
		VehicleID:  vehicleID,
		Lat:        lat,
		Lon:        lon,
		Heading:    heading,
		Speed:      speed,
		RecordedAt: row.RecordedAt.Time,
	}, nil
}

func (r *pgVehiclePositionsRepository) FindNearby(ctx context.Context, lat, lon, radiusMeters float64) ([]NearbyVehicle, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	rows, err := r.q.FindNearbyVehicles(ctx, db.FindNearbyVehiclesParams{
		Lon:     lon,
		Lat:     lat,
		RadiusM: radiusMeters,
	})
	if err != nil {
		return nil, fmt.Errorf("storage: FindNearby: %w", err)
	}

	vehicles := make([]NearbyVehicle, 0, len(rows))
	for _, row := range rows {
		vehicles = append(vehicles, NearbyVehicle{
			ID:          row.ID,
			PlateNumber: row.PlateNumber,
			RouteName:   row.RouteName,
			Lat:         toFloat64(row.Lat),
			Lon:         toFloat64(row.Lon),
		})
	}
	return vehicles, nil
}

// ---------------------------------------------------------------------------
// RatingsRepository
// ---------------------------------------------------------------------------

type pgRatingsRepository struct {
	q *db.Queries
}

// NewRatingsRepository creates a RatingsRepository backed by the given pool.
func NewRatingsRepository(pool *pgxpool.Pool) RatingsRepository {
	return &pgRatingsRepository{q: db.New(pool)}
}

func (r *pgRatingsRepository) CreateRating(ctx context.Context, rt *Rating) (*Rating, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	row, err := r.q.CreateRating(ctx, db.CreateRatingParams{
		TripID:   rt.TripID,
		Rating:   rt.Rating,
		DeviceID: rt.DeviceID,
	})
	if err != nil {
		// Check for unique constraint violation (device already rated this trip).
		if strings.Contains(err.Error(), "duplicate key") ||
			strings.Contains(err.Error(), "unique constraint") {
			return nil, fmt.Errorf("storage: CreateRating: already rated: %w", err)
		}
		return nil, fmt.Errorf("storage: CreateRating: %w", err)
	}

	rt.ID = row.ID
	rt.CreatedAt = row.CreatedAt.Time
	return rt, nil
}

// ---------------------------------------------------------------------------
// FavoritesRepository
// ---------------------------------------------------------------------------

type pgFavoritesRepository struct {
	q *db.Queries
}

// NewFavoritesRepository creates a FavoritesRepository backed by the given pool.
func NewFavoritesRepository(pool *pgxpool.Pool) FavoritesRepository {
	return &pgFavoritesRepository{q: db.New(pool)}
}

func (r *pgFavoritesRepository) ListByDevice(ctx context.Context, deviceID string) ([]Favorite, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	rows, err := r.q.ListFavoritesByDevice(ctx, deviceID)
	if err != nil {
		return nil, fmt.Errorf("storage: ListByDevice: %w", err)
	}

	favs := make([]Favorite, 0, len(rows))
	for _, row := range rows {
		favs = append(favs, Favorite{
			ID:        row.ID,
			DeviceID:  row.DeviceID,
			RouteID:   row.RouteID,
			RouteName: row.RouteName,
			CreatedAt: row.CreatedAt.Time,
		})
	}
	return favs, nil
}

func (r *pgFavoritesRepository) Add(ctx context.Context, deviceID string, routeID int32) (*Favorite, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	row, err := r.q.AddFavorite(ctx, db.AddFavoriteParams{
		DeviceID: deviceID,
		RouteID:  routeID,
	})
	if err != nil {
		return nil, fmt.Errorf("storage: AddFavorite: %w", err)
	}

	return &Favorite{
		ID:        row.ID,
		DeviceID:  row.DeviceID,
		RouteID:   row.RouteID,
		CreatedAt: row.CreatedAt.Time,
	}, nil
}

func (r *pgFavoritesRepository) Remove(ctx context.Context, deviceID string, routeID int32) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	err := r.q.RemoveFavorite(ctx, db.RemoveFavoriteParams{
		DeviceID: deviceID,
		RouteID:  routeID,
	})
	if err != nil {
		return fmt.Errorf("storage: RemoveFavorite: %w", err)
	}
	return nil
}
