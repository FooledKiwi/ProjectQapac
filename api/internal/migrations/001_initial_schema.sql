-- Migration: 001_initial_schema
-- Requires: PostGIS extension

-- Enable PostGIS if not already enabled
CREATE EXTENSION IF NOT EXISTS postgis;

-- =====================
-- Base tables
-- =====================

CREATE TABLE IF NOT EXISTS stops (
  id         SERIAL PRIMARY KEY,
  name       VARCHAR(255) NOT NULL,
  geom       GEOMETRY(POINT, 4326) NOT NULL,
  active     BOOLEAN DEFAULT true,
  created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_stops_geom ON stops USING GIST(geom);

CREATE TABLE IF NOT EXISTS routes (
  id     SERIAL PRIMARY KEY,
  name   VARCHAR(255) NOT NULL,
  active BOOLEAN DEFAULT true
);

CREATE TABLE IF NOT EXISTS route_stops (
  id       SERIAL PRIMARY KEY,
  route_id INT NOT NULL REFERENCES routes(id),
  stop_id  INT NOT NULL REFERENCES stops(id),
  sequence INT NOT NULL,
  UNIQUE(route_id, stop_id)
);

CREATE TABLE IF NOT EXISTS route_shapes (
  id         SERIAL PRIMARY KEY,
  route_id   INT NOT NULL REFERENCES routes(id) UNIQUE,
  geom       GEOMETRY(LINESTRING, 4326) NOT NULL,
  updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_route_shapes_geom ON route_shapes USING GIST(geom);

-- =====================
-- Cache tables (unlogged: no WAL, fast writes, data lost on crash)
-- =====================

CREATE UNLOGGED TABLE IF NOT EXISTS stop_eta_cache (
  id         SERIAL PRIMARY KEY,
  stop_id    INT NOT NULL REFERENCES stops(id),
  eta_seconds INT NOT NULL,
  calc_ts    TIMESTAMP DEFAULT NOW(),
  expires_at TIMESTAMP NOT NULL,
  UNIQUE(stop_id)
);

CREATE INDEX IF NOT EXISTS idx_eta_cache_stop_id ON stop_eta_cache(stop_id);

CREATE UNLOGGED TABLE IF NOT EXISTS route_to_stop_cache (
  id          SERIAL PRIMARY KEY,
  origin_hash VARCHAR(50) NOT NULL,
  stop_id     INT NOT NULL REFERENCES stops(id),
  polyline    TEXT NOT NULL,
  distance_m  INT NOT NULL,
  duration_s  INT NOT NULL,
  calc_ts     TIMESTAMP DEFAULT NOW(),
  expires_at  TIMESTAMP NOT NULL,
  UNIQUE(origin_hash, stop_id)
);

CREATE INDEX IF NOT EXISTS idx_route_cache_hash_stop ON route_to_stop_cache(origin_hash, stop_id);
