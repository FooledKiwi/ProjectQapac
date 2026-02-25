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
	stopsRepo    storage.StopsAdminRepository
	routesRepo   storage.RoutesAdminRepository
}

// NewAdminHandler creates an AdminHandler with the given repositories.
func NewAdminHandler(
	usersRepo storage.UsersRepository,
	vehiclesRepo storage.VehiclesRepository,
	alertsRepo storage.AlertsRepository,
	stopsRepo storage.StopsAdminRepository,
	routesRepo storage.RoutesAdminRepository,
) *AdminHandler {
	return &AdminHandler{
		usersRepo:    usersRepo,
		vehiclesRepo: vehiclesRepo,
		alertsRepo:   alertsRepo,
		stopsRepo:    stopsRepo,
		routesRepo:   routesRepo,
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

// ---------------------------------------------------------------------------
// Stop management
// ---------------------------------------------------------------------------

type createStopRequest struct {
	Name string  `json:"name" binding:"required"`
	Lat  float64 `json:"lat" binding:"required"`
	Lon  float64 `json:"lon" binding:"required"`
}

// CreateStop handles POST /api/v1/admin/stops
func (h *AdminHandler) CreateStop(c *gin.Context) {
	var req createStopRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	stop := &storage.AdminStop{
		Name: req.Name,
		Lat:  req.Lat,
		Lon:  req.Lon,
	}

	created, err := h.stopsRepo.CreateStop(c.Request.Context(), stop)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create stop"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":         created.ID,
		"name":       created.Name,
		"lat":        created.Lat,
		"lon":        created.Lon,
		"active":     created.Active,
		"created_at": created.CreatedAt,
	})
}

// ListStops handles GET /api/v1/admin/stops
func (h *AdminHandler) ListStops(c *gin.Context) {
	activeOnly := c.DefaultQuery("active", "true") == "true"

	stops, err := h.stopsRepo.ListStops(c.Request.Context(), activeOnly)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list stops"})
		return
	}

	out := make([]gin.H, len(stops))
	for i, s := range stops {
		out[i] = gin.H{
			"id":         s.ID,
			"name":       s.Name,
			"lat":        s.Lat,
			"lon":        s.Lon,
			"active":     s.Active,
			"created_at": s.CreatedAt,
		}
	}

	c.JSON(http.StatusOK, out)
}

// GetStop handles GET /api/v1/admin/stops/:id
func (h *AdminHandler) GetStop(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	stop, err := h.stopsRepo.GetStopByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query stop"})
		return
	}
	if stop == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "stop not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":         stop.ID,
		"name":       stop.Name,
		"lat":        stop.Lat,
		"lon":        stop.Lon,
		"active":     stop.Active,
		"created_at": stop.CreatedAt,
	})
}

type updateStopRequest struct {
	Name   string  `json:"name"`
	Lat    float64 `json:"lat"`
	Lon    float64 `json:"lon"`
	Active *bool   `json:"active"`
}

// UpdateStop handles PUT /api/v1/admin/stops/:id
func (h *AdminHandler) UpdateStop(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	existing, err := h.stopsRepo.GetStopByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query stop"})
		return
	}
	if existing == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "stop not found"})
		return
	}

	var req updateStopRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Apply partial updates.
	if req.Name != "" {
		existing.Name = req.Name
	}
	if req.Lat != 0 {
		existing.Lat = req.Lat
	}
	if req.Lon != 0 {
		existing.Lon = req.Lon
	}
	if req.Active != nil {
		existing.Active = *req.Active
	}

	if err := h.stopsRepo.UpdateStop(c.Request.Context(), existing); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update stop"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":     existing.ID,
		"name":   existing.Name,
		"lat":    existing.Lat,
		"lon":    existing.Lon,
		"active": existing.Active,
	})
}

// DeactivateStop handles DELETE /api/v1/admin/stops/:id
func (h *AdminHandler) DeactivateStop(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	if err := h.stopsRepo.DeactivateStop(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to deactivate stop"})
		return
	}

	c.Status(http.StatusNoContent)
}

// ---------------------------------------------------------------------------
// Route management
// ---------------------------------------------------------------------------

type createRouteRequest struct {
	Name string `json:"name" binding:"required"`
}

// CreateRoute handles POST /api/v1/admin/routes
func (h *AdminHandler) CreateRoute(c *gin.Context) {
	var req createRouteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	route := &storage.AdminRoute{
		Name: req.Name,
	}

	created, err := h.routesRepo.CreateRoute(c.Request.Context(), route)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create route"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":     created.ID,
		"name":   created.Name,
		"active": created.Active,
	})
}

// ListRoutes handles GET /api/v1/admin/routes
func (h *AdminHandler) ListRoutes(c *gin.Context) {
	activeOnly := c.DefaultQuery("active", "true") == "true"

	routes, err := h.routesRepo.ListRoutes(c.Request.Context(), activeOnly)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list routes"})
		return
	}

	out := make([]gin.H, len(routes))
	for i, rt := range routes {
		out[i] = gin.H{
			"id":     rt.ID,
			"name":   rt.Name,
			"active": rt.Active,
		}
	}

	c.JSON(http.StatusOK, out)
}

// GetRoute handles GET /api/v1/admin/routes/:id
func (h *AdminHandler) GetRoute(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	detail, err := h.routesRepo.GetRouteByID(c.Request.Context(), id)
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
			"stop_id":  s.StopID,
			"sequence": s.Sequence,
		}
	}

	resp := gin.H{
		"id":     detail.ID,
		"name":   detail.Name,
		"active": detail.Active,
		"stops":  stops,
	}
	if detail.ShapeGeomWKT != "" {
		resp["shape_geom_wkt"] = detail.ShapeGeomWKT
	}

	c.JSON(http.StatusOK, resp)
}

type updateRouteRequest struct {
	Name   string `json:"name"`
	Active *bool  `json:"active"`
}

// UpdateRoute handles PUT /api/v1/admin/routes/:id
func (h *AdminHandler) UpdateRoute(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	existing, err := h.routesRepo.GetRouteByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query route"})
		return
	}
	if existing == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "route not found"})
		return
	}

	var req updateRouteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	route := &storage.AdminRoute{
		ID:     existing.ID,
		Name:   existing.Name,
		Active: existing.Active,
	}

	if req.Name != "" {
		route.Name = req.Name
	}
	if req.Active != nil {
		route.Active = *req.Active
	}

	if err := h.routesRepo.UpdateRoute(c.Request.Context(), route); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update route"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":     route.ID,
		"name":   route.Name,
		"active": route.Active,
	})
}

// DeactivateRoute handles DELETE /api/v1/admin/routes/:id
func (h *AdminHandler) DeactivateRoute(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	if err := h.routesRepo.DeactivateRoute(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to deactivate route"})
		return
	}

	c.Status(http.StatusNoContent)
}

type replaceRouteStopsRequest struct {
	Stops []routeStopEntryRequest `json:"stops" binding:"required,dive"`
}

type routeStopEntryRequest struct {
	StopID   int32 `json:"stop_id" binding:"required"`
	Sequence int   `json:"sequence" binding:"required"`
}

// ReplaceRouteStops handles PUT /api/v1/admin/routes/:id/stops
func (h *AdminHandler) ReplaceRouteStops(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	// Verify route exists.
	existing, err := h.routesRepo.GetRouteByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query route"})
		return
	}
	if existing == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "route not found"})
		return
	}

	var req replaceRouteStopsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	entries := make([]storage.RouteStopEntry, len(req.Stops))
	for i, s := range req.Stops {
		entries[i] = storage.RouteStopEntry{
			StopID:   s.StopID,
			Sequence: s.Sequence,
		}
	}

	if err := h.routesRepo.ReplaceRouteStops(c.Request.Context(), id, entries); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to replace route stops"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"route_id": id, "stops_count": len(entries)})
}

type updateRouteShapeRequest struct {
	GeomWKT string `json:"geom_wkt" binding:"required"`
}

// UpdateRouteShape handles PUT /api/v1/admin/routes/:id/shape
func (h *AdminHandler) UpdateRouteShape(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	// Verify route exists.
	existing, err := h.routesRepo.GetRouteByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query route"})
		return
	}
	if existing == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "route not found"})
		return
	}

	var req updateRouteShapeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.routesRepo.UpsertRouteShape(c.Request.Context(), id, req.GeomWKT); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update route shape"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"route_id": id, "shape_updated": true})
}
