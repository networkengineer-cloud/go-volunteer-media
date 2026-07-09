package middleware

import (
	"context"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/auth"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/logging"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
	"gorm.io/gorm"
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

// AuthRequired middleware to protect routes. Accepts either a JWT (issued at
// login) or an API token (prefixed "pat_", generated via the admin API
// tokens endpoints) in the Authorization header.
func AuthRequired(db *gorm.DB) gin.HandlerFunc {
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

		if auth.IsAPIToken(token) {
			userID, isAdmin, ok := authenticateAPIToken(ctx, db, token)
			if !ok {
				logger := GetLogger(c)
				logger.WithFields(map[string]interface{}{
					"ip":       c.ClientIP(),
					"endpoint": c.Request.URL.Path,
					"method":   c.Request.Method,
				}).Warn("Invalid or expired API token")

				logging.LogUnauthorizedAccess(ctx, c.ClientIP(), c.Request.URL.Path, "invalid_api_token")

				c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
				c.Abort()
				return
			}

			c.Set("user_id", userID)
			c.Set("is_admin", isAdmin)
			c.Next()
			return
		}

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

// authenticateAPIToken looks up a presented API token, verifying it is
// unexpired and unrevoked, and returns the owning user's *current* identity.
// is_admin is read fresh from the User row (not cached on the token) so
// demoting or deleting the admin immediately invalidates their tokens.
// LastUsedAt is updated best-effort; a failure to record it does not fail auth.
func authenticateAPIToken(ctx context.Context, db *gorm.DB, token string) (userID uint, isAdmin bool, ok bool) {
	hash := auth.HashAPIToken(token)

	var apiToken models.APIToken
	if err := db.WithContext(ctx).
		Where("token_hash = ? AND expires_at > ?", hash, time.Now()).
		First(&apiToken).Error; err != nil {
		return 0, false, false
	}

	var user models.User
	if err := db.WithContext(ctx).First(&user, apiToken.UserID).Error; err != nil {
		return 0, false, false
	}

	now := time.Now()
	db.WithContext(ctx).Model(&apiToken).Update("last_used_at", &now)

	return user.ID, user.IsAdmin, true
}

// AdminRequired middleware to restrict access to admin users only
func AdminRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		if !GetIsAdmin(c) {
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

// IsSiteAdmin checks if the current user is a site-wide admin
func IsSiteAdmin(c *gin.Context) bool {
	v, exists := c.Get("is_admin")
	if !exists {
		return false
	}
	b, ok := v.(bool)
	return ok && b
}

// GetUserID retrieves the authenticated user's ID from the Gin context.
// Returns (0, false) if the key is missing or has an unexpected type.
func GetUserID(c *gin.Context) (uint, bool) {
	v, exists := c.Get("user_id")
	if !exists {
		return 0, false
	}
	id, ok := v.(uint)
	return id, ok
}

// GetIsAdmin retrieves the is_admin flag from the Gin context.
// Returns false if the key is missing or has an unexpected type.
func GetIsAdmin(c *gin.Context) bool {
	v, exists := c.Get("is_admin")
	if !exists {
		return false
	}
	b, ok := v.(bool)
	return ok && b
}
