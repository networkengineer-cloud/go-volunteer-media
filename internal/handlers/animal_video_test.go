package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupVideoTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)
	assert.NoError(t, db.AutoMigrate(
		&models.User{},
		&models.Group{},
		&models.UserGroup{},
		&models.Animal{},
		&models.AnimalImage{},
		&models.AnimalVideo{},
	))
	return db
}

func TestGetAnimalMedia(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupVideoTestDB(t)

	group := models.Group{Name: "Dogs", Description: "Dog group"}
	assert.NoError(t, db.Create(&group).Error)

	user := models.User{Username: "vol", Email: "vol@test.com", Password: "x"}
	assert.NoError(t, db.Create(&user).Error)
	assert.NoError(t, db.Model(&user).Association("Groups").Append(&group))

	animal := models.Animal{Name: "Rex", Species: "Dog", GroupID: group.ID, Status: "available"}
	assert.NoError(t, db.Create(&animal).Error)

	// Seed one image and one video
	animalIDRef := animal.ID
	assert.NoError(t, db.Create(&models.AnimalImage{
		AnimalID: &animalIDRef,
		UserID:   user.ID,
		ImageURL: "/images/test.jpg",
	}).Error)
	assert.NoError(t, db.Create(&models.AnimalVideo{
		AnimalID:     &animalIDRef,
		UserID:       user.ID,
		VideoURL:     "/videos/test.mp4",
		ThumbnailURL: "/images/thumb.jpg",
	}).Error)

	r := gin.New()
	r.GET("/groups/:id/animals/:animalId/media", func(c *gin.Context) {
		c.Set("user_id", user.ID)
		c.Set("is_admin", false)
	}, GetAnimalMedia(db))

	req := httptest.NewRequest(http.MethodGet,
		"/groups/"+itoa(group.ID)+"/animals/"+itoa(animal.ID)+"/media", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var body struct {
		Images []models.AnimalImage `json:"images"`
		Videos []models.AnimalVideo `json:"videos"`
	}
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Len(t, body.Images, 1)
	assert.Len(t, body.Videos, 1)
}

// itoa converts uint to string for URL building in tests.
func itoa(n uint) string {
	return fmt.Sprintf("%d", n)
}
