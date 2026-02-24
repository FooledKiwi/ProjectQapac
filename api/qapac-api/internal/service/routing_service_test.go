package service

import (
	"context"
	"errors"
	"testing"

	"github.com/FooledKiwi/ProjectQapac/api/qapac-api/internal/routing"
	"github.com/FooledKiwi/ProjectQapac/api/qapac-api/internal/storage"
)

// --- mock StopsRepository ---

type mockStopsRepo struct {
	stop *storage.Stop
	err  error
}

func (m *mockStopsRepo) FindStopsNear(_ context.Context, _, _, _ float64) ([]storage.Stop, error) {
	return nil, nil
}

func (m *mockStopsRepo) GetStop(_ context.Context, _ int32) (*storage.Stop, error) {
	return m.stop, m.err
}

// --- mock Router ---

type mockRouter struct {
	resp  *routing.RoutingResponse
	err   error
	calls int
}

func (m *mockRouter) Route(_ context.Context, _ routing.RoutingRequest) (*routing.RoutingResponse, error) {
	m.calls++
	return m.resp, m.err
}

// --- tests ---

func TestRoutingService_GetRouteTo_StopNotFound(t *testing.T) {
	svc := NewRoutingService(
		&mockRouter{resp: &routing.RoutingResponse{}},
		&mockStopsRepo{stop: nil, err: nil},
	)
	_, err := svc.GetRouteTo(context.Background(), -12.0464, -77.0428, 99)
	if err == nil {
		t.Fatal("expected error when stop not found, got nil")
	}
}

func TestRoutingService_GetRouteTo_StopFetchError(t *testing.T) {
	svc := NewRoutingService(
		&mockRouter{resp: &routing.RoutingResponse{}},
		&mockStopsRepo{stop: nil, err: errors.New("db error")},
	)
	_, err := svc.GetRouteTo(context.Background(), -12.0464, -77.0428, 1)
	if err == nil {
		t.Fatal("expected error on store failure, got nil")
	}
}

func TestRoutingService_GetRouteTo_Success(t *testing.T) {
	stop := &storage.Stop{ID: 5, Name: "Centro", Lat: -12.055, Lon: -77.053}
	routerResp := &routing.RoutingResponse{Polyline: "encodedABC", DistanceM: 1200, DurationS: 300}

	inner := &mockRouter{resp: routerResp}
	svc := NewRoutingService(inner, &mockStopsRepo{stop: stop})

	got, err := svc.GetRouteTo(context.Background(), -12.0464, -77.0428, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Polyline != "encodedABC" {
		t.Errorf("polyline = %q, want %q", got.Polyline, "encodedABC")
	}
	if got.DistanceM != 1200 {
		t.Errorf("distance = %d, want 1200", got.DistanceM)
	}
	if inner.calls != 1 {
		t.Errorf("inner router called %d times, want 1", inner.calls)
	}
}

func TestRoutingService_GetRouteTo_RouterError(t *testing.T) {
	stop := &storage.Stop{ID: 5, Name: "Centro", Lat: -12.055, Lon: -77.053}
	inner := &mockRouter{err: errors.New("google unreachable")}
	svc := NewRoutingService(inner, &mockStopsRepo{stop: stop})

	_, err := svc.GetRouteTo(context.Background(), -12.0464, -77.0428, 5)
	if err == nil {
		t.Fatal("expected error when router fails, got nil")
	}
}

func TestRoutingService_GetRouteTo_StopIDInContext(t *testing.T) {
	// Verify that WithStopID is called so that CachedRouter can build the right key.
	// We indirectly test this by wrapping in a spy router that checks the context.
	stop := &storage.Stop{ID: 7, Name: "Miraflores", Lat: -18.067, Lon: -70.232}

	var capturedCtx context.Context
	spy := &spyRouter{fn: func(ctx context.Context, req routing.RoutingRequest) (*routing.RoutingResponse, error) {
		capturedCtx = ctx
		return &routing.RoutingResponse{Polyline: "poly", DistanceM: 500, DurationS: 100}, nil
	}}

	svc := NewRoutingService(spy, &mockStopsRepo{stop: stop})
	_, err := svc.GetRouteTo(context.Background(), -12.0464, -77.0428, 7)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The stop ID must have been embedded in the context.
	id, ok := routing.StopIDFromContext(capturedCtx)
	if !ok {
		t.Fatal("stop ID not found in context passed to router")
	}
	if id != 7 {
		t.Errorf("context stop ID = %d, want 7", id)
	}
}

// spyRouter calls a user-supplied function.
type spyRouter struct {
	fn func(context.Context, routing.RoutingRequest) (*routing.RoutingResponse, error)
}

func (s *spyRouter) Route(ctx context.Context, req routing.RoutingRequest) (*routing.RoutingResponse, error) {
	return s.fn(ctx, req)
}
