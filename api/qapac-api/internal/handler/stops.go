package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

const defaultRadiusMeters = 1000.0
const maxRadiusMeters = 50_000.0

// ListStopsNear handles GET /api/v1/stops/nearby
//
// Query params:
//   - lat    (required) float64 — WGS-84 latitude
//   - lon    (required) float64 — WGS-84 longitude
//   - radius (optional) float64 — search radius in metres; default 1000
//
// Response 200:
//
//	[{"id":1,"name":"Paradero Centro","lat":-12.123,"lon":-76.456}]
//
// Response 400: missing or invalid query parameters.
// Response 500: storage error.
func (h *Handler) ListStopsNear(c *gin.Context) {
	lat, ok := parseRequiredFloat(c, "lat")
	if !ok {
		return
	}

	lon, ok := parseRequiredFloat(c, "lon")
	if !ok {
		return
	}

	radius := defaultRadiusMeters
	if raw := c.Query("radius"); raw != "" {
		v, err := strconv.ParseFloat(raw, 64)
		if err != nil || v <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "radius must be a positive number"})
			return
		}
		if v > maxRadiusMeters {
			c.JSON(http.StatusBadRequest, gin.H{"error": "radius must not exceed 50000 metres"})
			return
		}
		radius = v
	}

	stops, err := h.stopsRepo.FindStopsNear(c.Request.Context(), lat, lon, radius)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query stops"})
		return
	}

	type stopJSON struct {
		ID   int32   `json:"id"`
		Name string  `json:"name"`
		Lat  float64 `json:"lat"`
		Lon  float64 `json:"lon"`
	}

	out := make([]stopJSON, len(stops))
	for i, s := range stops {
		out[i] = stopJSON{ID: s.ID, Name: s.Name, Lat: s.Lat, Lon: s.Lon}
	}

	c.JSON(http.StatusOK, out)
}

// GetStop handles GET /api/v1/stops/:id
//
// Path param:
//   - id (required) int32 — stop identifier
//
// Response 200:
//
//	{"id":1,"name":"Paradero Centro","lat":-12.123,"lon":-76.456,"eta_seconds":300}
//
// Response 400: id is not a valid integer.
// Response 404: stop does not exist.
// Response 500: storage or ETA error.
func (h *Handler) GetStop(c *gin.Context) {
	idRaw := c.Param("id")
	id64, err := strconv.ParseInt(idRaw, 10, 32)
	if err != nil || id64 <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id must be a positive integer"})
		return
	}
	id := int32(id64)

	stop, err := h.stopsRepo.GetStop(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query stop"})
		return
	}
	if stop == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "stop not found"})
		return
	}

	etaSecs, _, _ := h.etaService.GetETAForStop(c.Request.Context(), id) //nolint:errcheck // ETA errors are non-fatal: we still return stop data with eta_seconds = 0.

	c.JSON(http.StatusOK, gin.H{
		"id":          stop.ID,
		"name":        stop.Name,
		"lat":         stop.Lat,
		"lon":         stop.Lon,
		"eta_seconds": etaSecs,
	})
}

// parseRequiredFloat extracts a required float64 query parameter.
// On failure it writes a 400 response and returns (0, false).
func parseRequiredFloat(c *gin.Context, name string) (float64, bool) {
	raw := c.Query(name)
	if raw == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": name + " query parameter is required"})
		return 0, false
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": name + " must be a valid number"})
		return 0, false
	}
	return v, true
}
