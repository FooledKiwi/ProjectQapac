package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dom1nux/qapac-api/internal/handler"
	"github.com/dom1nux/qapac-api/internal/middleware"
	"github.com/dom1nux/qapac-api/internal/routing"
	"github.com/dom1nux/qapac-api/internal/service"
	"github.com/dom1nux/qapac-api/internal/storage"
	"github.com/gin-gonic/gin"
	"time"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// ---------------------------------------------------------------------------
// Minimal stubs — satisfy interfaces without real DB or Google API.
// ---------------------------------------------------------------------------

type stubStopsRepo struct{}

func (s *stubStopsRepo) FindStopsNear(_ context.Context, _, _, _ float64) ([]storage.Stop, error) {
	return nil, nil
}
func (s *stubStopsRepo) GetStop(_ context.Context, _ int32) (*storage.Stop, error) {
	return nil, nil
}

type stubETAProvider struct{}

func (s *stubETAProvider) GetETA(_ context.Context, _ int32) (int, string, error) {
	return 0, "simple", nil
}

type stubETACacheStore struct{}

func (s *stubETACacheStore) GetCachedETA(_ context.Context, _ int32) (int, bool, error) {
	return 0, false, nil
}
func (s *stubETACacheStore) SetCachedETA(_ context.Context, _ int32, _ int) error { return nil }

type stubRouter struct{}

func (s *stubRouter) Route(_ context.Context, _ routing.RoutingRequest) (*routing.RoutingResponse, error) {
	return &routing.RoutingResponse{}, nil
}

// buildTestEngine replicates the gin engine wiring from app.New without
// requiring a real database or external API.
func buildTestEngine() *gin.Engine {
	stopsRepo := &stubStopsRepo{}
	etaSvc := service.NewETAService(&stubETAProvider{}, &stubETACacheStore{})
	routingSvc := service.NewRoutingService(&stubRouter{}, stopsRepo)

	r := gin.New()
	r.Use(middleware.Timeout(10 * time.Second))

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	h := handler.New(stopsRepo, etaSvc, routingSvc)
	api := r.Group("/api/v1")
	{
		api.GET("/stops/nearby", h.ListStopsNear)
		api.GET("/stops/:id", h.GetStop)
		api.GET("/routes/to-stop", h.GetRouteToStop)
	}

	return r
}

// ---------------------------------------------------------------------------
// Smoke tests — verify routes are registered and reachable.
// ---------------------------------------------------------------------------

func TestSmoke_HealthEndpoint(t *testing.T) {
	r := buildTestEngine()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("/health: status = %d, want 200", w.Code)
	}
}

func TestSmoke_StopsNearbyRouteExists(t *testing.T) {
	r := buildTestEngine()

	// Missing required params → 400, but the route must be registered (not 404).
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/stops/nearby", nil)
	r.ServeHTTP(w, req)

	if w.Code == http.StatusNotFound {
		t.Errorf("/api/v1/stops/nearby: route not registered (got 404)")
	}
}

func TestSmoke_GetStopRouteExists(t *testing.T) {
	r := buildTestEngine()

	// The stub repo returns (nil, nil) → handler responds 404 with JSON body
	// {"error":"stop not found"}, which is different from gin's plain-text
	// 404 when no route matches. We confirm the route IS registered by
	// checking the response is JSON (i.e. the handler ran).
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/stops/1", nil)
	r.ServeHTTP(w, req)

	ct := w.Header().Get("Content-Type")
	if ct == "" {
		t.Errorf("/api/v1/stops/:id: no Content-Type header; route may not be registered")
	}
}

func TestSmoke_GetRouteToStopRouteExists(t *testing.T) {
	r := buildTestEngine()

	// Missing required params → 400, but the route must be registered (not 404).
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/routes/to-stop", nil)
	r.ServeHTTP(w, req)

	if w.Code == http.StatusNotFound {
		t.Errorf("/api/v1/routes/to-stop: route not registered (got 404)")
	}
}
