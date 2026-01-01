package middleware

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/auth"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/logging"
)

// CORS middleware to handle cross-origin requests
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get allowed origins from environment variable
		allowedOrigins := os.Getenv("ALLOWED_ORIGINS")
		if allowedOrigins == "" {
			// Default for development
			allowedOrigins = "http://localhost:5173,http://localhost:3000"
		}

		origin := c.Request.Header.Get("Origin")
		// Check if the origin is in the allowed list
		if origin != "" && contains(strings.Split(allowedOrigins, ","), origin) {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
		} else if allowedOrigins == "*" {
			// Allow wildcard only if explicitly set to "*"
			c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		}

		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// contains checks if a string is in a slice
func contains(slice []string, str string) bool {
	for _, item := range slice {
		if strings.TrimSpace(item) == str {
			return true
		}
	}
	return false
}

// AuthRequired middleware to protect routes
func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			// Log unauthorized access attempt
			logger := GetLogger(c)
			logger.WithFields(map[string]interface{}{
				"ip":       c.ClientIP(),
				"endpoint": c.Request.URL.Path,
				"method":   c.Request.Method,
			}).Warn("Authorization header missing")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		// Extract token from "Bearer <token>"
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			logger := GetLogger(c)
			logger.WithFields(map[string]interface{}{
				"ip":       c.ClientIP(),
				"endpoint": c.Request.URL.Path,
				"method":   c.Request.Method,
			}).Warn("Invalid authorization format")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization format"})
			c.Abort()
			return
		}

		token := parts[1]
		claims, err := auth.ValidateToken(token)
		if err != nil {
			// Log invalid token attempt
			logger := GetLogger(c)
			logger.WithFields(map[string]interface{}{
				"ip":       c.ClientIP(),
				"endpoint": c.Request.URL.Path,
				"method":   c.Request.Method,
				"error":    err.Error(),
			}).Warn("Invalid or expired token")

			// Use audit logger for security event
			logging.LogUnauthorizedAccess(ctx, c.ClientIP(), c.Request.URL.Path, "invalid_token")

			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		// Store user info in context
		c.Set("user_id", claims.UserID)
		c.Set("is_admin", claims.IsAdmin)
		c.Set("is_group_admin", claims.IsGroupAdmin)
		c.Next()
	}
}

// AdminRequired middleware to restrict access to admin users only
func AdminRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		isAdmin, exists := c.Get("is_admin")
		if !exists || !isAdmin.(bool) {
			// Log unauthorized admin access attempt
			logger := GetLogger(c)
			userID, _ := c.Get("user_id")
			logger.WithFields(map[string]interface{}{
				"ip":       c.ClientIP(),
				"endpoint": c.Request.URL.Path,
				"method":   c.Request.Method,
				"user_id":  userID,
			}).Warn("Admin access denied - insufficient privileges")

			// Use audit logger for security event
			logging.LogUnauthorizedAccess(ctx, c.ClientIP(), c.Request.URL.Path, "insufficient_privileges")

			c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
			c.Abort()
			return
		}
		c.Next()
	}
}

// AdminOrGroupAdminRequired middleware to restrict access to admin or group admin users
// This allows site admins and group admins to access certain management endpoints
func AdminOrGroupAdminRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		// Check if site admin
		isAdmin, exists := c.Get("is_admin")
		if exists && isAdmin.(bool) {
			c.Next()
			return
		}

		// Check if group admin by looking for is_group_admin flag from GetCurrentUser
		// For now, we'll check if user is group admin by verifying they have group memberships with admin role
		// This will be validated by the handler itself when needed
		userID, exists := c.Get("user_id")
		if !exists {
			logger := GetLogger(c)
			logger.WithFields(map[string]interface{}{
				"ip":       c.ClientIP(),
				"endpoint": c.Request.URL.Path,
				"method":   c.Request.Method,
			}).Warn("User ID not found in context")
			logging.LogUnauthorizedAccess(ctx, c.ClientIP(), c.Request.URL.Path, "invalid_context")
			c.JSON(http.StatusForbidden, gin.H{"error": "Admin or group admin access required"})
			c.Abort()
			return
		}

		// Check if user has is_group_admin flag set (from JWT or session)
		// The frontend should set this in the JWT or we verify it per-request
		// For now, we allow the request through and let handlers validate group membership
		// This is acceptable because handlers will verify the user is actually a group admin
		// for the specific group being managed
		isGroupAdmin, exists := c.Get("is_group_admin")
		if exists && isGroupAdmin.(bool) {
			c.Next()
			return
		}

		// Neither site admin nor group admin
		logger := GetLogger(c)
		logger.WithFields(map[string]interface{}{
			"ip":       c.ClientIP(),
			"endpoint": c.Request.URL.Path,
			"method":   c.Request.Method,
			"user_id":  userID,
		}).Warn("Admin or group admin access denied - insufficient privileges")

		logging.LogUnauthorizedAccess(ctx, c.ClientIP(), c.Request.URL.Path, "insufficient_privileges")

		c.JSON(http.StatusForbidden, gin.H{"error": "Admin or group admin access required"})
		c.Abort()
	}
}

// IsSiteAdmin checks if the current user is a site-wide admin
func IsSiteAdmin(c *gin.Context) bool {
	isAdmin, exists := c.Get("is_admin")
	return exists && isAdmin.(bool)
}
