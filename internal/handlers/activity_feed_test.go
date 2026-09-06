package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupActivityFeedTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	// Migrate models
	err = db.AutoMigrate(
		&models.User{},
		&models.Group{},
		&models.Animal{},
		&models.AnimalComment{},
		&models.Update{},
		&models.CommentTag{},
		&models.ShiftCoverageRequest{},
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

	comment := models.AnimalComment{
		AnimalID: animal.ID,
		UserID:   user.ID,
		Content:  "Test comment",
	}
	db.Create(&comment)

	update := models.Update{
		GroupID: group.ID,
		UserID:  user.ID,
		Title:   "Test Update",
		Content: "Test update content",
	}
	db.Create(&update)

	return db
}

func TestGetGroupActivityFeed(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		setupContext   func(*gin.Context)
		queryString    string
		expectedStatus int
		expectedError  string
	}{
		{
			name: "successful retrieval of activity feed",
			setupContext: func(c *gin.Context) {
				c.Set("user_id", uint(1))
				c.Set("is_admin", false)
				c.Params = gin.Params{{Key: "id", Value: "1"}}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "forbidden when no group access",
			setupContext: func(c *gin.Context) {
				c.Set("user_id", uint(999))
				c.Set("is_admin", false)
				c.Params = gin.Params{{Key: "id", Value: "1"}}
			},
			expectedStatus: http.StatusForbidden,
			expectedError:  "Access denied",
		},
		{
			name: "successful with limit parameter",
			setupContext: func(c *gin.Context) {
				c.Set("user_id", uint(1))
				c.Set("is_admin", false)
				c.Params = gin.Params{{Key: "id", Value: "1"}}
			},
			queryString:    "?limit=5",
			expectedStatus: http.StatusOK,
		},
		{
			name: "successful with offset parameter",
			setupContext: func(c *gin.Context) {
				c.Set("user_id", uint(1))
				c.Set("is_admin", false)
				c.Params = gin.Params{{Key: "id", Value: "1"}}
			},
			queryString:    "?offset=0&limit=10",
			expectedStatus: http.StatusOK,
		},
		{
			name: "successful with type filter for comments",
			setupContext: func(c *gin.Context) {
				c.Set("user_id", uint(1))
				c.Set("is_admin", false)
				c.Params = gin.Params{{Key: "id", Value: "1"}}
			},
			queryString:    "?type=comments",
			expectedStatus: http.StatusOK,
		},
		{
			name: "successful with type filter for announcements",
			setupContext: func(c *gin.Context) {
				c.Set("user_id", uint(1))
				c.Set("is_admin", false)
				c.Params = gin.Params{{Key: "id", Value: "1"}}
			},
			queryString:    "?type=announcements",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			db := setupActivityFeedTestDB(t)
			defer func() {
				sqlDB, _ := db.DB()
				sqlDB.Close()
			}()

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", "/groups/1/activity"+tt.queryString, nil)
			tt.setupContext(c)

			// Execute
			handler := GetGroupActivityFeed(db)
			handler(c)

			// Assert
			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedError != "" {
				assert.Contains(t, w.Body.String(), tt.expectedError)
			}
		})
	}
}

// fetchActivityFeed calls GetGroupActivityFeed as user 1 (the fixture user
// from setupActivityFeedTestDB) and decodes the JSON body, mirroring the
// shape of feedRequest in activity_feed_postgres_test.go but kept local
// here since this file's tests run against the in-memory SQLite DB, not a
// real Postgres instance.
func fetchActivityFeed(t *testing.T, db *gorm.DB, query string) map[string]interface{} {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("user_id", uint(1))
	c.Set("is_admin", false)
	c.Params = gin.Params{{Key: "id", Value: "1"}}
	url := "/groups/1/activity"
	if query != "" {
		url += "?" + query
	}
	c.Request = httptest.NewRequest("GET", url, nil)

	GetGroupActivityFeed(db)(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v (body: %s)", err, w.Body.String())
	}
	return body
}

func itemsOfType(t *testing.T, body map[string]interface{}, itemType string) []map[string]interface{} {
	t.Helper()
	items, _ := body["items"].([]interface{})
	var matched []map[string]interface{}
	for _, raw := range items {
		item := raw.(map[string]interface{})
		if item["type"] == itemType {
			matched = append(matched, item)
		}
	}
	return matched
}

func TestGetGroupActivityFeed_IncludesOpenCoverageRequest(t *testing.T) {
	db := setupActivityFeedTestDB(t)
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	req := models.ShiftCoverageRequest{
		GroupID:           1,
		RequestedByUserID: 1,
		Date:              time.Date(2026, 9, 12, 0, 0, 0, 0, time.UTC),
		Hour:              9,
		Status:            models.CoverageRequestOpen,
	}
	if err := db.Create(&req).Error; err != nil {
		t.Fatalf("create coverage request: %v", err)
	}

	body := fetchActivityFeed(t, db, "")

	coverageItems := itemsOfType(t, body, "coverage_request")
	if len(coverageItems) != 1 {
		t.Fatalf("expected exactly 1 coverage_request item, got %d: %v", len(coverageItems), body["items"])
	}
	item := coverageItems[0]
	if item["status"] != "open" {
		t.Fatalf("expected status \"open\", got %v", item["status"])
	}
	if item["claimed_by_user"] != nil {
		t.Fatalf("expected no claimed_by_user on an open request, got %v", item["claimed_by_user"])
	}
	if item["hour"] != float64(9) {
		t.Fatalf("expected hour=9, got %v", item["hour"])
	}
}

func TestGetGroupActivityFeed_ClaimedCoverageRequestIncludesClaimer(t *testing.T) {
	db := setupActivityFeedTestDB(t)
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	claimer := models.User{Username: "claimer", Email: "claimer@example.com", Password: "hashedpassword"}
	if err := db.Create(&claimer).Error; err != nil {
		t.Fatalf("create claimer: %v", err)
	}

	req := models.ShiftCoverageRequest{
		GroupID:           1,
		RequestedByUserID: 1,
		Date:              time.Date(2026, 9, 12, 0, 0, 0, 0, time.UTC),
		Hour:              9,
		Status:            models.CoverageRequestClaimed,
		ClaimedByUserID:   &claimer.ID,
	}
	if err := db.Create(&req).Error; err != nil {
		t.Fatalf("create coverage request: %v", err)
	}

	body := fetchActivityFeed(t, db, "")

	coverageItems := itemsOfType(t, body, "coverage_request")
	if len(coverageItems) != 1 {
		t.Fatalf("expected exactly 1 coverage_request item, got %d: %v", len(coverageItems), body["items"])
	}
	item := coverageItems[0]
	if item["status"] != "claimed" {
		t.Fatalf("expected status \"claimed\", got %v", item["status"])
	}
	claimedByUser, ok := item["claimed_by_user"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected claimed_by_user to be present on a claimed request, got %v", item["claimed_by_user"])
	}
	if uint(claimedByUser["id"].(float64)) != claimer.ID {
		t.Fatalf("expected claimed_by_user.id=%d, got %v", claimer.ID, claimedByUser["id"])
	}
}

func TestGetGroupActivityFeed_ExcludesCancelledCoverageRequest(t *testing.T) {
	db := setupActivityFeedTestDB(t)
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	req := models.ShiftCoverageRequest{
		GroupID:           1,
		RequestedByUserID: 1,
		Date:              time.Date(2026, 9, 12, 0, 0, 0, 0, time.UTC),
		Hour:              9,
		Status:            models.CoverageRequestCancelled,
	}
	if err := db.Create(&req).Error; err != nil {
		t.Fatalf("create coverage request: %v", err)
	}

	body := fetchActivityFeed(t, db, "")

	if got := itemsOfType(t, body, "coverage_request"); len(got) != 0 {
		t.Fatalf("expected cancelled coverage requests to be excluded from the feed, got %v", got)
	}
}

func TestGetGroupActivityFeed_FilterTypeCoverageRequestsOnlyExcludesCommentsAndAnnouncements(t *testing.T) {
	db := setupActivityFeedTestDB(t)
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	req := models.ShiftCoverageRequest{
		GroupID:           1,
		RequestedByUserID: 1,
		Date:              time.Date(2026, 9, 12, 0, 0, 0, 0, time.UTC),
		Hour:              9,
		Status:            models.CoverageRequestOpen,
	}
	if err := db.Create(&req).Error; err != nil {
		t.Fatalf("create coverage request: %v", err)
	}

	body := fetchActivityFeed(t, db, "type=coverage_requests")

	items, _ := body["items"].([]interface{})
	if len(items) != 1 {
		t.Fatalf("expected exactly 1 item with type=coverage_requests (the setup fixture's comment/announcement excluded), got %d: %v", len(items), items)
	}
	if items[0].(map[string]interface{})["type"] != "coverage_request" {
		t.Fatalf("expected the one item to be a coverage_request, got %v", items[0])
	}
}
