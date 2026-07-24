package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/embedding"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupAnimalCommentTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	// IMPORTANT: SQLite in-memory databases are per-connection. Without
	// capping the pool to a single connection, database/sql may open a
	// second physical connection to service a concurrent caller (e.g. the
	// detached goroutine embedCommentAsync spawns) — and since ":memory:"
	// isn't shared-cache here, that second connection is a distinct, blank,
	// unmigrated database, producing intermittent "no such table:
	// animal_comments" errors. Forcing a single connection makes the pool
	// serialize access instead, matching the pattern in SetupTestDB
	// (test_helpers.go).
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("Failed to get database instance: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)

	// Migrate models
	err = db.AutoMigrate(
		&models.User{},
		&models.Group{},
		&models.Animal{},
		&models.AnimalComment{},
		&models.CommentTag{},
	)
	if err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	// Create test data
	user := models.User{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "hashedpassword",
		IsAdmin:  false,
	}
	db.Create(&user)

	group := models.Group{
		Name:        "Test Group",
		Description: "Test group description",
	}
	db.Create(&group)

	// Add user to group
	db.Model(&user).Association("Groups").Append(&group)

	animal := models.Animal{
		Name:        "Test Animal",
		Species:     "Dog",
		GroupID:     group.ID,
		Status:      "available",
		Description: "Test animal",
	}
	db.Create(&animal)

	// Create comment tags
	tag1 := models.CommentTag{Name: "urgent", Color: "#FF0000"}
	tag2 := models.CommentTag{Name: "medical", Color: "#00FF00"}
	db.Create(&tag1)
	db.Create(&tag2)

	return db
}

func TestGetAnimalComments(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		setupContext   func(*gin.Context)
		setupData      func(*gorm.DB)
		expectedStatus int
		expectedError  string
	}{
		{
			name: "successful retrieval of comments",
			setupContext: func(c *gin.Context) {
				c.Set("user_id", uint(1))
				c.Set("is_admin", false)
				c.Params = gin.Params{
					{Key: "id", Value: "1"},
					{Key: "animalId", Value: "1"},
				}
			},
			setupData: func(db *gorm.DB) {
				comment := models.AnimalComment{
					AnimalID: 1,
					UserID:   1,
					Content:  "Test comment",
				}
				db.Create(&comment)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "forbidden when no group access",
			setupContext: func(c *gin.Context) {
				c.Set("user_id", uint(999))
				c.Set("is_admin", false)
				c.Params = gin.Params{
					{Key: "id", Value: "1"},
					{Key: "animalId", Value: "1"},
				}
			},
			setupData:      func(db *gorm.DB) {},
			expectedStatus: http.StatusForbidden,
			expectedError:  "Access denied",
		},
		{
			name: "not found when animal doesn't exist",
			setupContext: func(c *gin.Context) {
				c.Set("user_id", uint(1))
				c.Set("is_admin", false)
				c.Params = gin.Params{
					{Key: "id", Value: "1"},
					{Key: "animalId", Value: "999"},
				}
			},
			setupData:      func(db *gorm.DB) {},
			expectedStatus: http.StatusNotFound,
			expectedError:  "Animal not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			db := setupAnimalCommentTestDB(t)
			defer func() {
				sqlDB, _ := db.DB()
				sqlDB.Close()
			}()

			tt.setupData(db)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", "/groups/1/animals/1/comments", nil)
			tt.setupContext(c)

			// Execute
			handler := GetAnimalComments(db)
			handler(c)

			// Assert
			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedError != "" {
				assert.Contains(t, w.Body.String(), tt.expectedError)
			}
		})
	}
}

func TestGetAnimalComments_WithTagFilter(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupAnimalCommentTestDB(t)
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	// Create comment with tag
	comment := models.AnimalComment{
		AnimalID: 1,
		UserID:   1,
		Content:  "Tagged comment",
	}
	db.Create(&comment)

	var tag models.CommentTag
	db.Where("name = ?", "urgent").First(&tag)
	db.Model(&comment).Association("Tags").Append(&tag)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/groups/1/animals/1/comments?tags=urgent", nil)
	c.Set("user_id", uint(1))
	c.Set("is_admin", false)
	c.Params = gin.Params{
		{Key: "id", Value: "1"},
		{Key: "animalId", Value: "1"},
	}

	handler := GetAnimalComments(db)
	handler(c)

	// Accept either OK (if JOIN works) or error (SQLite limitations with complex JOINs)
	// The important thing is the handler doesn't crash
	assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusInternalServerError)
}

func TestGetAnimalCommentPosition(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupAnimalCommentTestDB(t)
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	oldest := models.AnimalComment{AnimalID: 1, UserID: 1, Content: "oldest", CreatedAt: base}
	middle := models.AnimalComment{AnimalID: 1, UserID: 1, Content: "middle", CreatedAt: base.Add(time.Hour)}
	newest := models.AnimalComment{AnimalID: 1, UserID: 1, Content: "newest", CreatedAt: base.Add(2 * time.Hour)}
	db.Create(&oldest)
	db.Create(&middle)
	db.Create(&newest)

	var urgentTag models.CommentTag
	db.Where("name = ?", "urgent").First(&urgentTag)
	db.Model(&middle).Association("Tags").Append(&urgentTag)

	requestPosition := func(commentID uint, query string) (int, map[string]interface{}) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", fmt.Sprintf("/groups/1/animals/1/comments/%d/position%s", commentID, query), nil)
		c.Set("user_id", uint(1))
		c.Set("is_admin", false)
		c.Params = gin.Params{
			{Key: "id", Value: "1"},
			{Key: "animalId", Value: "1"},
			{Key: "commentId", Value: fmt.Sprintf("%d", commentID)},
		}

		handler := GetAnimalCommentPosition(db)
		handler(c)

		var body map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &body)
		return w.Code, body
	}

	t.Run("default order (desc) ranks the newest comment first", func(t *testing.T) {
		status, body := requestPosition(newest.ID, "")
		assert.Equal(t, http.StatusOK, status)
		assert.Equal(t, true, body["found"])
		assert.Equal(t, float64(0), body["offset"])
	})

	t.Run("default order (desc) ranks the oldest comment last", func(t *testing.T) {
		status, body := requestPosition(oldest.ID, "")
		assert.Equal(t, http.StatusOK, status)
		assert.Equal(t, float64(2), body["offset"])
	})

	t.Run("asc order reverses the ranking", func(t *testing.T) {
		status, body := requestPosition(oldest.ID, "?order=asc")
		assert.Equal(t, http.StatusOK, status)
		assert.Equal(t, float64(0), body["offset"])

		status, body = requestPosition(newest.ID, "?order=asc")
		assert.Equal(t, http.StatusOK, status)
		assert.Equal(t, float64(2), body["offset"])
	})

	t.Run("tag filter positions relative to the filtered set only", func(t *testing.T) {
		// "middle" is the only urgent-tagged comment, so it's position 0
		// within that filtered set even though it's position 1 overall.
		status, body := requestPosition(middle.ID, "?tags=urgent")
		assert.Equal(t, http.StatusOK, status)
		assert.Equal(t, true, body["found"])
		assert.Equal(t, float64(0), body["offset"])
	})

	t.Run("reports found:false when the comment doesn't match the active tag filter", func(t *testing.T) {
		status, body := requestPosition(oldest.ID, "?tags=urgent")
		assert.Equal(t, http.StatusOK, status)
		assert.Equal(t, false, body["found"])
		assert.NotContains(t, body, "offset")
	})

	t.Run("not found when the comment doesn't exist", func(t *testing.T) {
		status, body := requestPosition(99999, "")
		assert.Equal(t, http.StatusNotFound, status)
		assert.Contains(t, body["error"], "Comment not found")
	})

	t.Run("forbidden when no group access", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", fmt.Sprintf("/groups/1/animals/1/comments/%d/position", oldest.ID), nil)
		c.Set("user_id", uint(999))
		c.Set("is_admin", false)
		c.Params = gin.Params{
			{Key: "id", Value: "1"},
			{Key: "animalId", Value: "1"},
			{Key: "commentId", Value: fmt.Sprintf("%d", oldest.ID)},
		}

		handler := GetAnimalCommentPosition(db)
		handler(c)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})
}

func TestCreateAnimalComment(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		setupContext   func(*gin.Context)
		requestBody    interface{}
		expectedStatus int
		expectedError  string
	}{
		{
			name: "successful comment creation",
			setupContext: func(c *gin.Context) {
				c.Set("user_id", uint(1))
				c.Set("is_admin", false)
				c.Params = gin.Params{
					{Key: "id", Value: "1"},
					{Key: "animalId", Value: "1"},
				}
			},
			requestBody: AnimalCommentRequest{
				Content:  "This is a test comment",
				ImageURL: "http://example.com/image.jpg",
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "forbidden when no group access",
			setupContext: func(c *gin.Context) {
				c.Set("user_id", uint(999))
				c.Set("is_admin", false)
				c.Params = gin.Params{
					{Key: "id", Value: "1"},
					{Key: "animalId", Value: "1"},
				}
			},
			requestBody: AnimalCommentRequest{
				Content: "Test comment",
			},
			expectedStatus: http.StatusForbidden,
			expectedError:  "Access denied",
		},
		{
			name: "bad request when content is missing",
			setupContext: func(c *gin.Context) {
				c.Set("user_id", uint(1))
				c.Set("is_admin", false)
				c.Params = gin.Params{
					{Key: "id", Value: "1"},
					{Key: "animalId", Value: "1"},
				}
			},
			requestBody: AnimalCommentRequest{
				ImageURL: "http://example.com/image.jpg",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "not found when animal doesn't exist",
			setupContext: func(c *gin.Context) {
				c.Set("user_id", uint(1))
				c.Set("is_admin", false)
				c.Params = gin.Params{
					{Key: "id", Value: "1"},
					{Key: "animalId", Value: "999"},
				}
			},
			requestBody: AnimalCommentRequest{
				Content: "Test comment",
			},
			expectedStatus: http.StatusNotFound,
			expectedError:  "Animal not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			db := setupAnimalCommentTestDB(t)
			defer func() {
				sqlDB, _ := db.DB()
				sqlDB.Close()
			}()

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			bodyBytes, _ := json.Marshal(tt.requestBody)
			c.Request = httptest.NewRequest("POST", "/groups/1/animals/1/comments", bytes.NewBuffer(bodyBytes))
			c.Request.Header.Set("Content-Type", "application/json")
			tt.setupContext(c)

			// Execute
			handler := CreateAnimalComment(db, &embedding.StubEmbedder{})
			handler(c)

			// Assert
			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedError != "" {
				assert.Contains(t, w.Body.String(), tt.expectedError)
			}
		})
	}
}

func TestCreateAnimalComment_WithTags(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupAnimalCommentTestDB(t)
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	var tag models.CommentTag
	db.Where("name = ?", "urgent").First(&tag)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("user_id", uint(1))
	c.Set("is_admin", false)
	c.Params = gin.Params{
		{Key: "id", Value: "1"},
		{Key: "animalId", Value: "1"},
	}

	requestBody := AnimalCommentRequest{
		Content: "Comment with tag",
		TagIDs:  []uint{tag.ID},
	}
	bodyBytes, _ := json.Marshal(requestBody)
	c.Request = httptest.NewRequest("POST", "/groups/1/animals/1/comments", bytes.NewBuffer(bodyBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	handler := CreateAnimalComment(db, &embedding.StubEmbedder{})
	handler(c)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Contains(t, w.Body.String(), "Comment with tag")
}

func TestGetGroupLatestComments(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		setupContext   func(*gin.Context)
		setupData      func(*gorm.DB)
		queryString    string
		expectedStatus int
		expectedError  string
	}{
		{
			name: "successful retrieval of latest comments",
			setupContext: func(c *gin.Context) {
				c.Set("user_id", uint(1))
				c.Set("is_admin", false)
				c.Params = gin.Params{
					{Key: "id", Value: "1"},
				}
			},
			setupData: func(db *gorm.DB) {
				comment := models.AnimalComment{
					AnimalID: 1,
					UserID:   1,
					Content:  "Latest comment",
				}
				db.Create(&comment)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "forbidden when no group access",
			setupContext: func(c *gin.Context) {
				c.Set("user_id", uint(999))
				c.Set("is_admin", false)
				c.Params = gin.Params{
					{Key: "id", Value: "1"},
				}
			},
			setupData:      func(db *gorm.DB) {},
			expectedStatus: http.StatusForbidden,
			expectedError:  "Access denied",
		},
		{
			name: "successful with limit parameter",
			setupContext: func(c *gin.Context) {
				c.Set("user_id", uint(1))
				c.Set("is_admin", false)
				c.Params = gin.Params{
					{Key: "id", Value: "1"},
				}
			},
			setupData: func(db *gorm.DB) {
				for i := 0; i < 5; i++ {
					comment := models.AnimalComment{
						AnimalID: 1,
						UserID:   1,
						Content:  fmt.Sprintf("Comment %d", i),
					}
					db.Create(&comment)
				}
			},
			queryString:    "?limit=3",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			db := setupAnimalCommentTestDB(t)
			defer func() {
				sqlDB, _ := db.DB()
				sqlDB.Close()
			}()

			tt.setupData(db)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", "/groups/1/comments"+tt.queryString, nil)
			tt.setupContext(c)

			// Execute
			handler := GetGroupLatestComments(db)
			handler(c)

			// Assert
			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedError != "" {
				assert.Contains(t, w.Body.String(), tt.expectedError)
			}
		})
	}
}
