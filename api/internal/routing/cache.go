package routing

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mmcloughlin/geohash"
)

const (
	// cacheTTL is how long a cached route entry remains valid.
	cacheTTL = 120 * time.Second

	// cacheQueryTimeout is the deadline for each cache read/write query.
	cacheQueryTimeout = 5 * time.Second

	// geohashPrecision controls the spatial resolution of the origin hash.
	// Precision 7 ≈ ±76m latitude / ±152m longitude cell — appropriate for
	// urban transit where stops can be as close as 200m apart.
	geohashPrecision = 7

	// noStopID is the sentinel value used when no stop ID is present in the
	// context. Stop IDs in the DB are SERIAL starting at 1, so 0 is safe as
	// a "no stop" cache-key component. CachedRouter should only see this when
	// used outside of RoutingService (e.g. direct callers in tests).
	noStopID int32 = 0
)

// CacheStore abstracts the persistence layer for route caching.
// This interface makes it easy to swap the real pgx implementation with a
// test double in unit tests.
type CacheStore interface {
	// GetCachedRoute returns a cached RoutingResponse for the given key, or
	// (nil, nil) when there is no valid (non-expired) entry.
	GetCachedRoute(ctx context.Context, originHash string, stopID int32) (*RoutingResponse, error)

	// SetCachedRoute upserts a route entry with an expiry of now + cacheTTL.
	SetCachedRoute(ctx context.Context, originHash string, stopID int32, resp *RoutingResponse) error
}

// Logger is a printf-style logging function injected into CachedRouter.
// Using a function type (rather than an interface) keeps the dependency minimal
// and makes test doubles trivial to write.
type Logger func(format string, args ...any)

// CachedRouter wraps another Router and transparently caches its results.
// Cache keys are composed of a geohash of the origin and the destination stop ID.
type CachedRouter struct {
	inner      Router
	store      CacheStore
	logger     Logger // called when async cache writes fail; nil = silent
	afterStore func() // optional hook called after every async store attempt; used in tests for synchronization
}

// CachedRouterOption configures a CachedRouter.
type CachedRouterOption func(*CachedRouter)

// WithLogger sets a logger that is called when the async cache write fails.
// In production, pass a log.Printf-compatible function. If not set, errors
// are silently dropped (preserves previous behavior and keeps the hot path clean).
func WithLogger(l Logger) CachedRouterOption {
	return func(r *CachedRouter) { r.logger = l }
}

// withAfterStore sets a hook called after every async store attempt (success or
// failure). Intended exclusively for test synchronization — do not use in production.
func withAfterStore(fn func()) CachedRouterOption {
	return func(r *CachedRouter) { r.afterStore = fn }
}

// NewCachedRouter wraps inner with a cache-aside layer backed by store.
// Optional behavior (logging, test synchronization) is configured via opts.
func NewCachedRouter(inner Router, store CacheStore, opts ...CachedRouterOption) *CachedRouter {
	r := &CachedRouter{inner: inner, store: store}
	for _, o := range opts {
		o(r)
	}
	return r
}

// Route satisfies the Router interface.
// It checks the cache first; on a miss it delegates to the inner Router and
// persists the result.
func (r *CachedRouter) Route(ctx context.Context, req RoutingRequest) (*RoutingResponse, error) {
	key := originHash(req.OriginLat, req.OriginLon)
	stopID := noStopID // sentinel: no stop ID in context

	// When callers supply the stop ID through the context (see WithStopID),
	// we use it to make the cache key more precise.
	if id, ok := stopIDFromContext(ctx); ok {
		stopID = id
	}

	cached, err := r.store.GetCachedRoute(ctx, key, stopID)
	if err != nil {
		// Cache read failures are non-fatal: fall through to the real router.
		_ = err
	}
	if cached != nil {
		return cached, nil
	}

	// Cache miss — call the inner router.
	resp, err := r.inner.Route(ctx, req)
	if err != nil {
		return nil, err
	}

	// Persist asynchronously so we don't add cache-write latency to the hot path.
	// We use a background context to avoid cancellation if the caller's context
	// expires right after the API call returns.
	go func() {
		storeCtx, cancel := context.WithTimeout(context.Background(), cacheQueryTimeout)
		defer cancel()

		if err := r.store.SetCachedRoute(storeCtx, key, stopID, resp); err != nil {
			if r.logger != nil {
				r.logger("routing: cache: async write failed (origin=%s stop=%d): %v", key, stopID, err)
			}
		}

		if r.afterStore != nil {
			r.afterStore()
		}
	}()

	return resp, nil
}

// originHash returns a geohash string that uniquely identifies the origin cell.
func originHash(lat, lon float64) string {
	return geohash.EncodeWithPrecision(lat, lon, geohashPrecision)
}

// --- pgx-backed CacheStore implementation ---

// pgCacheStore is the production implementation of CacheStore backed by pgx.
type pgCacheStore struct {
	pool *pgxpool.Pool
}

// NewPgCacheStore creates a CacheStore backed by the given connection pool.
func NewPgCacheStore(pool *pgxpool.Pool) CacheStore {
	return &pgCacheStore{pool: pool}
}

// GetCachedRoute queries route_to_stop_cache for a valid (non-expired) entry.
func (s *pgCacheStore) GetCachedRoute(ctx context.Context, originHash string, stopID int32) (*RoutingResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, cacheQueryTimeout)
	defer cancel()

	const q = `
		SELECT polyline, distance_m, duration_s
		FROM route_to_stop_cache
		WHERE origin_hash = $1
		  AND stop_id     = $2
		  AND expires_at  > NOW()`

	var (
		polyline  string
		distanceM int32
		durationS int32
	)

	err := s.pool.QueryRow(ctx, q, originHash, stopID).Scan(&polyline, &distanceM, &durationS)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil // cache miss
	}
	if err != nil {
		return nil, fmt.Errorf("routing: cache: get: %w", err)
	}

	return &RoutingResponse{
		Polyline:  polyline,
		DistanceM: int(distanceM),
		DurationS: int(durationS),
	}, nil
}

// SetCachedRoute upserts a route entry into route_to_stop_cache.
// The expiry time is computed in Go from cacheTTL so that the constant is the
// single source of truth — the SQL never encodes the TTL value directly.
func (s *pgCacheStore) SetCachedRoute(ctx context.Context, originHash string, stopID int32, resp *RoutingResponse) error {
	ctx, cancel := context.WithTimeout(ctx, cacheQueryTimeout)
	defer cancel()

	expiresAt := time.Now().Add(cacheTTL)

	const q = `
		INSERT INTO route_to_stop_cache
			(origin_hash, stop_id, polyline, distance_m, duration_s, calc_ts, expires_at)
		VALUES
			($1, $2, $3, $4, $5, NOW(), $6)
		ON CONFLICT (origin_hash, stop_id)
		DO UPDATE SET
			polyline   = EXCLUDED.polyline,
			distance_m = EXCLUDED.distance_m,
			duration_s = EXCLUDED.duration_s,
			calc_ts    = EXCLUDED.calc_ts,
			expires_at = EXCLUDED.expires_at`

	_, err := s.pool.Exec(ctx, q,
		originHash,
		stopID,
		resp.Polyline,
		int32(resp.DistanceM),
		int32(resp.DurationS),
		expiresAt,
	)
	if err != nil {
		return fmt.Errorf("routing: cache: set: %w", err)
	}
	return nil
}

// --- Context helpers for passing stop ID through the routing call chain ---

type contextKey int

const stopIDKey contextKey = iota

// WithStopID returns a new context carrying the stop ID.
// CachedRouter reads this to build a more precise cache key.
func WithStopID(ctx context.Context, stopID int32) context.Context {
	return context.WithValue(ctx, stopIDKey, stopID)
}

// StopIDFromContext extracts the stop ID stored by WithStopID, if present.
// Exported so that the service layer and tests can inspect the context.
func StopIDFromContext(ctx context.Context) (int32, bool) {
	v, ok := ctx.Value(stopIDKey).(int32)
	return v, ok
}

// stopIDFromContext is the internal alias used within the package.
func stopIDFromContext(ctx context.Context) (int32, bool) {
	return StopIDFromContext(ctx)
}
