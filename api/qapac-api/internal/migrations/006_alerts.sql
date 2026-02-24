-- Migration: 006_alerts
-- Route change notifications and incident reports.
-- Can be created by admins or drivers.

CREATE TABLE IF NOT EXISTS alerts (
  id            SERIAL PRIMARY KEY,
  title         VARCHAR(255) NOT NULL,
  description   TEXT,
  route_id      INT          REFERENCES routes(id) ON DELETE SET NULL,
  vehicle_plate VARCHAR(20),
  image_path    VARCHAR(500),
  created_by    INT          REFERENCES users(id) ON DELETE SET NULL,
  created_at    TIMESTAMP    DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_alerts_route_id   ON alerts(route_id);
CREATE INDEX IF NOT EXISTS idx_alerts_created_at ON alerts(created_at DESC);
