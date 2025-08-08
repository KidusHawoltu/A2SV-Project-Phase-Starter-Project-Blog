package infrastructure

import (
	domain "A2SV_Starter_Project_Blog/Domain"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware creates a gin middleware for JWT authentication.
func AuthMiddleware(jwtService JWTService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is required"})
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header format must be Bearer {token}"})
			return
		}

		tokenString := parts[1]
		claims, err := jwtService.ValidateToken(tokenString)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			return
		}

		// Set user info into the context for later use in handlers
		c.Set("userID", claims.UserID)
		c.Set("userRole", claims.Role)

		c.Next()
	}
}

// AdminOnlyMiddleware checks if the authenticated user has the 'admin' role.
// It should be used *after* the AuthMiddleware.
func AdminOnlyMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. Get the role from the context, which was set by AuthMiddleware.
		// Note: your existing middleware uses the key "userRole".
		role, exists := c.Get("userRole")
		if !exists {
			// This case should not happen if AuthMiddleware is used correctly.
			// It means the token was valid but did not contain a role claim.
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Access denied: Role information is missing."})
			return
		}

		// 2. Check if the role is 'admin'.
		// We cast the role to domain.Role for type safety.
		userRole, ok := role.(domain.Role)
		if !ok || userRole != domain.RoleAdmin {
			// If the type assertion fails or the user is not an admin, abort the request.
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Access denied: Admin privileges required."})
			return
		}

		// 3. If the user is an admin, continue to the next handler.
		c.Next()
	}
}
