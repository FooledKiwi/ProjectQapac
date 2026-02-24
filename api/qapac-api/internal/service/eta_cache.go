package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	// etaCacheTTL is how long a cached ETA entry remains valid.
	// ETA changes frequently, so we keep the TTL short.
	etaCacheTTL = 60 * time.Second

	// etaCacheQueryTimeout is the deadline for each cache read/write query.
	etaCacheQueryTimeout = 5 * time.Second
)

// ETACacheStore abstracts the persistence layer for ETA caching.
// Using an interface lets unit tests swap in an in-memory double without a DB.
type ETACacheStore interface {
	// GetCachedETA returns the cached ETA seconds for stopID, or (0, false, nil)
	// when there is no valid (non-expired) entry.
	GetCachedETA(ctx context.Context, stopID int32) (seconds int, found bool, err error)

	// SetCachedETA upserts an ETA entry with an expiry of now + etaCacheTTL.
	SetCachedETA(ctx context.Context, stopID int32, seconds int) error
}

// ETAService wraps one or two ETAProviders with a database-backed cache.
//
// Resolution order on a cache miss:
//  1. Call primary provider.
//  2. If primary returns ErrNoVehicleData AND a fallback is configured,
//     call the fallback provider instead.
//  3. Write the result to the cache and return it.
//
// This design supports the MVP v1 → v2 migration path:
//   - MVP v1: primary = SimpleETAProvider, no fallback.
//   - MVP v2: primary = GPSETAProvider,    fallback = SimpleETAProvider.
//     Swapping primary is the only change required in app.go.
type ETAService struct {
	primary  ETAProvider
	fallback ETAProvider // optional; nil means no fallback
	store    ETACacheStore
}

// NewETAService creates an ETAService with a single provider and no fallback.
// Use this for MVP v1 where SimpleETAProvider is the only source.
//
//   - primary is the strategy used to compute ETA on a cache miss.
//   - store   is the cache backend (use NewPgETACacheStore for production).
func NewETAService(primary ETAProvider, store ETACacheStore) *ETAService {
	return &ETAService{primary: primary, store: store}
}

// NewETAServiceWithFallback creates an ETAService where fallback is called
// whenever primary returns ErrNoVehicleData.  Use this for MVP v2:
//
//	NewETAServiceWithFallback(gpsProvider, simpleProvider, pgStore)
func NewETAServiceWithFallback(primary, fallback ETAProvider, store ETACacheStore) *ETAService {
	return &ETAService{primary: primary, fallback: fallback, store: store}
}

// GetETAForStop returns the estimated bus arrival time (in seconds) for the
// given stop.
//
// Resolution order:
//  1. Cache hit  → returns immediately with source="cache".
//  2. Primary provider → used on cache miss.
//  3. Fallback provider → used only when primary returns ErrNoVehicleData.
//
// The source string in the return value identifies where the value came from
// ("cache", "simple", "gps", "simple_fallback", …) for telemetry purposes.
//
// Errors:
//   - stopID ≤ 0 → immediate error, no provider is called.
//   - Primary fails with a non-ErrNoVehicleData error → error is returned.
//   - Primary returns ErrNoVehicleData but no fallback is set → error is returned.
//   - Cache read/write failures are non-fatal; the computed value is still returned.
func (s *ETAService) GetETAForStop(ctx context.Context, stopID int32) (seconds int, source string, err error) {
	if stopID <= 0 {
		return 0, "", fmt.Errorf("eta: GetETAForStop: invalid stop ID %d", stopID)
	}

	// --- cache read ---
	cached, found, _ := s.store.GetCachedETA(ctx, stopID) //nolint:errcheck // cache miss is non-fatal
	if found {
		return cached, "cache", nil
	}
	// Cache failures are non-fatal; fall through to the provider.

	// --- primary provider ---
	secs, src, provErr := s.primary.GetETA(ctx, stopID)
	if provErr != nil {
		// If the primary has no data and a fallback is available, use it.
		if errors.Is(provErr, ErrNoVehicleData) && s.fallback != nil {
			secs, src, provErr = s.fallback.GetETA(ctx, stopID)
			if provErr != nil {
				return 0, "", fmt.Errorf("eta: GetETAForStop: fallback provider: %w", provErr)
			}
			// Annotate source so callers/telemetry know GPS was attempted.
			src = src + "_fallback"
		} else {
			return 0, "", fmt.Errorf("eta: GetETAForStop: primary provider: %w", provErr)
		}
	}

	// --- cache write (best-effort) ---
	_ = s.store.SetCachedETA(ctx, stopID, secs) //nolint:errcheck // best-effort cache write

	return secs, src, nil
}

// --- pgx-backed ETACacheStore ---

// pgETACacheStore is the production implementation backed by pgx.
type pgETACacheStore struct {
	pool *pgxpool.Pool
}

// NewPgETACacheStore creates an ETACacheStore backed by the given connection pool.
func NewPgETACacheStore(pool *pgxpool.Pool) ETACacheStore {
	return &pgETACacheStore{pool: pool}
}

// GetCachedETA queries stop_eta_cache for a valid (non-expired) entry.
func (s *pgETACacheStore) GetCachedETA(ctx context.Context, stopID int32) (seconds int, found bool, err error) {
	ctx, cancel := context.WithTimeout(ctx, etaCacheQueryTimeout)
	defer cancel()

	const q = `
		SELECT eta_seconds
		FROM stop_eta_cache
		WHERE stop_id   = $1
		  AND expires_at > NOW()`

	var etaSecs int32
	err = s.pool.QueryRow(ctx, q, stopID).Scan(&etaSecs)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, false, nil // cache miss
	}
	if err != nil {
		return 0, false, fmt.Errorf("eta: cache: get: %w", err)
	}

	return int(etaSecs), true, nil
}

// SetCachedETA upserts an ETA entry into stop_eta_cache.
// The expiry time is computed in Go so that etaCacheTTL is the single source of truth.
func (s *pgETACacheStore) SetCachedETA(ctx context.Context, stopID int32, seconds int) error {
	ctx, cancel := context.WithTimeout(ctx, etaCacheQueryTimeout)
	defer cancel()

	expiresAt := time.Now().Add(etaCacheTTL)

	const q = `
		INSERT INTO stop_eta_cache (stop_id, eta_seconds, calc_ts, expires_at)
		VALUES ($1, $2, NOW(), $3)
		ON CONFLICT (stop_id)
		DO UPDATE SET
			eta_seconds = EXCLUDED.eta_seconds,
			calc_ts     = EXCLUDED.calc_ts,
			expires_at  = EXCLUDED.expires_at`

	_, err := s.pool.Exec(ctx, q, stopID, int32(seconds), expiresAt)
	if err != nil {
		return fmt.Errorf("eta: cache: set: %w", err)
	}
	return nil
}
