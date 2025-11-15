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
