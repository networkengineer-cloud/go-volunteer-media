package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// HealthCheck returns basic health status
func HealthCheck() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "healthy",
			"time":   time.Now().UTC().Format(time.RFC3339),
		})
	}
}

// ReadinessCheck checks if the application is ready to serve traffic
// This includes checking database connectivity
func ReadinessCheck(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		
		// Check database connectivity
		sqlDB, err := db.DB()
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "not ready",
				"error":  "database connection unavailable",
			})
			return
		}
		
		// Ping database with context timeout
		if err := sqlDB.PingContext(ctx); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "not ready",
				"error":  "database ping failed",
			})
			return
		}
		
		c.JSON(http.StatusOK, gin.H{
			"status":   "ready",
			"time":     time.Now().UTC().Format(time.RFC3339),
			"database": "connected",
		})
	}
}
