// Package shelterclock answers "what day is it at the shelter?" (roadmap
// item AR-5). The server runs in UTC (internal/database forces time.Local
// to UTC), but shifts, check-ins, "late" and "no-show" are wall-clock
// concepts at the shelter. Anything that decides which calendar day or
// week "now" falls in must go through this package, not time.Now().UTC().
//
// The zone is the "shelter_timezone" site setting (an IANA name such as
// "America/Chicago"), editable by site admins. Unset or invalid falls back
// to UTC, which is what the app did before the setting existed.
//
// Date-only values (ShiftCoverageRequest.Date and the like) are stored as
// UTC midnight of the calendar date. Today and WeekStart return values in
// that same form, so they compare directly against stored dates.
package shelterclock

import (
	"errors"
	"strings"
	"time"

	// Embed the IANA time zone database so zone lookups work regardless
	// of what the runtime image ships.
	_ "time/tzdata"

	"github.com/networkengineer-cloud/go-volunteer-media/internal/logging"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
	"gorm.io/gorm"
)

// SettingKey is the SiteSetting holding the shelter's IANA time zone.
const SettingKey = "shelter_timezone"

// DefaultZone is used when the setting is unset or invalid.
const DefaultZone = "UTC"

// ParseZone validates an IANA zone name for the setting. It rejects the
// empty string and "Local", whose meaning depends on the server.
func ParseZone(name string) (*time.Location, error) {
	name = strings.TrimSpace(name)
	if name == "" || name == "Local" {
		return nil, errors.New("time zone must be an IANA name such as America/Chicago")
	}
	loc, err := time.LoadLocation(name)
	if err != nil {
		return nil, errors.New("unknown time zone " + name + "; use an IANA name such as America/Chicago")
	}
	return loc, nil
}

// Location returns the shelter's configured zone, or UTC if the setting is
// missing or invalid (logged, so a bad value is noticed). Pass the
// request-scoped db (middleware.GetDB) so the lookup carries the request's
// context.
func Location(db *gorm.DB) *time.Location {
	var setting models.SiteSetting
	res := db.Where("key = ?", SettingKey).Limit(1).Find(&setting)
	if res.Error != nil {
		logging.WithField("error", res.Error.Error()).Warn("Failed to read shelter time zone; using UTC")
		return time.UTC
	}
	if res.RowsAffected == 0 || strings.TrimSpace(setting.Value) == "" {
		return time.UTC
	}
	loc, err := ParseZone(setting.Value)
	if err != nil {
		logging.WithField("value", setting.Value).Warn("Invalid shelter time zone setting; using UTC")
		return time.UTC
	}
	return loc
}

// DateOf returns the calendar date that instant t falls on in loc, as UTC
// midnight of that date.
func DateOf(t time.Time, loc *time.Location) time.Time {
	y, m, d := t.In(loc).Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// Today returns the shelter's current calendar date as UTC midnight.
func Today(loc *time.Location) time.Time {
	return DateOf(time.Now(), loc)
}

// WeekStart returns the Sunday on or before date (a UTC-midnight calendar
// date, as from Today or DateOf).
func WeekStart(date time.Time) time.Time {
	return date.AddDate(0, 0, -int(date.Weekday()))
}
