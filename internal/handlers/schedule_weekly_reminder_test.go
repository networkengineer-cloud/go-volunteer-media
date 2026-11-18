package handlers

import (
	"strings"
	"testing"
	"time"

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
	req := &models.ShiftCoverageRequest{GroupID: groupID, RequestedByUserID: userID, Date: date, Hour: hour, Status: status}
	if err := db.Create(req).Error; err != nil {
		t.Fatalf("create request: %v", err)
	}
	return req
}

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
		provider := &recordingEmailProvider{}
		return db, provider, email.NewServiceWithProvider(provider, db), group, requester
	}

	t.Run("emails the coming week's open requests once", func(t *testing.T) {
		t.Setenv("SCHEDULE_EMAIL_NOTIFICATIONS_ENABLED", "true")
		db, provider, svc, group, requester := setup(t)

		inWeek := createWeeklyRequest(t, db, group.ID, requester.ID, day(24), 10, models.CoverageRequestOpen)
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
		if !strings.Contains(sent.subject, "Dogs") {
			t.Errorf("subject %q should name the group", sent.subject)
		}
		if !strings.Contains(sent.body, "Tuesday, November 24") {
			t.Errorf("body should list the in-week shift:\n%s", sent.body)
		}
		for _, excluded := range []string{"November 22", "November 30", "November 25"} {
			if strings.Contains(sent.body, excluded) {
				t.Errorf("body should not mention %s:\n%s", excluded, sent.body)
			}
		}

		var stamped models.ShiftCoverageRequest
		db.First(&stamped, inWeek.ID)
		if stamped.NotifiedAt == nil {
			t.Error("announced request should be stamped so the digest sweep doesn't repeat it")
		}
	})

	t.Run("does nothing outside the Sunday window", func(t *testing.T) {
		t.Setenv("SCHEDULE_EMAIL_NOTIFICATIONS_ENABLED", "true")
		db, provider, svc, group, requester := setup(t)
		createWeeklyRequest(t, db, group.ID, requester.ID, day(24), 10, models.CoverageRequestOpen)

		sendWeeklyCoverageReminders(db, svc, time.Date(2026, 11, 22, 12, 0, 0, 0, loc), loc, 17)
		if len(provider.sends) != 0 {
			t.Fatalf("expected no email before the send hour, got %d", len(provider.sends))
		}
	})

	t.Run("does nothing while schedule email is off", func(t *testing.T) {
		t.Setenv("SCHEDULE_EMAIL_NOTIFICATIONS_ENABLED", "")
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

	t.Run("skips groups with scheduling disabled", func(t *testing.T) {
		t.Setenv("SCHEDULE_EMAIL_NOTIFICATIONS_ENABLED", "true")
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
