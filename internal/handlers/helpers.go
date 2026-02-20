package handlers

import "gorm.io/gorm"

// activeGroupsPreload excludes soft-deleted groups for association preloads.
func activeGroupsPreload(db *gorm.DB) *gorm.DB {
	return db.Where("groups.deleted_at IS NULL")
}
