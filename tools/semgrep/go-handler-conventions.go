//go:build ignore

// Semgrep rule fixture for go-handler-conventions.yaml — never compiled.

package handlers

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/networkengineer-cloud/go-volunteer-media/internal/middleware"
)

func GoodHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		db := middleware.GetDB(c, db)
		// ok: handler-reassigns-shared-db
		db = db.Where("group_id = ?", 1)
		var n int64
		db.Model(nil).Count(&n)
		c.JSON(http.StatusOK, n)
	}
}

func ReassignsSharedDB(db *gorm.DB) gin.HandlerFunc {
	// ruleid: handler-missing-request-scoped-db
	return func(c *gin.Context) {
		// ruleid: handler-reassigns-shared-db
		db = db.Where("group_id = ?", 1)
		var n int64
		db.Model(nil).Count(&n)
	}
}

func MissingGetDB(db *gorm.DB) gin.HandlerFunc {
	// ruleid: handler-missing-request-scoped-db
	return func(c *gin.Context) {
		var n int64
		db.Model(nil).Count(&n)
	}
}

func MissingGetDBPassesDB(db *gorm.DB) gin.HandlerFunc {
	// ruleid: handler-missing-request-scoped-db
	return func(c *gin.Context) {
		if !checkGroupAccess(db, nil, nil, "1") {
			return
		}
	}
}

func ScopedUnderAnotherName(db *gorm.DB) gin.HandlerFunc {
	// ok: handler-missing-request-scoped-db
	return func(c *gin.Context) {
		dbCtx := middleware.GetDB(c, db)
		dbCtx.Model(nil)
		go func() { db.Model(nil) }()
	}
}

func NoDBUse(db *gorm.DB) gin.HandlerFunc {
	// ok: handler-missing-request-scoped-db
	return func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	}
}

func LeaksError(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		db := middleware.GetDB(c, db)
		if err := db.Error; err != nil {
			// ruleid: internal-error-leaks-to-client
			respondInternalError(c, err.Error())
			// ruleid: internal-error-leaks-to-client
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			// ok: internal-error-leaks-to-client
			respondInternalError(c, "Failed to load")
			// ok: internal-error-leaks-to-client
			respondBadRequest(c, err.Error())
		}
	}
}

func WritesFiles(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		db := middleware.GetDB(c, db)
		_ = db
		file, _ := c.FormFile("image")
		// ruleid: handler-writes-local-filesystem
		_ = c.SaveUploadedFile(file, "/tmp/x")
		// ruleid: handler-writes-local-filesystem
		_ = os.WriteFile("/tmp/y", nil, 0o600)
	}
}
