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

// singleShift asserts a coverage_request item's coverage_shifts array has
// exactly one entry and returns it, for tests exercising a single (ungrouped)
// request.
func singleShift(t *testing.T, item map[string]interface{}) map[string]interface{} {
	t.Helper()
	shifts, _ := item["coverage_shifts"].([]interface{})
	if len(shifts) != 1 {
		t.Fatalf("expected exactly 1 shift in coverage_shifts, got %d: %v", len(shifts), item["coverage_shifts"])
	}
	shift, ok := shifts[0].(map[string]interface{})
	if !ok {
		t.Fatalf("expected coverage_shifts[0] to be an object, got %v", shifts[0])
	}
	return shift
}

func TestGetGroupActivityFeed_IncludesOpenCoverageRequest(t *testing.T) {
	t.Setenv("COVERAGE_REQUESTS_FEED_ENABLED", "true")
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
	shift := singleShift(t, coverageItems[0])
	if shift["status"] != "open" {
		t.Fatalf("expected status \"open\", got %v", shift["status"])
	}
	if shift["claimed_by_user"] != nil {
		t.Fatalf("expected no claimed_by_user on an open request, got %v", shift["claimed_by_user"])
	}
	if shift["hour"] != float64(9) {
		t.Fatalf("expected hour=9, got %v", shift["hour"])
	}
	// Regression: a raw time.Time here marshals as a full RFC3339 timestamp
	// ("2026-09-12T00:00:00Z"), not the plain date scheduleGrid.ts's
	// formatDateLabel/dayOfWeekFromIso expect - they append their own
	// "T00:00:00Z" suffix, so a timestamp value here double-suffixes into an
	// invalid date string on the frontend.
	if shift["date"] != "2026-09-12" {
		t.Fatalf("expected date=\"2026-09-12\" (date-only, not a timestamp), got %v", shift["date"])
	}
}

func TestGetGroupActivityFeed_ClaimedCoverageRequestIncludesClaimer(t *testing.T) {
	t.Setenv("COVERAGE_REQUESTS_FEED_ENABLED", "true")
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
	shift := singleShift(t, coverageItems[0])
	if shift["status"] != "claimed" {
		t.Fatalf("expected status \"claimed\", got %v", shift["status"])
	}
	claimedByUser, ok := shift["claimed_by_user"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected claimed_by_user to be present on a claimed request, got %v", shift["claimed_by_user"])
	}
	if uint(claimedByUser["id"].(float64)) != claimer.ID {
		t.Fatalf("expected claimed_by_user.id=%d, got %v", claimer.ID, claimedByUser["id"])
	}
}

func TestGetGroupActivityFeed_ExcludesCancelledCoverageRequest(t *testing.T) {
	t.Setenv("COVERAGE_REQUESTS_FEED_ENABLED", "true")
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
	t.Setenv("COVERAGE_REQUESTS_FEED_ENABLED", "true")
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

// TestGetGroupActivityFeed_CoverageRequestsHiddenByDefault guards the
// COVERAGE_REQUESTS_FEED_ENABLED feature flag's opt-in default: this feed
// item type is still rolling out, so an operator who hasn't explicitly set
// the env var must never see coverage requests, under "all" or even under an
// explicit type=coverage_requests filter.
func TestGetGroupActivityFeed_CoverageRequestsHiddenByDefault(t *testing.T) {
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

	if body := fetchActivityFeed(t, db, ""); len(itemsOfType(t, body, "coverage_request")) != 0 {
		t.Fatalf("expected no coverage_request items under the default-disabled flag, got %v", body["items"])
	}

	body := fetchActivityFeed(t, db, "type=coverage_requests")
	items, _ := body["items"].([]interface{})
	if len(items) != 0 {
		t.Fatalf("expected type=coverage_requests to return nothing under the default-disabled flag, got %v", items)
	}
}

// TestGetGroupActivityFeed_GroupsBatchedCoverageRequestsFromSameRequester
// covers the main case groupCoverageRequests exists for: a single batch
// submission (CreateCoverageRequestsBatch) creates several
// ShiftCoverageRequest rows in quick succession, and the feed should show
// that as one card listing every shift, not one card per row.
func TestGetGroupActivityFeed_GroupsBatchedCoverageRequestsFromSameRequester(t *testing.T) {
	t.Setenv("COVERAGE_REQUESTS_FEED_ENABLED", "true")
	db := setupActivityFeedTestDB(t)
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	base := time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC)
	first := models.ShiftCoverageRequest{
		GroupID: 1, RequestedByUserID: 1,
		Date: time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC), Hour: 15,
		Status: models.CoverageRequestOpen, CreatedAt: base,
	}
	second := models.ShiftCoverageRequest{
		GroupID: 1, RequestedByUserID: 1,
		Date: time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC), Hour: 14,
		Status: models.CoverageRequestOpen, CreatedAt: base.Add(3 * time.Second),
	}
	if err := db.Create(&first).Error; err != nil {
		t.Fatalf("create first coverage request: %v", err)
	}
	if err := db.Create(&second).Error; err != nil {
		t.Fatalf("create second coverage request: %v", err)
	}

	body := fetchActivityFeed(t, db, "")

	coverageItems := itemsOfType(t, body, "coverage_request")
	if len(coverageItems) != 1 {
		t.Fatalf("expected the two batched requests to collapse into 1 item, got %d: %v", len(coverageItems), body["items"])
	}
	shifts, _ := coverageItems[0]["coverage_shifts"].([]interface{})
	if len(shifts) != 2 {
		t.Fatalf("expected 2 shifts in the grouped item, got %d: %v", len(shifts), coverageItems[0]["coverage_shifts"])
	}
	// Display order is by date/hour, not creation order, so the 14:00 shift
	// (created second) should come first.
	if shifts[0].(map[string]interface{})["hour"] != float64(14) {
		t.Fatalf("expected shifts sorted by hour, got %v", shifts)
	}
	if shifts[1].(map[string]interface{})["hour"] != float64(15) {
		t.Fatalf("expected shifts sorted by hour, got %v", shifts)
	}

	// Scoped to just coverage requests, total should count the group once,
	// not the 2 underlying rows - it drives this endpoint's pagination.
	scoped := fetchActivityFeed(t, db, "type=coverage_requests")
	if scoped["total"] != float64(1) {
		t.Fatalf("expected total to count the group once, got %v", scoped["total"])
	}
}

// TestGetGroupActivityFeed_DoesNotGroupCoverageRequestsAcrossBatchWindow
// ensures two requests from the same person well outside
// coverageRequestBatchWindow of each other are treated as separate actions,
// not merged into one card.
func TestGetGroupActivityFeed_DoesNotGroupCoverageRequestsAcrossBatchWindow(t *testing.T) {
	t.Setenv("COVERAGE_REQUESTS_FEED_ENABLED", "true")
	db := setupActivityFeedTestDB(t)
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	base := time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC)
	first := models.ShiftCoverageRequest{
		GroupID: 1, RequestedByUserID: 1,
		Date: time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC), Hour: 14,
		Status: models.CoverageRequestOpen, CreatedAt: base,
	}
	second := models.ShiftCoverageRequest{
		GroupID: 1, RequestedByUserID: 1,
		Date: time.Date(2026, 9, 26, 0, 0, 0, 0, time.UTC), Hour: 14,
		Status: models.CoverageRequestOpen, CreatedAt: base.Add(coverageRequestBatchWindow + time.Second),
	}
	if err := db.Create(&first).Error; err != nil {
		t.Fatalf("create first coverage request: %v", err)
	}
	if err := db.Create(&second).Error; err != nil {
		t.Fatalf("create second coverage request: %v", err)
	}

	body := fetchActivityFeed(t, db, "")

	coverageItems := itemsOfType(t, body, "coverage_request")
	if len(coverageItems) != 2 {
		t.Fatalf("expected 2 separate items outside the batch window, got %d: %v", len(coverageItems), body["items"])
	}
}

// TestGetGroupActivityFeed_DoesNotGroupCoverageRequestsAcrossRequesters
// ensures two different people requesting coverage at the same instant never
// merge into one card, even though the timing alone would otherwise chain
// them.
func TestGetGroupActivityFeed_DoesNotGroupCoverageRequestsAcrossRequesters(t *testing.T) {
	t.Setenv("COVERAGE_REQUESTS_FEED_ENABLED", "true")
	db := setupActivityFeedTestDB(t)
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	otherUser := models.User{Username: "otherrequester", Email: "other@example.com", Password: "hashedpassword"}
	if err := db.Create(&otherUser).Error; err != nil {
		t.Fatalf("create other user: %v", err)
	}

	base := time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC)
	first := models.ShiftCoverageRequest{
		GroupID: 1, RequestedByUserID: 1,
		Date: time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC), Hour: 14,
		Status: models.CoverageRequestOpen, CreatedAt: base,
	}
	second := models.ShiftCoverageRequest{
		GroupID: 1, RequestedByUserID: otherUser.ID,
		Date: time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC), Hour: 15,
		Status: models.CoverageRequestOpen, CreatedAt: base,
	}
	if err := db.Create(&first).Error; err != nil {
		t.Fatalf("create first coverage request: %v", err)
	}
	if err := db.Create(&second).Error; err != nil {
		t.Fatalf("create second coverage request: %v", err)
	}

	body := fetchActivityFeed(t, db, "")

	coverageItems := itemsOfType(t, body, "coverage_request")
	if len(coverageItems) != 2 {
		t.Fatalf("expected 2 separate items across different requesters, got %d: %v", len(coverageItems), body["items"])
	}
}

// TestGetGroupActivityFeed_CoverageRequestGroupSpanIsCappedAtBatchWindow
// guards against chaining off only the previous row: four single requests,
// each spaced just under coverageRequestBatchWindow/2 apart (well inside the
// window pairwise) but well over the window apart end-to-end, must NOT all
// collapse into one group - only into 2, since rows 0-1 and rows 2-3 are
// each within the window of their own group's first row, but row 2 is not
// within the window of row 0. Spacing is derived from the constant itself
// so this test keeps testing the same relationship if the window changes.
func TestGetGroupActivityFeed_CoverageRequestGroupSpanIsCappedAtBatchWindow(t *testing.T) {
	t.Setenv("COVERAGE_REQUESTS_FEED_ENABLED", "true")
	db := setupActivityFeedTestDB(t)
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	step := coverageRequestBatchWindow/2 + time.Second
	base := time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC)
	for i := 0; i < 4; i++ {
		req := models.ShiftCoverageRequest{
			GroupID: 1, RequestedByUserID: 1,
			Date: time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC), Hour: 9 + i,
			Status: models.CoverageRequestOpen, CreatedAt: base.Add(time.Duration(i) * step),
		}
		if err := db.Create(&req).Error; err != nil {
			t.Fatalf("create coverage request %d: %v", i, err)
		}
	}

	body := fetchActivityFeed(t, db, "")

	coverageItems := itemsOfType(t, body, "coverage_request")
	if len(coverageItems) != 2 {
		t.Fatalf("expected a %s total span to split into 2 groups (rows 0-1 within the window of row 0, rows 2-3 within the window of row 2 but not row 0), got %d: %v", 3*step, len(coverageItems), body["items"])
	}
}

// TestGetGroupActivityFeed_CoverageRequestsAreCappedAtFetchLimit guards
// against the unbounded-forever-growing query this cap exists to fix: an
// open request never expires (not even once its own shift date has passed),
// so a group's history of never-claimed, never-cancelled requests -
// "optional" ones especially - can otherwise accumulate without limit.
// Creates well more than coverageRequestFetchCap rows, each an hour apart
// (comfortably outside coverageRequestBatchWindow, so none group together)
// and asserts only coverageRequestFetchCap of them - the most recent -
// come back.
func TestGetGroupActivityFeed_CoverageRequestsAreCappedAtFetchLimit(t *testing.T) {
	t.Setenv("COVERAGE_REQUESTS_FEED_ENABLED", "true")
	db := setupActivityFeedTestDB(t)
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	const overCap = coverageRequestFetchCap + 50
	for i := 0; i < overCap; i++ {
		req := models.ShiftCoverageRequest{
			GroupID: 1, RequestedByUserID: 1,
			Date: base.AddDate(0, 0, i/9), Hour: 9 + i%9,
			Status: models.CoverageRequestOpen, CreatedAt: base.Add(time.Duration(i) * time.Hour),
		}
		if err := db.Create(&req).Error; err != nil {
			t.Fatalf("create coverage request %d: %v", i, err)
		}
	}

	// limit is capped at 100 by the endpoint itself (see GetGroupActivityFeed),
	// so the *page* of items can never directly show coverageRequestFetchCap
	// (300) at once - total is the field that reflects the capped fetch.
	body := fetchActivityFeed(t, db, "type=coverage_requests&limit=100")

	coverageItems := itemsOfType(t, body, "coverage_request")
	if len(coverageItems) != 100 {
		t.Fatalf("expected a full page of 100 items, got %d", len(coverageItems))
	}
	if body["total"] != float64(coverageRequestFetchCap) {
		t.Fatalf("expected total to reflect the capped fetch (%d rows created, %d cap), got %v", overCap, coverageRequestFetchCap, body["total"])
	}
}
