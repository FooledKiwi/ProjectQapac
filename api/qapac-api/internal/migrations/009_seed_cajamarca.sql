-- Migration: 009_seed_cajamarca
-- Replaces Lima demo data with realistic Cajamarca bus transit seed data.
-- Coordinates are WGS84 (SRID 4326): ST_Point(longitude, latitude).
--
-- Test credentials (bcrypt cost 10):
--   admin    / admin123
--   conductor1 / driver123
--   conductor2 / driver123

-- =====================
-- 1. Remove Lima seed data
-- =====================

-- Dependent tables first (FK order).
DELETE FROM route_to_stop_cache;
DELETE FROM stop_eta_cache;
DELETE FROM route_shapes  WHERE route_id IN (SELECT id FROM routes WHERE id <= 2);
DELETE FROM route_stops   WHERE route_id IN (SELECT id FROM routes WHERE id <= 2);
DELETE FROM routes        WHERE id <= 2;
DELETE FROM stops         WHERE id <= 8;

-- =====================
-- 2. Routes (5 plausible Cajamarca combi routes)
-- =====================

INSERT INTO routes (id, name, active) VALUES
  (1, 'Ruta 1 - Centro a Baños del Inca',       true),
  (2, 'Ruta 2 - Mollepampa a Universidad',       true),
  (3, 'Ruta 3 - Centro a Hoyos Rubio',           true),
  (4, 'Ruta 4 - San Sebastián a Lucmacucho',     true),
  (5, 'Ruta 5 - Terminal Terrestre a El Ingenio', true)
ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, active = EXCLUDED.active;

-- =====================
-- 3. Stops (~28 real Cajamarca locations)
--    Cajamarca city center: lat ~-7.163, lon ~-78.512
-- =====================

INSERT INTO stops (id, name, geom, active) VALUES
  -- Central / historic
  ( 1, 'Plaza de Armas',                ST_SetSRID(ST_Point(-78.5120, -7.1631), 4326), true),
  ( 2, 'Catedral de Cajamarca',         ST_SetSRID(ST_Point(-78.5125, -7.1626), 4326), true),
  ( 3, 'Mercado Central',               ST_SetSRID(ST_Point(-78.5098, -7.1618), 4326), true),
  ( 4, 'Complejo de Belén',             ST_SetSRID(ST_Point(-78.5143, -7.1641), 4326), true),
  ( 5, 'Plazuela Bolognesi',            ST_SetSRID(ST_Point(-78.5100, -7.1656), 4326), true),

  -- Av. Hoyos Rubio corridor (north-south main road)
  ( 6, 'Av. Hoyos Rubio / Jr. Del Batan', ST_SetSRID(ST_Point(-78.5170, -7.1590), 4326), true),
  ( 7, 'Av. Hoyos Rubio / Ovalo Musical',  ST_SetSRID(ST_Point(-78.5200, -7.1540), 4326), true),
  ( 8, 'Qhapaq Ñan (Hoyos Rubio)',         ST_SetSRID(ST_Point(-78.5225, -7.1500), 4326), true),

  -- Av. Atahualpa corridor (east toward Baños del Inca)
  ( 9, 'Cuarto del Rescate',            ST_SetSRID(ST_Point(-78.5107, -7.1613), 4326), true),
  (10, 'Av. Atahualpa / Jr. Angamos',   ST_SetSRID(ST_Point(-78.5050, -7.1600), 4326), true),
  (11, 'Av. Atahualpa / Ovalo El Inca',  ST_SetSRID(ST_Point(-78.4980, -7.1585), 4326), true),
  (12, 'Av. Manco Cápac',               ST_SetSRID(ST_Point(-78.4900, -7.1570), 4326), true),
  (13, 'Baños del Inca - Centro',        ST_SetSRID(ST_Point(-78.4680, -7.1635), 4326), true),
  (14, 'Baños del Inca - Termas',        ST_SetSRID(ST_Point(-78.4665, -7.1650), 4326), true),

  -- South: Mollepampa / Av. Industrial
  (15, 'Mollepampa',                     ST_SetSRID(ST_Point(-78.5190, -7.1810), 4326), true),
  (16, 'Av. Industrial / Jr. Chanchamayo', ST_SetSRID(ST_Point(-78.5160, -7.1760), 4326), true),
  (17, 'Av. San Martín de Porres',       ST_SetSRID(ST_Point(-78.5130, -7.1720), 4326), true),

  -- University area (UNC / UPAGU)
  (18, 'Universidad Nacional de Cajamarca', ST_SetSRID(ST_Point(-78.5080, -7.1680), 4326), true),
  (19, 'UPAGU',                          ST_SetSRID(ST_Point(-78.5060, -7.1700), 4326), true),

  -- Hospital / health
  (20, 'Hospital Regional de Cajamarca', ST_SetSRID(ST_Point(-78.5042, -7.1575), 4326), true),

  -- San Sebastián / north neighborhoods
  (21, 'Iglesia San Sebastián',          ST_SetSRID(ST_Point(-78.5145, -7.1595), 4326), true),
  (22, 'Jr. Tayabamba / Av. El Maestro', ST_SetSRID(ST_Point(-78.5170, -7.1620), 4326), true),

  -- Lucmacucho (west hills)
  (23, 'Lucmacucho',                     ST_SetSRID(ST_Point(-78.5250, -7.1600), 4326), true),
  (24, 'Mirador Santa Apolonia',         ST_SetSRID(ST_Point(-78.5150, -7.1670), 4326), true),

  -- Terminal / access points
  (25, 'Terminal Terrestre',             ST_SetSRID(ST_Point(-78.5230, -7.1480), 4326), true),
  (26, 'Paradero Evitamiento Norte',     ST_SetSRID(ST_Point(-78.5260, -7.1440), 4326), true),

  -- East: El Ingenio
  (27, 'El Ingenio',                     ST_SetSRID(ST_Point(-78.4780, -7.1550), 4326), true),
  (28, 'Puente Atahualpa',              ST_SetSRID(ST_Point(-78.4850, -7.1560), 4326), true)
ON CONFLICT (id) DO UPDATE SET
  name   = EXCLUDED.name,
  geom   = EXCLUDED.geom,
  active = EXCLUDED.active;

-- Reset sequences past the highest explicit ID.
SELECT setval('stops_id_seq',  (SELECT MAX(id) FROM stops));
SELECT setval('routes_id_seq', (SELECT MAX(id) FROM routes));

-- =====================
-- 4. Route-stop mappings (stop sequences per route)
-- =====================

-- Clear any previous mappings for these routes.
DELETE FROM route_stops WHERE route_id IN (1, 2, 3, 4, 5);

-- Ruta 1: Centro → Baños del Inca (east corridor along Av. Atahualpa)
INSERT INTO route_stops (route_id, stop_id, sequence) VALUES
  (1,  1, 1),   -- Plaza de Armas
  (1,  9, 2),   -- Cuarto del Rescate
  (1, 10, 3),   -- Av. Atahualpa / Jr. Angamos
  (1, 11, 4),   -- Ovalo El Inca
  (1, 12, 5),   -- Av. Manco Cápac
  (1, 13, 6),   -- Baños del Inca - Centro
  (1, 14, 7)    -- Baños del Inca - Termas
ON CONFLICT DO NOTHING;

-- Ruta 2: Mollepampa → Universidad (south to east)
INSERT INTO route_stops (route_id, stop_id, sequence) VALUES
  (2, 15, 1),   -- Mollepampa
  (2, 16, 2),   -- Av. Industrial
  (2, 17, 3),   -- Av. San Martín de Porres
  (2,  5, 4),   -- Plazuela Bolognesi
  (2,  1, 5),   -- Plaza de Armas
  (2, 18, 6),   -- UNC
  (2, 19, 7)    -- UPAGU
ON CONFLICT DO NOTHING;

-- Ruta 3: Centro → Hoyos Rubio (north corridor)
INSERT INTO route_stops (route_id, stop_id, sequence) VALUES
  (3,  1, 1),   -- Plaza de Armas
  (3,  4, 2),   -- Complejo de Belén
  (3, 22, 3),   -- Jr. Tayabamba / Av. El Maestro
  (3,  6, 4),   -- Av. Hoyos Rubio / Jr. Del Batan
  (3,  7, 5),   -- Ovalo Musical
  (3,  8, 6)    -- Qhapaq Ñan
ON CONFLICT DO NOTHING;

-- Ruta 4: San Sebastián → Lucmacucho (west loop through center)
INSERT INTO route_stops (route_id, stop_id, sequence) VALUES
  (4, 21, 1),   -- Iglesia San Sebastián
  (4,  2, 2),   -- Catedral
  (4,  1, 3),   -- Plaza de Armas
  (4, 24, 4),   -- Mirador Santa Apolonia
  (4, 23, 5)    -- Lucmacucho
ON CONFLICT DO NOTHING;

-- Ruta 5: Terminal Terrestre → El Ingenio (cross-city east-west)
INSERT INTO route_stops (route_id, stop_id, sequence) VALUES
  (5, 25, 1),   -- Terminal Terrestre
  (5, 26, 2),   -- Evitamiento Norte
  (5,  7, 3),   -- Ovalo Musical
  (5,  6, 4),   -- Hoyos Rubio / Del Batan
  (5,  1, 5),   -- Plaza de Armas
  (5, 20, 6),   -- Hospital Regional
  (5, 28, 7),   -- Puente Atahualpa
  (5, 27, 8)    -- El Ingenio
ON CONFLICT DO NOTHING;

-- =====================
-- 5. Route shapes (simplified linestrings following stop order)
-- =====================

DELETE FROM route_shapes WHERE route_id IN (1, 2, 3, 4, 5);

INSERT INTO route_shapes (route_id, geom) VALUES
  -- Ruta 1: Centro → Baños del Inca
  (1, ST_SetSRID(ST_MakeLine(ARRAY[
    ST_Point(-78.5120, -7.1631),  -- Plaza de Armas
    ST_Point(-78.5107, -7.1613),  -- Cuarto del Rescate
    ST_Point(-78.5050, -7.1600),  -- Av. Atahualpa / Angamos
    ST_Point(-78.4980, -7.1585),  -- Ovalo El Inca
    ST_Point(-78.4900, -7.1570),  -- Manco Cápac
    ST_Point(-78.4680, -7.1635),  -- Baños del Inca Centro
    ST_Point(-78.4665, -7.1650)   -- Baños del Inca Termas
  ]), 4326)),

  -- Ruta 2: Mollepampa → Universidad
  (2, ST_SetSRID(ST_MakeLine(ARRAY[
    ST_Point(-78.5190, -7.1810),  -- Mollepampa
    ST_Point(-78.5160, -7.1760),  -- Av. Industrial
    ST_Point(-78.5130, -7.1720),  -- San Martín de Porres
    ST_Point(-78.5100, -7.1656),  -- Plazuela Bolognesi
    ST_Point(-78.5120, -7.1631),  -- Plaza de Armas
    ST_Point(-78.5080, -7.1680),  -- UNC
    ST_Point(-78.5060, -7.1700)   -- UPAGU
  ]), 4326)),

  -- Ruta 3: Centro → Hoyos Rubio
  (3, ST_SetSRID(ST_MakeLine(ARRAY[
    ST_Point(-78.5120, -7.1631),  -- Plaza de Armas
    ST_Point(-78.5143, -7.1641),  -- Complejo de Belén
    ST_Point(-78.5170, -7.1620),  -- Jr. Tayabamba
    ST_Point(-78.5170, -7.1590),  -- Hoyos Rubio / Del Batan
    ST_Point(-78.5200, -7.1540),  -- Ovalo Musical
    ST_Point(-78.5225, -7.1500)   -- Qhapaq Ñan
  ]), 4326)),

  -- Ruta 4: San Sebastián → Lucmacucho
  (4, ST_SetSRID(ST_MakeLine(ARRAY[
    ST_Point(-78.5145, -7.1595),  -- Iglesia San Sebastián
    ST_Point(-78.5125, -7.1626),  -- Catedral
    ST_Point(-78.5120, -7.1631),  -- Plaza de Armas
    ST_Point(-78.5150, -7.1670),  -- Mirador Santa Apolonia
    ST_Point(-78.5250, -7.1600)   -- Lucmacucho
  ]), 4326)),

  -- Ruta 5: Terminal Terrestre → El Ingenio
  (5, ST_SetSRID(ST_MakeLine(ARRAY[
    ST_Point(-78.5230, -7.1480),  -- Terminal Terrestre
    ST_Point(-78.5260, -7.1440),  -- Evitamiento Norte
    ST_Point(-78.5200, -7.1540),  -- Ovalo Musical
    ST_Point(-78.5170, -7.1590),  -- Hoyos Rubio / Del Batan
    ST_Point(-78.5120, -7.1631),  -- Plaza de Armas
    ST_Point(-78.5042, -7.1575),  -- Hospital Regional
    ST_Point(-78.4850, -7.1560),  -- Puente Atahualpa
    ST_Point(-78.4780, -7.1550)   -- El Ingenio
  ]), 4326))
ON CONFLICT (route_id) DO UPDATE SET geom = EXCLUDED.geom, updated_at = NOW();

-- =====================
-- 6. Users (1 admin + 2 drivers)
--    Passwords hashed with bcrypt cost 10.
-- =====================

INSERT INTO users (id, username, password_hash, full_name, phone, role, active) VALUES
  (1, 'admin',
   '$2a$10$Dj1Oxb.KR2TDHwtI8.231.K5IW53CsgDGrUjf2TJWgLJGY4VXqzZS',
   'Administrador Qapac', '976000001', 'admin', true),
  (2, 'conductor1',
   '$2a$10$jWv/mzugZmGSJSwpx6pSwOxlTPEVR7RETPUUJ9EY7rCSeZCNsOtve',
   'Carlos Quispe Rojas', '976000002', 'driver', true),
  (3, 'conductor2',
   '$2a$10$jWv/mzugZmGSJSwpx6pSwOxlTPEVR7RETPUUJ9EY7rCSeZCNsOtve',
   'María Elena Torres', '976000003', 'driver', true)
ON CONFLICT (id) DO UPDATE SET
  username      = EXCLUDED.username,
  password_hash = EXCLUDED.password_hash,
  full_name     = EXCLUDED.full_name,
  phone         = EXCLUDED.phone,
  role          = EXCLUDED.role,
  active        = EXCLUDED.active;

SELECT setval('users_id_seq', (SELECT MAX(id) FROM users));

-- =====================
-- 7. Vehicles (8 buses with Peruvian-style plates, assigned to routes)
-- =====================

INSERT INTO vehicles (id, plate_number, route_id, status) VALUES
  (1, 'ABC-123', 1, 'active'),
  (2, 'ABD-456', 1, 'active'),
  (3, 'CAJ-100', 2, 'active'),
  (4, 'CAJ-201', 2, 'active'),
  (5, 'CAJ-302', 3, 'active'),
  (6, 'CAJ-403', 4, 'active'),
  (7, 'CAJ-504', 5, 'active'),
  (8, 'CAJ-605', 5, 'inactive')
ON CONFLICT (id) DO UPDATE SET
  plate_number = EXCLUDED.plate_number,
  route_id     = EXCLUDED.route_id,
  status       = EXCLUDED.status;

SELECT setval('vehicles_id_seq', (SELECT MAX(id) FROM vehicles));

-- =====================
-- 8. Vehicle assignments (link drivers to vehicles)
-- =====================

-- Deactivate any prior assignments for these vehicles.
UPDATE vehicle_assignments SET active = false
  WHERE vehicle_id IN (1, 2, 3, 4, 5, 6, 7, 8) AND active = true;

INSERT INTO vehicle_assignments (vehicle_id, driver_id, active) VALUES
  (1, 2, true),   -- conductor1 drives vehicle 1 (Ruta 1)
  (3, 3, true)    -- conductor2 drives vehicle 3 (Ruta 2)
ON CONFLICT DO NOTHING;

-- =====================
-- 9. Sample vehicle positions (so nearby-vehicles queries return data)
-- =====================

INSERT INTO vehicle_positions (vehicle_id, geom, heading, speed, recorded_at) VALUES
  -- Vehicle 1 near Plaza de Armas (Ruta 1)
  (1, ST_SetSRID(ST_Point(-78.5115, -7.1628), 4326), 90.0, 15.5, NOW()),
  -- Vehicle 3 near Mollepampa (Ruta 2)
  (3, ST_SetSRID(ST_Point(-78.5185, -7.1805), 4326), 0.0, 22.0, NOW())
ON CONFLICT (vehicle_id) DO UPDATE SET
  geom        = EXCLUDED.geom,
  heading     = EXCLUDED.heading,
  speed       = EXCLUDED.speed,
  recorded_at = EXCLUDED.recorded_at;
