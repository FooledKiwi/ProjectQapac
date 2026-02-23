package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dom1nux/qapac-api/internal/routing"
	"github.com/dom1nux/qapac-api/internal/service"
	"github.com/dom1nux/qapac-api/internal/storage"
	"github.com/gin-gonic/gin"
)

func init() {
	// Suppress gin debug output in tests.
	gin.SetMode(gin.TestMode)
}

// ---------------------------------------------------------------------------
// Test doubles
// ---------------------------------------------------------------------------

type mockStopsRepo struct {
	findResult []storage.Stop
	findErr    error
	getResult  *storage.Stop
	getErr     error
}

func (m *mockStopsRepo) FindStopsNear(_ context.Context, _, _, _ float64) ([]storage.Stop, error) {
	return m.findResult, m.findErr
}

func (m *mockStopsRepo) GetStop(_ context.Context, _ int32) (*storage.Stop, error) {
	return m.getResult, m.getErr
}

// mockETAProvider satisfies service.ETAProvider.
type mockETAProvider struct {
	seconds int
	source  string
	err     error
}

func (m *mockETAProvider) GetETA(_ context.Context, _ int32) (int, string, error) {
	return m.seconds, m.source, m.err
}

// mockETACacheStore satisfies service.ETACacheStore; always misses.
type mockETACacheStore struct{}

func (m *mockETACacheStore) GetCachedETA(_ context.Context, _ int32) (int, bool, error) {
	return 0, false, nil
}

func (m *mockETACacheStore) SetCachedETA(_ context.Context, _ int32, _ int) error {
	return nil
}

// mockRoutingService is a thin wrapper so we can control GetRouteTo responses.
type mockRoutingServiceRouter struct {
	resp *routing.RoutingResponse
	err  error
}

func (m *mockRoutingServiceRouter) Route(_ context.Context, _ routing.RoutingRequest) (*routing.RoutingResponse, error) {
	return m.resp, m.err
}

// newTestHandler builds a Handler backed by the given test doubles.
// etaSeconds is what the ETA provider will return.
func newTestHandler(
	stopsRepo storage.StopsRepository,
	etaProvider service.ETAProvider,
	routingRouter routing.Router,
	stopsRepoForRouting storage.StopsRepository,
) *Handler {
	etaSvc := service.NewETAService(etaProvider, &mockETACacheStore{})
	routingSvc := service.NewRoutingService(routingRouter, stopsRepoForRouting)
	return New(stopsRepo, etaSvc, routingSvc)
}

// newRouter builds a minimal gin engine with the handler routes registered.
func newRouter(h *Handler) *gin.Engine {
	r := gin.New()
	api := r.Group("/api/v1")
	api.GET("/stops/nearby", h.ListStopsNear)
	api.GET("/stops/:id", h.GetStop)
	api.GET("/routes/to-stop", h.GetRouteToStop)
	return r
}

// ---------------------------------------------------------------------------
// ListStopsNear tests
// ---------------------------------------------------------------------------

func TestListStopsNear_MissingLat(t *testing.T) {
	h := newTestHandler(&mockStopsRepo{}, &mockETAProvider{}, &mockRoutingServiceRouter{}, &mockStopsRepo{})
	r := newRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/stops/nearby?lon=-77.04", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestListStopsNear_MissingLon(t *testing.T) {
	h := newTestHandler(&mockStopsRepo{}, &mockETAProvider{}, &mockRoutingServiceRouter{}, &mockStopsRepo{})
	r := newRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/stops/nearby?lat=-12.05", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestListStopsNear_InvalidLat(t *testing.T) {
	h := newTestHandler(&mockStopsRepo{}, &mockETAProvider{}, &mockRoutingServiceRouter{}, &mockStopsRepo{})
	r := newRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/stops/nearby?lat=notanumber&lon=-77.04", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestListStopsNear_InvalidRadius(t *testing.T) {
	h := newTestHandler(&mockStopsRepo{}, &mockETAProvider{}, &mockRoutingServiceRouter{}, &mockStopsRepo{})
	r := newRouter(h)

	for _, tc := range []struct {
		name   string
		radius string
	}{
		{"negative", "-5"},
		{"zero", "0"},
		{"exceeds cap", "50001"},
	} {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/stops/nearby?lat=-12.05&lon=-77.04&radius="+tc.radius, nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want 400", tc.name, w.Code)
		}
	}
}

func TestListStopsNear_StorageError(t *testing.T) {
	repo := &mockStopsRepo{findErr: errors.New("db unreachable")}
	h := newTestHandler(repo, &mockETAProvider{}, &mockRoutingServiceRouter{}, &mockStopsRepo{})
	r := newRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/stops/nearby?lat=-12.05&lon=-77.04", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", w.Code)
	}
}

func TestListStopsNear_EmptyResult(t *testing.T) {
	repo := &mockStopsRepo{findResult: []storage.Stop{}}
	h := newTestHandler(repo, &mockETAProvider{}, &mockRoutingServiceRouter{}, &mockStopsRepo{})
	r := newRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/stops/nearby?lat=-12.05&lon=-77.04", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	var result []interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty array, got %d items", len(result))
	}
}

func TestListStopsNear_Success(t *testing.T) {
	stops := []storage.Stop{
		{ID: 1, Name: "Paradero Centro", Lat: -12.05, Lon: -77.04},
		{ID: 2, Name: "Paradero Norte", Lat: -12.03, Lon: -77.02},
	}
	repo := &mockStopsRepo{findResult: stops}
	h := newTestHandler(repo, &mockETAProvider{}, &mockRoutingServiceRouter{}, &mockStopsRepo{})
	r := newRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/stops/nearby?lat=-12.05&lon=-77.04&radius=500", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}

	var result []map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 stops, got %d", len(result))
	}
	if result[0]["name"] != "Paradero Centro" {
		t.Errorf("first stop name = %q, want %q", result[0]["name"], "Paradero Centro")
	}
	// JSON numbers decode to float64.
	if result[0]["id"].(float64) != 1 {
		t.Errorf("first stop id = %v, want 1", result[0]["id"])
	}
}

func TestListStopsNear_DefaultRadius(t *testing.T) {
	// Verify the handler accepts requests without a radius param (uses default 1000m).
	repo := &mockStopsRepo{findResult: []storage.Stop{}}
	h := newTestHandler(repo, &mockETAProvider{}, &mockRoutingServiceRouter{}, &mockStopsRepo{})
	r := newRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/stops/nearby?lat=-12.05&lon=-77.04", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

// ---------------------------------------------------------------------------
// GetStop tests
// ---------------------------------------------------------------------------

func TestGetStop_InvalidID_NonInteger(t *testing.T) {
	h := newTestHandler(&mockStopsRepo{}, &mockETAProvider{}, &mockRoutingServiceRouter{}, &mockStopsRepo{})
	r := newRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/stops/abc", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestGetStop_InvalidID_Zero(t *testing.T) {
	h := newTestHandler(&mockStopsRepo{}, &mockETAProvider{}, &mockRoutingServiceRouter{}, &mockStopsRepo{})
	r := newRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/stops/0", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestGetStop_NotFound(t *testing.T) {
	repo := &mockStopsRepo{getResult: nil, getErr: nil}
	h := newTestHandler(repo, &mockETAProvider{}, &mockRoutingServiceRouter{}, &mockStopsRepo{})
	r := newRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/stops/99", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

func TestGetStop_StorageError(t *testing.T) {
	repo := &mockStopsRepo{getErr: errors.New("db error")}
	h := newTestHandler(repo, &mockETAProvider{}, &mockRoutingServiceRouter{}, &mockStopsRepo{})
	r := newRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/stops/1", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", w.Code)
	}
}

func TestGetStop_Success(t *testing.T) {
	stop := &storage.Stop{ID: 3, Name: "Miraflores", Lat: -18.067, Lon: -70.232}
	repo := &mockStopsRepo{getResult: stop}
	etaProv := &mockETAProvider{seconds: 240, source: "simple"}
	h := newTestHandler(repo, etaProv, &mockRoutingServiceRouter{}, &mockStopsRepo{})
	r := newRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/stops/3", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if result["name"] != "Miraflores" {
		t.Errorf("name = %q, want %q", result["name"], "Miraflores")
	}
	if result["eta_seconds"].(float64) != 240 {
		t.Errorf("eta_seconds = %v, want 240", result["eta_seconds"])
	}
}

func TestGetStop_ETAErrorNonFatal(t *testing.T) {
	// When ETA fails, the stop data should still be returned with eta_seconds = 0.
	stop := &storage.Stop{ID: 4, Name: "Centro", Lat: -12.05, Lon: -77.04}
	repo := &mockStopsRepo{getResult: stop}
	etaProv := &mockETAProvider{err: errors.New("eta provider down")}
	h := newTestHandler(repo, etaProv, &mockRoutingServiceRouter{}, &mockStopsRepo{})
	r := newRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/stops/4", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if result["eta_seconds"].(float64) != 0 {
		t.Errorf("eta_seconds = %v, want 0 when ETA fails", result["eta_seconds"])
	}
}

// ---------------------------------------------------------------------------
// GetRouteToStop tests
// ---------------------------------------------------------------------------

func TestGetRouteToStop_MissingLat(t *testing.T) {
	h := newTestHandler(&mockStopsRepo{}, &mockETAProvider{}, &mockRoutingServiceRouter{}, &mockStopsRepo{})
	r := newRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/routes/to-stop?lon=-77.04&stop_id=1", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestGetRouteToStop_MissingLon(t *testing.T) {
	h := newTestHandler(&mockStopsRepo{}, &mockETAProvider{}, &mockRoutingServiceRouter{}, &mockStopsRepo{})
	r := newRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/routes/to-stop?lat=-12.05&stop_id=1", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestGetRouteToStop_MissingStopID(t *testing.T) {
	h := newTestHandler(&mockStopsRepo{}, &mockETAProvider{}, &mockRoutingServiceRouter{}, &mockStopsRepo{})
	r := newRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/routes/to-stop?lat=-12.05&lon=-77.04", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestGetRouteToStop_InvalidStopID(t *testing.T) {
	h := newTestHandler(&mockStopsRepo{}, &mockETAProvider{}, &mockRoutingServiceRouter{}, &mockStopsRepo{})
	r := newRouter(h)

	for _, id := range []string{"abc", "0", "-1"} {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/routes/to-stop?lat=-12.05&lon=-77.04&stop_id="+id, nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("stop_id=%q: status = %d, want 400", id, w.Code)
		}
	}
}

func TestGetRouteToStop_StopNotFound(t *testing.T) {
	// When the stop does not exist RoutingService wraps ErrStopNotFound;
	// the handler must translate that into a 404.
	stopsForRouting := &mockStopsRepo{getResult: nil, getErr: nil} // GetStop returns (nil, nil)
	h := newTestHandler(&mockStopsRepo{}, &mockETAProvider{}, &mockRoutingServiceRouter{}, stopsForRouting)
	r := newRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/routes/to-stop?lat=-12.05&lon=-77.04&stop_id=99", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

func TestGetRouteToStop_RouterError(t *testing.T) {
	stop := &storage.Stop{ID: 5, Name: "Centro", Lat: -12.05, Lon: -77.04}
	routingRouter := &mockRoutingServiceRouter{err: errors.New("google API down")}
	stopsForRouting := &mockStopsRepo{getResult: stop}
	h := newTestHandler(&mockStopsRepo{}, &mockETAProvider{}, routingRouter, stopsForRouting)
	r := newRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/routes/to-stop?lat=-12.05&lon=-77.04&stop_id=5", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", w.Code)
	}
}

func TestGetRouteToStop_Success(t *testing.T) {
	stop := &storage.Stop{ID: 5, Name: "Centro", Lat: -12.05, Lon: -77.04}
	routingResp := &routing.RoutingResponse{
		Polyline:   "encodedPoly",
		DistanceM:  800,
		DurationS:  420,
		IsFallback: false,
	}
	routingRouter := &mockRoutingServiceRouter{resp: routingResp}
	stopsForRouting := &mockStopsRepo{getResult: stop}
	h := newTestHandler(&mockStopsRepo{}, &mockETAProvider{}, routingRouter, stopsForRouting)
	r := newRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/routes/to-stop?lat=-12.05&lon=-77.04&stop_id=5", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if result["polyline"] != "encodedPoly" {
		t.Errorf("polyline = %q, want %q", result["polyline"], "encodedPoly")
	}
	if result["distance_m"].(float64) != 800 {
		t.Errorf("distance_m = %v, want 800", result["distance_m"])
	}
	if result["duration_s"].(float64) != 420 {
		t.Errorf("duration_s = %v, want 420", result["duration_s"])
	}
	if result["is_fallback"].(bool) != false {
		t.Errorf("is_fallback = %v, want false", result["is_fallback"])
	}
}

func TestGetRouteToStop_FallbackExposed(t *testing.T) {
	// When the router uses the straight-line fallback, is_fallback must be true in the response.
	stop := &storage.Stop{ID: 5, Name: "Centro", Lat: -12.05, Lon: -77.04}
	routingResp := &routing.RoutingResponse{
		Polyline:   "",
		DistanceM:  750,
		DurationS:  90,
		IsFallback: true,
	}
	routingRouter := &mockRoutingServiceRouter{resp: routingResp}
	stopsForRouting := &mockStopsRepo{getResult: stop}
	h := newTestHandler(&mockStopsRepo{}, &mockETAProvider{}, routingRouter, stopsForRouting)
	r := newRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/routes/to-stop?lat=-12.05&lon=-77.04&stop_id=5", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if result["is_fallback"].(bool) != true {
		t.Errorf("is_fallback = %v, want true", result["is_fallback"])
	}
	if result["polyline"] != "" {
		t.Errorf("polyline = %q, want empty string for fallback", result["polyline"])
	}
}
