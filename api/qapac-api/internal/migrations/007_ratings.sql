-- Migration: 007_ratings
-- Anonymous trip ratings submitted by app users (identified by device_id).
-- Each device can rate a given trip only once.

CREATE TABLE IF NOT EXISTS ratings (
  id         SERIAL PRIMARY KEY,
  trip_id    INT          NOT NULL REFERENCES trips(id) ON DELETE CASCADE,
  rating     SMALLINT     NOT NULL CHECK (rating BETWEEN 1 AND 5),
  device_id  VARCHAR(255) NOT NULL,
  created_at TIMESTAMP    DEFAULT NOW(),
  UNIQUE(trip_id, device_id)
);

CREATE INDEX IF NOT EXISTS idx_ratings_trip_id ON ratings(trip_id);
