-- ==========================================================================
-- Admin: Vehicles
-- ==========================================================================

-- name: CreateVehicle :one
INSERT INTO vehicles (plate_number, route_id, status)
VALUES (@plate_number, @route_id, @status)
RETURNING id, plate_number, route_id, status, created_at;

-- name: GetVehicleByID :one
SELECT id, plate_number, route_id, status, created_at
FROM vehicles
WHERE id = @id;

-- name: ListVehicles :many
SELECT id, plate_number, route_id, status, created_at
FROM vehicles
WHERE (sqlc.narg(route_id)::int IS NULL OR route_id = sqlc.narg(route_id)::int)
  AND (sqlc.narg(status)::text IS NULL OR status = sqlc.narg(status)::text)
ORDER BY created_at DESC;

-- name: UpdateVehicle :exec
UPDATE vehicles SET plate_number = @plate_number, route_id = @route_id, status = @status
WHERE id = @id;

-- name: DeactivateOldAssignment :exec
UPDATE vehicle_assignments SET active = false WHERE vehicle_id = @vehicle_id AND active = true;

-- name: InsertAssignment :one
INSERT INTO vehicle_assignments (vehicle_id, driver_id, collector_id, active)
VALUES (@vehicle_id, @driver_id, @collector_id, true)
RETURNING id, vehicle_id, driver_id, collector_id, assigned_at, active;

-- name: GetActiveAssignment :one
SELECT id, vehicle_id, driver_id, collector_id, assigned_at, active
FROM vehicle_assignments
WHERE vehicle_id = @vehicle_id AND active = true;

-- ==========================================================================
-- Admin: Alerts
-- ==========================================================================

-- name: CreateAlert :one
INSERT INTO alerts (title, description, route_id, vehicle_plate, image_path, created_by)
VALUES (@title, @description, @route_id, @vehicle_plate, @image_path, @created_by)
RETURNING id, title, COALESCE(description, '') AS description, route_id,
          COALESCE(vehicle_plate, '') AS vehicle_plate,
          COALESCE(image_path, '') AS image_path, created_by, created_at;

-- name: GetAlertByID :one
SELECT id, title, COALESCE(description, '') AS description, route_id,
       COALESCE(vehicle_plate, '') AS vehicle_plate,
       COALESCE(image_path, '') AS image_path, created_by, created_at
FROM alerts
WHERE id = @id;

-- name: ListAlerts :many
SELECT id, title, COALESCE(description, '') AS description, route_id,
       COALESCE(vehicle_plate, '') AS vehicle_plate,
       COALESCE(image_path, '') AS image_path, created_by, created_at
FROM alerts
WHERE (sqlc.narg(route_id)::int IS NULL OR route_id = sqlc.narg(route_id)::int)
ORDER BY created_at DESC;

-- name: DeleteAlert :exec
DELETE FROM alerts WHERE id = @id;

-- ==========================================================================
-- Admin: Stops
-- ==========================================================================

-- name: AdminCreateStop :one
INSERT INTO stops (name, geom, active)
VALUES (@name, ST_SetSRID(ST_MakePoint(sqlc.arg(lon)::float8, sqlc.arg(lat)::float8), 4326), true)
RETURNING id, name, ST_X(geom) AS lon, ST_Y(geom) AS lat, active, created_at;

-- name: AdminGetStopByID :one
SELECT id, name, ST_X(geom) AS lon, ST_Y(geom) AS lat, active, created_at
FROM stops
WHERE id = @id;

-- name: AdminListStops :many
SELECT id, name, ST_X(geom) AS lon, ST_Y(geom) AS lat, active, created_at
FROM stops
WHERE (sqlc.arg(active_only)::bool = false OR active = true)
ORDER BY id ASC;

-- name: AdminUpdateStop :exec
UPDATE stops
SET name = @name, geom = ST_SetSRID(ST_MakePoint(sqlc.arg(lon)::float8, sqlc.arg(lat)::float8), 4326), active = @active
WHERE id = @id;

-- name: AdminDeactivateStop :exec
UPDATE stops SET active = false WHERE id = @id;

-- ==========================================================================
-- Admin: Routes
-- ==========================================================================

-- name: AdminCreateRoute :one
INSERT INTO routes (name, active) VALUES (@name, true)
RETURNING id, name, active;

-- name: AdminGetRouteByID :one
SELECT id, name, active FROM routes WHERE id = @id;

-- name: AdminListRoutes :many
SELECT id, name, active FROM routes
WHERE (sqlc.arg(active_only)::bool = false OR active = true)
ORDER BY id ASC;

-- name: AdminUpdateRoute :exec
UPDATE routes SET name = @name, active = @active WHERE id = @id;

-- name: AdminDeactivateRoute :exec
UPDATE routes SET active = false WHERE id = @id;

-- name: AdminGetRouteStops :many
SELECT stop_id, sequence FROM route_stops
WHERE route_id = @route_id
ORDER BY sequence ASC;

-- name: AdminDeleteRouteStops :exec
DELETE FROM route_stops WHERE route_id = @route_id;

-- name: AdminInsertRouteStop :exec
INSERT INTO route_stops (route_id, stop_id, sequence) VALUES (@route_id, @stop_id, @sequence);

-- name: AdminGetRouteShapeWKT :one
SELECT ST_AsText(geom) AS geom_wkt FROM route_shapes WHERE route_id = @route_id;

-- name: AdminUpsertRouteShape :exec
INSERT INTO route_shapes (route_id, geom, updated_at)
VALUES (@route_id, ST_GeomFromText(@geom_wkt, 4326), NOW())
ON CONFLICT (route_id)
DO UPDATE SET geom = ST_GeomFromText(@geom_wkt, 4326), updated_at = NOW();
