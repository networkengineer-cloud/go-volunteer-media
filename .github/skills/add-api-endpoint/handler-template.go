// handler-template.go — Copy this file as internal/handlers/<feature>.go
// Replace "Foo"/"foo" with your entity name throughout.
package handlers

import (
	"net/http"
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
		userID, _ := c.Get("user_id")
		isAdmin, _ := c.Get("is_admin")

		if !checkGroupAccess(db, userID, isAdmin.(bool), groupID) {
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

// CreateFoo creates a new Foo in the given group.
func CreateFoo(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		groupID := c.Param("id")
		userID, _ := c.Get("user_id")
		isAdmin, _ := c.Get("is_admin")

		if !checkGroupAccess(db, userID, isAdmin.(bool), groupID) {
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
		c.JSON(http.StatusCreated, foo)
	}
}

// UpdateFoo updates an existing Foo by ID.
func UpdateFoo(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		groupID := c.Param("id")
		fooID := c.Param("fooId")
		userID, _ := c.Get("user_id")
		isAdmin, _ := c.Get("is_admin")

		if !checkGroupAccess(db, userID, isAdmin.(bool), groupID) {
			respondForbidden(c, "forbidden")
			return
		}

		var foo models.Foo
		if err := db.WithContext(ctx).Where("id = ? AND group_id = ?", fooID, groupID).First(&foo).Error; err != nil {
			respondNotFound(c, "not found")
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

		if input.Name != "" {
			foo.Name = input.Name
		}
		foo.Description = input.Description

		if err := db.WithContext(ctx).Save(&foo).Error; err != nil {
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
		userID, _ := c.Get("user_id")
		isAdmin, _ := c.Get("is_admin")

		if !checkGroupAccess(db, userID, isAdmin.(bool), groupID) {
			respondForbidden(c, "forbidden")
			return
		}

		var foo models.Foo
		if err := db.WithContext(ctx).Where("id = ? AND group_id = ?", fooID, groupID).First(&foo).Error; err != nil {
			respondNotFound(c, "not found")
			return
		}

		if err := db.WithContext(ctx).Delete(&foo).Error; err != nil {
			respondInternalError(c, err.Error())
			return
		}
		respondNoContent(c)
	}
}
