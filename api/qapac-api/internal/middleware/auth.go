package middleware

import (
	"net/http"
	"strings"

	"github.com/FooledKiwi/ProjectQapac/api/qapac-api/internal/service"
	"github.com/gin-gonic/gin"
)

// Context keys for storing auth claims in the request context.
const (
	// ContextKeyUserID stores the authenticated user's ID.
	ContextKeyUserID = "auth_user_id"
	// ContextKeyUsername stores the authenticated user's username.
	ContextKeyUsername = "auth_username"
	// ContextKeyRole stores the authenticated user's role.
	ContextKeyRole = "auth_role"
)

// JWTAuth returns a Gin middleware that validates a Bearer token from the
// Authorization header using the provided AuthService.
//
// On success, user claims are stored in the Gin context under ContextKey* keys.
// On failure, the request is aborted with a 401 response.
func JWTAuth(authService *service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing authorization header"})
			return
		}

		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization format; expected 'Bearer <token>'"})
			return
		}

		claims, err := authService.ValidateAccessToken(parts[1])
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}

		// Store claims in context for downstream handlers.
		c.Set(ContextKeyUserID, claims.UserID)
		c.Set(ContextKeyUsername, claims.Username)
		c.Set(ContextKeyRole, claims.Role)

		c.Next()
	}
}

// RequireRole returns a Gin middleware that checks whether the authenticated
// user has one of the allowed roles. Must be used after JWTAuth.
func RequireRole(allowed ...string) gin.HandlerFunc {
	roleSet := make(map[string]bool, len(allowed))
	for _, r := range allowed {
		roleSet[r] = true
	}

	return func(c *gin.Context) {
		role, exists := c.Get(ContextKeyRole)
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
			return
		}

		roleStr, ok := role.(string)
		if !ok || !roleSet[roleStr] {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
			return
		}

		c.Next()
	}
}
