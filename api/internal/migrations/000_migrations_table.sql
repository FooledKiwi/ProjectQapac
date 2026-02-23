-- Migration: 000_migrations_table
-- Must run before any other migration.
-- Tracks which migrations have already been applied so Run() is idempotent.

CREATE TABLE IF NOT EXISTS schema_migrations (
  version    VARCHAR(255) PRIMARY KEY,
  applied_at TIMESTAMP DEFAULT NOW()
);
