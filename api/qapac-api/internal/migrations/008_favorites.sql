-- Migration: 008_favorites
-- Anonymous user favorite routes, identified by device_id.
-- Each device can favorite a given route only once.

CREATE TABLE IF NOT EXISTS favorites (
  id         SERIAL PRIMARY KEY,
  device_id  VARCHAR(255) NOT NULL,
  route_id   INT          NOT NULL REFERENCES routes(id) ON DELETE CASCADE,
  created_at TIMESTAMP    DEFAULT NOW(),
  UNIQUE(device_id, route_id)
);

CREATE INDEX IF NOT EXISTS idx_favorites_device_id ON favorites(device_id);
CREATE INDEX IF NOT EXISTS idx_favorites_route_id  ON favorites(route_id);
