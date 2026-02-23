package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/dom1nux/qapac-api/internal/service"
	"github.com/gin-gonic/gin"
)

// GetRouteToStop handles GET /api/v1/routes/to-stop
//
// Query params:
//   - lat     (required) float64 — user's WGS-84 latitude
//   - lon     (required) float64 — user's WGS-84 longitude
//   - stop_id (required) int32   — destination stop identifier
//
// Response 200:
//
//	{"polyline":"...","distance_m":500,"duration_s":600,"is_fallback":false}
//
// Response 400: missing or invalid query parameters.
// Response 404: stop does not exist.
// Response 500: routing error.
func (h *Handler) GetRouteToStop(c *gin.Context) {
	lat, ok := parseRequiredFloat(c, "lat")
	if !ok {
		return
	}

	lon, ok := parseRequiredFloat(c, "lon")
	if !ok {
		return
	}

	stopIDRaw := c.Query("stop_id")
	if stopIDRaw == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "stop_id query parameter is required"})
		return
	}
	stopID64, err := strconv.ParseInt(stopIDRaw, 10, 32)
	if err != nil || stopID64 <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "stop_id must be a positive integer"})
		return
	}
	stopID := int32(stopID64)

	resp, err := h.routingService.GetRouteTo(c.Request.Context(), lat, lon, stopID)
	if err != nil {
		if errors.Is(err, service.ErrStopNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "stop not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to calculate route"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"polyline":    resp.Polyline,
		"distance_m":  resp.DistanceM,
		"duration_s":  resp.DurationS,
		"is_fallback": resp.IsFallback,
	})
}
