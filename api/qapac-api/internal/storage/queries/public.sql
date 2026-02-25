-- ==========================================================================
-- Public: Routes
-- ==========================================================================

-- name: PublicListRoutes :many
SELECT r.id, r.name, r.active,
       COUNT(v.id) FILTER (WHERE v.status = 'active')::int AS vehicle_count
FROM routes r
LEFT JOIN vehicles v ON v.route_id = r.id
WHERE r.active = true
GROUP BY r.id
ORDER BY r.name;

-- name: PublicGetRouteByID :one
SELECT id, name, active FROM routes WHERE id = @id;

-- name: PublicGetRouteStopsOrdered :many
SELECT s.id, s.name, ST_AsText(s.geom) AS geom, rs.sequence
FROM route_stops rs
JOIN stops s ON s.id = rs.stop_id
WHERE rs.route_id = @route_id
ORDER BY rs.sequence;

-- name: PublicGetRouteVehicles :many
SELECT v.id, v.plate_number, v.status,
       COALESCE(d.full_name, '') AS driver_name,
       COALESCE(col.full_name, '') AS collector_name
FROM vehicles v
LEFT JOIN vehicle_assignments va ON va.vehicle_id = v.id AND va.active = true
LEFT JOIN users d ON d.id = va.driver_id
LEFT JOIN users col ON col.id = va.collector_id
WHERE v.route_id = @route_id
ORDER BY v.plate_number;

-- name: PublicGetRouteShapeWKT :one
SELECT ST_AsText(geom) AS geom_wkt FROM route_shapes WHERE route_id = @route_id;

-- name: PublicGetRouteVehiclesWithPositions :many
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
WHERE v.route_id = @route_id AND v.status = 'active'
ORDER BY v.plate_number;

-- ==========================================================================
-- Public: Vehicle positions
-- ==========================================================================

-- name: GetVehiclePosition :one
SELECT ST_AsText(geom) AS geom_wkt, heading, speed, recorded_at
FROM vehicle_positions
WHERE vehicle_id = @vehicle_id;

-- name: FindNearbyVehicles :many
SELECT v.id, v.plate_number, COALESCE(rt.name, '') AS route_name,
       ST_Y(vp.geom) AS lat, ST_X(vp.geom) AS lon
FROM vehicle_positions vp
JOIN vehicles v ON v.id = vp.vehicle_id
LEFT JOIN routes rt ON rt.id = v.route_id
WHERE v.status = 'active'
  AND ST_DWithin(vp.geom::geography, ST_SetSRID(ST_MakePoint(sqlc.arg(lon)::float8, sqlc.arg(lat)::float8), 4326)::geography, sqlc.arg(radius_m)::float8)
ORDER BY vp.geom::geography <-> ST_SetSRID(ST_MakePoint(sqlc.arg(lon)::float8, sqlc.arg(lat)::float8), 4326)::geography;

-- ==========================================================================
-- Public: Ratings
-- ==========================================================================

-- name: CreateRating :one
INSERT INTO ratings (trip_id, rating, device_id)
VALUES (@trip_id, @rating, @device_id)
RETURNING id, trip_id, rating, device_id, created_at;

-- ==========================================================================
-- Public: Favorites
-- ==========================================================================

-- name: ListFavoritesByDevice :many
SELECT f.id, f.device_id, f.route_id, COALESCE(rt.name, '') AS route_name, f.created_at
FROM favorites f
LEFT JOIN routes rt ON rt.id = f.route_id
WHERE f.device_id = @device_id
ORDER BY f.created_at DESC;

-- name: AddFavorite :one
INSERT INTO favorites (device_id, route_id)
VALUES (@device_id, @route_id)
ON CONFLICT (device_id, route_id) DO UPDATE SET device_id = EXCLUDED.device_id
RETURNING id, device_id, route_id, created_at;

-- name: RemoveFavorite :exec
DELETE FROM favorites WHERE device_id = @device_id AND route_id = @route_id;
