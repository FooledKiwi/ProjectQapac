package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func newTimeoutRouter(d time.Duration, handler gin.HandlerFunc) *gin.Engine {
	r := gin.New()
	r.Use(Timeout(d))
	r.GET("/test", handler)
	return r
}

func TestTimeout_HandlerCompletesInTime(t *testing.T) {
	r := newTimeoutRouter(100*time.Millisecond, func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

func TestTimeout_ContextHasDeadline(t *testing.T) {
	// The middleware must attach a deadline to the request context.
	r := newTimeoutRouter(500*time.Millisecond, func(c *gin.Context) {
		_, ok := c.Request.Context().Deadline()
		if !ok {
			t.Error("context has no deadline; middleware did not set one")
		}
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)
}

func TestTimeout_503WhenHandlerExitsWithoutWriting(t *testing.T) {
	// Use a very short timeout. The handler sleeps past the deadline, then
	// returns without writing. The middleware detects ctx expiry and writes 503.
	r := newTimeoutRouter(5*time.Millisecond, func(c *gin.Context) {
		// Wait until the middleware's context expires.
		<-c.Request.Context().Done()
		// Return without writing a response — middleware should write 503.
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", w.Code)
	}
}

func TestTimeout_HandlerResponseNotOverwritten(t *testing.T) {
	// When the handler writes a response, the middleware must not overwrite it
	// even if the context deadline subsequently expires.
	r := newTimeoutRouter(5*time.Millisecond, func(c *gin.Context) {
		c.JSON(http.StatusAccepted, gin.H{"done": true})
		// Simulate slow post-response work that runs past the deadline.
		time.Sleep(20 * time.Millisecond)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Errorf("status = %d, want 202 (handler response must not be overwritten)", w.Code)
	}
}

func TestTimeout_DeadlineIsRespectedByStorage(t *testing.T) {
	// End-to-end: confirm that the context deadline propagates to storage-style
	// code that respects context cancellation.
	r := newTimeoutRouter(10*time.Millisecond, func(c *gin.Context) {
		ctx := c.Request.Context()

		// Simulate a storage call that blocks but respects context.
		done := make(chan struct{})
		go func() {
			time.Sleep(200 * time.Millisecond)
			close(done)
		}()

		select {
		case <-ctx.Done():
			// Deadline hit — handler exits without writing, as expected.
			return
		case <-done:
			c.JSON(http.StatusOK, gin.H{"ok": true})
		}
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", w.Code)
	}
}

func TestTimeout_PreCancelledContext(t *testing.T) {
	// If the incoming request already carries a cancelled context the middleware
	// should propagate cancellation cleanly.
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already cancelled

	r := newTimeoutRouter(100*time.Millisecond, func(c *gin.Context) {
		if c.Request.Context().Err() == nil {
			t.Error("expected cancelled context, got nil error")
		}
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil).WithContext(ctx)
	r.ServeHTTP(w, req)
}
