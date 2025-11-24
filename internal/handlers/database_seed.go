package handlers

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/database"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/logging"
	"gorm.io/gorm"
)

// SeedDatabase re-seeds the database (admin only, dangerous operation, dev only)
func SeedDatabase(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		logger := logging.GetDefaultLogger()
		
		// Only allow in development environment
		env := os.Getenv("ENV")
		if env != "development" {
			logger.Warn("Database seed attempted in non-development environment: " + env)
			c.JSON(http.StatusForbidden, gin.H{"error": "Database seeding is only available in development environments"})
			return
		}
		
		logger.Info("Admin initiated database re-seed")
		
		// Force seed the database
		if err := database.SeedData(db, true); err != nil {
			logger.Error("Failed to seed database", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to seed database"})
			return
		}

		logger.Info("Database re-seeded successfully")
		c.JSON(http.StatusOK, gin.H{
			"message": "Database re-seeded successfully",
			"demo_accounts": gin.H{
				"admin": gin.H{
					"username": "admin",
					"password": "demo1234",
				},
				"volunteers": []string{
					"sarah_modsquad",
					"mike_modsquad",
					"jake_modsquad",
					"lisa_modsquad",
				},
				"password": "demo1234",
			},
		})
	}
}
