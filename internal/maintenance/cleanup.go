package maintenance

import (
	"time"

	"github.com/networkengineer-cloud/go-volunteer-media/internal/logging"
	"gorm.io/gorm"
)

// CleanupOrphanedImages deletes orphaned animal images that are older than the specified number of days
// Orphaned images are those with animal_id IS NULL (uploaded but never linked to an animal)
func CleanupOrphanedImages(db *gorm.DB, olderThanDays int) (int64, error) {
	if olderThanDays < 1 {
		olderThanDays = 7 // Default to 7 days if invalid value provided
	}

	cutoffDate := time.Now().AddDate(0, 0, -olderThanDays)

	logging.WithFields(map[string]interface{}{
		"cutoff_date":     cutoffDate,
		"older_than_days": olderThanDays,
	}).Info("Starting orphaned image cleanup")

	// First, count how many images will be deleted for logging
	var count int64
	countResult := db.Raw(`
		SELECT COUNT(*) 
		FROM animal_images 
		WHERE animal_id IS NULL 
		  AND created_at < ?
	`, cutoffDate).Scan(&count)

	if countResult.Error != nil {
		logging.WithField("error", countResult.Error.Error()).Warn("Failed to count orphaned images")
		return 0, countResult.Error
	}

	if count == 0 {
		logging.Info("No orphaned images found to clean up")
		return 0, nil
	}

	logging.WithField("count", count).Info("Found orphaned images to delete")

	// Delete orphaned images older than the cutoff date
	// Note: This performs a soft delete (sets deleted_at) due to GORM's default behavior
	result := db.Exec(`
		DELETE FROM animal_images 
		WHERE animal_id IS NULL 
		  AND created_at < ?
	`, cutoffDate)

	if result.Error != nil {
		logging.WithField("error", result.Error.Error()).Warn("Failed to delete orphaned images")
		return 0, result.Error
	}

	logging.WithFields(map[string]interface{}{
		"deleted_count": result.RowsAffected,
		"cutoff_date":   cutoffDate,
	}).Info("Successfully cleaned up orphaned images")

	return result.RowsAffected, nil
}

// CleanupOldSoftDeletedRecords permanently deletes soft-deleted records older than the specified number of days
// This helps reduce database size by removing old soft-deleted records that are no longer needed
func CleanupOldSoftDeletedRecords(db *gorm.DB, tableName string, olderThanDays int) (int64, error) {
	if olderThanDays < 30 {
		olderThanDays = 30 // Default to 30 days minimum to avoid accidental data loss
	}

	cutoffDate := time.Now().AddDate(0, 0, -olderThanDays)

	logging.WithFields(map[string]interface{}{
		"table":           tableName,
		"cutoff_date":     cutoffDate,
		"older_than_days": olderThanDays,
	}).Info("Starting soft-deleted records cleanup")

	// Count soft-deleted records
	var count int64
	countQuery := `SELECT COUNT(*) FROM ` + tableName + ` WHERE deleted_at IS NOT NULL AND deleted_at < ?`
	countResult := db.Raw(countQuery, cutoffDate).Scan(&count)

	if countResult.Error != nil {
		logging.WithField("error", countResult.Error.Error()).Warn("Failed to count soft-deleted records")
		return 0, countResult.Error
	}

	if count == 0 {
		logging.WithField("table", tableName).Info("No old soft-deleted records found to clean up")
		return 0, nil
	}

	logging.WithFields(map[string]interface{}{
		"table": tableName,
		"count": count,
	}).Info("Found soft-deleted records to permanently delete")

	// Permanently delete soft-deleted records
	deleteQuery := `DELETE FROM ` + tableName + ` WHERE deleted_at IS NOT NULL AND deleted_at < ?`
	result := db.Exec(deleteQuery, cutoffDate)

	if result.Error != nil {
		logging.WithField("error", result.Error.Error()).Warn("Failed to permanently delete soft-deleted records")
		return 0, result.Error
	}

	logging.WithFields(map[string]interface{}{
		"table":         tableName,
		"deleted_count": result.RowsAffected,
		"cutoff_date":   cutoffDate,
	}).Info("Successfully cleaned up soft-deleted records")

	return result.RowsAffected, nil
}
