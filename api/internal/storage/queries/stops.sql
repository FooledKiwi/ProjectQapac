-- name: FindStopsNear :many
SELECT id, name, ST_AsText(geom) AS geom
FROM stops
WHERE ST_DWithin(geom::geography, ST_SetSRID(ST_MakePoint(sqlc.arg(lon)::float8, sqlc.arg(lat)::float8), 4326)::geography, sqlc.arg(radius_m)::float8)
  AND active = true
ORDER BY ST_Distance(geom::geography, ST_SetSRID(ST_MakePoint(sqlc.arg(lon)::float8, sqlc.arg(lat)::float8), 4326)::geography);

-- name: GetStop :one
SELECT id, name, ST_AsText(geom) AS geom
FROM stops
WHERE id = sqlc.arg(id)::int AND active = true;

-- name: GetRouteShape :one
SELECT id, route_id, ST_AsText(geom) AS geom
FROM route_shapes
WHERE route_id = sqlc.arg(route_id)::int;
