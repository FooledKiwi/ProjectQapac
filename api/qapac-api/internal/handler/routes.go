package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/FooledKiwi/ProjectQapac/api/qapac-api/internal/service"
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
//
// # Polyline encoding
//
// The "polyline" field contains a string encoded with Google's Encoded Polyline
// Algorithm Format (https://developers.google.com/maps/documentation/utilities/polylinealgorithm).
// Each coordinate pair is encoded at 1e-5 precision (5 decimal places).
// This is the value returned verbatim from the Google Routes API v2
// `routes.polyline.encodedPolyline` field.
//
// When is_fallback is true the route could not be computed (Google API
// unavailable) and polyline will be an empty string — only distance_m and
// duration_s (straight-line estimates) are reliable in that case.
//
// # Android decoding
//
// Add the Maps SDK utility library to your build.gradle.kts:
//
//	implementation("com.google.maps.android:android-maps-utils:3.8.2")
//
// Then decode the polyline into a list of LatLng points and draw it on a
// GoogleMap:
//
//	import com.google.maps.android.PolyUtil
//	import com.google.android.gms.maps.model.LatLng
//	import com.google.android.gms.maps.model.PolylineOptions
//
//	val encoded: String = response.polyline   // from the JSON response
//	if (encoded.isNotEmpty()) {
//	    val points: List<LatLng> = PolyUtil.decode(encoded)
//	    googleMap.addPolyline(
//	        PolylineOptions()
//	            .addAll(points)
//	            .width(8f)
//	            .color(android.graphics.Color.BLUE)
//	    )
//	}
//
// PolyUtil.decode() returns a List<LatLng> where each element is a WGS-84
// coordinate pair. The list preserves the route direction (origin → destination)
// so it can be passed directly to PolylineOptions.addAll().
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
