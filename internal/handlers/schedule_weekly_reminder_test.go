package handlers

import (
	"bytes"
	"context"
	"errors"
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

func mustChicago(t *testing.T) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation("America/Chicago")
	if err != nil {
		t.Fatalf("load location: %v", err)
	}
	return loc
}

func TestWeeklyReminderWindow(t *testing.T) {
	loc := mustChicago(t)
	cases := []struct {
		name    string
		local   time.Time
		ok      bool
		weekKey string
		start   string
		end     string
	}{
		{"Sunday before send hour", time.Date(2026, 11, 22, 16, 59, 0, 0, loc), false, "", "", ""},
		{"Sunday at send hour", time.Date(2026, 11, 22, 17, 0, 0, 0, loc), true, "2026-11-22", "2026-11-23", "2026-11-30"},
		{"Sunday late evening (already Monday in UTC)", time.Date(2026, 11, 22, 23, 30, 0, 0, loc), true, "2026-11-22", "2026-11-23", "2026-11-30"},
		{"Monday", time.Date(2026, 11, 23, 17, 0, 0, 0, loc), false, "", "", ""},
		{"Saturday", time.Date(2026, 11, 21, 18, 0, 0, 0, loc), false, "", "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			weekKey, start, end, ok := weeklyReminderWindow(tc.local.UTC(), loc, 17)
			if ok != tc.ok {
				t.Fatalf("ok = %v, want %v", ok, tc.ok)
			}
			if !ok {
				return
			}
			if weekKey != tc.weekKey || start.Format("2006-01-02") != tc.start || end.Format("2006-01-02") != tc.end {
				t.Errorf("got (%s, %s, %s), want (%s, %s, %s)", weekKey, start.Format("2006-01-02"), end.Format("2006-01-02"), tc.weekKey, tc.start, tc.end)
			}
			if start.Location() != time.UTC {
				t.Errorf("range must be UTC midnights to match ShiftCoverageRequest.Date, got %v", start.Location())
			}
		})
	}
}

func TestClaimWeeklyReminder_OncePerGroupPerWeek(t *testing.T) {
	db := SetupTestDB(t)

	steps := []struct {
		group uint
		week  string
		want  bool
	}{
		{1, "2026-11-22", true},
		{1, "2026-11-22", false}, // second replica / next tick
		{2, "2026-11-22", true},  // other groups are independent
		{1, "2026-11-29", true},  // next week
		{1, "2026-11-29", false},
	}
	for i, s := range steps {
		got, err := claimWeeklyReminder(db, s.group, s.week)
		if err != nil {
			t.Fatalf("step %d: %v", i, err)
		}
		if got != s.want {
			t.Errorf("step %d (group %d, week %s): claimed = %v, want %v", i, s.group, s.week, got, s.want)
		}
	}

	var rows int64
	db.Model(&models.SiteSetting{}).Where("key LIKE ?", weeklyReminderMarkerPrefix+"%").Count(&rows)
	if rows != 2 {
		t.Errorf("expected one marker row per group, got %d", rows)
	}
}

func createWeeklyRequest(t *testing.T, db *gorm.DB, groupID, userID uint, date time.Time, hour int, status models.CoverageRequestStatus) *models.ShiftCoverageRequest {
	t.Helper()
	// Already announced by the coverage digest - the normal state of a
	// request by Sunday. See createUnannouncedWeeklyRequest for one that isn't.
	announced := time.Now()
	req := &models.ShiftCoverageRequest{GroupID: groupID, RequestedByUserID: userID, Date: date, Hour: hour, Status: status, NotifiedAt: &announced}
	if err := db.Create(req).Error; err != nil {
		t.Fatalf("create request: %v", err)
	}
	return req
}

func createUnannouncedWeeklyRequest(t *testing.T, db *gorm.DB, groupID, userID uint, date time.Time, hour int, age time.Duration) *models.ShiftCoverageRequest {
	t.Helper()
	req := &models.ShiftCoverageRequest{GroupID: groupID, RequestedByUserID: userID, Date: date, Hour: hour, Status: models.CoverageRequestOpen}
	if err := db.Create(req).Error; err != nil {
		t.Fatalf("create request: %v", err)
	}
	if err := db.Model(req).UpdateColumn("created_at", time.Now().Add(-age)).Error; err != nil {
		t.Fatalf("backdate request: %v", err)
	}
	return req
}

// failingEmailProvider fails every send.
type failingEmailProvider struct{}

func (failingEmailProvider) SendEmail(context.Context, string, string, string) error {
	return errors.New("provider down")
}
func (failingEmailProvider) IsConfigured() bool      { return true }
func (failingEmailProvider) GetProviderName() string { return "failing-mock" }

func TestSendWeeklyCoverageReminders(t *testing.T) {
	loc := mustChicago(t)
	sundayEvening := time.Date(2026, 11, 22, 17, 5, 0, 0, loc)
	day := func(d int) time.Time { return time.Date(2026, 11, d, 0, 0, 0, 0, time.UTC) }

	setup := func(t *testing.T) (*gorm.DB, *recordingEmailProvider, *email.Service, *models.Group, *models.User) {
		t.Helper()
		db := SetupTestDB(t)
		requester, other, group := setupCoverageTestGroup(t, db)
		if err := db.Model(other).Update("email_notifications_enabled", true).Error; err != nil {
			t.Fatalf("enable email: %v", err)
		}
		if err := db.Model(group).Update("weekly_coverage_reminder_enabled", true).Error; err != nil {
			t.Fatalf("opt group in: %v", err)
		}
		provider := &recordingEmailProvider{}
		return db, provider, email.NewServiceWithProvider(provider, db), group, requester
	}

	t.Run("emails the coming week's open requests once", func(t *testing.T) {
		t.Setenv("FRONTEND_URL", "https://portal.example.com")
		t.Setenv("SCHEDULE_EMAIL_NOTIFICATIONS_ENABLED", "true")
		t.Setenv("WEEKLY_COVERAGE_REMINDER_ENABLED", "true")
		db, provider, svc, group, requester := setup(t)

		createWeeklyRequest(t, db, group.ID, requester.ID, day(24), 10, models.CoverageRequestOpen)
		createWeeklyRequest(t, db, group.ID, requester.ID, day(22), 10, models.CoverageRequestOpen)    // today: excluded
		createWeeklyRequest(t, db, group.ID, requester.ID, day(30), 10, models.CoverageRequestOpen)    // week after: excluded
		createWeeklyRequest(t, db, group.ID, requester.ID, day(25), 11, models.CoverageRequestClaimed) // filled: excluded

		sendWeeklyCoverageReminders(db, svc, sundayEvening, loc, 17)
		sendWeeklyCoverageReminders(db, svc, sundayEvening.Add(10*time.Minute), loc, 17)

		if len(provider.sends) != 1 {
			t.Fatalf("expected 1 email (one opted-in member, one send per week), got %d", len(provider.sends))
		}
		sent := provider.sends[0]
		if sent.to != "other@example.com" {
			t.Errorf("sent to %q", sent.to)
		}
		if want := "Coverage needed this week in Dogs - " + models.DefaultSiteName; sent.subject != want {
			t.Errorf("subject = %q, want %q", sent.subject, want)
		}
		if want := fmt.Sprintf(`href="https://portal.example.com/groups/%d?view=schedule"`, group.ID); !strings.Contains(sent.body, want) {
			t.Errorf("body should link to the group's schedule (%s):\n%s", want, sent.body)
		}
		if !strings.Contains(sent.body, "Tuesday, November 24") {
			t.Errorf("body should list the in-week shift:\n%s", sent.body)
		}
		for _, excluded := range []string{"November 22", "November 30", "November 25"} {
			if strings.Contains(sent.body, excluded) {
				t.Errorf("body should not mention %s:\n%s", excluded, sent.body)
			}
		}
	})

	t.Run("leaves requests still waiting for their first announcement to the digest", func(t *testing.T) {
		t.Setenv("SCHEDULE_EMAIL_NOTIFICATIONS_ENABLED", "true")
		t.Setenv("WEEKLY_COVERAGE_REMINDER_ENABLED", "true")
		db, provider, svc, group, requester := setup(t)
		createWeeklyRequest(t, db, group.ID, requester.ID, day(24), 10, models.CoverageRequestOpen)
		// Filed a minute ago: inside the digest's quiet period.
		fresh := createUnannouncedWeeklyRequest(t, db, group.ID, requester.ID, day(25), 10, time.Minute)

		sendWeeklyCoverageReminders(db, svc, sundayEvening, loc, 17)

		if len(provider.sends) != 1 {
			t.Fatalf("expected 1 weekly email, got %d", len(provider.sends))
		}
		if strings.Contains(provider.sends[0].body, "November 25") {
			t.Errorf("weekly email must not list a request the digest hasn't announced yet:\n%s", provider.sends[0].body)
		}
		var reloaded models.ShiftCoverageRequest
		db.First(&reloaded, fresh.ID)
		if reloaded.NotifiedAt != nil {
			t.Fatal("weekly reminder must not stamp a request it didn't list")
		}

		// Once its quiet period is over, the digest still announces it.
		if err := db.Model(fresh).UpdateColumn("created_at", time.Now().Add(-10*time.Minute)).Error; err != nil {
			t.Fatalf("backdate: %v", err)
		}
		rec := &notifyRecorder{}
		sweepCoverageDigests(db, rec.record)
		if rec.count() != 1 {
			t.Errorf("digest should still announce the new request, got %d announcements", rec.count())
		}
	})

	t.Run("a request filed later on Sunday evening does not trigger a late weekly email", func(t *testing.T) {
		t.Setenv("SCHEDULE_EMAIL_NOTIFICATIONS_ENABLED", "true")
		t.Setenv("WEEKLY_COVERAGE_REMINDER_ENABLED", "true")
		db, provider, svc, group, requester := setup(t)

		// 17:05, nothing open: the group's turn is used up, no email.
		sendWeeklyCoverageReminders(db, svc, sundayEvening, loc, 17)
		// Announced later that evening, then the 23:55 tick.
		createWeeklyRequest(t, db, group.ID, requester.ID, day(24), 10, models.CoverageRequestOpen)
		sendWeeklyCoverageReminders(db, svc, time.Date(2026, 11, 22, 23, 55, 0, 0, loc), loc, 17)

		if len(provider.sends) != 0 {
			t.Fatalf("expected no weekly email after the 5 PM pass found nothing open, got %d", len(provider.sends))
		}
	})

	t.Run("each group gets only its own requests", func(t *testing.T) {
		t.Setenv("SCHEDULE_EMAIL_NOTIFICATIONS_ENABLED", "true")
		t.Setenv("WEEKLY_COVERAGE_REMINDER_ENABLED", "true")
		db, provider, svc, dogs, requester := setup(t)
		cats := CreateTestGroup(t, db, "Cats", "Cat volunteers")
		if err := db.Model(cats).Updates(map[string]interface{}{"scheduling_enabled": true, "weekly_coverage_reminder_enabled": true}).Error; err != nil {
			t.Fatalf("enable cats: %v", err)
		}
		catPerson := CreateTestUser(t, db, "catperson", "cat@example.com", "password123", false)
		AddUserToGroupWithAdmin(t, db, catPerson.ID, cats.ID, false)
		if err := db.Model(catPerson).Update("email_notifications_enabled", true).Error; err != nil {
			t.Fatalf("enable email: %v", err)
		}
		createWeeklyRequest(t, db, dogs.ID, requester.ID, day(24), 10, models.CoverageRequestOpen)
		createWeeklyRequest(t, db, cats.ID, catPerson.ID, day(26), 12, models.CoverageRequestOpen)

		sendWeeklyCoverageReminders(db, svc, sundayEvening, loc, 17)

		if len(provider.sends) != 2 {
			t.Fatalf("expected one email per group, got %d", len(provider.sends))
		}
		for _, sent := range provider.sends {
			switch sent.to {
			case "other@example.com":
				if !strings.Contains(sent.body, "November 24") || strings.Contains(sent.body, "November 26") {
					t.Errorf("Dogs email has the wrong shifts:\n%s", sent.body)
				}
			case "cat@example.com":
				if !strings.Contains(sent.body, "November 26") || strings.Contains(sent.body, "November 24") {
					t.Errorf("Cats email has the wrong shifts:\n%s", sent.body)
				}
			default:
				t.Errorf("unexpected recipient %q", sent.to)
			}
		}
	})

	t.Run("reports when every send fails", func(t *testing.T) {
		db, _, _, group, requester := setup(t)
		createWeeklyRequest(t, db, group.ID, requester.ID, day(24), 10, models.CoverageRequestOpen)
		svc := email.NewServiceWithProvider(failingEmailProvider{}, db)

		sent, total, err := sendGroupEmailsCounted(context.Background(), db, group.ID, func(to []string) []error {
			return svc.SendNotificationEmailBatch(context.Background(), to, email.NotificationEmail{Title: "t", Content: "c"})
		})
		if err != nil {
			t.Fatalf("per-recipient failures are counted, not returned: %v", err)
		}
		if sent != 0 || total != 1 {
			t.Errorf("got sent=%d total=%d, want 0 of 1", sent, total)
		}
	})

	t.Run("does nothing outside the Sunday window", func(t *testing.T) {
		t.Setenv("SCHEDULE_EMAIL_NOTIFICATIONS_ENABLED", "true")
		t.Setenv("WEEKLY_COVERAGE_REMINDER_ENABLED", "true")
		db, provider, svc, group, requester := setup(t)
		createWeeklyRequest(t, db, group.ID, requester.ID, day(24), 10, models.CoverageRequestOpen)

		sendWeeklyCoverageReminders(db, svc, time.Date(2026, 11, 22, 12, 0, 0, 0, loc), loc, 17)
		if len(provider.sends) != 0 {
			t.Fatalf("expected no email before the send hour, got %d", len(provider.sends))
		}
	})

	t.Run("does nothing while schedule email is off", func(t *testing.T) {
		t.Setenv("SCHEDULE_EMAIL_NOTIFICATIONS_ENABLED", "")
		t.Setenv("WEEKLY_COVERAGE_REMINDER_ENABLED", "true")
		db, provider, svc, group, requester := setup(t)
		createWeeklyRequest(t, db, group.ID, requester.ID, day(24), 10, models.CoverageRequestOpen)

		sendWeeklyCoverageReminders(db, svc, sundayEvening, loc, 17)
		if len(provider.sends) != 0 {
			t.Fatalf("expected no email with the flag off, got %d", len(provider.sends))
		}
		var claims int64
		db.Model(&models.SiteSetting{}).Where("key LIKE ?", weeklyReminderMarkerPrefix+"%").Count(&claims)
		if claims != 0 {
			t.Error("must not claim the week when nothing was sent, or enabling the flag later that evening would skip it")
		}
	})

	t.Run("does nothing while the weekly reminder flag is off", func(t *testing.T) {
		t.Setenv("SCHEDULE_EMAIL_NOTIFICATIONS_ENABLED", "true")
		t.Setenv("WEEKLY_COVERAGE_REMINDER_ENABLED", "")
		db, provider, svc, group, requester := setup(t)
		createWeeklyRequest(t, db, group.ID, requester.ID, day(24), 10, models.CoverageRequestOpen)

		sendWeeklyCoverageReminders(db, svc, sundayEvening, loc, 17)
		if len(provider.sends) != 0 {
			t.Fatalf("expected no email with WEEKLY_COVERAGE_REMINDER_ENABLED unset, got %d", len(provider.sends))
		}
		var claims int64
		db.Model(&models.SiteSetting{}).Where("key LIKE ?", weeklyReminderMarkerPrefix+"%").Count(&claims)
		if claims != 0 {
			t.Error("must not claim the week while the flag is off")
		}
	})

	t.Run("skips groups that haven't opted in", func(t *testing.T) {
		t.Setenv("SCHEDULE_EMAIL_NOTIFICATIONS_ENABLED", "true")
		t.Setenv("WEEKLY_COVERAGE_REMINDER_ENABLED", "true")
		db, provider, svc, group, requester := setup(t)
		createWeeklyRequest(t, db, group.ID, requester.ID, day(24), 10, models.CoverageRequestOpen)
		if err := db.Model(group).Update("weekly_coverage_reminder_enabled", false).Error; err != nil {
			t.Fatalf("opt group out: %v", err)
		}

		sendWeeklyCoverageReminders(db, svc, sundayEvening, loc, 17)
		if len(provider.sends) != 0 {
			t.Fatalf("expected no email for a group that hasn't opted in, got %d", len(provider.sends))
		}
	})

	t.Run("skips groups with scheduling disabled", func(t *testing.T) {
		t.Setenv("SCHEDULE_EMAIL_NOTIFICATIONS_ENABLED", "true")
		t.Setenv("WEEKLY_COVERAGE_REMINDER_ENABLED", "true")
		db, provider, svc, group, requester := setup(t)
		createWeeklyRequest(t, db, group.ID, requester.ID, day(24), 10, models.CoverageRequestOpen)
		if err := db.Model(group).Update("scheduling_enabled", false).Error; err != nil {
			t.Fatalf("disable scheduling: %v", err)
		}

		sendWeeklyCoverageReminders(db, svc, sundayEvening, loc, 17)
		if len(provider.sends) != 0 {
			t.Fatalf("expected no email for a group without scheduling, got %d", len(provider.sends))
		}
	})
}

func TestUpdateGroupWeeklyCoverageReminder(t *testing.T) {
	tests := []struct {
		name       string
		isAdmin    bool
		groupAdmin bool
		body       string
		wantStatus int
		wantStored bool
	}{
		{"site admin can opt a group in", true, false, `{"enabled":true}`, http.StatusOK, true},
		{"group admin cannot opt their group in", false, true, `{"enabled":true}`, http.StatusForbidden, false},
		{"missing enabled is rejected", true, false, `{}`, http.StatusBadRequest, false},
		{"invalid JSON is rejected", true, false, `{`, http.StatusBadRequest, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := SetupTestDB(t)
			user := CreateTestUser(t, db, "u", "u@test.com", "password123", tt.isAdmin)
			group := CreateTestGroup(t, db, "Dogs", "Dog volunteers")
			if tt.groupAdmin {
				AddUserToGroupWithAdmin(t, db, user.ID, group.ID, true)
			}

			c, w := setupGroupTestContext(user.ID, tt.isAdmin)
			c.Params = gin.Params{{Key: "id", Value: fmt.Sprintf("%d", group.ID)}}
			c.Request = httptest.NewRequest("PATCH", fmt.Sprintf("/api/groups/%d/weekly-coverage-reminder", group.ID), bytes.NewBufferString(tt.body))
			c.Request.Header.Set("Content-Type", "application/json")
			UpdateGroupWeeklyCoverageReminder(db)(c)

			if w.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d: %s", w.Code, tt.wantStatus, w.Body.String())
			}
			var reloaded models.Group
			if err := db.First(&reloaded, group.ID).Error; err != nil {
				t.Fatalf("reload: %v", err)
			}
			if reloaded.WeeklyCoverageReminderEnabled != tt.wantStored {
				t.Errorf("stored = %v, want %v", reloaded.WeeklyCoverageReminderEnabled, tt.wantStored)
			}
		})
	}
}

func TestGroupWeeklyCoverageReminderDefaultsOff(t *testing.T) {
	db := SetupTestDB(t)
	group := CreateTestGroup(t, db, "Dogs", "Dog volunteers")
	var reloaded models.Group
	if err := db.First(&reloaded, group.ID).Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	if reloaded.WeeklyCoverageReminderEnabled {
		t.Error("new groups must default to not receiving the weekly reminder")
	}
}

func TestUpdateGroupWeeklyCoverageReminder_EdgeCases(t *testing.T) {
	perform := func(db *gorm.DB, adminID uint, idParam, body string) *httptest.ResponseRecorder {
		c, w := setupGroupTestContext(adminID, true)
		c.Params = gin.Params{{Key: "id", Value: strings.ReplaceAll(idParam, "%20", " ")}}
		c.Request = httptest.NewRequest("PATCH", "/api/groups/"+idParam+"/weekly-coverage-reminder", bytes.NewBufferString(body))
		c.Request.Header.Set("Content-Type", "application/json")
		UpdateGroupWeeklyCoverageReminder(db)(c)
		return w
	}

	t.Run("site admin can opt a group back out", func(t *testing.T) {
		db := SetupTestDB(t)
		admin := CreateTestUser(t, db, "admin", "admin@test.com", "password123", true)
		group := CreateTestGroup(t, db, "Dogs", "Dog volunteers")
		if err := db.Model(group).Update("weekly_coverage_reminder_enabled", true).Error; err != nil {
			t.Fatalf("opt in: %v", err)
		}
		if w := perform(db, admin.ID, fmt.Sprintf("%d", group.ID), `{"enabled":false}`); w.Code != http.StatusOK {
			t.Fatalf("status = %d: %s", w.Code, w.Body.String())
		}
		var reloaded models.Group
		db.First(&reloaded, group.ID)
		if reloaded.WeeklyCoverageReminderEnabled {
			t.Error("expected the group to be opted out")
		}
	})

	t.Run("unknown group is 404", func(t *testing.T) {
		db := SetupTestDB(t)
		admin := CreateTestUser(t, db, "admin", "admin@test.com", "password123", true)
		if w := perform(db, admin.ID, "9999", `{"enabled":true}`); w.Code != http.StatusNotFound {
			t.Errorf("status = %d, want 404", w.Code)
		}
	})

	t.Run("non-numeric group id is 400", func(t *testing.T) {
		db := SetupTestDB(t)
		admin := CreateTestUser(t, db, "admin", "admin@test.com", "password123", true)
		if w := perform(db, admin.ID, "1%20OR%201=1", `{"enabled":true}`); w.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", w.Code)
		}
	})
}
