package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/FooledKiwi/ProjectQapac/api/qapac-api/internal/storage"
	"github.com/gin-gonic/gin"
)

// PublicHandler holds dependencies for unauthenticated public endpoints.
type PublicHandler struct {
	routesRepo    storage.PublicRoutesRepository
	positionsRepo storage.VehiclePositionsRepository
	alertsRepo    storage.AlertsRepository
	ratingsRepo   storage.RatingsRepository
	favoritesRepo storage.FavoritesRepository
}

// NewPublicHandler creates a PublicHandler with the given repositories.
func NewPublicHandler(
	routesRepo storage.PublicRoutesRepository,
	positionsRepo storage.VehiclePositionsRepository,
	alertsRepo storage.AlertsRepository,
	ratingsRepo storage.RatingsRepository,
	favoritesRepo storage.FavoritesRepository,
) *PublicHandler {
	return &PublicHandler{
		routesRepo:    routesRepo,
		positionsRepo: positionsRepo,
		alertsRepo:    alertsRepo,
		ratingsRepo:   ratingsRepo,
		favoritesRepo: favoritesRepo,
	}
}

// ---------------------------------------------------------------------------
// Routes
// ---------------------------------------------------------------------------

// ListRoutes handles GET /api/v1/routes
func (h *PublicHandler) ListRoutes(c *gin.Context) {
	routes, err := h.routesRepo.ListRoutes(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list routes"})
		return
	}

	out := make([]gin.H, len(routes))
	for i, r := range routes {
		out[i] = gin.H{
			"id":            r.ID,
			"name":          r.Name,
			"active":        r.Active,
			"vehicle_count": r.VehicleCount,
		}
	}

	c.JSON(http.StatusOK, out)
}

// GetRoute handles GET /api/v1/routes/:id
func (h *PublicHandler) GetRoute(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	detail, err := h.routesRepo.GetRouteDetail(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query route"})
		return
	}
	if detail == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "route not found"})
		return
	}

	stops := make([]gin.H, len(detail.Stops))
	for i, s := range detail.Stops {
		stops[i] = gin.H{
			"id":       s.ID,
			"name":     s.Name,
			"lat":      s.Lat,
			"lon":      s.Lon,
			"sequence": s.Sequence,
		}
	}

	vehicles := make([]gin.H, len(detail.Vehicles))
	for i, v := range detail.Vehicles {
		vehicles[i] = gin.H{
			"id":        v.ID,
			"plate":     v.PlateNumber,
			"driver":    v.DriverName,
			"collector": v.CollectorName,
			"status":    v.Status,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"id":             detail.ID,
		"name":           detail.Name,
		"active":         detail.Active,
		"stops":          stops,
		"vehicles":       vehicles,
		"shape_polyline": detail.ShapePolyline,
	})
}

// GetRouteVehicles handles GET /api/v1/routes/:id/vehicles
func (h *PublicHandler) GetRouteVehicles(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	vehicles, err := h.routesRepo.GetRouteVehiclesWithPositions(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query route vehicles"})
		return
	}

	out := make([]gin.H, len(vehicles))
	for i, v := range vehicles {
		entry := gin.H{
			"id":             v.ID,
			"plate":          v.PlateNumber,
			"driver_name":    v.DriverName,
			"collector_name": v.CollectorName,
			"status":         v.Status,
		}
		if v.Position != nil {
			entry["position"] = gin.H{
				"lat":         v.Position.Lat,
				"lon":         v.Position.Lon,
				"heading":     v.Position.Heading,
				"speed":       v.Position.Speed,
				"recorded_at": v.Position.RecordedAt,
			}
		}
		out[i] = entry
	}

	c.JSON(http.StatusOK, out)
}

// ---------------------------------------------------------------------------
// Vehicle Positions
// ---------------------------------------------------------------------------

// GetVehiclePosition handles GET /api/v1/vehicles/:id/position
func (h *PublicHandler) GetVehiclePosition(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	pos, err := h.positionsRepo.GetPosition(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query vehicle position"})
		return
	}
	if pos == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no position recorded for this vehicle"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"lat":         pos.Lat,
		"lon":         pos.Lon,
		"heading":     pos.Heading,
		"speed":       pos.Speed,
		"recorded_at": pos.RecordedAt,
	})
}

// NearbyVehicles handles GET /api/v1/vehicles/nearby
func (h *PublicHandler) NearbyVehicles(c *gin.Context) {
	lat, ok := parseRequiredFloat(c, "lat")
	if !ok {
		return
	}
	lon, ok := parseRequiredFloat(c, "lon")
	if !ok {
		return
	}

	radius := 1000.0 // default 1 km
	if raw := c.Query("radius"); raw != "" {
		v, err := strconv.ParseFloat(raw, 64)
		if err != nil || v <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "radius must be a positive number"})
			return
		}
		if v > 50000 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "radius must not exceed 50000 metres"})
			return
		}
		radius = v
	}

	vehicles, err := h.positionsRepo.FindNearby(c.Request.Context(), lat, lon, radius)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query nearby vehicles"})
		return
	}

	out := make([]gin.H, len(vehicles))
	for i, v := range vehicles {
		out[i] = gin.H{
			"id":         v.ID,
			"plate":      v.PlateNumber,
			"route_name": v.RouteName,
			"lat":        v.Lat,
			"lon":        v.Lon,
		}
	}

	c.JSON(http.StatusOK, out)
}

// ---------------------------------------------------------------------------
// Alerts (public read)
// ---------------------------------------------------------------------------

// ListAlerts handles GET /api/v1/alerts
func (h *PublicHandler) ListAlerts(c *gin.Context) {
	var routeID *int32
	if raw := c.Query("route_id"); raw != "" {
		v, err := strconv.ParseInt(raw, 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "route_id must be a valid integer"})
			return
		}
		id := int32(v)
		routeID = &id
	}

	alerts, err := h.alertsRepo.ListAlerts(c.Request.Context(), routeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list alerts"})
		return
	}

	out := make([]gin.H, len(alerts))
	for i, a := range alerts {
		out[i] = gin.H{
			"id":            a.ID,
			"title":         a.Title,
			"description":   a.Description,
			"route_id":      a.RouteID,
			"vehicle_plate": a.VehiclePlate,
			"image_path":    a.ImagePath,
			"created_by":    a.CreatedBy,
			"created_at":    a.CreatedAt,
		}
	}

	c.JSON(http.StatusOK, out)
}

// GetAlert handles GET /api/v1/alerts/:id
func (h *PublicHandler) GetAlert(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	alert, err := h.alertsRepo.GetAlertByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query alert"})
		return
	}
	if alert == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "alert not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":            alert.ID,
		"title":         alert.Title,
		"description":   alert.Description,
		"route_id":      alert.RouteID,
		"vehicle_plate": alert.VehiclePlate,
		"image_path":    alert.ImagePath,
		"created_by":    alert.CreatedBy,
		"created_at":    alert.CreatedAt,
	})
}

// ---------------------------------------------------------------------------
// Ratings
// ---------------------------------------------------------------------------

type createRatingRequest struct {
	TripID   int32  `json:"trip_id" binding:"required"`
	Rating   int16  `json:"rating" binding:"required,min=1,max=5"`
	DeviceID string `json:"device_id" binding:"required"`
}

// CreateRating handles POST /api/v1/ratings
func (h *PublicHandler) CreateRating(c *gin.Context) {
	var req createRatingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	rating := &storage.Rating{
		TripID:   req.TripID,
		Rating:   req.Rating,
		DeviceID: req.DeviceID,
	}

	created, err := h.ratingsRepo.CreateRating(c.Request.Context(), rating)
	if err != nil {
		if strings.Contains(err.Error(), "already rated") {
			c.JSON(http.StatusConflict, gin.H{"error": "device has already rated this trip"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create rating"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":         created.ID,
		"trip_id":    created.TripID,
		"rating":     created.Rating,
		"device_id":  created.DeviceID,
		"created_at": created.CreatedAt,
	})
}

// ---------------------------------------------------------------------------
// Favorites
// ---------------------------------------------------------------------------

// ListFavorites handles GET /api/v1/favorites
func (h *PublicHandler) ListFavorites(c *gin.Context) {
	deviceID := c.Query("device_id")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "device_id query parameter is required"})
		return
	}

	favs, err := h.favoritesRepo.ListByDevice(c.Request.Context(), deviceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list favorites"})
		return
	}

	out := make([]gin.H, len(favs))
	for i, f := range favs {
		out[i] = gin.H{
			"id":         f.ID,
			"device_id":  f.DeviceID,
			"route_id":   f.RouteID,
			"route_name": f.RouteName,
			"created_at": f.CreatedAt,
		}
	}

	c.JSON(http.StatusOK, out)
}

type addFavoriteRequest struct {
	DeviceID string `json:"device_id" binding:"required"`
	RouteID  int32  `json:"route_id" binding:"required"`
}

// AddFavorite handles POST /api/v1/favorites
func (h *PublicHandler) AddFavorite(c *gin.Context) {
	var req addFavoriteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	fav, err := h.favoritesRepo.Add(c.Request.Context(), req.DeviceID, req.RouteID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to add favorite"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":        fav.ID,
		"device_id": fav.DeviceID,
		"route_id":  fav.RouteID,
	})
}

type removeFavoriteRequest struct {
	DeviceID string `json:"device_id" binding:"required"`
	RouteID  int32  `json:"route_id" binding:"required"`
}

// RemoveFavorite handles DELETE /api/v1/favorites
func (h *PublicHandler) RemoveFavorite(c *gin.Context) {
	var req removeFavoriteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.favoritesRepo.Remove(c.Request.Context(), req.DeviceID, req.RouteID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to remove favorite"})
		return
	}

	c.Status(http.StatusNoContent)
}
