package handlers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
	"gorm.io/gorm"
)

func TestShiftCoverageRequestMigration(t *testing.T) {
	db := SetupTestDB(t)
	user := CreateTestUser(t, db, "volunteer1", "volunteer1@example.com", "password123", false)
	group := CreateTestGroup(t, db, "Dogs", "Dog volunteers")

	req := &models.ShiftCoverageRequest{
		GroupID:           group.ID,
		RequestedByUserID: user.ID,
		Date:              time.Date(2026, 8, 18, 0, 0, 0, 0, time.UTC),
		Hour:              10,
		Status:            models.CoverageRequestOpen,
	}
	if err := db.Create(req).Error; err != nil {
		t.Fatalf("Failed to create ShiftCoverageRequest: %v", err)
	}
	if req.ID == 0 {
		t.Fatal("Expected ShiftCoverageRequest to get an assigned ID")
	}
}

func setupCoverageTestGroup(t *testing.T, db *gorm.DB) (requester, other *models.User, group *models.Group) {
	requester = CreateTestUser(t, db, "requester", "requester@example.com", "password123", false)
	other = CreateTestUser(t, db, "other", "other@example.com", "password123", false)
	group = CreateTestGroup(t, db, "Dogs", "Dog volunteers")
	AddUserToGroupWithAdmin(t, db, requester.ID, group.ID, false)
	AddUserToGroupWithAdmin(t, db, other.ID, group.ID, false)
	if err := db.Model(group).Update("scheduling_enabled", true).Error; err != nil {
		t.Fatalf("Failed to enable scheduling: %v", err)
	}
	// requester has a recurring Tuesday 10am shift.
	slot := &models.ShiftSlot{UserID: requester.ID, GroupID: group.ID, DayOfWeek: 2, Hour: 10}
	if err := db.Create(slot).Error; err != nil {
		t.Fatalf("Failed to create shift slot: %v", err)
	}
	return requester, other, group
}

func performCreateCoverageRequest(db *gorm.DB, callerID uint, isAdmin bool, groupID uint, body string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", callerID)
		c.Set("is_admin", isAdmin)
		c.Next()
	})
	router.POST("/groups/:id/schedule/coverage-requests", CreateCoverageRequest(db, nil, nil))

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/groups/%d/schedule/coverage-requests", groupID), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

// nextWeekday returns the next future date (strictly after today) that
// falls on the given time.Weekday, formatted as "2006-01-02".
func nextWeekday(weekday time.Weekday) string {
	today := time.Now().UTC()
	daysAhead := (int(weekday) - int(today.Weekday()) + 7) % 7
	if daysAhead == 0 {
		daysAhead = 7
	}
	return today.AddDate(0, 0, daysAhead).Format("2006-01-02")
}

func TestCreateCoverageRequest(t *testing.T) {
	t.Run("happy path creates an open request", func(t *testing.T) {
		db := SetupTestDB(t)
		requester, _, group := setupCoverageTestGroup(t, db)
		date := nextWeekday(time.Tuesday)

		body := fmt.Sprintf(`{"date":"%s","hour":10}`, date)
		w := performCreateCoverageRequest(db, requester.ID, false, group.ID, body)

		if w.Code != http.StatusCreated {
			t.Fatalf("Expected 201, got %d: %s", w.Code, w.Body.String())
		}
		var created models.ShiftCoverageRequest
		if err := db.Where("group_id = ? AND requested_by_user_id = ?", group.ID, requester.ID).First(&created).Error; err != nil {
			t.Fatalf("Expected request to be persisted: %v", err)
		}
		if created.Status != models.CoverageRequestOpen {
			t.Fatalf("Expected status open, got %s", created.Status)
		}
	})

	t.Run("rejects a date whose weekday has no matching shift", func(t *testing.T) {
		db := SetupTestDB(t)
		requester, _, group := setupCoverageTestGroup(t, db)
		date := nextWeekday(time.Wednesday) // requester's slot is Tuesday

		body := fmt.Sprintf(`{"date":"%s","hour":10}`, date)
		w := performCreateCoverageRequest(db, requester.ID, false, group.ID, body)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("Expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("rejects a past or same-day date", func(t *testing.T) {
		db := SetupTestDB(t)
		requester, _, group := setupCoverageTestGroup(t, db)
		today := time.Now().UTC().Format("2006-01-02")

		body := fmt.Sprintf(`{"date":"%s","hour":10}`, today)
		w := performCreateCoverageRequest(db, requester.ID, false, group.ID, body)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("Expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("rejects a duplicate open request for the same date and hour", func(t *testing.T) {
		db := SetupTestDB(t)
		requester, _, group := setupCoverageTestGroup(t, db)
		date := nextWeekday(time.Tuesday)
		body := fmt.Sprintf(`{"date":"%s","hour":10}`, date)

		first := performCreateCoverageRequest(db, requester.ID, false, group.ID, body)
		if first.Code != http.StatusCreated {
			t.Fatalf("Expected first request to succeed, got %d: %s", first.Code, first.Body.String())
		}
		second := performCreateCoverageRequest(db, requester.ID, false, group.ID, body)
		if second.Code != http.StatusConflict {
			t.Fatalf("Expected 409 on duplicate, got %d: %s", second.Code, second.Body.String())
		}
	})

	t.Run("group admin can create on behalf of a member", func(t *testing.T) {
		db := SetupTestDB(t)
		requester, _, group := setupCoverageTestGroup(t, db)
		admin := CreateTestUser(t, db, "groupadmin", "groupadmin@example.com", "password123", false)
		AddUserToGroupWithAdmin(t, db, admin.ID, group.ID, true)
		date := nextWeekday(time.Tuesday)

		body := fmt.Sprintf(`{"date":"%s","hour":10,"user_id":%d}`, date, requester.ID)
		w := performCreateCoverageRequest(db, admin.ID, false, group.ID, body)

		if w.Code != http.StatusCreated {
			t.Fatalf("Expected 201, got %d: %s", w.Code, w.Body.String())
		}
		var created models.ShiftCoverageRequest
		if err := db.Where("group_id = ?", group.ID).First(&created).Error; err != nil {
			t.Fatalf("Expected request to be persisted: %v", err)
		}
		if created.RequestedByUserID != requester.ID {
			t.Fatalf("Expected request to be for requester %d, got %d", requester.ID, created.RequestedByUserID)
		}
	})

	t.Run("non-admin cannot create on behalf of another member", func(t *testing.T) {
		db := SetupTestDB(t)
		requester, other, group := setupCoverageTestGroup(t, db)
		date := nextWeekday(time.Tuesday)

		body := fmt.Sprintf(`{"date":"%s","hour":10,"user_id":%d}`, date, requester.ID)
		w := performCreateCoverageRequest(db, other.ID, false, group.ID, body)

		if w.Code != http.StatusForbidden {
			t.Fatalf("Expected 403, got %d: %s", w.Code, w.Body.String())
		}
	})
}

func performClaimCoverageRequest(db *gorm.DB, callerID uint, groupID, requestID uint) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", callerID)
		c.Set("is_admin", false)
		c.Next()
	})
	router.POST("/groups/:id/schedule/coverage-requests/:requestId/claim", ClaimCoverageRequest(db, nil, nil))

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/groups/%d/schedule/coverage-requests/%d/claim", groupID, requestID), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func createOpenCoverageRequest(t *testing.T, db *gorm.DB, groupID, userID uint, dayOfWeek, hour int, date time.Time) *models.ShiftCoverageRequest {
	req := &models.ShiftCoverageRequest{
		GroupID:           groupID,
		RequestedByUserID: userID,
		Date:              date,
		Hour:              hour,
		Status:            models.CoverageRequestOpen,
	}
	if err := db.Create(req).Error; err != nil {
		t.Fatalf("Failed to create coverage request: %v", err)
	}
	return req
}

func TestClaimCoverageRequest(t *testing.T) {
	t.Run("happy path claims and updates status", func(t *testing.T) {
		db := SetupTestDB(t)
		requester, claimant, group := setupCoverageTestGroup(t, db)
		date, _ := time.Parse("2006-01-02", nextWeekday(time.Tuesday))
		reqRow := createOpenCoverageRequest(t, db, group.ID, requester.ID, 2, 10, date)

		w := performClaimCoverageRequest(db, claimant.ID, group.ID, reqRow.ID)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var updated models.ShiftCoverageRequest
		if err := db.First(&updated, reqRow.ID).Error; err != nil {
			t.Fatalf("Failed to reload request: %v", err)
		}
		if updated.Status != models.CoverageRequestClaimed {
			t.Fatalf("Expected status claimed, got %s", updated.Status)
		}
		if updated.ClaimedByUserID == nil || *updated.ClaimedByUserID != claimant.ID {
			t.Fatalf("Expected claimed_by_user_id %d, got %v", claimant.ID, updated.ClaimedByUserID)
		}
	})

	t.Run("rejects claiming an already-claimed request", func(t *testing.T) {
		db := SetupTestDB(t)
		requester, claimant, group := setupCoverageTestGroup(t, db)
		thirdUser := CreateTestUser(t, db, "third", "third@example.com", "password123", false)
		AddUserToGroupWithAdmin(t, db, thirdUser.ID, group.ID, false)
		date, _ := time.Parse("2006-01-02", nextWeekday(time.Tuesday))
		reqRow := createOpenCoverageRequest(t, db, group.ID, requester.ID, 2, 10, date)

		first := performClaimCoverageRequest(db, claimant.ID, group.ID, reqRow.ID)
		if first.Code != http.StatusOK {
			t.Fatalf("Expected first claim to succeed, got %d: %s", first.Code, first.Body.String())
		}
		second := performClaimCoverageRequest(db, thirdUser.ID, group.ID, reqRow.ID)
		if second.Code != http.StatusConflict {
			t.Fatalf("Expected 409 on second claim, got %d: %s", second.Code, second.Body.String())
		}
	})

	t.Run("rejects the requester claiming their own request", func(t *testing.T) {
		db := SetupTestDB(t)
		requester, _, group := setupCoverageTestGroup(t, db)
		date, _ := time.Parse("2006-01-02", nextWeekday(time.Tuesday))
		reqRow := createOpenCoverageRequest(t, db, group.ID, requester.ID, 2, 10, date)

		w := performClaimCoverageRequest(db, requester.ID, group.ID, reqRow.ID)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("Expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("rejects a claimant with a conflicting shift at that exact date/hour", func(t *testing.T) {
		db := SetupTestDB(t)
		requester, claimant, group := setupCoverageTestGroup(t, db)
		date, _ := time.Parse("2006-01-02", nextWeekday(time.Tuesday))
		reqRow := createOpenCoverageRequest(t, db, group.ID, requester.ID, 2, 10, date)

		// claimant already has their own Tuesday 10am shift in a different group.
		otherGroup := CreateTestGroup(t, db, "Cats", "Cat volunteers")
		AddUserToGroupWithAdmin(t, db, claimant.ID, otherGroup.ID, false)
		if err := db.Create(&models.ShiftSlot{UserID: claimant.ID, GroupID: otherGroup.ID, DayOfWeek: 2, Hour: 10}).Error; err != nil {
			t.Fatalf("Failed to create conflicting shift slot: %v", err)
		}

		w := performClaimCoverageRequest(db, claimant.ID, group.ID, reqRow.ID)

		if w.Code != http.StatusConflict {
			t.Fatalf("Expected 409 on conflict, got %d: %s", w.Code, w.Body.String())
		}
	})
}

func performCancelCoverageRequest(db *gorm.DB, callerID uint, isAdmin bool, groupID, requestID uint) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", callerID)
		c.Set("is_admin", isAdmin)
		c.Next()
	})
	router.DELETE("/groups/:id/schedule/coverage-requests/:requestId", CancelCoverageRequest(db))

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/groups/%d/schedule/coverage-requests/%d", groupID, requestID), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func TestCancelCoverageRequest(t *testing.T) {
	t.Run("requester can cancel their own open request", func(t *testing.T) {
		db := SetupTestDB(t)
		requester, _, group := setupCoverageTestGroup(t, db)
		date, _ := time.Parse("2006-01-02", nextWeekday(time.Tuesday))
		reqRow := createOpenCoverageRequest(t, db, group.ID, requester.ID, 2, 10, date)

		w := performCancelCoverageRequest(db, requester.ID, false, group.ID, reqRow.ID)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var updated models.ShiftCoverageRequest
		db.First(&updated, reqRow.ID)
		if updated.Status != models.CoverageRequestCancelled {
			t.Fatalf("Expected status cancelled, got %s", updated.Status)
		}
	})

	t.Run("requester cannot cancel after it's claimed", func(t *testing.T) {
		db := SetupTestDB(t)
		requester, claimant, group := setupCoverageTestGroup(t, db)
		date, _ := time.Parse("2006-01-02", nextWeekday(time.Tuesday))
		reqRow := createOpenCoverageRequest(t, db, group.ID, requester.ID, 2, 10, date)
		if claim := performClaimCoverageRequest(db, claimant.ID, group.ID, reqRow.ID); claim.Code != http.StatusOK {
			t.Fatalf("Expected claim to succeed, got %d: %s", claim.Code, claim.Body.String())
		}

		w := performCancelCoverageRequest(db, requester.ID, false, group.ID, reqRow.ID)

		if w.Code != http.StatusForbidden {
			t.Fatalf("Expected 403, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("group admin can cancel any request", func(t *testing.T) {
		db := SetupTestDB(t)
		requester, _, group := setupCoverageTestGroup(t, db)
		admin := CreateTestUser(t, db, "groupadmin", "groupadmin@example.com", "password123", false)
		AddUserToGroupWithAdmin(t, db, admin.ID, group.ID, true)
		date, _ := time.Parse("2006-01-02", nextWeekday(time.Tuesday))
		reqRow := createOpenCoverageRequest(t, db, group.ID, requester.ID, 2, 10, date)

		w := performCancelCoverageRequest(db, admin.ID, false, group.ID, reqRow.ID)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("non-admin, non-requester cannot cancel", func(t *testing.T) {
		db := SetupTestDB(t)
		requester, other, group := setupCoverageTestGroup(t, db)
		date, _ := time.Parse("2006-01-02", nextWeekday(time.Tuesday))
		reqRow := createOpenCoverageRequest(t, db, group.ID, requester.ID, 2, 10, date)

		w := performCancelCoverageRequest(db, other.ID, false, group.ID, reqRow.ID)

		if w.Code != http.StatusForbidden {
			t.Fatalf("Expected 403, got %d: %s", w.Code, w.Body.String())
		}
	})
}
