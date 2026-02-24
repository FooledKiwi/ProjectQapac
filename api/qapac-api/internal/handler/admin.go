package handler

import (
	"net/http"
	"strconv"

	"github.com/FooledKiwi/ProjectQapac/api/qapac-api/internal/service"
	"github.com/FooledKiwi/ProjectQapac/api/qapac-api/internal/storage"
	"github.com/gin-gonic/gin"
)

// AdminHandler holds dependencies for admin CRUD endpoints.
type AdminHandler struct {
	usersRepo    storage.UsersRepository
	vehiclesRepo storage.VehiclesRepository
	alertsRepo   storage.AlertsRepository
}

// NewAdminHandler creates an AdminHandler with the given repositories.
func NewAdminHandler(
	usersRepo storage.UsersRepository,
	vehiclesRepo storage.VehiclesRepository,
	alertsRepo storage.AlertsRepository,
) *AdminHandler {
	return &AdminHandler{
		usersRepo:    usersRepo,
		vehiclesRepo: vehiclesRepo,
		alertsRepo:   alertsRepo,
	}
}

// ---------------------------------------------------------------------------
// User management
// ---------------------------------------------------------------------------

type createUserRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required,min=6"`
	FullName string `json:"full_name" binding:"required"`
	Phone    string `json:"phone"`
	Role     string `json:"role" binding:"required,oneof=driver admin"`
}

// CreateUser handles POST /api/v1/admin/users
func (h *AdminHandler) CreateUser(c *gin.Context) {
	var req createUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	hash, err := service.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}

	user := &storage.User{
		Username:     req.Username,
		PasswordHash: hash,
		FullName:     req.FullName,
		Phone:        req.Phone,
		Role:         req.Role,
	}

	created, err := h.usersRepo.CreateUser(c.Request.Context(), user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":        created.ID,
		"username":  created.Username,
		"full_name": created.FullName,
		"phone":     created.Phone,
		"role":      created.Role,
		"active":    created.Active,
	})
}

// ListUsers handles GET /api/v1/admin/users
func (h *AdminHandler) ListUsers(c *gin.Context) {
	role := c.Query("role")
	activeOnly := c.DefaultQuery("active", "true") == "true"

	users, err := h.usersRepo.ListUsers(c.Request.Context(), role, activeOnly)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list users"})
		return
	}

	out := make([]gin.H, len(users))
	for i, u := range users {
		out[i] = gin.H{
			"id":        u.ID,
			"username":  u.Username,
			"full_name": u.FullName,
			"phone":     u.Phone,
			"role":      u.Role,
			"active":    u.Active,
		}
	}

	c.JSON(http.StatusOK, out)
}

// GetUser handles GET /api/v1/admin/users/:id
func (h *AdminHandler) GetUser(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	user, err := h.usersRepo.GetUserByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query user"})
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
		"role":               user.Role,
		"profile_image_path": user.ProfileImagePath,
		"active":             user.Active,
		"created_at":         user.CreatedAt,
		"updated_at":         user.UpdatedAt,
	})
}

type updateUserRequest struct {
	FullName         string `json:"full_name"`
	Phone            string `json:"phone"`
	Role             string `json:"role" binding:"omitempty,oneof=driver admin"`
	ProfileImagePath string `json:"profile_image_path"`
	Active           *bool  `json:"active"`
}

// UpdateUser handles PUT /api/v1/admin/users/:id
func (h *AdminHandler) UpdateUser(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	existing, err := h.usersRepo.GetUserByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query user"})
		return
	}
	if existing == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	var req updateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Apply partial updates.
	if req.FullName != "" {
		existing.FullName = req.FullName
	}
	if req.Phone != "" {
		existing.Phone = req.Phone
	}
	if req.Role != "" {
		existing.Role = req.Role
	}
	if req.ProfileImagePath != "" {
		existing.ProfileImagePath = req.ProfileImagePath
	}
	if req.Active != nil {
		existing.Active = *req.Active
	}

	if err := h.usersRepo.UpdateUser(c.Request.Context(), existing); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":        existing.ID,
		"username":  existing.Username,
		"full_name": existing.FullName,
		"phone":     existing.Phone,
		"role":      existing.Role,
		"active":    existing.Active,
	})
}

// DeactivateUser handles DELETE /api/v1/admin/users/:id
func (h *AdminHandler) DeactivateUser(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	if err := h.usersRepo.DeactivateUser(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to deactivate user"})
		return
	}

	c.Status(http.StatusNoContent)
}

// ---------------------------------------------------------------------------
// Vehicle management
// ---------------------------------------------------------------------------

type createVehicleRequest struct {
	PlateNumber string `json:"plate_number" binding:"required"`
	RouteID     *int32 `json:"route_id"`
}

// CreateVehicle handles POST /api/v1/admin/vehicles
func (h *AdminHandler) CreateVehicle(c *gin.Context) {
	var req createVehicleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	vehicle := &storage.Vehicle{
		PlateNumber: req.PlateNumber,
		RouteID:     req.RouteID,
		Status:      "inactive",
	}

	created, err := h.vehiclesRepo.CreateVehicle(c.Request.Context(), vehicle)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create vehicle"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":           created.ID,
		"plate_number": created.PlateNumber,
		"route_id":     created.RouteID,
		"status":       created.Status,
	})
}

// ListVehicles handles GET /api/v1/admin/vehicles
func (h *AdminHandler) ListVehicles(c *gin.Context) {
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

	status := c.Query("status")

	vehicles, err := h.vehiclesRepo.ListVehicles(c.Request.Context(), routeID, status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list vehicles"})
		return
	}

	out := make([]gin.H, len(vehicles))
	for i, v := range vehicles {
		out[i] = gin.H{
			"id":           v.ID,
			"plate_number": v.PlateNumber,
			"route_id":     v.RouteID,
			"status":       v.Status,
		}
	}

	c.JSON(http.StatusOK, out)
}

// GetVehicle handles GET /api/v1/admin/vehicles/:id
func (h *AdminHandler) GetVehicle(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	vehicle, err := h.vehiclesRepo.GetVehicleByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query vehicle"})
		return
	}
	if vehicle == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "vehicle not found"})
		return
	}

	// Also fetch assignment (non-fatal if absent).
	assignment, _ := h.vehiclesRepo.GetActiveAssignment(c.Request.Context(), id) //nolint:errcheck

	resp := gin.H{
		"id":           vehicle.ID,
		"plate_number": vehicle.PlateNumber,
		"route_id":     vehicle.RouteID,
		"status":       vehicle.Status,
		"created_at":   vehicle.CreatedAt,
	}

	if assignment != nil {
		resp["assignment"] = gin.H{
			"driver_id":    assignment.DriverID,
			"collector_id": assignment.CollectorID,
			"assigned_at":  assignment.AssignedAt,
		}
	}

	c.JSON(http.StatusOK, resp)
}

type updateVehicleRequest struct {
	PlateNumber string `json:"plate_number"`
	RouteID     *int32 `json:"route_id"`
	Status      string `json:"status" binding:"omitempty,oneof=active inactive maintenance"`
}

// UpdateVehicle handles PUT /api/v1/admin/vehicles/:id
func (h *AdminHandler) UpdateVehicle(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	existing, err := h.vehiclesRepo.GetVehicleByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query vehicle"})
		return
	}
	if existing == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "vehicle not found"})
		return
	}

	var req updateVehicleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.PlateNumber != "" {
		existing.PlateNumber = req.PlateNumber
	}
	if req.RouteID != nil {
		existing.RouteID = req.RouteID
	}
	if req.Status != "" {
		existing.Status = req.Status
	}

	if err := h.vehiclesRepo.UpdateVehicle(c.Request.Context(), existing); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update vehicle"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":           existing.ID,
		"plate_number": existing.PlateNumber,
		"route_id":     existing.RouteID,
		"status":       existing.Status,
	})
}

type assignVehicleRequest struct {
	DriverID    int32  `json:"driver_id" binding:"required"`
	CollectorID *int32 `json:"collector_id"`
}

// AssignVehicle handles POST /api/v1/admin/vehicles/:id/assign
func (h *AdminHandler) AssignVehicle(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	vehicle, err := h.vehiclesRepo.GetVehicleByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query vehicle"})
		return
	}
	if vehicle == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "vehicle not found"})
		return
	}

	var req assignVehicleRequest
	if err = c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	assignment := &storage.VehicleAssignment{
		VehicleID:   id,
		DriverID:    req.DriverID,
		CollectorID: req.CollectorID,
	}

	created, err := h.vehiclesRepo.AssignVehicle(c.Request.Context(), assignment)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to assign vehicle"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":           created.ID,
		"vehicle_id":   created.VehicleID,
		"driver_id":    created.DriverID,
		"collector_id": created.CollectorID,
		"assigned_at":  created.AssignedAt,
	})
}

// ---------------------------------------------------------------------------
// Alert management
// ---------------------------------------------------------------------------

type createAlertRequest struct {
	Title        string `json:"title" binding:"required"`
	Description  string `json:"description"`
	RouteID      *int32 `json:"route_id"`
	VehiclePlate string `json:"vehicle_plate"`
	ImagePath    string `json:"image_path"`
}

// CreateAlert handles POST /api/v1/admin/alerts
func (h *AdminHandler) CreateAlert(c *gin.Context) {
	var req createAlertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get creator ID from JWT claims.
	var createdBy *int32
	if uid, exists := c.Get("auth_user_id"); exists {
		if id, ok := uid.(int32); ok {
			createdBy = &id
		}
	}

	alert := &storage.Alert{
		Title:        req.Title,
		Description:  req.Description,
		RouteID:      req.RouteID,
		VehiclePlate: req.VehiclePlate,
		ImagePath:    req.ImagePath,
		CreatedBy:    createdBy,
	}

	created, err := h.alertsRepo.CreateAlert(c.Request.Context(), alert)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create alert"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":            created.ID,
		"title":         created.Title,
		"description":   created.Description,
		"route_id":      created.RouteID,
		"vehicle_plate": created.VehiclePlate,
		"image_path":    created.ImagePath,
		"created_by":    created.CreatedBy,
		"created_at":    created.CreatedAt,
	})
}

// DeleteAlert handles DELETE /api/v1/admin/alerts/:id
func (h *AdminHandler) DeleteAlert(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	if err := h.alertsRepo.DeleteAlert(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete alert"})
		return
	}

	c.Status(http.StatusNoContent)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// parseID extracts and validates a positive int32 :id path parameter.
func parseID(c *gin.Context) (int32, bool) {
	raw := c.Param("id")
	v, err := strconv.ParseInt(raw, 10, 32)
	if err != nil || v <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id must be a positive integer"})
		return 0, false
	}
	return int32(v), true
}
