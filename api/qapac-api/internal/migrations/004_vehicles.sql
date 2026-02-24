-- Migration: 004_vehicles
-- Adds vehicle fleet management: vehicles, driver/collector assignments,
-- and real-time GPS position tracking.

-- =====================
-- Vehicles (bus fleet)
-- =====================

CREATE TABLE IF NOT EXISTS vehicles (
  id           SERIAL PRIMARY KEY,
  plate_number VARCHAR(20)  NOT NULL UNIQUE,
  route_id     INT          REFERENCES routes(id) ON DELETE SET NULL,
  status       VARCHAR(20)  DEFAULT 'inactive' CHECK (status IN ('active', 'inactive', 'maintenance')),
  created_at   TIMESTAMP    DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_vehicles_route_id ON vehicles(route_id);
CREATE INDEX IF NOT EXISTS idx_vehicles_status   ON vehicles(status);

-- =====================
-- Vehicle assignments (driver + optional collector per vehicle)
-- Only one active assignment per vehicle at a time.
-- =====================

CREATE TABLE IF NOT EXISTS vehicle_assignments (
  id           SERIAL PRIMARY KEY,
  vehicle_id   INT       NOT NULL REFERENCES vehicles(id) ON DELETE CASCADE,
  driver_id    INT       NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  collector_id INT       REFERENCES users(id) ON DELETE SET NULL,
  assigned_at  TIMESTAMP DEFAULT NOW(),
  active       BOOLEAN   DEFAULT true
);

-- Ensure only one active assignment per vehicle
CREATE UNIQUE INDEX IF NOT EXISTS idx_vehicle_assignments_active
  ON vehicle_assignments(vehicle_id) WHERE active = true;

CREATE INDEX IF NOT EXISTS idx_vehicle_assignments_driver    ON vehicle_assignments(driver_id);
CREATE INDEX IF NOT EXISTS idx_vehicle_assignments_collector ON vehicle_assignments(collector_id);

-- =====================
-- Vehicle positions (latest GPS position per vehicle)
-- One row per vehicle, updated on each GPS report from the driver app.
-- =====================

CREATE TABLE IF NOT EXISTS vehicle_positions (
  id          SERIAL PRIMARY KEY,
  vehicle_id  INT            NOT NULL REFERENCES vehicles(id) ON DELETE CASCADE UNIQUE,
  geom        GEOMETRY(POINT, 4326) NOT NULL,
  heading     FLOAT,
  speed       FLOAT,
  recorded_at TIMESTAMP      NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_vehicle_positions_geom ON vehicle_positions USING GIST(geom);
