package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
)

// TestSetAnimalProfilePictureGroupScoped tests the profile picture setting functionality
func TestSetAnimalProfilePictureGroupScoped(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	// Auto-migrate
	err = db.AutoMigrate(&models.User{}, &models.Group{}, &models.UserGroup{}, &models.Animal{}, &models.AnimalImage{})
	assert.NoError(t, err)

	// Create test data
	group1 := models.Group{Name: "Dogs", Description: "Dog volunteers"}
	assert.NoError(t, db.Create(&group1).Error)

	group2 := models.Group{Name: "Cats", Description: "Cat volunteers"}
	assert.NoError(t, db.Create(&group2).Error)

	// Create users
	user1 := models.User{
		Username: "dogvolunteer",
		Email:    "dog@example.com",
		Password: "hashedpassword",
		IsAdmin:  false,
	}
	assert.NoError(t, db.Create(&user1).Error)
	assert.NoError(t, db.Model(&user1).Association("Groups").Append(&group1))

	user2 := models.User{
		Username: "catvolunteer",
		Email:    "cat@example.com",
		Password: "hashedpassword",
		IsAdmin:  false,
	}
	assert.NoError(t, db.Create(&user2).Error)
	assert.NoError(t, db.Model(&user2).Association("Groups").Append(&group2))

	admin := models.User{
		Username: "admin",
		Email:    "admin@example.com",
		Password: "hashedpassword",
		IsAdmin:  true,
	}
	assert.NoError(t, db.Create(&admin).Error)

	// Create animals
	animalID1 := uint(1)
	animal1 := models.Animal{
		Name:     "Rex",
		Species:  "Dog",
		GroupID:  group1.ID,
		Status:   "available",
		ImageURL: "/api/images/old-image",
	}
	assert.NoError(t, db.Create(&animal1).Error)
	animalID1 = animal1.ID

	animal2 := models.Animal{
		Name:    "Fluffy",
		Species: "Cat",
		GroupID: group2.ID,
		Status:  "available",
	}
	assert.NoError(t, db.Create(&animal2).Error)

	// Create images for animal1
	image1 := models.AnimalImage{
		AnimalID:         &animalID1,
		UserID:           user1.ID,
		ImageURL:         "/api/images/image1",
		IsProfilePicture: true,
		MimeType:         "image/jpeg",
	}
	assert.NoError(t, db.Create(&image1).Error)

	image2 := models.AnimalImage{
		AnimalID:         &animalID1,
		UserID:           user1.ID,
		ImageURL:         "/api/images/image2",
		IsProfilePicture: false,
		MimeType:         "image/jpeg",
	}
	assert.NoError(t, db.Create(&image2).Error)

	tests := []struct {
		name           string
		groupID        uint
		animalID       uint
		imageID        uint
		userID         uint
		isAdmin        bool
		expectedStatus int
		checkResult    func(*testing.T, *httptest.ResponseRecorder, *gorm.DB)
	}{
		{
			name:           "group member can set profile picture",
			groupID:        group1.ID,
			animalID:       animalID1,
			imageID:        image2.ID,
			userID:         user1.ID,
			isAdmin:        false,
			expectedStatus: http.StatusOK,
			checkResult: func(t *testing.T, w *httptest.ResponseRecorder, db *gorm.DB) {
				// Verify the image is now the profile picture
				var updatedImage models.AnimalImage
				err := db.First(&updatedImage, image2.ID).Error
				assert.NoError(t, err)
				assert.True(t, updatedImage.IsProfilePicture)

				// Verify the old profile picture is no longer marked as profile
				var oldImage models.AnimalImage
				err = db.First(&oldImage, image1.ID).Error
				assert.NoError(t, err)
				assert.False(t, oldImage.IsProfilePicture)

				// Verify the animal's image_url is updated
				var updatedAnimal models.Animal
				err = db.First(&updatedAnimal, animalID1).Error
				assert.NoError(t, err)
				assert.Equal(t, "/api/images/image2", updatedAnimal.ImageURL)
			},
		},
		{
			name:           "non-member cannot set profile picture",
			groupID:        group1.ID,
			animalID:       animalID1,
			imageID:        image2.ID,
			userID:         user2.ID, // Cat volunteer trying to access dog group
			isAdmin:        false,
			expectedStatus: http.StatusForbidden,
			checkResult: func(t *testing.T, w *httptest.ResponseRecorder, db *gorm.DB) {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Contains(t, response["error"], "Access denied")
			},
		},
		{
			name:           "admin can set profile picture regardless of group",
			groupID:        group1.ID,
			animalID:       animalID1,
			imageID:        image2.ID,
			userID:         admin.ID,
			isAdmin:        true,
			expectedStatus: http.StatusOK,
			checkResult:    func(t *testing.T, w *httptest.ResponseRecorder, db *gorm.DB) {},
		},
		{
			name:           "invalid group returns 403 (access denied)",
			groupID:        999,
			animalID:       animalID1,
			imageID:        image2.ID,
			userID:         user1.ID,
			isAdmin:        false,
			expectedStatus: http.StatusForbidden,
			checkResult:    func(t *testing.T, w *httptest.ResponseRecorder, db *gorm.DB) {},
		},
		{
			name:           "invalid animal returns 404",
			groupID:        group1.ID,
			animalID:       999,
			imageID:        image2.ID,
			userID:         user1.ID,
			isAdmin:        false,
			expectedStatus: http.StatusNotFound,
			checkResult:    func(t *testing.T, w *httptest.ResponseRecorder, db *gorm.DB) {},
		},
		{
			name:           "invalid image returns 404",
			groupID:        group1.ID,
			animalID:       animalID1,
			imageID:        999,
			userID:         user1.ID,
			isAdmin:        false,
			expectedStatus: http.StatusNotFound,
			checkResult:    func(t *testing.T, w *httptest.ResponseRecorder, db *gorm.DB) {},
		},
		{
			name:           "animal from wrong group returns 404",
			groupID:        group2.ID,
			animalID:       animalID1, // Dog animal in cat group
			imageID:        image2.ID,
			userID:         user2.ID,
			isAdmin:        false,
			expectedStatus: http.StatusNotFound,
			checkResult:    func(t *testing.T, w *httptest.ResponseRecorder, db *gorm.DB) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test context
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			// Setup request
			c.Request = httptest.NewRequest("PUT", fmt.Sprintf("/api/groups/%d/animals/%d/images/%d/set-profile", tt.groupID, tt.animalID, tt.imageID), nil)
			c.Params = gin.Params{
				{Key: "id", Value: fmt.Sprintf("%d", tt.groupID)},
				{Key: "animalId", Value: fmt.Sprintf("%d", tt.animalID)},
				{Key: "imageId", Value: fmt.Sprintf("%d", tt.imageID)},
			}

			// Setup authentication
			c.Set("user_id", tt.userID)
			c.Set("is_admin", tt.isAdmin)

			// Call handler
			handler := SetAnimalProfilePictureGroupScoped(db)
			handler(c)

			// Assert status code
			assert.Equal(t, tt.expectedStatus, w.Code, "Status code mismatch")

			// Run additional checks
			if tt.checkResult != nil {
				tt.checkResult(t, w, db)
			}
		})
	}
}

// TestSetAnimalProfilePicture_Transaction tests that the operation is atomic
func TestSetAnimalProfilePicture_Transaction(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	// Auto-migrate
	err = db.AutoMigrate(&models.User{}, &models.Group{}, &models.UserGroup{}, &models.Animal{}, &models.AnimalImage{})
	assert.NoError(t, err)

	// Create test data
	group := models.Group{Name: "Dogs", Description: "Dog volunteers"}
	assert.NoError(t, db.Create(&group).Error)

	user := models.User{
		Username: "volunteer",
		Email:    "volunteer@example.com",
		Password: "hashedpassword",
		IsAdmin:  false,
	}
	assert.NoError(t, db.Create(&user).Error)
	assert.NoError(t, db.Model(&user).Association("Groups").Append(&group))

	animalID := uint(1)
	animal := models.Animal{
		Name:     "Rex",
		Species:  "Dog",
		GroupID:  group.ID,
		Status:   "available",
		ImageURL: "/api/images/old-image",
	}
	assert.NoError(t, db.Create(&animal).Error)
	animalID = animal.ID

	// Create two images
	image1 := models.AnimalImage{
		AnimalID:         &animalID,
		UserID:           user.ID,
		ImageURL:         "/api/images/image1",
		IsProfilePicture: true,
		MimeType:         "image/jpeg",
	}
	assert.NoError(t, db.Create(&image1).Error)

	image2 := models.AnimalImage{
		AnimalID:         &animalID,
		UserID:           user.ID,
		ImageURL:         "/api/images/image2",
		IsProfilePicture: false,
		MimeType:         "image/jpeg",
	}
	assert.NoError(t, db.Create(&image2).Error)

	// Set image2 as profile picture
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("PUT", fmt.Sprintf("/api/groups/%d/animals/%d/images/%d/set-profile", group.ID, animalID, image2.ID), nil)
	c.Params = gin.Params{
		{Key: "id", Value: fmt.Sprintf("%d", group.ID)},
		{Key: "animalId", Value: fmt.Sprintf("%d", animalID)},
		{Key: "imageId", Value: fmt.Sprintf("%d", image2.ID)},
	}
	c.Set("user_id", user.ID)
	c.Set("is_admin", false)

	handler := SetAnimalProfilePictureGroupScoped(db)
	handler(c)

	assert.Equal(t, http.StatusOK, w.Code)

	// Verify exactly one image is marked as profile picture
	var profileImages []models.AnimalImage
	err = db.Where("animal_id = ? AND is_profile_picture = ?", animalID, true).Find(&profileImages).Error
	assert.NoError(t, err)
	assert.Equal(t, 1, len(profileImages), "Should have exactly one profile picture")
	assert.Equal(t, image2.ID, profileImages[0].ID, "The new image should be the profile picture")
}

// TestSetAnimalProfilePicture_ConcurrentRequests tests handling of concurrent profile picture updates
func TestSetAnimalProfilePicture_ConcurrentRequests(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	// Auto-migrate
	err = db.AutoMigrate(&models.User{}, &models.Group{}, &models.UserGroup{}, &models.Animal{}, &models.AnimalImage{})
	assert.NoError(t, err)

	// Create test data
	group := models.Group{Name: "Dogs", Description: "Dog volunteers"}
	assert.NoError(t, db.Create(&group).Error)

	user := models.User{
		Username: "volunteer",
		Email:    "volunteer@example.com",
		Password: "hashedpassword",
		IsAdmin:  false,
	}
	assert.NoError(t, db.Create(&user).Error)
	assert.NoError(t, db.Model(&user).Association("Groups").Append(&group))

	animalID := uint(1)
	animal := models.Animal{
		Name:     "Rex",
		Species:  "Dog",
		GroupID:  group.ID,
		Status:   "available",
		ImageURL: "/api/images/old-image",
	}
	assert.NoError(t, db.Create(&animal).Error)
	animalID = animal.ID

	// Create multiple images
	images := make([]models.AnimalImage, 3)
	for i := 0; i < 3; i++ {
		images[i] = models.AnimalImage{
			AnimalID:         &animalID,
			UserID:           user.ID,
			ImageURL:         fmt.Sprintf("/api/images/image%d", i+1),
			IsProfilePicture: i == 0, // First one is profile
			MimeType:         "image/jpeg",
		}
		assert.NoError(t, db.Create(&images[i]).Error)
	}

	// Simulate concurrent requests by making multiple calls
	// Note: SQLite doesn't support true concurrency well, but this tests the transaction logic
	for i := 1; i < 3; i++ {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("PUT", fmt.Sprintf("/api/groups/%d/animals/%d/images/%d/set-profile", group.ID, animalID, images[i].ID), nil)
		c.Params = gin.Params{
			{Key: "id", Value: fmt.Sprintf("%d", group.ID)},
			{Key: "animalId", Value: fmt.Sprintf("%d", animalID)},
			{Key: "imageId", Value: fmt.Sprintf("%d", images[i].ID)},
		}
		c.Set("user_id", user.ID)
		c.Set("is_admin", false)

		handler := SetAnimalProfilePictureGroupScoped(db)
		handler(c)

		assert.Equal(t, http.StatusOK, w.Code)
	}

	// Final check: exactly one profile picture should exist
	var profileImages []models.AnimalImage
	err = db.Where("animal_id = ? AND is_profile_picture = ?", animalID, true).Find(&profileImages).Error
	assert.NoError(t, err)
	assert.Equal(t, 1, len(profileImages), "Should have exactly one profile picture after concurrent updates")
}
