//go:build ignore

// handler-template.go — Copy this file as internal/handlers/<feature>.go
// Replace "Foo"/"foo" with your entity name throughout.
package handlers

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
)

// GetFoos returns all Foos for a group.
func GetFoos(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		groupID := c.Param("id")
		userID, exists := c.Get("user_id")
		if !exists {
			respondUnauthorized(c, "unauthorized")
			return
		}
		// is_admin is always set by AuthRequired alongside user_id; false default is safe (conservatively denies access).
		isAdmin, _ := c.Get("is_admin")
		isAdminBool, _ := isAdmin.(bool)

		if !checkGroupAccess(db, userID, isAdminBool, groupID) {
			respondForbidden(c, "forbidden")
			return
		}

		var foos []models.Foo
		if err := db.WithContext(ctx).Where("group_id = ?", groupID).Find(&foos).Error; err != nil {
			respondInternalError(c, err.Error())
			return
		}
		respondOK(c, foos)
	}
}

// GetFooByID returns a single Foo by ID.
func GetFooByID(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		groupID := c.Param("id")
		fooID := c.Param("fooId")
		userID, exists := c.Get("user_id")
		if !exists {
			respondUnauthorized(c, "unauthorized")
			return
		}
		// is_admin is always set by AuthRequired alongside user_id; false default is safe (conservatively denies access).
		isAdmin, _ := c.Get("is_admin")
		isAdminBool, _ := isAdmin.(bool)

		if !checkGroupAccess(db, userID, isAdminBool, groupID) {
			respondForbidden(c, "forbidden")
			return
		}

		parsedFooID, err := strconv.ParseUint(fooID, 10, 64)
		if err != nil {
			respondBadRequest(c, "invalid foo id")
			return
		}

		var foo models.Foo
		if err := db.WithContext(ctx).Where("id = ? AND group_id = ?", parsedFooID, groupID).First(&foo).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				respondNotFound(c, "not found")
			} else {
				respondInternalError(c, err.Error())
			}
			return
		}
		respondOK(c, foo)
	}
}

// CreateFoo creates a new Foo in the given group.
func CreateFoo(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		groupID := c.Param("id")
		userID, exists := c.Get("user_id")
		if !exists {
			respondUnauthorized(c, "unauthorized")
			return
		}
		// is_admin is always set by AuthRequired alongside user_id; false default is safe (conservatively denies access).
		isAdmin, _ := c.Get("is_admin")
		isAdminBool, _ := isAdmin.(bool)

		if !checkGroupAdminAccess(db, userID, isAdminBool, groupID) {
			respondForbidden(c, "forbidden")
			return
		}

		var input struct {
			Name        string `json:"name" binding:"required"`
			Description string `json:"description"`
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			respondBadRequest(c, err.Error())
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
		if err := db.WithContext(ctx).Create(&foo).Error; err != nil {
			respondInternalError(c, err.Error())
			return
		}
		respondCreated(c, foo)
	}
}

// UpdateFoo updates an existing Foo by ID.
func UpdateFoo(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		groupID := c.Param("id")
		fooID := c.Param("fooId")
		userID, exists := c.Get("user_id")
		if !exists {
			respondUnauthorized(c, "unauthorized")
			return
		}
		// is_admin is always set by AuthRequired alongside user_id; false default is safe (conservatively denies access).
		isAdmin, _ := c.Get("is_admin")
		isAdminBool, _ := isAdmin.(bool)

		if !checkGroupAdminAccess(db, userID, isAdminBool, groupID) {
			respondForbidden(c, "forbidden")
			return
		}

		parsedFooID, err := strconv.ParseUint(fooID, 10, 64)
		if err != nil {
			respondBadRequest(c, "invalid foo id")
			return
		}

		var foo models.Foo
		if err := db.WithContext(ctx).Where("id = ? AND group_id = ?", parsedFooID, groupID).First(&foo).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				respondNotFound(c, "not found")
			} else {
				respondInternalError(c, err.Error())
			}
			return
		}

		var input struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			respondBadRequest(c, err.Error())
			return
		}

		updates := map[string]interface{}{}
		if input.Name != "" {
			updates["name"] = input.Name
		}
		if input.Description != "" {
			updates["description"] = input.Description
		}

		if err := db.WithContext(ctx).Model(&foo).Updates(updates).Error; err != nil {
			respondInternalError(c, err.Error())
			return
		}
		// Reload to return DB-generated values (e.g. updated_at) that map-based Updates may not back-fill.
		if err := db.WithContext(ctx).First(&foo, foo.ID).Error; err != nil {
			respondInternalError(c, err.Error())
			return
		}
		respondOK(c, foo)
	}
}

// DeleteFoo soft-deletes a Foo by ID.
func DeleteFoo(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		groupID := c.Param("id")
		fooID := c.Param("fooId")
		userID, exists := c.Get("user_id")
		if !exists {
			respondUnauthorized(c, "unauthorized")
			return
		}
		// is_admin is always set by AuthRequired alongside user_id; false default is safe (conservatively denies access).
		isAdmin, _ := c.Get("is_admin")
		isAdminBool, _ := isAdmin.(bool)

		if !checkGroupAdminAccess(db, userID, isAdminBool, groupID) {
			respondForbidden(c, "forbidden")
			return
		}

		parsedFooID, err := strconv.ParseUint(fooID, 10, 64)
		if err != nil {
			respondBadRequest(c, "invalid foo id")
			return
		}

		var foo models.Foo
		if err := db.WithContext(ctx).Where("id = ? AND group_id = ?", parsedFooID, groupID).First(&foo).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				respondNotFound(c, "not found")
			} else {
				respondInternalError(c, err.Error())
			}
			return
		}

		if err := db.WithContext(ctx).Delete(&foo).Error; err != nil {
			respondInternalError(c, err.Error())
			return
		}
		respondNoContent(c)
	}
}
