-- ==========================================================================
-- Driver: Assignment
-- ==========================================================================

-- name: GetAssignmentByDriver :one
SELECT va.vehicle_id, v.plate_number,
       COALESCE(rt.name, '') AS route_name,
       COALESCE(col.full_name, '') AS collector_name,
       va.assigned_at
FROM vehicle_assignments va
JOIN vehicles v ON v.id = va.vehicle_id
LEFT JOIN routes rt ON rt.id = v.route_id
LEFT JOIN users col ON col.id = va.collector_id
WHERE va.driver_id = @driver_id AND va.active = true;

-- ==========================================================================
-- Driver: GPS position
-- ==========================================================================

-- name: UpsertPosition :exec
INSERT INTO vehicle_positions (vehicle_id, geom, heading, speed, recorded_at)
VALUES (@vehicle_id, ST_SetSRID(ST_MakePoint(sqlc.arg(lon)::float8, sqlc.arg(lat)::float8), 4326), @heading, @speed, NOW())
ON CONFLICT (vehicle_id)
DO UPDATE SET
    geom = ST_SetSRID(ST_MakePoint(sqlc.arg(lon)::float8, sqlc.arg(lat)::float8), 4326),
    heading = @heading,
    speed = @speed,
    recorded_at = NOW();

-- ==========================================================================
-- Driver: Trips
-- ==========================================================================

-- name: StartTrip :one
INSERT INTO trips (vehicle_id, route_id, driver_id, status)
VALUES (@vehicle_id, @route_id, @driver_id, 'active')
RETURNING id, vehicle_id, route_id, driver_id, started_at, ended_at, status;

-- name: GetVehicleRouteID :one
SELECT route_id FROM vehicles WHERE id = @id;

-- name: GetActiveTrip :one
SELECT id, vehicle_id, route_id, driver_id, started_at, ended_at, status
FROM trips
WHERE driver_id = @driver_id AND status = 'active'
ORDER BY started_at DESC
LIMIT 1;

-- name: EndTrip :execrows
UPDATE trips
SET status = 'completed', ended_at = NOW()
WHERE driver_id = @driver_id AND status = 'active';
