package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/logging"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/maintenance"
	"gorm.io/gorm"
)

// RunOrphanedImageCleanup triggers cleanup of orphaned animal images (admin only)
func RunOrphanedImageCleanup(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		// Get optional days parameter (default 7 days)
		days := 7
		if daysParam := c.Query("days"); daysParam != "" {
			if parsedDays, err := strconv.Atoi(daysParam); err == nil && parsedDays > 0 {
				days = parsedDays
			}
		}

		logging.WithField("days", days).Info("Admin triggered orphaned image cleanup")

		// Run cleanup using the maintenance package
		deletedCount, err := maintenance.CleanupOrphanedImages(db.WithContext(ctx), days)
		if err != nil {
			logging.WithField("error", err.Error()).Warn("Orphaned image cleanup failed")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Cleanup failed"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message":       "Cleanup completed successfully",
			"deleted_count": deletedCount,
			"days":          days,
		})
	}
}

// RunSoftDeletedRecordsCleanup triggers cleanup of old soft-deleted records (admin only)
func RunSoftDeletedRecordsCleanup(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		// Get required table parameter
		tableName := c.Query("table")
		if tableName == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "table parameter is required"})
			return
		}

		// Validate table name against whitelist to prevent SQL injection
		validTables := map[string]bool{
			"animal_comments":     true,
			"animal_images":       true,
			"animal_tags":         true,
			"animals":             true,
			"announcements":       true,
			"comment_tags":        true,
			"groups":              true,
			"protocols":           true,
			"updates":             true,
			"users":               true,
			"animal_name_history": true,
		}

		if !validTables[tableName] {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid table name"})
			return
		}

		// Get optional days parameter (default 90 days, minimum 30)
		days := 90
		if daysParam := c.Query("days"); daysParam != "" {
			if parsedDays, err := strconv.Atoi(daysParam); err == nil && parsedDays >= 30 {
				days = parsedDays
			}
		}

		logging.WithFields(map[string]interface{}{
			"table": tableName,
			"days":  days,
		}).Info("Admin triggered soft-deleted records cleanup")

		// Run cleanup using the maintenance package
		deletedCount, err := maintenance.CleanupOldSoftDeletedRecords(db.WithContext(ctx), tableName, days)
		if err != nil {
			logging.WithField("error", err.Error()).Warn("Soft-deleted records cleanup failed")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Cleanup failed"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message":       "Cleanup completed successfully",
			"table":         tableName,
			"deleted_count": deletedCount,
			"days":          days,
		})
	}
}
