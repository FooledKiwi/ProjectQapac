-- Migration: 005_trips
-- Tracks bus trips (a driver starting and completing a route run).

CREATE TABLE IF NOT EXISTS trips (
  id         SERIAL PRIMARY KEY,
  vehicle_id INT          NOT NULL REFERENCES vehicles(id) ON DELETE CASCADE,
  route_id   INT          NOT NULL REFERENCES routes(id) ON DELETE CASCADE,
  driver_id  INT          NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  started_at TIMESTAMP    NOT NULL DEFAULT NOW(),
  ended_at   TIMESTAMP,
  status     VARCHAR(20)  DEFAULT 'active' CHECK (status IN ('active', 'completed', 'cancelled'))
);

CREATE INDEX IF NOT EXISTS idx_trips_vehicle_id ON trips(vehicle_id);
CREATE INDEX IF NOT EXISTS idx_trips_route_id   ON trips(route_id);
CREATE INDEX IF NOT EXISTS idx_trips_driver_id  ON trips(driver_id);
CREATE INDEX IF NOT EXISTS idx_trips_status     ON trips(status);
CREATE INDEX IF NOT EXISTS idx_trips_started_at ON trips(started_at);
