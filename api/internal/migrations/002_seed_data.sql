-- Migration: 002_seed_data
-- Synthetic fixtures: 8 stops + 2 routes in Lima metropolitan area (demo zone)
-- Coordinates are WGS84 (SRID 4326): lon, lat order for ST_Point

-- =====================
-- Routes
-- =====================

INSERT INTO routes (id, name, active) VALUES
  (1, 'Ruta A - Centro a Miraflores', true),
  (2, 'Ruta B - Miraflores a San Isidro', true)
ON CONFLICT DO NOTHING;

-- =====================
-- Stops
-- Coordinates around Lima (lat: ~-12.0, lon: ~-77.0)
-- =====================

INSERT INTO stops (id, name, geom, active) VALUES
  (1,  'Paradero Plaza Mayor',       ST_SetSRID(ST_Point(-77.0282, -12.0464), 4326), true),
  (2,  'Paradero Breña',             ST_SetSRID(ST_Point(-77.0450, -12.0580), 4326), true),
  (3,  'Paradero La Victoria',       ST_SetSRID(ST_Point(-77.0196, -12.0650), 4326), true),
  (4,  'Paradero San Borja Norte',   ST_SetSRID(ST_Point(-77.0050, -12.0870), 4326), true),
  (5,  'Paradero Miraflores Centro', ST_SetSRID(ST_Point(-77.0300, -12.1170), 4326), true),
  (6,  'Paradero Ovalo Gutierrez',   ST_SetSRID(ST_Point(-77.0350, -12.1050), 4326), true),
  (7,  'Paradero San Isidro',        ST_SetSRID(ST_Point(-77.0400, -12.0960), 4326), true),
  (8,  'Paradero Surquillo',         ST_SetSRID(ST_Point(-77.0270, -12.1080), 4326), true)
ON CONFLICT DO NOTHING;

-- Reset sequence to avoid PK conflicts on future inserts
SELECT setval('stops_id_seq',  (SELECT MAX(id) FROM stops));
SELECT setval('routes_id_seq', (SELECT MAX(id) FROM routes));

-- =====================
-- Route stops (stop sequences per route)
-- =====================

INSERT INTO route_stops (route_id, stop_id, sequence) VALUES
  -- Ruta A: Plaza Mayor -> Breña -> La Victoria -> San Borja -> Miraflores
  (1, 1, 1),
  (1, 2, 2),
  (1, 3, 3),
  (1, 4, 4),
  (1, 5, 5),
  -- Ruta B: Miraflores -> Ovalo Gutierrez -> San Isidro -> Surquillo
  (2, 5, 1),
  (2, 6, 2),
  (2, 7, 3),
  (2, 8, 4)
ON CONFLICT DO NOTHING;

-- =====================
-- Route shapes (simplified linestrings following the routes)
-- =====================

INSERT INTO route_shapes (route_id, geom) VALUES
  (1, ST_SetSRID(
    ST_MakeLine(ARRAY[
      ST_Point(-77.0282, -12.0464),
      ST_Point(-77.0450, -12.0580),
      ST_Point(-77.0196, -12.0650),
      ST_Point(-77.0050, -12.0870),
      ST_Point(-77.0300, -12.1170)
    ]), 4326)
  ),
  (2, ST_SetSRID(
    ST_MakeLine(ARRAY[
      ST_Point(-77.0300, -12.1170),
      ST_Point(-77.0350, -12.1050),
      ST_Point(-77.0400, -12.0960),
      ST_Point(-77.0270, -12.1080)
    ]), 4326)
  )
ON CONFLICT (route_id) DO NOTHING;
