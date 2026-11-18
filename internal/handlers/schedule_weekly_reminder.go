package handlers

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	// Embeds the IANA time zone database so the shelter time zone resolves
	// even on a runtime image without /usr/share/zoneinfo.
	_ "time/tzdata"

	"github.com/networkengineer-cloud/go-volunteer-media/internal/email"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/logging"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// The weekly coverage reminder emails every group that has opted in
// (Group.WeeklyCoverageReminderEnabled, set in Manage Groups; off by default)
// and has scheduling enabled, on
// Sunday afternoon shelter time, the coverage requests still open for the
// coming Monday-Sunday. It is the automatic counterpart to the admin "Send
// reminder" button (SendCoverageReminder): the digest sweep announces a
// request once, when it is created, and nothing re-announces it after that
// unless an admin remembers to - this does, once a week, for the week ahead.
//
// Email only. Three gates, all required: the email service must be
// configured, SCHEDULE_EMAIL_NOTIFICATIONS_ENABLED must be on (as for every
// schedule email), and WEEKLY_COVERAGE_REMINDER_ENABLED must be on, so the
// weekly send can be turned off without silencing the rest of schedule email.

const (
	// defaultShelterTimeZone is used when SHELTER_TIMEZONE is unset or invalid.
	// The rest of the app runs in UTC (see database.init); this is the one
	// place that needs to know what "Sunday 5 PM" means locally.
	defaultShelterTimeZone = "America/Chicago"

	// defaultWeeklyReminderHour is the local hour on Sunday at or after which
	// the reminder goes out. WEEKLY_COVERAGE_REMINDER_HOUR overrides it.
	defaultWeeklyReminderHour = 17

	// weeklyReminderMarkerPrefix keys the per-group SiteSetting row recording
	// which week's reminder that group last received. One row per group,
	// overwritten weekly, rather than one row per week, so the table doesn't
	// grow forever.
	weeklyReminderMarkerPrefix = models.InternalSettingPrefix + "weekly_coverage_reminder.group."

	weeklyReminderStopTimeout = 10 * time.Second
)

// weeklyCoverageReminderEnabled reports whether WEEKLY_COVERAGE_REMINDER_ENABLED
// is on. Opt-in, like the other feature flags: unset means off.
func weeklyCoverageReminderEnabled() bool {
	v := os.Getenv("WEEKLY_COVERAGE_REMINDER_ENABLED")
	return v == "true" || v == "1"
}

// shelterLocation returns the shelter's time zone from SHELTER_TIMEZONE,
// falling back to defaultShelterTimeZone.
func shelterLocation() *time.Location {
	name := os.Getenv("SHELTER_TIMEZONE")
	if name == "" {
		name = defaultShelterTimeZone
	}
	loc, err := time.LoadLocation(name)
	if err != nil {
		logging.Warn(fmt.Sprintf("Invalid SHELTER_TIMEZONE %q; using %s", name, defaultShelterTimeZone))
		loc, _ = time.LoadLocation(defaultShelterTimeZone)
	}
	return loc
}

// weeklyReminderHour returns the local send hour on Sunday (0-23).
func weeklyReminderHour() int {
	if v := os.Getenv("WEEKLY_COVERAGE_REMINDER_HOUR"); v != "" {
		if h, err := strconv.Atoi(v); err == nil && h >= 0 && h <= 23 {
			return h
		}
		logging.Warn(fmt.Sprintf("Invalid WEEKLY_COVERAGE_REMINDER_HOUR %q; using %d", v, defaultWeeklyReminderHour))
	}
	return defaultWeeklyReminderHour
}

// weeklyReminderWindow reports whether now falls in the send window - Sunday,
// from the send hour until local midnight - and if so returns the week key
// (the Sunday's local date) and the coming week's date range [start, end) as
// UTC midnights, which is how ShiftCoverageRequest.Date stores calendar dates.
//
// The window runs to midnight rather than being a single tick so a deploy or
// restart that straddles the send hour still sends, just late.
func weeklyReminderWindow(now time.Time, loc *time.Location, sendHour int) (weekKey string, start, end time.Time, ok bool) {
	local := now.In(loc)
	if local.Weekday() != time.Sunday || local.Hour() < sendHour {
		return "", time.Time{}, time.Time{}, false
	}
	sunday := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, time.UTC)
	return sunday.Format("2006-01-02"), sunday.AddDate(0, 0, 1), sunday.AddDate(0, 0, 8), true
}

// claimWeeklyReminder records that groupID's reminder for weekKey is being
// sent and reports whether this caller won it. Prod runs several replicas,
// each with its own ticker; the marker row is the mutex. The insert wins the
// very first week; after that, the conditional update wins only if the row
// still names an older week, and a concurrent second replica's identical
// update re-evaluates its WHERE after the first commits and matches nothing.
//
// Claimed before sending, like the coverage digest: a crash in between drops
// one weekly reminder rather than duplicating it.
func claimWeeklyReminder(db *gorm.DB, groupID uint, weekKey string) (bool, error) {
	key := weeklyReminderMarkerPrefix + strconv.FormatUint(uint64(groupID), 10)
	created := db.Clauses(clause.OnConflict{DoNothing: true}).
		Create(&models.SiteSetting{Key: key, Value: weekKey})
	if created.Error != nil {
		return false, created.Error
	}
	if created.RowsAffected > 0 {
		return true, nil
	}
	updated := db.Model(&models.SiteSetting{}).
		Where("key = ? AND value <> ?", key, weekKey).
		Update("value", weekKey)
	if updated.Error != nil {
		return false, updated.Error
	}
	return updated.RowsAffected > 0, nil
}

// sendWeeklyCoverageReminders runs one pass. It is a no-op outside the
// Sunday window, when the flags are off, or for a group that already had
// this week's turn.
//
// Every opted-in group is claimed on the first tick of the window, whether
// or not it has anything open, and a group with nothing to list simply gets
// no email. Claiming only groups that have open requests would instead let a
// request filed later that evening trigger the "weekly" email at, say,
// 11:55 PM, taking over that request's normal announcement.
func sendWeeklyCoverageReminders(db *gorm.DB, emailService *email.Service, now time.Time, loc *time.Location, sendHour int) {
	if !weeklyCoverageReminderEnabled() || emailService == nil || !emailService.IsConfigured() || !scheduleEmailNotificationsEnabled() {
		return
	}
	weekKey, start, end, ok := weeklyReminderWindow(now, loc, sendHour)
	if !ok {
		return
	}

	var groups []models.Group
	if err := db.Select("id", "name").
		Where("scheduling_enabled = ? AND weekly_coverage_reminder_enabled = ?", true, true).
		Order("id").
		Find(&groups).Error; err != nil {
		logging.Error("Failed to load groups for weekly coverage reminder", err)
		return
	}

	for _, grp := range groups {
		log := logging.WithField("group_id", grp.ID)

		won, err := claimWeeklyReminder(db, grp.ID, weekKey)
		if err != nil {
			log.Error("Failed to claim weekly coverage reminder", err)
			continue
		}
		if !won {
			continue // already handled this week, here or on another replica
		}

		// Only requests the group has already been told about. One still
		// waiting out the coverage digest's quiet period (notified_at NULL)
		// is left to the digest, which announces it within minutes as a new
		// request - listing it here too would either announce it twice or,
		// if stamped, rob it of that announcement. Nothing here is stamped.
		var requests []models.ShiftCoverageRequest
		if err := db.Preload("RequestedByUser").
			Where("group_id = ? AND status = ? AND notified_at IS NOT NULL AND date >= ? AND date < ?",
				grp.ID, models.CoverageRequestOpen, start, end).
			Order("requested_by_user_id, date, hour").
			Find(&requests).Error; err != nil {
			log.Error("Failed to load open coverage requests for weekly reminder; this group's reminder is skipped this week", err)
			continue
		}
		if len(requests) == 0 {
			continue
		}

		title := fmt.Sprintf("This week: coverage still needed in %s", grp.Name)
		content := fmt.Sprintf("These shifts for the week of %s still need coverage. Open the Schedule tab in the portal to pick one up.\n\n%s",
			start.Format("Monday, January 2"), buildMultiRequesterCoverageContent(requests, nil))

		// Synchronous, like the digest sweep: the stop function waits for the
		// tick, so shutdown doesn't close the DB pool under an in-flight send.
		sent, total, err := sendGroupAnnouncementEmailsCounted(context.Background(), db, emailService, grp.ID, title, content)
		if err != nil {
			log.Error("Error sending weekly coverage reminder emails; this group's reminder is lost for this week", err)
			continue
		}
		if total > 0 && sent == 0 {
			log.Errorf("Weekly coverage reminder failed for all %d recipient(s); this group's reminder is lost for this week", total)
			continue
		}
		log.Info(fmt.Sprintf("Sent weekly coverage reminder covering %d open request(s) to %d of %d recipient(s)", len(requests), sent, total))
	}
}

// StartWeeklyCoverageReminder starts the background loop that sends the
// Sunday coverage reminder. Safe with multiple replicas: each group's send is
// claimed in the database first. Returns a stop function for graceful
// shutdown that waits (bounded) for an in-flight pass.
func StartWeeklyCoverageReminder(db *gorm.DB, emailService *email.Service, interval time.Duration) (stop func()) {
	loc := shelterLocation()
	sendHour := weeklyReminderHour()

	ticker := time.NewTicker(interval)
	done := make(chan struct{})
	finished := make(chan struct{})

	go func() {
		defer close(finished)
		for {
			select {
			case now := <-ticker.C:
				sendWeeklyCoverageReminders(db, emailService, now, loc, sendHour)
			case <-done:
				ticker.Stop()
				return
			}
		}
	}()

	var once sync.Once
	return func() {
		once.Do(func() { close(done) })
		select {
		case <-finished:
		case <-time.After(weeklyReminderStopTimeout):
			logging.Warn(fmt.Sprintf("Weekly coverage reminder did not stop within %s of shutdown signal; proceeding with shutdown anyway", weeklyReminderStopTimeout))
		}
	}
}
