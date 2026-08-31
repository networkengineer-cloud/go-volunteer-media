package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/email"
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

func TestShiftCoverageRequestPriorityDefaultsToNormal(t *testing.T) {
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
	if req.Priority != "normal" {
		t.Errorf("expected default Priority %q, got %q", "normal", req.Priority)
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

// nextWeekdayWithParity returns the next future date (strictly after today)
// that falls on the given weekday AND whose week matches the given parity
// ("a" or "b"), formatted as "2006-01-02". Walks forward in 7-day steps from
// nextWeekday's result, since consecutive same-weekday dates alternate
// parity, so at most one extra step is ever needed.
func nextWeekdayWithParity(t *testing.T, weekday time.Weekday, parity string) string {
	t.Helper()
	candidate, err := time.Parse("2006-01-02", nextWeekday(weekday))
	if err != nil {
		t.Fatalf("failed to parse candidate date: %v", err)
	}
	for i := 0; i < 2; i++ {
		if weekParity(weekStartOf(candidate)) == parity {
			return candidate.Format("2006-01-02")
		}
		candidate = candidate.AddDate(0, 0, 7)
	}
	t.Fatalf("could not find a %s-parity %s within 2 weeks", parity, weekday)
	return ""
}

func TestCreateCoverageRequest(t *testing.T) {
	t.Run("defaults to normal priority when omitted", func(t *testing.T) {
		db := SetupTestDB(t)
		requester, _, group := setupCoverageTestGroup(t, db)
		date := nextWeekday(time.Tuesday)

		body := fmt.Sprintf(`{"date":"%s","hour":10}`, date)
		w := performCreateCoverageRequest(db, requester.ID, false, group.ID, body)

		if w.Code != http.StatusCreated {
			t.Fatalf("Expected 201, got %d: %s", w.Code, w.Body.String())
		}
		var resp coverageRequestResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}
		if resp.Priority != "normal" {
			t.Fatalf("Expected default priority %q, got %q", "normal", resp.Priority)
		}
	})

	t.Run("accepts an explicit optional priority", func(t *testing.T) {
		db := SetupTestDB(t)
		requester, _, group := setupCoverageTestGroup(t, db)
		date := nextWeekday(time.Tuesday)

		body := fmt.Sprintf(`{"date":"%s","hour":10,"priority":"optional"}`, date)
		w := performCreateCoverageRequest(db, requester.ID, false, group.ID, body)

		if w.Code != http.StatusCreated {
			t.Fatalf("Expected 201, got %d: %s", w.Code, w.Body.String())
		}
		var resp coverageRequestResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}
		if resp.Priority != "optional" {
			t.Fatalf("Expected priority %q, got %q", "optional", resp.Priority)
		}
	})

	t.Run("rejects an invalid priority value", func(t *testing.T) {
		db := SetupTestDB(t)
		requester, _, group := setupCoverageTestGroup(t, db)
		date := nextWeekday(time.Tuesday)

		body := fmt.Sprintf(`{"date":"%s","hour":10,"priority":"urgent"}`, date)
		w := performCreateCoverageRequest(db, requester.ID, false, group.ID, body)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("Expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})

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

	t.Run("rejects a past date", func(t *testing.T) {
		db := SetupTestDB(t)
		requester, _, group := setupCoverageTestGroup(t, db)
		yesterday := time.Now().UTC().AddDate(0, 0, -1).Format("2006-01-02")

		body := fmt.Sprintf(`{"date":"%s","hour":10}`, yesterday)
		w := performCreateCoverageRequest(db, requester.ID, false, group.ID, body)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("Expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("accepts a same-day date", func(t *testing.T) {
		// Same-day requests are legitimate (e.g. realizing this afternoon you
		// can't make a shift later today) and must match the frontend's UTC
		// calendar-date notion of "today" exactly. The requester needs a
		// ShiftSlot matching today's actual weekday/hour, not the fixed
		// Tuesday 10am slot setupCoverageTestGroup wires up, since "today"
		// varies with when the test runs.
		db := SetupTestDB(t)
		requester := CreateTestUser(t, db, "requester", "requester@example.com", "password123", false)
		group := CreateTestGroup(t, db, "Dogs", "Dog volunteers")
		AddUserToGroupWithAdmin(t, db, requester.ID, group.ID, false)
		if err := db.Model(group).Update("scheduling_enabled", true).Error; err != nil {
			t.Fatalf("Failed to enable scheduling: %v", err)
		}
		now := time.Now().UTC()
		slot := &models.ShiftSlot{UserID: requester.ID, GroupID: group.ID, DayOfWeek: int(now.Weekday()), Hour: 10}
		if err := db.Create(slot).Error; err != nil {
			t.Fatalf("Failed to create shift slot: %v", err)
		}
		today := now.Format("2006-01-02")

		body := fmt.Sprintf(`{"date":"%s","hour":10}`, today)
		w := performCreateCoverageRequest(db, requester.ID, false, group.ID, body)

		if w.Code != http.StatusCreated {
			t.Fatalf("Expected 201, got %d: %s", w.Code, w.Body.String())
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

	t.Run("second request within the cooldown window still succeeds", func(t *testing.T) {
		// Guards the notification-cooldown check added in CreateCoverageRequest:
		// the recent-request count query now runs unconditionally (even with
		// nil emailService/groupMeService, as performCreateCoverageRequest
		// always passes), so a rapid cancel-and-recreate cycle by the same
		// user must not error or panic - it should just skip notifications,
		// which this test can't directly observe without mocking the
		// email/GroupMe services, so it asserts the structural outcome
		// instead: both rows get created successfully.
		db := SetupTestDB(t)
		requester, _, group := setupCoverageTestGroup(t, db)
		date := nextWeekday(time.Tuesday)
		body := fmt.Sprintf(`{"date":"%s","hour":10}`, date)

		first := performCreateCoverageRequest(db, requester.ID, false, group.ID, body)
		if first.Code != http.StatusCreated {
			t.Fatalf("Expected first request to succeed, got %d: %s", first.Code, first.Body.String())
		}
		var firstCreated models.ShiftCoverageRequest
		if err := db.Where("group_id = ? AND requested_by_user_id = ?", group.ID, requester.ID).First(&firstCreated).Error; err != nil {
			t.Fatalf("Expected first request to be persisted: %v", err)
		}
		if err := db.Model(&firstCreated).Update("status", models.CoverageRequestCancelled).Error; err != nil {
			t.Fatalf("Failed to cancel first request: %v", err)
		}

		second := performCreateCoverageRequest(db, requester.ID, false, group.ID, body)
		if second.Code != http.StatusCreated {
			t.Fatalf("Expected second request within cooldown to still succeed, got %d: %s", second.Code, second.Body.String())
		}

		var count int64
		if err := db.Model(&models.ShiftCoverageRequest{}).
			Where("group_id = ? AND requested_by_user_id = ?", group.ID, requester.ID).
			Count(&count).Error; err != nil {
			t.Fatalf("Failed to count requests: %v", err)
		}
		if count != 2 {
			t.Fatalf("Expected 2 requests to exist (one cancelled, one active), got %d", count)
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

	t.Run("rejects an hour beyond the weekend cap before even checking for a matching slot", func(t *testing.T) {
		db := SetupTestDB(t)
		requester := CreateTestUser(t, db, "requester", "requester@example.com", "password123", false)
		group := CreateTestGroup(t, db, "Dogs", "Dog volunteers")
		AddUserToGroupWithAdmin(t, db, requester.ID, group.ID, false)
		if err := db.Model(group).Update("scheduling_enabled", true).Error; err != nil {
			t.Fatalf("Failed to enable scheduling: %v", err)
		}
		// Give the requester a (currently invalid-once-this-lands, but
		// pre-existing) Saturday 4pm slot to prove the rejection comes from
		// the hour-bound check, not "no matching slot".
		if err := db.Exec("INSERT INTO shift_slots (user_id, group_id, day_of_week, hour, created_at, updated_at) VALUES (?, ?, 6, 16, datetime('now'), datetime('now'))", requester.ID, group.ID).Error; err != nil {
			t.Fatalf("Failed to seed slot: %v", err)
		}
		date := nextWeekday(time.Saturday)

		body := fmt.Sprintf(`{"date":"%s","hour":16}`, date)
		w := performCreateCoverageRequest(db, requester.ID, false, group.ID, body)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("Expected 400 for hour 16 on a Saturday, got %d: %s", w.Code, w.Body.String())
		}
		if !strings.Contains(w.Body.String(), "hour") {
			t.Errorf("Expected error to mention hour, got %q", w.Body.String())
		}
	})

	t.Run("rejects a date on a biweekly slot's off-week", func(t *testing.T) {
		db := SetupTestDB(t)
		requester := CreateTestUser(t, db, "requester", "requester@example.com", "password123", false)
		group := CreateTestGroup(t, db, "Dogs", "Dog volunteers")
		AddUserToGroupWithAdmin(t, db, requester.ID, group.ID, false)
		if err := db.Model(group).Update("scheduling_enabled", true).Error; err != nil {
			t.Fatalf("Failed to enable scheduling: %v", err)
		}
		if err := db.Create(&models.ShiftSlot{UserID: requester.ID, GroupID: group.ID, DayOfWeek: 2, Hour: 10, Cadence: "biweekly_a"}).Error; err != nil {
			t.Fatalf("Failed to create shift slot: %v", err)
		}

		// requester's slot is "A", so a "B"-week date has no active slot.
		date := nextWeekdayWithParity(t, time.Tuesday, "b")
		body := fmt.Sprintf(`{"date":"%s","hour":10}`, date)
		w := performCreateCoverageRequest(db, requester.ID, false, group.ID, body)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("Expected 400 for an off-week date, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("accepts a date on a biweekly slot's on-week", func(t *testing.T) {
		db := SetupTestDB(t)
		requester := CreateTestUser(t, db, "requester", "requester@example.com", "password123", false)
		group := CreateTestGroup(t, db, "Dogs", "Dog volunteers")
		AddUserToGroupWithAdmin(t, db, requester.ID, group.ID, false)
		if err := db.Model(group).Update("scheduling_enabled", true).Error; err != nil {
			t.Fatalf("Failed to enable scheduling: %v", err)
		}
		if err := db.Create(&models.ShiftSlot{UserID: requester.ID, GroupID: group.ID, DayOfWeek: 2, Hour: 10, Cadence: "biweekly_a"}).Error; err != nil {
			t.Fatalf("Failed to create shift slot: %v", err)
		}

		date := nextWeekdayWithParity(t, time.Tuesday, "a")
		body := fmt.Sprintf(`{"date":"%s","hour":10}`, date)
		w := performCreateCoverageRequest(db, requester.ID, false, group.ID, body)

		if w.Code != http.StatusCreated {
			t.Fatalf("Expected 201 for an on-week date, got %d: %s", w.Code, w.Body.String())
		}
	})
}

func TestBuildCoverageRequestSummary(t *testing.T) {
	t.Run("a single open request renders as one sentence, not a list", func(t *testing.T) {
		date, _ := time.Parse("2006-01-02", nextWeekday(time.Tuesday))
		summary := buildCoverageRequestSummary("Jane Doe", []models.ShiftCoverageRequest{
			{Date: date, Hour: 10},
		})
		want := fmt.Sprintf("Jane Doe needs coverage for their 10:00 AM shift on %s.", date.Format("Monday, January 2"))
		if summary != want {
			t.Fatalf("Expected %q, got %q", want, summary)
		}
	})

	t.Run("multiple open requests render as one bulk list, not separate messages", func(t *testing.T) {
		tue, _ := time.Parse("2006-01-02", nextWeekday(time.Tuesday))
		thu, _ := time.Parse("2006-01-02", nextWeekday(time.Thursday))
		summary := buildCoverageRequestSummary("Jane Doe", []models.ShiftCoverageRequest{
			{Date: tue, Hour: 10},
			{Date: thu, Hour: 14},
		})
		want := fmt.Sprintf("Jane Doe needs coverage for 2 shifts:\n- %s at 10:00 AM\n- %s at 2:00 PM",
			tue.Format("Monday, January 2"), thu.Format("Monday, January 2"))
		if summary != want {
			t.Fatalf("Expected %q, got %q", want, summary)
		}
	})

	t.Run("a terminal-hour (90-min) request shows the full start-end range, not just the start time", func(t *testing.T) {
		// nextWeekday(Tuesday) is a weekday, whose terminal hour is 17
		// (maxHourFor) - a 90-min slot ending 6:30 PM. Someone deciding
		// whether to claim this from the notification text needs to see
		// the actual end time, not just "5:00 PM".
		date, _ := time.Parse("2006-01-02", nextWeekday(time.Tuesday))
		summary := buildCoverageRequestSummary("Jane Doe", []models.ShiftCoverageRequest{
			{Date: date, Hour: 17},
		})
		want := fmt.Sprintf("Jane Doe needs coverage for their 5:00–6:30 PM shift on %s.", date.Format("Monday, January 2"))
		if summary != want {
			t.Fatalf("Expected %q, got %q", want, summary)
		}
	})

	t.Run("a single optional request uses softer phrasing than a normal one", func(t *testing.T) {
		date, _ := time.Parse("2006-01-02", nextWeekday(time.Tuesday))
		summary := buildCoverageRequestSummary("Jane Doe", []models.ShiftCoverageRequest{
			{Date: date, Hour: 10, Priority: "optional"},
		})
		want := fmt.Sprintf("Jane Doe could use coverage for their 10:00 AM shift on %s, if anyone's available.", date.Format("Monday, January 2"))
		if summary != want {
			t.Fatalf("Expected %q, got %q", want, summary)
		}
	})

	t.Run("a mixed-priority list marks only the optional lines", func(t *testing.T) {
		tue, _ := time.Parse("2006-01-02", nextWeekday(time.Tuesday))
		thu, _ := time.Parse("2006-01-02", nextWeekday(time.Thursday))
		summary := buildCoverageRequestSummary("Jane Doe", []models.ShiftCoverageRequest{
			{Date: tue, Hour: 10, Priority: "normal"},
			{Date: thu, Hour: 14, Priority: "optional"},
		})
		want := fmt.Sprintf("Jane Doe needs coverage for 2 shifts:\n- %s at 10:00 AM\n- %s at 2:00 PM (optional)",
			tue.Format("Monday, January 2"), thu.Format("Monday, January 2"))
		if summary != want {
			t.Fatalf("Expected %q, got %q", want, summary)
		}
	})

	t.Run("an all-optional list uses softer header phrasing and skips redundant per-line marks", func(t *testing.T) {
		tue, _ := time.Parse("2006-01-02", nextWeekday(time.Tuesday))
		thu, _ := time.Parse("2006-01-02", nextWeekday(time.Thursday))
		summary := buildCoverageRequestSummary("Jane Doe", []models.ShiftCoverageRequest{
			{Date: tue, Hour: 10, Priority: "optional"},
			{Date: thu, Hour: 14, Priority: "optional"},
		})
		want := fmt.Sprintf("Jane Doe could use coverage for 2 shifts, if anyone's available:\n- %s at 10:00 AM\n- %s at 2:00 PM",
			tue.Format("Monday, January 2"), thu.Format("Monday, January 2"))
		if summary != want {
			t.Fatalf("Expected %q, got %q", want, summary)
		}
	})
}

func TestScheduleEmailNotificationsEnabled(t *testing.T) {
	t.Run("defaults to disabled when unset", func(t *testing.T) {
		t.Setenv("SCHEDULE_EMAIL_NOTIFICATIONS_ENABLED", "")
		if scheduleEmailNotificationsEnabled() {
			t.Fatal("expected scheduleEmailNotificationsEnabled to default to false when unset")
		}
	})

	t.Run("enabled when set to true", func(t *testing.T) {
		t.Setenv("SCHEDULE_EMAIL_NOTIFICATIONS_ENABLED", "true")
		if !scheduleEmailNotificationsEnabled() {
			t.Fatal("expected scheduleEmailNotificationsEnabled to be true")
		}
	})

	t.Run("disabled for any other value", func(t *testing.T) {
		t.Setenv("SCHEDULE_EMAIL_NOTIFICATIONS_ENABLED", "yes")
		if scheduleEmailNotificationsEnabled() {
			t.Fatal("expected scheduleEmailNotificationsEnabled to be false for a non-true/1 value")
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

	t.Run("a claimant with a biweekly slot can claim a same-day/hour request on their own off-week", func(t *testing.T) {
		db := SetupTestDB(t)
		requester, claimant, group := setupCoverageTestGroup(t, db)
		// claimant has their own Tuesday 10am biweekly_a slot in a different group.
		otherGroup := CreateTestGroup(t, db, "Cats", "Cat volunteers")
		AddUserToGroupWithAdmin(t, db, claimant.ID, otherGroup.ID, false)
		if err := db.Create(&models.ShiftSlot{UserID: claimant.ID, GroupID: otherGroup.ID, DayOfWeek: 2, Hour: 10, Cadence: "biweekly_a"}).Error; err != nil {
			t.Fatalf("Failed to create claimant's biweekly slot: %v", err)
		}
		offWeekDate, err := time.Parse("2006-01-02", nextWeekdayWithParity(t, time.Tuesday, "b"))
		if err != nil {
			t.Fatalf("failed to parse date: %v", err)
		}
		reqRow := createOpenCoverageRequest(t, db, group.ID, requester.ID, 2, 10, offWeekDate)

		w := performClaimCoverageRequest(db, claimant.ID, group.ID, reqRow.ID)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200 (claimant's slot is inactive this week), got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("a claimant with a biweekly slot cannot claim a same-day/hour request on their own on-week", func(t *testing.T) {
		db := SetupTestDB(t)
		requester, claimant, group := setupCoverageTestGroup(t, db)
		otherGroup := CreateTestGroup(t, db, "Cats", "Cat volunteers")
		AddUserToGroupWithAdmin(t, db, claimant.ID, otherGroup.ID, false)
		if err := db.Create(&models.ShiftSlot{UserID: claimant.ID, GroupID: otherGroup.ID, DayOfWeek: 2, Hour: 10, Cadence: "biweekly_a"}).Error; err != nil {
			t.Fatalf("Failed to create claimant's biweekly slot: %v", err)
		}
		onWeekDate, err := time.Parse("2006-01-02", nextWeekdayWithParity(t, time.Tuesday, "a"))
		if err != nil {
			t.Fatalf("failed to parse date: %v", err)
		}
		reqRow := createOpenCoverageRequest(t, db, group.ID, requester.ID, 2, 10, onWeekDate)

		w := performClaimCoverageRequest(db, claimant.ID, group.ID, reqRow.ID)

		if w.Code != http.StatusConflict {
			t.Fatalf("Expected 409 (claimant's slot is active this week), got %d: %s", w.Code, w.Body.String())
		}
	})
}

// TestCreateCoverageRequest_EmailGatedByScheduleFlag exercises the actual
// SendEmail call, not just scheduleEmailNotificationsEnabled() in isolation,
// so a regression that only gates one of the two call sites (or gates the
// wrong condition) would be caught here.
func TestCreateCoverageRequest_EmailGatedByScheduleFlag(t *testing.T) {
	t.Run("no email is sent while the flag is unset (default)", func(t *testing.T) {
		t.Setenv("SCHEDULE_EMAIL_NOTIFICATIONS_ENABLED", "")
		db := SetupTestDB(t)
		requester, other, group := setupCoverageTestGroup(t, db)
		if err := db.Model(other).Update("email_notifications_enabled", true).Error; err != nil {
			t.Fatalf("Failed to enable other's email notifications: %v", err)
		}
		provider := &mockEmailProvider{}
		emailSvc := email.NewServiceWithProvider(provider, db)

		gin.SetMode(gin.TestMode)
		router := gin.New()
		router.Use(func(c *gin.Context) {
			c.Set("user_id", requester.ID)
			c.Set("is_admin", false)
			c.Next()
		})
		router.POST("/groups/:id/schedule/coverage-requests", CreateCoverageRequest(db, emailSvc, nil))

		date := nextWeekday(time.Tuesday)
		body := fmt.Sprintf(`{"date":"%s","hour":10}`, date)
		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/groups/%d/schedule/coverage-requests", group.ID), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("Expected 201, got %d: %s", w.Code, w.Body.String())
		}
		time.Sleep(50 * time.Millisecond)
		if got := provider.sendCount(); got != 0 {
			t.Fatalf("Expected no coverage-request email while the flag is off, got %d", got)
		}
	})

	t.Run("an email is sent once the flag is enabled", func(t *testing.T) {
		t.Setenv("SCHEDULE_EMAIL_NOTIFICATIONS_ENABLED", "true")
		db := SetupTestDB(t)
		requester, other, group := setupCoverageTestGroup(t, db)
		if err := db.Model(other).Update("email_notifications_enabled", true).Error; err != nil {
			t.Fatalf("Failed to enable other's email notifications: %v", err)
		}
		provider := &mockEmailProvider{}
		emailSvc := email.NewServiceWithProvider(provider, db)

		gin.SetMode(gin.TestMode)
		router := gin.New()
		router.Use(func(c *gin.Context) {
			c.Set("user_id", requester.ID)
			c.Set("is_admin", false)
			c.Next()
		})
		router.POST("/groups/:id/schedule/coverage-requests", CreateCoverageRequest(db, emailSvc, nil))

		date := nextWeekday(time.Tuesday)
		body := fmt.Sprintf(`{"date":"%s","hour":10}`, date)
		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/groups/%d/schedule/coverage-requests", group.ID), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("Expected 201, got %d: %s", w.Code, w.Body.String())
		}
		time.Sleep(50 * time.Millisecond)
		if got := provider.sendCount(); got == 0 {
			t.Fatal("Expected a coverage-request email to be sent while the flag is on, got none")
		}
	})
}

// TestClaimCoverageRequest_EmailGatedByScheduleFlag covers notifyRequesterOfClaim,
// which has its own separate emailService/IsConfigured/EmailNotificationsEnabled
// checks from the create path above - the flag must gate both call sites.
func TestClaimCoverageRequest_EmailGatedByScheduleFlag(t *testing.T) {
	t.Run("no email is sent while the flag is unset (default)", func(t *testing.T) {
		t.Setenv("SCHEDULE_EMAIL_NOTIFICATIONS_ENABLED", "")
		db := SetupTestDB(t)
		requester, claimant, group := setupCoverageTestGroup(t, db)
		if err := db.Model(requester).Update("email_notifications_enabled", true).Error; err != nil {
			t.Fatalf("Failed to enable requester's email notifications: %v", err)
		}
		date, _ := time.Parse("2006-01-02", nextWeekday(time.Tuesday))
		reqRow := createOpenCoverageRequest(t, db, group.ID, requester.ID, 2, 10, date)
		provider := &mockEmailProvider{}
		emailSvc := email.NewServiceWithProvider(provider, db)

		gin.SetMode(gin.TestMode)
		router := gin.New()
		router.Use(func(c *gin.Context) {
			c.Set("user_id", claimant.ID)
			c.Set("is_admin", false)
			c.Next()
		})
		router.POST("/groups/:id/schedule/coverage-requests/:requestId/claim", ClaimCoverageRequest(db, emailSvc, nil))

		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/groups/%d/schedule/coverage-requests/%d/claim", group.ID, reqRow.ID), nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}
		time.Sleep(50 * time.Millisecond)
		if got := provider.sendCount(); got != 0 {
			t.Fatalf("Expected no claim email while the flag is off, got %d", got)
		}
	})

	t.Run("an email is sent once the flag is enabled", func(t *testing.T) {
		t.Setenv("SCHEDULE_EMAIL_NOTIFICATIONS_ENABLED", "true")
		db := SetupTestDB(t)
		requester, claimant, group := setupCoverageTestGroup(t, db)
		if err := db.Model(requester).Update("email_notifications_enabled", true).Error; err != nil {
			t.Fatalf("Failed to enable requester's email notifications: %v", err)
		}
		date, _ := time.Parse("2006-01-02", nextWeekday(time.Tuesday))
		reqRow := createOpenCoverageRequest(t, db, group.ID, requester.ID, 2, 10, date)
		provider := &mockEmailProvider{}
		emailSvc := email.NewServiceWithProvider(provider, db)

		gin.SetMode(gin.TestMode)
		router := gin.New()
		router.Use(func(c *gin.Context) {
			c.Set("user_id", claimant.ID)
			c.Set("is_admin", false)
			c.Next()
		})
		router.POST("/groups/:id/schedule/coverage-requests/:requestId/claim", ClaimCoverageRequest(db, emailSvc, nil))

		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/groups/%d/schedule/coverage-requests/%d/claim", group.ID, reqRow.ID), nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}
		time.Sleep(50 * time.Millisecond)
		if got := provider.sendCount(); got == 0 {
			t.Fatal("Expected a claim email to be sent while the flag is on, got none")
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

func performCreateCoverageRequestsBatch(db *gorm.DB, callerID uint, groupID uint, body string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", callerID)
		c.Set("is_admin", false)
		c.Next()
	})
	router.POST("/groups/:id/schedule/coverage-requests/batch", CreateCoverageRequestsBatch(db, nil, nil))

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/groups/%d/schedule/coverage-requests/batch", groupID), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func performListCoverageRequests(db *gorm.DB, callerID uint, isAdmin bool, groupID uint) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", callerID)
		c.Set("is_admin", isAdmin)
		c.Next()
	})
	router.GET("/groups/:id/schedule/coverage-requests", ListCoverageRequests(db))

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/groups/%d/schedule/coverage-requests", groupID), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func TestListCoverageRequests(t *testing.T) {
	t.Run("surfaces the request's priority", func(t *testing.T) {
		db := SetupTestDB(t)
		requester, other, group := setupCoverageTestGroup(t, db)
		date, _ := time.Parse("2006-01-02", nextWeekday(time.Tuesday))
		req := &models.ShiftCoverageRequest{
			GroupID:           group.ID,
			RequestedByUserID: requester.ID,
			Date:              date,
			Hour:              10,
			Status:            models.CoverageRequestOpen,
			Priority:          "optional",
		}
		if err := db.Create(req).Error; err != nil {
			t.Fatalf("Failed to create coverage request: %v", err)
		}

		w := performListCoverageRequests(db, other.ID, false, group.ID)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var items []coverageRequestListItem
		if err := json.Unmarshal(w.Body.Bytes(), &items); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}
		if len(items) != 1 {
			t.Fatalf("Expected 1 open request, got %d", len(items))
		}
		if items[0].Priority != "optional" {
			t.Fatalf("Expected priority %q, got %q", "optional", items[0].Priority)
		}
	})

	t.Run("returns an open upcoming request with requester name and claimable true for another member", func(t *testing.T) {
		db := SetupTestDB(t)
		requester, other, group := setupCoverageTestGroup(t, db)
		date, _ := time.Parse("2006-01-02", nextWeekday(time.Tuesday))
		createOpenCoverageRequest(t, db, group.ID, requester.ID, 2, 10, date)

		w := performListCoverageRequests(db, other.ID, false, group.ID)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var items []coverageRequestListItem
		if err := json.Unmarshal(w.Body.Bytes(), &items); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}
		if len(items) != 1 {
			t.Fatalf("Expected 1 open request, got %d", len(items))
		}
		if items[0].RequestedByName != "requester" {
			t.Fatalf("Expected requested_by_name %q, got %q", "requester", items[0].RequestedByName)
		}
		if !items[0].Claimable {
			t.Fatalf("Expected claimable true for another member")
		}
	})

	t.Run("excludes claimed and cancelled requests", func(t *testing.T) {
		db := SetupTestDB(t)
		requester, claimant, group := setupCoverageTestGroup(t, db)
		thirdUser := CreateTestUser(t, db, "third", "third@example.com", "password123", false)
		AddUserToGroupWithAdmin(t, db, thirdUser.ID, group.ID, false)
		tue, _ := time.Parse("2006-01-02", nextWeekday(time.Tuesday))

		claimedReq := createOpenCoverageRequest(t, db, group.ID, requester.ID, 2, 10, tue)
		if claim := performClaimCoverageRequest(db, claimant.ID, group.ID, claimedReq.ID); claim.Code != http.StatusOK {
			t.Fatalf("Expected claim to succeed, got %d: %s", claim.Code, claim.Body.String())
		}
		cancelledReq := createOpenCoverageRequest(t, db, group.ID, requester.ID, 4, 14, tue.AddDate(0, 0, 2))
		if cancel := performCancelCoverageRequest(db, requester.ID, false, group.ID, cancelledReq.ID); cancel.Code != http.StatusOK {
			t.Fatalf("Expected cancel to succeed, got %d: %s", cancel.Code, cancel.Body.String())
		}

		w := performListCoverageRequests(db, thirdUser.ID, false, group.ID)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var items []coverageRequestListItem
		if err := json.Unmarshal(w.Body.Bytes(), &items); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}
		if len(items) != 0 {
			t.Fatalf("Expected 0 open requests, got %d", len(items))
		}
	})

	t.Run("excludes a request whose date has since passed", func(t *testing.T) {
		db := SetupTestDB(t)
		requester, other, group := setupCoverageTestGroup(t, db)
		yesterday := time.Now().UTC().AddDate(0, 0, -1).Truncate(24 * time.Hour)
		createOpenCoverageRequest(t, db, group.ID, requester.ID, int(yesterday.Weekday()), 10, yesterday)

		w := performListCoverageRequests(db, other.ID, false, group.ID)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var items []coverageRequestListItem
		if err := json.Unmarshal(w.Body.Bytes(), &items); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}
		if len(items) != 0 {
			t.Fatalf("Expected past-dated request to be excluded, got %d", len(items))
		}
	})

	t.Run("marks the requester's own request as not claimable", func(t *testing.T) {
		db := SetupTestDB(t)
		requester, _, group := setupCoverageTestGroup(t, db)
		date, _ := time.Parse("2006-01-02", nextWeekday(time.Tuesday))
		createOpenCoverageRequest(t, db, group.ID, requester.ID, 2, 10, date)

		w := performListCoverageRequests(db, requester.ID, false, group.ID)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var items []coverageRequestListItem
		if err := json.Unmarshal(w.Body.Bytes(), &items); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}
		if len(items) != 1 {
			t.Fatalf("Expected 1 open request, got %d", len(items))
		}
		if items[0].Claimable {
			t.Fatalf("Expected claimable false for the requester's own request")
		}
	})

	t.Run("marks as not claimable when the viewer has a conflicting shift at that date/hour", func(t *testing.T) {
		db := SetupTestDB(t)
		requester, other, group := setupCoverageTestGroup(t, db)
		date, _ := time.Parse("2006-01-02", nextWeekday(time.Tuesday))
		createOpenCoverageRequest(t, db, group.ID, requester.ID, 2, 10, date)

		otherGroup := CreateTestGroup(t, db, "Cats", "Cat volunteers")
		AddUserToGroupWithAdmin(t, db, other.ID, otherGroup.ID, false)
		if err := db.Create(&models.ShiftSlot{UserID: other.ID, GroupID: otherGroup.ID, DayOfWeek: 2, Hour: 10}).Error; err != nil {
			t.Fatalf("Failed to create conflicting shift slot: %v", err)
		}

		w := performListCoverageRequests(db, other.ID, false, group.ID)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var items []coverageRequestListItem
		if err := json.Unmarshal(w.Body.Bytes(), &items); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}
		if len(items) != 1 {
			t.Fatalf("Expected 1 open request, got %d", len(items))
		}
		if items[0].Claimable {
			t.Fatalf("Expected claimable false when the viewer has a conflicting shift")
		}
	})

	t.Run("marks as not claimable when the viewer already has a different claimed request at that date/hour", func(t *testing.T) {
		// Distinct from the ShiftSlot-conflict case above: this exercises the
		// claimKeys half of loadUserConflictKeys/isRequestClaimableGiven,
		// which a recurring-slot conflict never touches.
		db := SetupTestDB(t)
		requester, other, group := setupCoverageTestGroup(t, db)
		thirdUser := CreateTestUser(t, db, "third", "third@example.com", "password123", false)
		AddUserToGroupWithAdmin(t, db, thirdUser.ID, group.ID, false)
		date, _ := time.Parse("2006-01-02", nextWeekday(time.Tuesday))

		// other already covers requester's Tuesday 10am shift.
		firstReq := createOpenCoverageRequest(t, db, group.ID, requester.ID, 2, 10, date)
		if claim := performClaimCoverageRequest(db, other.ID, group.ID, firstReq.ID); claim.Code != http.StatusOK {
			t.Fatalf("Expected claim to succeed, got %d: %s", claim.Code, claim.Body.String())
		}

		// thirdUser separately needs coverage for that exact same date/hour.
		secondReq := createOpenCoverageRequest(t, db, group.ID, thirdUser.ID, 2, 10, date)

		w := performListCoverageRequests(db, other.ID, false, group.ID)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var items []coverageRequestListItem
		if err := json.Unmarshal(w.Body.Bytes(), &items); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}
		if len(items) != 1 || items[0].ID != secondReq.ID {
			t.Fatalf("Expected exactly thirdUser's open request, got %+v", items)
		}
		if items[0].Claimable {
			t.Fatalf("Expected claimable false since the viewer already has a claimed shift at that date/hour")
		}
	})

	t.Run("marks claimable true when the viewer's conflicting shift is a biweekly slot on its off-week", func(t *testing.T) {
		db := SetupTestDB(t)
		requester, other, group := setupCoverageTestGroup(t, db)
		otherGroup := CreateTestGroup(t, db, "Cats", "Cat volunteers")
		AddUserToGroupWithAdmin(t, db, other.ID, otherGroup.ID, false)
		if err := db.Create(&models.ShiftSlot{UserID: other.ID, GroupID: otherGroup.ID, DayOfWeek: 2, Hour: 10, Cadence: "biweekly_a"}).Error; err != nil {
			t.Fatalf("Failed to create conflicting biweekly slot: %v", err)
		}
		offWeekDate, err := time.Parse("2006-01-02", nextWeekdayWithParity(t, time.Tuesday, "b"))
		if err != nil {
			t.Fatalf("failed to parse date: %v", err)
		}
		createOpenCoverageRequest(t, db, group.ID, requester.ID, 2, 10, offWeekDate)

		w := performListCoverageRequests(db, other.ID, false, group.ID)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var items []coverageRequestListItem
		if err := json.Unmarshal(w.Body.Bytes(), &items); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}
		if len(items) != 1 || !items[0].Claimable {
			t.Fatalf("Expected claimable true since the conflicting slot is inactive this (off) week, got %+v", items)
		}
	})

	t.Run("non-member is denied", func(t *testing.T) {
		db := SetupTestDB(t)
		_, _, group := setupCoverageTestGroup(t, db)
		outsider := CreateTestUser(t, db, "outsider", "outsider@example.com", "password123", false)

		w := performListCoverageRequests(db, outsider.ID, false, group.ID)

		if w.Code != http.StatusForbidden {
			t.Fatalf("Expected 403, got %d: %s", w.Code, w.Body.String())
		}
	})
}

func TestCreateCoverageRequestsBatch(t *testing.T) {
	t.Run("applies the batch priority to every created item", func(t *testing.T) {
		db := SetupTestDB(t)
		requester, _, group := setupCoverageTestGroup(t, db)
		if err := db.Create(&models.ShiftSlot{UserID: requester.ID, GroupID: group.ID, DayOfWeek: 4, Hour: 14}).Error; err != nil {
			t.Fatalf("Failed to create second shift slot: %v", err)
		}
		tue := nextWeekday(time.Tuesday)
		thu := nextWeekday(time.Thursday)
		body := fmt.Sprintf(`{"priority":"optional","requests":[{"date":"%s","hour":10},{"date":"%s","hour":14}]}`, tue, thu)

		w := performCreateCoverageRequestsBatch(db, requester.ID, group.ID, body)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var resp coverageRequestBatchResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}
		if len(resp.Created) != 2 {
			t.Fatalf("Expected 2 created, got %d", len(resp.Created))
		}
		for _, created := range resp.Created {
			if created.Priority != "optional" {
				t.Errorf("Expected priority %q, got %q for request %d", "optional", created.Priority, created.ID)
			}
		}
	})

	t.Run("rejects an invalid batch priority value", func(t *testing.T) {
		db := SetupTestDB(t)
		requester, _, group := setupCoverageTestGroup(t, db)
		tue := nextWeekday(time.Tuesday)
		body := fmt.Sprintf(`{"priority":"urgent","requests":[{"date":"%s","hour":10}]}`, tue)

		w := performCreateCoverageRequestsBatch(db, requester.ID, group.ID, body)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("Expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("an item's own priority overrides the batch default", func(t *testing.T) {
		db := SetupTestDB(t)
		requester, _, group := setupCoverageTestGroup(t, db)
		if err := db.Create(&models.ShiftSlot{UserID: requester.ID, GroupID: group.ID, DayOfWeek: 4, Hour: 14}).Error; err != nil {
			t.Fatalf("Failed to create second shift slot: %v", err)
		}
		tue := nextWeekday(time.Tuesday)
		thu := nextWeekday(time.Thursday)
		body := fmt.Sprintf(
			`{"priority":"normal","requests":[{"date":"%s","hour":10,"priority":"optional"},{"date":"%s","hour":14}]}`,
			tue, thu,
		)

		w := performCreateCoverageRequestsBatch(db, requester.ID, group.ID, body)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var resp coverageRequestBatchResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}
		if len(resp.Created) != 2 {
			t.Fatalf("Expected 2 created, got %d", len(resp.Created))
		}
		byHour := map[int]string{}
		for _, created := range resp.Created {
			byHour[created.Hour] = created.Priority
		}
		if byHour[10] != "optional" {
			t.Errorf("Expected the Tuesday item's own priority %q to win over the batch default, got %q", "optional", byHour[10])
		}
		if byHour[14] != "normal" {
			t.Errorf("Expected the Thursday item (no override) to fall back to the batch default %q, got %q", "normal", byHour[14])
		}
	})

	t.Run("rejects an invalid per-item priority value", func(t *testing.T) {
		db := SetupTestDB(t)
		requester, _, group := setupCoverageTestGroup(t, db)
		tue := nextWeekday(time.Tuesday)
		body := fmt.Sprintf(`{"requests":[{"date":"%s","hour":10,"priority":"urgent"}]}`, tue)

		w := performCreateCoverageRequestsBatch(db, requester.ID, group.ID, body)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("Expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("happy path creates multiple open requests and reports none skipped", func(t *testing.T) {
		db := SetupTestDB(t)
		requester, _, group := setupCoverageTestGroup(t, db)
		// requester already has Tuesday 10am (from setupCoverageTestGroup); add Thursday 2pm too.
		if err := db.Create(&models.ShiftSlot{UserID: requester.ID, GroupID: group.ID, DayOfWeek: 4, Hour: 14}).Error; err != nil {
			t.Fatalf("Failed to create second shift slot: %v", err)
		}
		tue := nextWeekday(time.Tuesday)
		thu := nextWeekday(time.Thursday)
		body := fmt.Sprintf(`{"requests":[{"date":"%s","hour":10},{"date":"%s","hour":14}]}`, tue, thu)

		w := performCreateCoverageRequestsBatch(db, requester.ID, group.ID, body)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var resp coverageRequestBatchResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}
		if len(resp.Created) != 2 {
			t.Fatalf("Expected 2 created, got %d", len(resp.Created))
		}
		if len(resp.Skipped) != 0 {
			t.Fatalf("Expected 0 skipped, got %d", len(resp.Skipped))
		}
	})

	t.Run("a duplicate item is skipped with a reason while the rest still succeed", func(t *testing.T) {
		db := SetupTestDB(t)
		requester, _, group := setupCoverageTestGroup(t, db)
		if err := db.Create(&models.ShiftSlot{UserID: requester.ID, GroupID: group.ID, DayOfWeek: 4, Hour: 14}).Error; err != nil {
			t.Fatalf("Failed to create second shift slot: %v", err)
		}
		tue := nextWeekday(time.Tuesday)
		thu := nextWeekday(time.Thursday)

		// Pre-create an open request for Tuesday so the batch's Tuesday item collides.
		firstBody := fmt.Sprintf(`{"date":"%s","hour":10}`, tue)
		if first := performCreateCoverageRequest(db, requester.ID, false, group.ID, firstBody); first.Code != http.StatusCreated {
			t.Fatalf("Expected setup request to succeed, got %d: %s", first.Code, first.Body.String())
		}

		body := fmt.Sprintf(`{"requests":[{"date":"%s","hour":10},{"date":"%s","hour":14}]}`, tue, thu)
		w := performCreateCoverageRequestsBatch(db, requester.ID, group.ID, body)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var resp coverageRequestBatchResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}
		if len(resp.Created) != 1 {
			t.Fatalf("Expected 1 created (Thursday), got %d", len(resp.Created))
		}
		if len(resp.Skipped) != 1 {
			t.Fatalf("Expected 1 skipped (Tuesday, duplicate), got %d", len(resp.Skipped))
		}
		if resp.Skipped[0].Hour != 10 {
			t.Fatalf("Expected the skipped item to be the Tuesday 10am one, got hour %d", resp.Skipped[0].Hour)
		}
	})

	t.Run("empty requests array is rejected", func(t *testing.T) {
		db := SetupTestDB(t)
		requester, _, group := setupCoverageTestGroup(t, db)
		w := performCreateCoverageRequestsBatch(db, requester.ID, group.ID, `{"requests":[]}`)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("Expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("a batch exceeding the item cap is rejected and creates nothing", func(t *testing.T) {
		db := SetupTestDB(t)
		requester, _, group := setupCoverageTestGroup(t, db)

		items := make([]string, 0, 201)
		for i := 0; i < 201; i++ {
			items = append(items, `{"date":"2027-01-01","hour":10}`)
		}
		body := fmt.Sprintf(`{"requests":[%s]}`, strings.Join(items, ","))
		w := performCreateCoverageRequestsBatch(db, requester.ID, group.ID, body)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("Expected 400, got %d: %s", w.Code, w.Body.String())
		}
		var count int64
		db.Model(&models.ShiftCoverageRequest{}).Where("group_id = ?", group.ID).Count(&count)
		if count != 0 {
			t.Fatalf("Expected no requests created when the batch exceeds the item cap, got %d", count)
		}
	})

	t.Run("a structurally invalid item rejects the whole batch", func(t *testing.T) {
		db := SetupTestDB(t)
		requester, _, group := setupCoverageTestGroup(t, db)
		tue := nextWeekday(time.Tuesday)
		body := fmt.Sprintf(`{"requests":[{"date":"%s","hour":10},{"date":"not-a-date","hour":10}]}`, tue)
		w := performCreateCoverageRequestsBatch(db, requester.ID, group.ID, body)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("Expected 400, got %d: %s", w.Code, w.Body.String())
		}
		var count int64
		db.Model(&models.ShiftCoverageRequest{}).Where("group_id = ?", group.ID).Count(&count)
		if count != 0 {
			t.Fatalf("Expected no requests created when the batch is rejected, got %d", count)
		}
	})

	t.Run("non-member is denied", func(t *testing.T) {
		db := SetupTestDB(t)
		_, _, group := setupCoverageTestGroup(t, db)
		outsider := CreateTestUser(t, db, "outsider", "outsider@example.com", "password123", false)
		tue := nextWeekday(time.Tuesday)
		body := fmt.Sprintf(`{"requests":[{"date":"%s","hour":10}]}`, tue)
		w := performCreateCoverageRequestsBatch(db, outsider.ID, group.ID, body)
		if w.Code != http.StatusForbidden {
			t.Fatalf("Expected 403, got %d: %s", w.Code, w.Body.String())
		}
	})
}

func performCancelCoverageRequestsBatch(db *gorm.DB, callerID uint, isAdmin bool, groupID uint, body string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", callerID)
		c.Set("is_admin", isAdmin)
		c.Next()
	})
	router.POST("/groups/:id/schedule/coverage-requests/cancel-batch", CancelCoverageRequestsBatch(db))

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/groups/%d/schedule/coverage-requests/cancel-batch", groupID), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func TestCancelCoverageRequestsBatch(t *testing.T) {
	t.Run("requester can bulk-cancel their own open requests", func(t *testing.T) {
		db := SetupTestDB(t)
		requester, _, group := setupCoverageTestGroup(t, db)
		if err := db.Create(&models.ShiftSlot{UserID: requester.ID, GroupID: group.ID, DayOfWeek: 4, Hour: 14}).Error; err != nil {
			t.Fatalf("Failed to create second shift slot: %v", err)
		}
		tue, _ := time.Parse("2006-01-02", nextWeekday(time.Tuesday))
		thu, _ := time.Parse("2006-01-02", nextWeekday(time.Thursday))
		reqA := createOpenCoverageRequest(t, db, group.ID, requester.ID, 2, 10, tue)
		reqB := createOpenCoverageRequest(t, db, group.ID, requester.ID, 4, 14, thu)

		body := fmt.Sprintf(`{"request_ids":[%d,%d]}`, reqA.ID, reqB.ID)
		w := performCancelCoverageRequestsBatch(db, requester.ID, false, group.ID, body)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var resp coverageRequestCancelBatchResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}
		if len(resp.Cancelled) != 2 {
			t.Fatalf("Expected 2 cancelled, got %d", len(resp.Cancelled))
		}
		if len(resp.Skipped) != 0 {
			t.Fatalf("Expected 0 skipped, got %d", len(resp.Skipped))
		}
		var updatedA, updatedB models.ShiftCoverageRequest
		db.First(&updatedA, reqA.ID)
		db.First(&updatedB, reqB.ID)
		if updatedA.Status != models.CoverageRequestCancelled || updatedB.Status != models.CoverageRequestCancelled {
			t.Fatalf("Expected both requests cancelled, got %s and %s", updatedA.Status, updatedB.Status)
		}
	})

	t.Run("an already-claimed request is skipped while the rest still cancel", func(t *testing.T) {
		db := SetupTestDB(t)
		requester, claimant, group := setupCoverageTestGroup(t, db)
		if err := db.Create(&models.ShiftSlot{UserID: requester.ID, GroupID: group.ID, DayOfWeek: 4, Hour: 14}).Error; err != nil {
			t.Fatalf("Failed to create second shift slot: %v", err)
		}
		tue, _ := time.Parse("2006-01-02", nextWeekday(time.Tuesday))
		thu, _ := time.Parse("2006-01-02", nextWeekday(time.Thursday))
		claimed := createOpenCoverageRequest(t, db, group.ID, requester.ID, 2, 10, tue)
		if claim := performClaimCoverageRequest(db, claimant.ID, group.ID, claimed.ID); claim.Code != http.StatusOK {
			t.Fatalf("Expected claim to succeed, got %d: %s", claim.Code, claim.Body.String())
		}
		stillOpen := createOpenCoverageRequest(t, db, group.ID, requester.ID, 4, 14, thu)

		body := fmt.Sprintf(`{"request_ids":[%d,%d]}`, claimed.ID, stillOpen.ID)
		w := performCancelCoverageRequestsBatch(db, requester.ID, false, group.ID, body)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var resp coverageRequestCancelBatchResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}
		if len(resp.Cancelled) != 1 || resp.Cancelled[0].ID != stillOpen.ID {
			t.Fatalf("Expected only the still-open request cancelled, got %+v", resp.Cancelled)
		}
		if len(resp.Skipped) != 1 || resp.Skipped[0].ID != claimed.ID {
			t.Fatalf("Expected the claimed request to be reported skipped, got %+v", resp.Skipped)
		}
	})

	t.Run("group admin can bulk-cancel another member's open requests", func(t *testing.T) {
		db := SetupTestDB(t)
		requester, _, group := setupCoverageTestGroup(t, db)
		admin := CreateTestUser(t, db, "groupadmin", "groupadmin@example.com", "password123", false)
		AddUserToGroupWithAdmin(t, db, admin.ID, group.ID, true)
		tue, _ := time.Parse("2006-01-02", nextWeekday(time.Tuesday))
		reqRow := createOpenCoverageRequest(t, db, group.ID, requester.ID, 2, 10, tue)

		body := fmt.Sprintf(`{"request_ids":[%d]}`, reqRow.ID)
		w := performCancelCoverageRequestsBatch(db, admin.ID, false, group.ID, body)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var resp coverageRequestCancelBatchResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}
		if len(resp.Cancelled) != 1 {
			t.Fatalf("Expected 1 cancelled, got %d", len(resp.Cancelled))
		}
	})

	t.Run("non-admin, non-requester's items are skipped rather than failing the batch", func(t *testing.T) {
		db := SetupTestDB(t)
		requester, other, group := setupCoverageTestGroup(t, db)
		tue, _ := time.Parse("2006-01-02", nextWeekday(time.Tuesday))
		reqRow := createOpenCoverageRequest(t, db, group.ID, requester.ID, 2, 10, tue)

		body := fmt.Sprintf(`{"request_ids":[%d]}`, reqRow.ID)
		w := performCancelCoverageRequestsBatch(db, other.ID, false, group.ID, body)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var resp coverageRequestCancelBatchResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}
		if len(resp.Cancelled) != 0 || len(resp.Skipped) != 1 {
			t.Fatalf("Expected the item skipped, not cancelled or errored, got cancelled=%d skipped=%d", len(resp.Cancelled), len(resp.Skipped))
		}
		var unchanged models.ShiftCoverageRequest
		db.First(&unchanged, reqRow.ID)
		if unchanged.Status != models.CoverageRequestOpen {
			t.Fatalf("Expected the request to remain open, got %s", unchanged.Status)
		}
	})

	t.Run("empty request_ids is rejected", func(t *testing.T) {
		db := SetupTestDB(t)
		_, _, group := setupCoverageTestGroup(t, db)
		w := performCancelCoverageRequestsBatch(db, 1, false, group.ID, `{"request_ids":[]}`)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("Expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("a batch exceeding the item cap is rejected and cancels nothing", func(t *testing.T) {
		db := SetupTestDB(t)
		requester, _, group := setupCoverageTestGroup(t, db)
		tue, _ := time.Parse("2006-01-02", nextWeekday(time.Tuesday))
		reqRow := createOpenCoverageRequest(t, db, group.ID, requester.ID, 2, 10, tue)

		ids := make([]string, 0, 201)
		for i := 0; i < 201; i++ {
			ids = append(ids, "999999")
		}
		body := fmt.Sprintf(`{"request_ids":[%s]}`, strings.Join(ids, ","))
		w := performCancelCoverageRequestsBatch(db, requester.ID, false, group.ID, body)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("Expected 400, got %d: %s", w.Code, w.Body.String())
		}
		var unchanged models.ShiftCoverageRequest
		db.First(&unchanged, reqRow.ID)
		if unchanged.Status != models.CoverageRequestOpen {
			t.Fatalf("Expected the unrelated request untouched, got %s", unchanged.Status)
		}
	})

	t.Run("non-member is denied", func(t *testing.T) {
		db := SetupTestDB(t)
		requester, _, group := setupCoverageTestGroup(t, db)
		outsider := CreateTestUser(t, db, "outsider", "outsider@example.com", "password123", false)
		tue, _ := time.Parse("2006-01-02", nextWeekday(time.Tuesday))
		reqRow := createOpenCoverageRequest(t, db, group.ID, requester.ID, 2, 10, tue)

		body := fmt.Sprintf(`{"request_ids":[%d]}`, reqRow.ID)
		w := performCancelCoverageRequestsBatch(db, outsider.ID, false, group.ID, body)

		if w.Code != http.StatusForbidden {
			t.Fatalf("Expected 403, got %d: %s", w.Code, w.Body.String())
		}
	})
}

func performUpdateCoverageRequestPriority(db *gorm.DB, callerID uint, isAdmin bool, groupID, requestID uint, body string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", callerID)
		c.Set("is_admin", isAdmin)
		c.Next()
	})
	router.PATCH("/groups/:id/schedule/coverage-requests/:requestId/priority", UpdateCoverageRequestPriority(db))

	req := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/groups/%d/schedule/coverage-requests/%d/priority", groupID, requestID), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func TestUpdateCoverageRequestPriority(t *testing.T) {
	t.Run("group admin can override priority to optional", func(t *testing.T) {
		db := SetupTestDB(t)
		requester, _, group := setupCoverageTestGroup(t, db)
		admin := CreateTestUser(t, db, "groupadmin", "groupadmin@example.com", "password123", false)
		AddUserToGroupWithAdmin(t, db, admin.ID, group.ID, true)
		date, _ := time.Parse("2006-01-02", nextWeekday(time.Tuesday))
		reqRow := createOpenCoverageRequest(t, db, group.ID, requester.ID, 2, 10, date)

		w := performUpdateCoverageRequestPriority(db, admin.ID, false, group.ID, reqRow.ID, `{"priority":"optional"}`)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var resp coverageRequestResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}
		if resp.Priority != "optional" {
			t.Fatalf("Expected priority %q, got %q", "optional", resp.Priority)
		}
		var updated models.ShiftCoverageRequest
		if err := db.First(&updated, reqRow.ID).Error; err != nil {
			t.Fatalf("Failed to reload request: %v", err)
		}
		if updated.Priority != "optional" {
			t.Fatalf("Expected persisted priority %q, got %q", "optional", updated.Priority)
		}
	})

	t.Run("site admin can override priority", func(t *testing.T) {
		db := SetupTestDB(t)
		requester, _, group := setupCoverageTestGroup(t, db)
		siteAdmin := CreateTestUser(t, db, "admin", "admin@example.com", "password123", true)
		date, _ := time.Parse("2006-01-02", nextWeekday(time.Tuesday))
		reqRow := createOpenCoverageRequest(t, db, group.ID, requester.ID, 2, 10, date)

		w := performUpdateCoverageRequestPriority(db, siteAdmin.ID, true, group.ID, reqRow.ID, `{"priority":"optional"}`)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("non-admin, non-requester member cannot override priority", func(t *testing.T) {
		db := SetupTestDB(t)
		requester, other, group := setupCoverageTestGroup(t, db)
		date, _ := time.Parse("2006-01-02", nextWeekday(time.Tuesday))
		reqRow := createOpenCoverageRequest(t, db, group.ID, requester.ID, 2, 10, date)

		w := performUpdateCoverageRequestPriority(db, other.ID, false, group.ID, reqRow.ID, `{"priority":"optional"}`)

		if w.Code != http.StatusForbidden {
			t.Fatalf("Expected 403, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("requester themself cannot override priority without being an admin", func(t *testing.T) {
		db := SetupTestDB(t)
		requester, _, group := setupCoverageTestGroup(t, db)
		date, _ := time.Parse("2006-01-02", nextWeekday(time.Tuesday))
		reqRow := createOpenCoverageRequest(t, db, group.ID, requester.ID, 2, 10, date)

		w := performUpdateCoverageRequestPriority(db, requester.ID, false, group.ID, reqRow.ID, `{"priority":"optional"}`)

		if w.Code != http.StatusForbidden {
			t.Fatalf("Expected 403, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("rejects an invalid priority value", func(t *testing.T) {
		db := SetupTestDB(t)
		requester, _, group := setupCoverageTestGroup(t, db)
		admin := CreateTestUser(t, db, "groupadmin", "groupadmin@example.com", "password123", false)
		AddUserToGroupWithAdmin(t, db, admin.ID, group.ID, true)
		date, _ := time.Parse("2006-01-02", nextWeekday(time.Tuesday))
		reqRow := createOpenCoverageRequest(t, db, group.ID, requester.ID, 2, 10, date)

		w := performUpdateCoverageRequestPriority(db, admin.ID, false, group.ID, reqRow.ID, `{"priority":"urgent"}`)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("Expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("rejects an omitted priority rather than silently defaulting to normal", func(t *testing.T) {
		// Unlike creation (where an omitted priority is a reasonable "I
		// didn't think about it, use normal" default), an explicit override
		// call with no priority is almost certainly a client bug - silently
		// defaulting here would let a malformed request quietly downgrade
		// nothing while reporting success.
		db := SetupTestDB(t)
		requester, _, group := setupCoverageTestGroup(t, db)
		admin := CreateTestUser(t, db, "groupadmin", "groupadmin@example.com", "password123", false)
		AddUserToGroupWithAdmin(t, db, admin.ID, group.ID, true)
		date, _ := time.Parse("2006-01-02", nextWeekday(time.Tuesday))
		reqRow := createOpenCoverageRequest(t, db, group.ID, requester.ID, 2, 10, date)

		w := performUpdateCoverageRequestPriority(db, admin.ID, false, group.ID, reqRow.ID, `{}`)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("Expected 400, got %d: %s", w.Code, w.Body.String())
		}
		var updated models.ShiftCoverageRequest
		if err := db.First(&updated, reqRow.ID).Error; err != nil {
			t.Fatalf("Failed to reload request: %v", err)
		}
		if updated.Priority != "normal" {
			t.Fatalf("Expected priority to remain unchanged at %q, got %q", "normal", updated.Priority)
		}
	})

	t.Run("returns 404 for a request in a different group", func(t *testing.T) {
		db := SetupTestDB(t)
		requester, _, group := setupCoverageTestGroup(t, db)
		otherGroup := createSchedulingEnabledGroup(t, db, "Cats", "Cat volunteers")
		admin := CreateTestUser(t, db, "groupadmin", "groupadmin@example.com", "password123", false)
		AddUserToGroupWithAdmin(t, db, admin.ID, otherGroup.ID, true)
		date, _ := time.Parse("2006-01-02", nextWeekday(time.Tuesday))
		reqRow := createOpenCoverageRequest(t, db, group.ID, requester.ID, 2, 10, date)

		w := performUpdateCoverageRequestPriority(db, admin.ID, false, otherGroup.ID, reqRow.ID, `{"priority":"optional"}`)

		if w.Code != http.StatusNotFound {
			t.Fatalf("Expected 404, got %d: %s", w.Code, w.Body.String())
		}
	})
}

func performReassignShiftsBatch(db *gorm.DB, callerID uint, isAdmin bool, groupID uint, body string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", callerID)
		c.Set("is_admin", isAdmin)
		c.Next()
	})
	router.POST("/groups/:id/schedule/reassign", ReassignShiftsBatch(db, nil, nil))

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/groups/%d/schedule/reassign", groupID), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

// TestReassignShiftsBatch covers the "previously approved / agreed in
// person" admin shortcut, extended to several hours in one call so a
// multi-hour swap sends one notification instead of one per hour.
func TestReassignShiftsBatch(t *testing.T) {
	t.Run("group admin can reassign several hours to another member in one call, with one consolidated response", func(t *testing.T) {
		db := SetupTestDB(t)
		requester, other, group := setupCoverageTestGroup(t, db)
		admin := CreateTestUser(t, db, "groupadmin", "groupadmin@example.com", "password123", false)
		AddUserToGroupWithAdmin(t, db, admin.ID, group.ID, true)
		// requester already has Tuesday 10am (from setupCoverageTestGroup); add 11am and 12pm too.
		if err := db.Create(&models.ShiftSlot{UserID: requester.ID, GroupID: group.ID, DayOfWeek: 2, Hour: 11}).Error; err != nil {
			t.Fatalf("Failed to create second shift slot: %v", err)
		}
		if err := db.Create(&models.ShiftSlot{UserID: requester.ID, GroupID: group.ID, DayOfWeek: 2, Hour: 12}).Error; err != nil {
			t.Fatalf("Failed to create third shift slot: %v", err)
		}
		date := nextWeekday(time.Tuesday)
		body := fmt.Sprintf(`{"from_user_id":%d,"to_user_id":%d,"date":"%s","hours":[10,11,12]}`, requester.ID, other.ID, date)

		w := performReassignShiftsBatch(db, admin.ID, false, group.ID, body)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var resp reassignShiftsBatchResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}
		if len(resp.Created) != 3 {
			t.Fatalf("Expected 3 created, got %d: %+v", len(resp.Created), resp.Created)
		}
		if len(resp.Skipped) != 0 {
			t.Fatalf("Expected 0 skipped, got %d: %+v", len(resp.Skipped), resp.Skipped)
		}
		for _, created := range resp.Created {
			if created.ClaimedByUserID == nil || *created.ClaimedByUserID != other.ID {
				t.Errorf("Expected claimed_by_user_id %d, got %v", other.ID, created.ClaimedByUserID)
			}
		}
	})

	t.Run("a conflicting hour is skipped while the rest of the batch still succeeds", func(t *testing.T) {
		db := SetupTestDB(t)
		requester, other, group := setupCoverageTestGroup(t, db)
		admin := CreateTestUser(t, db, "groupadmin", "groupadmin@example.com", "password123", false)
		AddUserToGroupWithAdmin(t, db, admin.ID, group.ID, true)
		if err := db.Create(&models.ShiftSlot{UserID: requester.ID, GroupID: group.ID, DayOfWeek: 2, Hour: 11}).Error; err != nil {
			t.Fatalf("Failed to create second shift slot: %v", err)
		}
		// other already has a conflicting Tuesday 11am shift in a different group.
		otherGroup := CreateTestGroup(t, db, "Cats", "Cat volunteers")
		AddUserToGroupWithAdmin(t, db, other.ID, otherGroup.ID, false)
		if err := db.Create(&models.ShiftSlot{UserID: other.ID, GroupID: otherGroup.ID, DayOfWeek: 2, Hour: 11}).Error; err != nil {
			t.Fatalf("Failed to create conflicting shift slot: %v", err)
		}
		date := nextWeekday(time.Tuesday)
		body := fmt.Sprintf(`{"from_user_id":%d,"to_user_id":%d,"date":"%s","hours":[10,11]}`, requester.ID, other.ID, date)

		w := performReassignShiftsBatch(db, admin.ID, false, group.ID, body)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var resp reassignShiftsBatchResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}
		if len(resp.Created) != 1 || resp.Created[0].Hour != 10 {
			t.Fatalf("Expected hour 10 created, got %+v", resp.Created)
		}
		if len(resp.Skipped) != 1 || resp.Skipped[0].Hour != 11 {
			t.Fatalf("Expected hour 11 skipped, got %+v", resp.Skipped)
		}
	})

	t.Run("skips (not fails) an hour where from_user has no matching slot, while other hours still succeed", func(t *testing.T) {
		db := SetupTestDB(t)
		requester, other, group := setupCoverageTestGroup(t, db)
		admin := CreateTestUser(t, db, "groupadmin", "groupadmin@example.com", "password123", false)
		AddUserToGroupWithAdmin(t, db, admin.ID, group.ID, true)
		// requester's only slot is Tuesday 10am (from setupCoverageTestGroup) - no 11am slot exists.
		date := nextWeekday(time.Tuesday)
		body := fmt.Sprintf(`{"from_user_id":%d,"to_user_id":%d,"date":"%s","hours":[10,11]}`, requester.ID, other.ID, date)

		w := performReassignShiftsBatch(db, admin.ID, false, group.ID, body)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var resp reassignShiftsBatchResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}
		if len(resp.Created) != 1 || resp.Created[0].Hour != 10 {
			t.Fatalf("Expected hour 10 created, got %+v", resp.Created)
		}
		if len(resp.Skipped) != 1 || resp.Skipped[0].Hour != 11 {
			t.Fatalf("Expected hour 11 skipped, got %+v", resp.Skipped)
		}
	})

	t.Run("skips (not fails) an hour where an active coverage request already exists for from_user, while other hours still succeed", func(t *testing.T) {
		db := SetupTestDB(t)
		requester, other, group := setupCoverageTestGroup(t, db)
		admin := CreateTestUser(t, db, "groupadmin", "groupadmin@example.com", "password123", false)
		AddUserToGroupWithAdmin(t, db, admin.ID, group.ID, true)
		if err := db.Create(&models.ShiftSlot{UserID: requester.ID, GroupID: group.ID, DayOfWeek: 2, Hour: 11}).Error; err != nil {
			t.Fatalf("Failed to create second shift slot: %v", err)
		}
		parsedDate, _ := time.Parse("2006-01-02", nextWeekday(time.Tuesday))
		// requester already has an open coverage request at hour 10.
		createOpenCoverageRequest(t, db, group.ID, requester.ID, 2, 10, parsedDate)
		date := parsedDate.Format("2006-01-02")
		body := fmt.Sprintf(`{"from_user_id":%d,"to_user_id":%d,"date":"%s","hours":[10,11]}`, requester.ID, other.ID, date)

		w := performReassignShiftsBatch(db, admin.ID, false, group.ID, body)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var resp reassignShiftsBatchResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}
		if len(resp.Created) != 1 || resp.Created[0].Hour != 11 {
			t.Fatalf("Expected hour 11 created, got %+v", resp.Created)
		}
		if len(resp.Skipped) != 1 || resp.Skipped[0].Hour != 10 || !strings.Contains(resp.Skipped[0].Reason, "already exists") {
			t.Fatalf("Expected hour 10 skipped with an 'already exists' reason, got %+v", resp.Skipped)
		}
	})

	t.Run("non-admin is forbidden", func(t *testing.T) {
		db := SetupTestDB(t)
		requester, other, group := setupCoverageTestGroup(t, db)
		date := nextWeekday(time.Tuesday)
		body := fmt.Sprintf(`{"from_user_id":%d,"to_user_id":%d,"date":"%s","hours":[10]}`, requester.ID, other.ID, date)

		w := performReassignShiftsBatch(db, other.ID, false, group.ID, body)

		if w.Code != http.StatusForbidden {
			t.Fatalf("Expected 403, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("rejects a past date", func(t *testing.T) {
		db := SetupTestDB(t)
		requester, other, group := setupCoverageTestGroup(t, db)
		admin := CreateTestUser(t, db, "groupadmin", "groupadmin@example.com", "password123", false)
		AddUserToGroupWithAdmin(t, db, admin.ID, group.ID, true)
		past := time.Now().UTC().AddDate(0, 0, -1).Format("2006-01-02")
		body := fmt.Sprintf(`{"from_user_id":%d,"to_user_id":%d,"date":"%s","hours":[10]}`, requester.ID, other.ID, past)

		w := performReassignShiftsBatch(db, admin.ID, false, group.ID, body)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("Expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("rejects an empty hours list", func(t *testing.T) {
		db := SetupTestDB(t)
		requester, other, group := setupCoverageTestGroup(t, db)
		admin := CreateTestUser(t, db, "groupadmin", "groupadmin@example.com", "password123", false)
		AddUserToGroupWithAdmin(t, db, admin.ID, group.ID, true)
		date := nextWeekday(time.Tuesday)
		body := fmt.Sprintf(`{"from_user_id":%d,"to_user_id":%d,"date":"%s","hours":[]}`, requester.ID, other.ID, date)

		w := performReassignShiftsBatch(db, admin.ID, false, group.ID, body)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("Expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("rejects a batch exceeding the max hours per call", func(t *testing.T) {
		db := SetupTestDB(t)
		requester, other, group := setupCoverageTestGroup(t, db)
		admin := CreateTestUser(t, db, "groupadmin", "groupadmin@example.com", "password123", false)
		AddUserToGroupWithAdmin(t, db, admin.ID, group.ID, true)
		date := nextWeekday(time.Tuesday)
		hours := make([]string, 0, 21)
		for h := 0; h < 21; h++ {
			hours = append(hours, fmt.Sprintf("%d", h))
		}
		body := fmt.Sprintf(`{"from_user_id":%d,"to_user_id":%d,"date":"%s","hours":[%s]}`, requester.ID, other.ID, date, strings.Join(hours, ","))

		w := performReassignShiftsBatch(db, admin.ID, false, group.ID, body)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("Expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("rejects a duplicate hour in the list", func(t *testing.T) {
		db := SetupTestDB(t)
		requester, other, group := setupCoverageTestGroup(t, db)
		admin := CreateTestUser(t, db, "groupadmin", "groupadmin@example.com", "password123", false)
		AddUserToGroupWithAdmin(t, db, admin.ID, group.ID, true)
		date := nextWeekday(time.Tuesday)
		body := fmt.Sprintf(`{"from_user_id":%d,"to_user_id":%d,"date":"%s","hours":[10,10]}`, requester.ID, other.ID, date)

		w := performReassignShiftsBatch(db, admin.ID, false, group.ID, body)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("Expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("rejects reassigning to the same person", func(t *testing.T) {
		db := SetupTestDB(t)
		requester, _, group := setupCoverageTestGroup(t, db)
		admin := CreateTestUser(t, db, "groupadmin", "groupadmin@example.com", "password123", false)
		AddUserToGroupWithAdmin(t, db, admin.ID, group.ID, true)
		date := nextWeekday(time.Tuesday)
		body := fmt.Sprintf(`{"from_user_id":%d,"to_user_id":%d,"date":"%s","hours":[10]}`, requester.ID, requester.ID, date)

		w := performReassignShiftsBatch(db, admin.ID, false, group.ID, body)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("Expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("rejects when to_user is not a member of the group", func(t *testing.T) {
		db := SetupTestDB(t)
		requester, _, group := setupCoverageTestGroup(t, db)
		admin := CreateTestUser(t, db, "groupadmin", "groupadmin@example.com", "password123", false)
		AddUserToGroupWithAdmin(t, db, admin.ID, group.ID, true)
		outsider := CreateTestUser(t, db, "outsider", "outsider@example.com", "password123", false)
		date := nextWeekday(time.Tuesday)
		body := fmt.Sprintf(`{"from_user_id":%d,"to_user_id":%d,"date":"%s","hours":[10]}`, requester.ID, outsider.ID, date)

		w := performReassignShiftsBatch(db, admin.ID, false, group.ID, body)

		if w.Code != http.StatusNotFound {
			t.Fatalf("Expected 404, got %d: %s", w.Code, w.Body.String())
		}
	})
}
