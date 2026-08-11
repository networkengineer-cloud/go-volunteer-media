package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
	"gorm.io/gorm"
)

// These tests exist because a manual smoke test against a real Postgres
// instance surfaced a bug no SQLite-backed test (every other test in this
// package) could catch: gorm.io/driver/postgres (via pgx) decodes
// `timestamptz` columns into time.Time using time.Local, not UTC. Any
// ShiftCoverageRequest.Date read back from the database - as opposed to a
// value freshly parsed from a request body - came back shifted to whatever
// the process's local timezone was, silently corrupting every
// .Format()/.Weekday() call on it: ClaimCoverageRequest's response date,
// its cross-group weekday conflict check, GetGroupScheduleOverview's
// date-keyed roster lookup (which broke entirely - a claimed request never
// showed as "covering"), and notification content. Fixed by forcing
// time.Local = time.UTC in internal/database's init() (see that file).
// These tests fail against that pre-fix state and pass after it, using the
// same reachable-Postgres-or-skip contract as search_postgres_test.go's
// openSearchTestPostgres/envOrDefault/tcpReachable helpers (shared, defined
// in that file, package-wide).

func TestShiftCoverageRequestDate_PostgresRoundTrip(t *testing.T) {
	db := openSearchTestPostgres(t)
	tx := db.Begin()
	t.Cleanup(func() { tx.Rollback() })

	unique := time.Now().UnixNano()
	requester := &models.User{
		Username: fmt.Sprintf("pgdate-requester-%d", unique),
		Email:    fmt.Sprintf("pgdate-requester-%d@example.com", unique),
		Password: "x",
	}
	if err := tx.Create(requester).Error; err != nil {
		t.Fatalf("create requester: %v", err)
	}
	group := &models.Group{Name: fmt.Sprintf("PgDateTest-%d", unique)}
	if err := tx.Create(group).Error; err != nil {
		t.Fatalf("create group: %v", err)
	}

	// 2026-08-11 is a Tuesday (weekday 2).
	utcMidnight := time.Date(2026, time.August, 11, 0, 0, 0, 0, time.UTC)
	created := &models.ShiftCoverageRequest{
		GroupID:           group.ID,
		RequestedByUserID: requester.ID,
		Date:              utcMidnight,
		Hour:              10,
		Status:            models.CoverageRequestOpen,
	}
	if err := tx.Create(created).Error; err != nil {
		t.Fatalf("create ShiftCoverageRequest: %v", err)
	}

	var fetched models.ShiftCoverageRequest
	if err := tx.Where("id = ?", created.ID).First(&fetched).Error; err != nil {
		t.Fatalf("re-fetch ShiftCoverageRequest: %v", err)
	}

	if got := fetched.Date.Format("2006-01-02"); got != "2026-08-11" {
		t.Fatalf("Date.Format(\"2006-01-02\") after a real Postgres round-trip = %q, want \"2026-08-11\" (time.Local must be UTC - see internal/database's init())", got)
	}
	if got := fetched.Date.Weekday(); got != time.Tuesday {
		t.Fatalf("Date.Weekday() after a real Postgres round-trip = %v, want Tuesday (time.Local must be UTC - see internal/database's init())", got)
	}
}

func TestGetGroupScheduleOverview_Postgres_ClaimedRequestShowsAsCovering(t *testing.T) {
	// End-to-end version of the same bug: this is the exact sequence (create
	// -> claim -> view overview) manually verified against a real Postgres
	// instance to surface it in the first place. Before the fix, the
	// claimant never appeared as "covering" - GetGroupScheduleOverview's
	// date-keyed lookup, built from a DB-read ShiftCoverageRequest.Date,
	// silently failed to match the date computed by iterating the requested
	// week, because the two representations disagreed by one day.
	db := openSearchTestPostgres(t)
	tx := db.Begin()
	t.Cleanup(func() { tx.Rollback() })

	unique := time.Now().UnixNano()
	requester := CreateTestUser(t, tx, fmt.Sprintf("pgcov-req-%d", unique), fmt.Sprintf("pgcov-req-%d@example.com", unique), "password123", false)
	claimant := CreateTestUser(t, tx, fmt.Sprintf("pgcov-claim-%d", unique), fmt.Sprintf("pgcov-claim-%d@example.com", unique), "password123", false)
	group := CreateTestGroup(t, tx, fmt.Sprintf("PgCovTest-%d", unique), "Postgres coverage overlay test")
	AddUserToGroupWithAdmin(t, tx, requester.ID, group.ID, false)
	AddUserToGroupWithAdmin(t, tx, claimant.ID, group.ID, false)
	if err := tx.Model(group).Update("scheduling_enabled", true).Error; err != nil {
		t.Fatalf("enable scheduling: %v", err)
	}

	// A future Tuesday, computed relative to whenever this test actually
	// runs (nextWeekday is the same helper every other coverage-request
	// test in this package uses for exactly this reason) - createOneCoverageRequest
	// rejects a past date, so a hard-coded date here would start failing
	// the day after whatever date got hard-coded, with a misleading
	// "expected 201, got 400" error that points nowhere near the
	// timezone fix this test exists to guard.
	date := nextWeekday(time.Tuesday)
	parsedDate, err := time.Parse("2006-01-02", date)
	if err != nil {
		t.Fatalf("parse computed date: %v", err)
	}
	dayOfWeek := int(parsedDate.Weekday())
	if err := tx.Create(&models.ShiftSlot{UserID: requester.ID, GroupID: group.ID, DayOfWeek: dayOfWeek, Hour: 10}).Error; err != nil {
		t.Fatalf("create shift slot: %v", err)
	}

	createBody := fmt.Sprintf(`{"date":"%s","hour":10}`, date)
	createResp := performCreateCoverageRequest(tx, requester.ID, false, group.ID, createBody)
	if createResp.Code != http.StatusCreated {
		t.Fatalf("expected 201 creating coverage request, got %d: %s", createResp.Code, createResp.Body.String())
	}
	var created coverageRequestResponse
	if err := json.Unmarshal(createResp.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}

	claimResp := performClaimCoverageRequest(tx, claimant.ID, group.ID, created.ID)
	if claimResp.Code != http.StatusOK {
		t.Fatalf("expected 200 claiming coverage request, got %d: %s", claimResp.Code, claimResp.Body.String())
	}

	weekStart := parsedDate.AddDate(0, 0, -dayOfWeek).Format("2006-01-02") // the Sunday that starts date's week
	overview := performGetGroupScheduleOverview(tx, requester.ID, group.ID, weekStart)
	if overview.Code != http.StatusOK {
		t.Fatalf("expected 200 fetching overview, got %d: %s", overview.Code, overview.Body.String())
	}
	var body struct {
		Slots []scheduleOverviewSlot `json:"slots"`
	}
	if err := json.Unmarshal(overview.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode overview response: %v", err)
	}

	var found bool
	for _, s := range body.Slots {
		if s.Date != date || s.Hour != 10 {
			continue
		}
		for _, m := range s.Members {
			if m.UserID == claimant.ID {
				found = true
				if m.Status != "covering" {
					t.Fatalf("claimant's status = %q, want \"covering\" (time.Local must be UTC - see internal/database's init())", m.Status)
				}
			}
			if m.UserID == requester.ID {
				t.Fatalf("original requester still appears in the roster after their shift was claimed; got status %q", m.Status)
			}
		}
	}
	if !found {
		t.Fatal("claimant never appears in the overview roster for the claimed date/hour at all - the date-keyed lookup silently failed to match")
	}
}

// performGetGroupScheduleOverview mirrors performCreateCoverageRequest's
// pattern for GetGroupScheduleOverview, which existing SQLite tests
// (TestGetGroupScheduleOverview_EffectiveRoster) set up inline rather than
// via a shared helper - factored out here since this file needs the exact
// same request shape.
func performGetGroupScheduleOverview(db *gorm.DB, callerID uint, groupID uint, weekStart string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", callerID)
		c.Set("is_admin", false)
		c.Next()
	})
	router.GET("/groups/:id/schedule/overview", GetGroupScheduleOverview(db))

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/groups/%d/schedule/overview?week_start=%s", groupID, weekStart), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}
