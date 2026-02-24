package handler

import (
	"net/http"
	"strings"

	"github.com/FooledKiwi/ProjectQapac/api/qapac-api/internal/storage"
	"github.com/gin-gonic/gin"
)

// DriverHandler holds dependencies for driver-specific endpoints.
type DriverHandler struct {
	usersRepo  storage.UsersRepository
	driverRepo storage.DriverRepository
	tripsRepo  storage.TripsRepository
}

// NewDriverHandler creates a DriverHandler with the given repositories.
func NewDriverHandler(
	usersRepo storage.UsersRepository,
	driverRepo storage.DriverRepository,
	tripsRepo storage.TripsRepository,
) *DriverHandler {
	return &DriverHandler{
		usersRepo:  usersRepo,
		driverRepo: driverRepo,
		tripsRepo:  tripsRepo,
	}
}

// authUserID extracts the authenticated user's ID from the gin context.
// Returns 0 and sends a 401 if the value is missing.
func authUserID(c *gin.Context) (int32, bool) {
	uid, exists := c.Get("auth_user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return 0, false
	}
	id, ok := uid.(int32)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid auth context"})
		return 0, false
	}
	return id, true
}

// ---------------------------------------------------------------------------
// GPS Position Reporting
// ---------------------------------------------------------------------------

type reportPositionRequest struct {
	Lat     float64  `json:"lat" binding:"required"`
	Lon     float64  `json:"lon" binding:"required"`
	Heading *float64 `json:"heading"`
	Speed   *float64 `json:"speed"`
}

// ReportPosition handles POST /api/v1/driver/position
func (h *DriverHandler) ReportPosition(c *gin.Context) {
	userID, ok := authUserID(c)
	if !ok {
		return
	}

	var req reportPositionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Find the driver's current vehicle assignment.
	assignment, err := h.driverRepo.GetAssignmentByDriver(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query assignment"})
		return
	}
	if assignment == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "no active vehicle assignment"})
		return
	}

	// Upsert the GPS position for the assigned vehicle.
	if err := h.driverRepo.UpsertPosition(
		c.Request.Context(), assignment.VehicleID,
		req.Lat, req.Lon, req.Heading, req.Speed,
	); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update position"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// ---------------------------------------------------------------------------
// Profile
// ---------------------------------------------------------------------------

// GetProfile handles GET /api/v1/driver/profile
func (h *DriverHandler) GetProfile(c *gin.Context) {
	userID, ok := authUserID(c)
	if !ok {
		return
	}

	user, err := h.usersRepo.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query profile"})
		return
	}
	if user == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":                 user.ID,
		"username":           user.Username,
		"full_name":          user.FullName,
		"phone":              user.Phone,
		"profile_image_path": user.ProfileImagePath,
	})
}

type updateProfileRequest struct {
	FullName string `json:"full_name"`
	Phone    string `json:"phone"`
}

// UpdateProfile handles PUT /api/v1/driver/profile
func (h *DriverHandler) UpdateProfile(c *gin.Context) {
	userID, ok := authUserID(c)
	if !ok {
		return
	}

	user, err := h.usersRepo.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query profile"})
		return
	}
	if user == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	var req updateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.FullName != "" {
		user.FullName = req.FullName
	}
	if req.Phone != "" {
		user.Phone = req.Phone
	}

	if err := h.usersRepo.UpdateUser(c.Request.Context(), user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update profile"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":        user.ID,
		"username":  user.Username,
		"full_name": user.FullName,
		"phone":     user.Phone,
	})
}

// ---------------------------------------------------------------------------
// Assignment
// ---------------------------------------------------------------------------

// GetAssignment handles GET /api/v1/driver/assignment
func (h *DriverHandler) GetAssignment(c *gin.Context) {
	userID, ok := authUserID(c)
	if !ok {
		return
	}

	assignment, err := h.driverRepo.GetAssignmentByDriver(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query assignment"})
		return
	}
	if assignment == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no active assignment"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"vehicle": gin.H{
			"id":         assignment.VehicleID,
			"plate":      assignment.PlateNumber,
			"route_name": assignment.RouteName,
		},
		"collector_name": assignment.CollectorName,
		"assigned_at":    assignment.AssignedAt,
	})
}

// ---------------------------------------------------------------------------
// Trips
// ---------------------------------------------------------------------------

// StartTrip handles POST /api/v1/driver/trips/start
func (h *DriverHandler) StartTrip(c *gin.Context) {
	userID, ok := authUserID(c)
	if !ok {
		return
	}

	// Check for existing active trip.
	existing, err := h.tripsRepo.GetActiveTrip(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check active trip"})
		return
	}
	if existing != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "an active trip already exists", "trip_id": existing.ID})
		return
	}

	// Driver must have an active assignment.
	assignment, err := h.driverRepo.GetAssignmentByDriver(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query assignment"})
		return
	}
	if assignment == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "no active vehicle assignment"})
		return
	}

	// Start trip using the assignment's vehicle; route_id is resolved inside the repository.
	trip, err := h.tripsRepo.StartTripFromAssignment(c.Request.Context(), userID, assignment.VehicleID)
	if err != nil {
		if strings.Contains(err.Error(), "no route assigned") {
			c.JSON(http.StatusConflict, gin.H{"error": "vehicle has no route assigned"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to start trip"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"trip_id":    trip.ID,
		"vehicle_id": trip.VehicleID,
		"route_id":   trip.RouteID,
		"started_at": trip.StartedAt,
	})
}

// EndTrip handles POST /api/v1/driver/trips/end
func (h *DriverHandler) EndTrip(c *gin.Context) {
	userID, ok := authUserID(c)
	if !ok {
		return
	}

	if err := h.tripsRepo.EndTrip(c.Request.Context(), userID); err != nil {
		if strings.Contains(err.Error(), "no active trip") {
			c.JSON(http.StatusNotFound, gin.H{"error": "no active trip to end"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to end trip"})
		return
	}

	c.Status(http.StatusNoContent)
}
