package service

import (
	"context"
	"errors"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Test doubles
// ---------------------------------------------------------------------------

// mockETAProvider is a configurable ETAProvider test double.
type mockETAProvider struct {
	seconds int
	source  string
	err     error
	calls   int
}

func (m *mockETAProvider) GetETA(_ context.Context, _ int32) (int, string, error) {
	m.calls++
	return m.seconds, m.source, m.err
}

// memETACacheStore is an in-memory ETACacheStore with per-entry expiry.
type memETACacheStore struct {
	entries  map[int32]memCacheEntry
	setErr   error
	getErr   error
	setCalls int
	getCalls int
}

type memCacheEntry struct {
	seconds   int
	expiresAt time.Time
}

func newMemStore() *memETACacheStore {
	return &memETACacheStore{entries: make(map[int32]memCacheEntry)}
}

func (m *memETACacheStore) GetCachedETA(_ context.Context, stopID int32) (int, bool, error) {
	m.getCalls++
	if m.getErr != nil {
		return 0, false, m.getErr
	}
	e, ok := m.entries[stopID]
	if !ok || time.Now().After(e.expiresAt) {
		return 0, false, nil
	}
	return e.seconds, true, nil
}

func (m *memETACacheStore) SetCachedETA(_ context.Context, stopID int32, seconds int) error {
	m.setCalls++
	if m.setErr != nil {
		return m.setErr
	}
	m.entries[stopID] = memCacheEntry{
		seconds:   seconds,
		expiresAt: time.Now().Add(etaCacheTTL),
	}
	return nil
}

// putExpired inserts an already-expired entry to simulate a stale cache row.
func (m *memETACacheStore) putExpired(stopID int32, seconds int) {
	m.entries[stopID] = memCacheEntry{
		seconds:   seconds,
		expiresAt: time.Now().Add(-time.Second),
	}
}

// ---------------------------------------------------------------------------
// ETAService — cache behaviour
// ---------------------------------------------------------------------------

func TestETAService_CacheHit(t *testing.T) {
	store := newMemStore()
	primary := &mockETAProvider{seconds: 999, source: "gps"}
	svc := NewETAService(primary, store)

	_ = store.SetCachedETA(context.Background(), 1, 120)

	secs, source, err := svc.GetETAForStop(context.Background(), 1)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if source != "cache" {
		t.Errorf("source = %q, want %q", source, "cache")
	}
	if secs != 120 {
		t.Errorf("seconds = %d, want 120", secs)
	}
	if primary.calls != 0 {
		t.Errorf("primary called %d times on cache hit, want 0", primary.calls)
	}
}

func TestETAService_CacheMiss_CallsPrimary(t *testing.T) {
	primary := &mockETAProvider{seconds: 240, source: "simple"}
	svc := NewETAService(primary, newMemStore())

	secs, source, err := svc.GetETAForStop(context.Background(), 2)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if source != "simple" {
		t.Errorf("source = %q, want %q", source, "simple")
	}
	if secs != 240 {
		t.Errorf("seconds = %d, want 240", secs)
	}
	if primary.calls != 1 {
		t.Errorf("primary calls = %d, want 1", primary.calls)
	}
}

func TestETAService_CacheMiss_WritesCache(t *testing.T) {
	store := newMemStore()
	primary := &mockETAProvider{seconds: 300, source: "simple"}
	svc := NewETAService(primary, store)

	_, _, _ = svc.GetETAForStop(context.Background(), 3)

	if store.setCalls != 1 {
		t.Errorf("SetCachedETA called %d times, want 1", store.setCalls)
	}
	// Second call must be a cache hit — primary not called again.
	_, src, _ := svc.GetETAForStop(context.Background(), 3)
	if src != "cache" {
		t.Errorf("second call source = %q, want %q", src, "cache")
	}
	if primary.calls != 1 {
		t.Errorf("primary calls after second request = %d, want 1", primary.calls)
	}
}

func TestETAService_ExpiredCache_CallsPrimary(t *testing.T) {
	store := newMemStore()
	store.putExpired(4, 42)

	primary := &mockETAProvider{seconds: 180, source: "simple"}
	svc := NewETAService(primary, store)

	secs, src, err := svc.GetETAForStop(context.Background(), 4)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if src != "simple" {
		t.Errorf("source = %q, want %q", src, "simple")
	}
	if secs != 180 {
		t.Errorf("seconds = %d, want 180", secs)
	}
}

func TestETAService_InvalidStopID(t *testing.T) {
	svc := NewETAService(&mockETAProvider{}, newMemStore())

	for _, id := range []int32{0, -1, -100} {
		_, _, err := svc.GetETAForStop(context.Background(), id)
		if err == nil {
			t.Errorf("stopID=%d: expected error, got nil", id)
		}
	}
}

func TestETAService_PrimaryError_NoFallback(t *testing.T) {
	primary := &mockETAProvider{err: errors.New("db down")}
	svc := NewETAService(primary, newMemStore())

	_, _, err := svc.GetETAForStop(context.Background(), 5)
	if err == nil {
		t.Fatal("expected error when primary fails, got nil")
	}
}

func TestETAService_CacheGetError_FallsBackToPrimary(t *testing.T) {
	store := newMemStore()
	store.getErr = errors.New("db read error")
	primary := &mockETAProvider{seconds: 240, source: "simple"}
	svc := NewETAService(primary, store)

	secs, src, err := svc.GetETAForStop(context.Background(), 6)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if src != "simple" {
		t.Errorf("source = %q, want %q", src, "simple")
	}
	if secs != 240 {
		t.Errorf("seconds = %d, want 240", secs)
	}
	if primary.calls != 1 {
		t.Errorf("primary calls = %d, want 1", primary.calls)
	}
}

func TestETAService_CacheSetError_StillReturnsValue(t *testing.T) {
	store := newMemStore()
	store.setErr = errors.New("db write error")
	primary := &mockETAProvider{seconds: 360, source: "simple"}
	svc := NewETAService(primary, store)

	secs, src, err := svc.GetETAForStop(context.Background(), 7)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if src != "simple" {
		t.Errorf("source = %q, want %q", src, "simple")
	}
	if secs != 360 {
		t.Errorf("seconds = %d, want 360", secs)
	}
}

// ---------------------------------------------------------------------------
// ETAService — fallback behaviour (MVP v2 path)
// ---------------------------------------------------------------------------

func TestETAService_Fallback_UsedWhenPrimaryReturnsNoVehicleData(t *testing.T) {
	primary := &mockETAProvider{err: ErrNoVehicleData}
	fallback := &mockETAProvider{seconds: 200, source: "simple"}
	svc := NewETAServiceWithFallback(primary, fallback, newMemStore())

	secs, src, err := svc.GetETAForStop(context.Background(), 10)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// source should be annotated with "_fallback" suffix.
	if src != "simple_fallback" {
		t.Errorf("source = %q, want %q", src, "simple_fallback")
	}
	if secs != 200 {
		t.Errorf("seconds = %d, want 200", secs)
	}
	if primary.calls != 1 {
		t.Errorf("primary calls = %d, want 1", primary.calls)
	}
	if fallback.calls != 1 {
		t.Errorf("fallback calls = %d, want 1", fallback.calls)
	}
}

func TestETAService_Fallback_NotCalledOnPrimarySuccess(t *testing.T) {
	primary := &mockETAProvider{seconds: 150, source: "gps"}
	fallback := &mockETAProvider{seconds: 999, source: "simple"}
	svc := NewETAServiceWithFallback(primary, fallback, newMemStore())

	secs, src, err := svc.GetETAForStop(context.Background(), 11)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if src != "gps" {
		t.Errorf("source = %q, want %q", src, "gps")
	}
	if secs != 150 {
		t.Errorf("seconds = %d, want 150", secs)
	}
	if fallback.calls != 0 {
		t.Errorf("fallback calls = %d, want 0 (should not be called on primary success)", fallback.calls)
	}
}

func TestETAService_Fallback_ErrorPropagatesWhenFallbackAlsoFails(t *testing.T) {
	primary := &mockETAProvider{err: ErrNoVehicleData}
	fallback := &mockETAProvider{err: errors.New("fallback also down")}
	svc := NewETAServiceWithFallback(primary, fallback, newMemStore())

	_, _, err := svc.GetETAForStop(context.Background(), 12)
	if err == nil {
		t.Fatal("expected error when both primary and fallback fail, got nil")
	}
}

func TestETAService_Fallback_NoVehicleDataWithoutFallbackIsError(t *testing.T) {
	// ErrNoVehicleData from primary with no fallback configured must return error.
	primary := &mockETAProvider{err: ErrNoVehicleData}
	svc := NewETAService(primary, newMemStore())

	_, _, err := svc.GetETAForStop(context.Background(), 13)
	if err == nil {
		t.Fatal("expected error when primary returns ErrNoVehicleData and no fallback is set")
	}
}

func TestETAService_Fallback_ResultCached(t *testing.T) {
	store := newMemStore()
	primary := &mockETAProvider{err: ErrNoVehicleData}
	fallback := &mockETAProvider{seconds: 180, source: "simple"}
	svc := NewETAServiceWithFallback(primary, fallback, store)

	_, _, _ = svc.GetETAForStop(context.Background(), 14)

	// Cache must have been written with the fallback value.
	if store.setCalls != 1 {
		t.Errorf("SetCachedETA calls = %d, want 1", store.setCalls)
	}
	// Second call must be served from cache.
	_, src, _ := svc.GetETAForStop(context.Background(), 14)
	if src != "cache" {
		t.Errorf("second call source = %q, want %q", src, "cache")
	}
}

// ---------------------------------------------------------------------------
// SimpleETAProvider — ETA of bus arriving at stop
// ---------------------------------------------------------------------------

func TestSimpleETAProvider_OffPeak_BasePlusOffset(t *testing.T) {
	// 14:00 is off-peak. stopID=5 → offset=5. Expected: 180+5 = 185s.
	fakeNow := time.Date(2026, 1, 1, 14, 0, 0, 0, time.Local)
	p := NewSimpleETAProvider(withClock(func() time.Time { return fakeNow }))

	secs, src, err := p.GetETA(context.Background(), 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if src != "simple" {
		t.Errorf("source = %q, want %q", src, "simple")
	}
	if secs != 185 {
		t.Errorf("off-peak secs = %d, want 185", secs)
	}
}

func TestSimpleETAProvider_PeakHour_BasePlusOffset(t *testing.T) {
	// 08:00 is peak. stopID=10 → offset=10. Expected: 360+10 = 370s.
	fakeNow := time.Date(2026, 1, 1, 8, 0, 0, 0, time.Local)
	p := NewSimpleETAProvider(withClock(func() time.Time { return fakeNow }))

	secs, _, err := p.GetETA(context.Background(), 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if secs != 370 {
		t.Errorf("peak-hour secs = %d, want 370", secs)
	}
}

func TestSimpleETAProvider_PeakSlowerThanOffPeak(t *testing.T) {
	// Same stopID, peak vs off-peak: peak ETA must be higher.
	peak := time.Date(2026, 1, 1, 7, 0, 0, 0, time.Local)
	offPeak := time.Date(2026, 1, 1, 11, 0, 0, 0, time.Local)

	pPeak := NewSimpleETAProvider(withClock(func() time.Time { return peak }))
	pOff := NewSimpleETAProvider(withClock(func() time.Time { return offPeak }))

	secsPeak, _, _ := pPeak.GetETA(context.Background(), 1)
	secsOff, _, _ := pOff.GetETA(context.Background(), 1)

	if secsPeak <= secsOff {
		t.Errorf("peak ETA (%d) should be > off-peak ETA (%d)", secsPeak, secsOff)
	}
}

func TestSimpleETAProvider_DifferentStopsDifferentETAs(t *testing.T) {
	// Different stopIDs must produce different ETA values (offset = stopID % 60).
	fakeNow := time.Date(2026, 1, 1, 12, 0, 0, 0, time.Local)
	p := NewSimpleETAProvider(withClock(func() time.Time { return fakeNow }))

	s1, _, _ := p.GetETA(context.Background(), 1)
	s2, _, _ := p.GetETA(context.Background(), 2)

	if s1 == s2 {
		t.Errorf("expected different ETAs for stopID=1 (%d) and stopID=2 (%d)", s1, s2)
	}
}

func TestSimpleETAProvider_OffsetWrapsAt60(t *testing.T) {
	// stopID=60 → offset=0, same as stopID=0 (base only).
	// stopID=61 → offset=1, same as stopID=1.
	fakeNow := time.Date(2026, 1, 1, 12, 0, 0, 0, time.Local)
	p := NewSimpleETAProvider(withClock(func() time.Time { return fakeNow }))

	s0, _, _ := p.GetETA(context.Background(), 60)  // 180 + 0
	s1, _, _ := p.GetETA(context.Background(), 1)   // 180 + 1
	s61, _, _ := p.GetETA(context.Background(), 61) // 180 + 1

	if s0 != 180 {
		t.Errorf("stopID=60: secs = %d, want 180", s0)
	}
	if s1 != s61 {
		t.Errorf("stopID=1 (%d) and stopID=61 (%d) should have the same ETA", s1, s61)
	}
}

func TestSimpleETAProvider_CustomOptions(t *testing.T) {
	// Custom: base 60s normal, 120s peak; peak hours = {10}.
	// At 10:00 with stopID=5: 120 + 5 = 125s.
	fakeNow := time.Date(2026, 1, 1, 10, 0, 0, 0, time.Local)
	p := NewSimpleETAProvider(
		WithBaseETAs(60, 120),
		WithPeakHours(10),
		withClock(func() time.Time { return fakeNow }),
	)

	secs, _, err := p.GetETA(context.Background(), 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if secs != 125 {
		t.Errorf("custom peak secs = %d, want 125", secs)
	}
}
