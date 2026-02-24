package handler

import (
	"errors"
	"net/http"

	"github.com/FooledKiwi/ProjectQapac/api/qapac-api/internal/service"
	"github.com/gin-gonic/gin"
)

// AuthHandler holds dependencies for authentication endpoints.
type AuthHandler struct {
	authService *service.AuthService
}

// NewAuthHandler creates an AuthHandler with the given auth service.
func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

// loginRequest is the expected body for POST /api/v1/auth/login.
type loginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// Login handles POST /api/v1/auth/login
//
// Request body:
//
//	{"username": "driver1", "password": "secret"}
//
// Response 200:
//
//	{"access_token": "...", "refresh_token": "...", "user": {"id":1, "username":"driver1", "full_name":"...", "role":"driver"}}
//
// Response 400: malformed request body.
// Response 401: invalid credentials.
// Response 500: internal error.
func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "username and password are required"})
		return
	}

	pair, user, err := h.authService.Login(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid username or password"})
			return
		}
		if errors.Is(err, service.ErrJWTSecretMissing) {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "authentication service not configured"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "login failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":  pair.AccessToken,
		"refresh_token": pair.RefreshToken,
		"user": gin.H{
			"id":        user.ID,
			"username":  user.Username,
			"full_name": user.FullName,
			"role":      user.Role,
		},
	})
}

// refreshRequest is the expected body for POST /api/v1/auth/refresh.
type refreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// Refresh handles POST /api/v1/auth/refresh
//
// Request body:
//
//	{"refresh_token": "..."}
//
// Response 200:
//
//	{"access_token": "...", "refresh_token": "..."}
//
// Response 400: malformed request body.
// Response 401: invalid, expired, or revoked token.
// Response 500: internal error.
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "refresh_token is required"})
		return
	}

	pair, err := h.authService.Refresh(c.Request.Context(), req.RefreshToken)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) ||
			errors.Is(err, service.ErrTokenExpired) ||
			errors.Is(err, service.ErrTokenRevoked) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired refresh token"})
			return
		}
		if errors.Is(err, service.ErrJWTSecretMissing) {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "authentication service not configured"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "token refresh failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":  pair.AccessToken,
		"refresh_token": pair.RefreshToken,
	})
}

// logoutRequest is the expected body for POST /api/v1/auth/logout.
type logoutRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// Logout handles POST /api/v1/auth/logout
//
// Request body:
//
//	{"refresh_token": "..."}
//
// Response 204: token revoked successfully.
// Response 400: malformed request body.
// Response 500: internal error.
func (h *AuthHandler) Logout(c *gin.Context) {
	var req logoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "refresh_token is required"})
		return
	}

	if err := h.authService.Logout(c.Request.Context(), req.RefreshToken); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "logout failed"})
		return
	}

	c.Status(http.StatusNoContent)
}
