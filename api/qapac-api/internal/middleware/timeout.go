package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Timeout returns a Gin middleware that attaches a deadline to the request
// context. It runs the handler chain synchronously (no goroutine spawning),
// which keeps gin.Context access single-threaded and avoids goroutine leaks.
//
// How it works:
//   - Before c.Next(), the context is replaced with one that has a deadline.
//   - After c.Next() returns, if the context expired AND no response has been
//     written yet, a 503 is sent. In practice this happens when a handler
//     returns early without writing (e.g. after detecting ctx.Err() != nil
//     in a select branch that doesn't call c.JSON).
//
// Limitation: this design cannot interrupt a handler that is blocked and does
// not check its context. That is an acceptable trade-off for MVP: all storage
// and routing calls propagate the context and will unblock when the deadline
// fires at the DB / HTTP level.
func Timeout(d time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), d)
		defer cancel()

		// Replace the request context so all downstream code sees the deadline.
		c.Request = c.Request.WithContext(ctx)

		c.Next()

		// If the deadline fired and the handler did not write a response, send
		// a 503. This covers the case where a handler exits via ctx.Done() in a
		// select without calling c.JSON/c.AbortWithStatus.
		if ctx.Err() != nil && !c.Writer.Written() {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
				"error": "request timed out",
			})
		}
	}
}
