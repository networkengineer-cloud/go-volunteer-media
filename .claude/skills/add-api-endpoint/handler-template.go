//go:build ignore

// handler-template.go — Copy this file as internal/handlers/<feature>.go
// Replace "Foo"/"foo" with your entity name throughout.
//
// Conventions shown here (match internal/handlers/schedule.go):
//   - Shadow the closure's db with middleware.GetDB(c, db) as the first line.
//     Never assign to the outer db with "=" — it is shared across requests.
//   - Authorize with callerCan(c, db, authz.<Action>, groupID) - the central
//     policy in internal/authz. Pick the Action that names what the endpoint
//     does; never check is_admin or group-admin flags inline. Identity comes
//     from the auth context, never from the request body.
//   - Return generic error messages to the client and log the real error.
package handlers

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/networkengineer-cloud/go-volunteer-media/internal/authz"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/middleware"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
)

// GetFoos returns all Foos for a group. Requires authz.ViewGroup (group member or site admin).
func GetFoos(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		db := middleware.GetDB(c, db)
		groupID := c.Param("id")

		if !callerCan(c, db, authz.ViewGroup, groupID) {
			respondForbidden(c, "forbidden")
			return
		}

		var foos []models.Foo
		if err := db.Where("group_id = ?", groupID).Find(&foos).Error; err != nil {
			middleware.GetLogger(c).Error("Failed to fetch foos", err)
			respondInternalError(c, "Failed to fetch foos")
			return
		}
		respondOK(c, foos)
	}
}

// GetFooByID returns a single Foo by ID. Requires authz.ViewGroup (group member or site admin).
func GetFooByID(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		db := middleware.GetDB(c, db)
		groupID := c.Param("id")

		if !callerCan(c, db, authz.ViewGroup, groupID) {
			respondForbidden(c, "forbidden")
			return
		}

		fooID, err := strconv.ParseUint(c.Param("fooId"), 10, 64)
		if err != nil {
			respondBadRequest(c, "invalid foo id")
			return
		}

		var foo models.Foo
		if err := db.Where("id = ? AND group_id = ?", fooID, groupID).First(&foo).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				respondNotFound(c, "not found")
				return
			}
			middleware.GetLogger(c).Error("Failed to fetch foo", err)
			respondInternalError(c, "Failed to fetch foo")
			return
		}
		respondOK(c, foo)
	}
}

// CreateFoo creates a new Foo in the given group. Requires authz.ManageContent (group admin or site admin).
func CreateFoo(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		db := middleware.GetDB(c, db)
		groupID := c.Param("id")

		if !callerCan(c, db, authz.ManageContent, groupID) {
			respondForbidden(c, "forbidden")
			return
		}

		var input struct {
			Name        string `json:"name" binding:"required"`
			Description string `json:"description"`
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			respondBadRequest(c, "invalid request body")
			return
		}

		parsedGroupID, err := strconv.ParseUint(groupID, 10, 64)
		if err != nil {
			respondBadRequest(c, "invalid group id")
			return
		}

		foo := models.Foo{
			GroupID:     uint(parsedGroupID),
			Name:        input.Name,
			Description: input.Description,
		}
		if err := db.Create(&foo).Error; err != nil {
			middleware.GetLogger(c).Error("Failed to create foo", err)
			respondInternalError(c, "Failed to create foo")
			return
		}
		respondCreated(c, foo)
	}
}

// UpdateFoo updates an existing Foo by ID. Requires authz.ManageContent (group admin or site admin).
func UpdateFoo(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		db := middleware.GetDB(c, db)
		groupID := c.Param("id")

		if !callerCan(c, db, authz.ManageContent, groupID) {
			respondForbidden(c, "forbidden")
			return
		}

		fooID, err := strconv.ParseUint(c.Param("fooId"), 10, 64)
		if err != nil {
			respondBadRequest(c, "invalid foo id")
			return
		}

		var foo models.Foo
		if err := db.Where("id = ? AND group_id = ?", fooID, groupID).First(&foo).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				respondNotFound(c, "not found")
				return
			}
			middleware.GetLogger(c).Error("Failed to fetch foo", err)
			respondInternalError(c, "Failed to fetch foo")
			return
		}

		// Pointer fields distinguish "not sent" from "set to empty".
		var input struct {
			Name        *string `json:"name"`
			Description *string `json:"description"`
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			respondBadRequest(c, "invalid request body")
			return
		}

		updates := map[string]interface{}{}
		if input.Name != nil {
			updates["name"] = *input.Name
		}
		if input.Description != nil {
			updates["description"] = *input.Description
		}

		if err := db.Model(&foo).Updates(updates).Error; err != nil {
			middleware.GetLogger(c).Error("Failed to update foo", err)
			respondInternalError(c, "Failed to update foo")
			return
		}
		// Reload to return DB-generated values (e.g. updated_at) that map-based Updates may not back-fill.
		if err := db.First(&foo, foo.ID).Error; err != nil {
			middleware.GetLogger(c).Error("Failed to reload foo", err)
			respondInternalError(c, "Failed to update foo")
			return
		}
		respondOK(c, foo)
	}
}

// DeleteFoo soft-deletes a Foo by ID. Requires authz.ManageContent (group admin or site admin).
func DeleteFoo(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		db := middleware.GetDB(c, db)
		groupID := c.Param("id")

		if !callerCan(c, db, authz.ManageContent, groupID) {
			respondForbidden(c, "forbidden")
			return
		}

		fooID, err := strconv.ParseUint(c.Param("fooId"), 10, 64)
		if err != nil {
			respondBadRequest(c, "invalid foo id")
			return
		}

		result := db.Where("id = ? AND group_id = ?", fooID, groupID).Delete(&models.Foo{})
		if result.Error != nil {
			middleware.GetLogger(c).Error("Failed to delete foo", result.Error)
			respondInternalError(c, "Failed to delete foo")
			return
		}
		if result.RowsAffected == 0 {
			respondNotFound(c, "not found")
			return
		}
		respondNoContent(c)
	}
}
