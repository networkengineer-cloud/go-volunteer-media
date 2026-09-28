package shelterclock

import (
	"testing"
	"time"

	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestParseZone(t *testing.T) {
	for name, ok := range map[string]bool{
		"America/Chicago":  true,
		"UTC":              true,
		" America/Denver ": true,
		"":                 false,
		"Local":            false,
		"Mars/Olympus":     false,
		"../../etc/passwd": false,
	} {
		_, err := ParseZone(name)
		if (err == nil) != ok {
			t.Errorf("ParseZone(%q) error = %v, want ok=%v", name, err, ok)
		}
	}
}

func TestDateOf(t *testing.T) {
	chicago, _ := ParseZone("America/Chicago")
	// 03:30 UTC on Sep 29 is 22:30 on Sep 28 in Chicago (CDT, UTC-5).
	instant := time.Date(2026, 9, 29, 3, 30, 0, 0, time.UTC)

	if got, want := DateOf(instant, chicago), time.Date(2026, 9, 28, 0, 0, 0, 0, time.UTC); !got.Equal(want) {
		t.Errorf("DateOf in Chicago = %s, want %s", got, want)
	}
	if got, want := DateOf(instant, time.UTC), time.Date(2026, 9, 29, 0, 0, 0, 0, time.UTC); !got.Equal(want) {
		t.Errorf("DateOf in UTC = %s, want %s", got, want)
	}
	if got := DateOf(instant, chicago); got.Location() != time.UTC {
		t.Errorf("DateOf must return UTC midnight to compare with stored dates, got %s", got.Location())
	}
}

func TestWeekStart(t *testing.T) {
	for in, want := range map[string]string{
		"2026-09-27": "2026-09-27", // Sunday
		"2026-09-28": "2026-09-27", // Monday
		"2026-10-03": "2026-09-27", // Saturday
	} {
		d, _ := time.Parse("2006-01-02", in)
		if got := WeekStart(d).Format("2006-01-02"); got != want {
			t.Errorf("WeekStart(%s) = %s, want %s", in, got, want)
		}
	}
}

func TestLocation(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.SiteSetting{}); err != nil {
		t.Fatal(err)
	}

	if got := Location(db); got != time.UTC {
		t.Errorf("unset: Location = %s, want UTC", got)
	}

	db.Create(&models.SiteSetting{Key: SettingKey, Value: "America/Chicago"})
	if got := Location(db).String(); got != "America/Chicago" {
		t.Errorf("set: Location = %s, want America/Chicago", got)
	}

	db.Model(&models.SiteSetting{}).Where("key = ?", SettingKey).Update("value", "Not/AZone")
	if got := Location(db); got != time.UTC {
		t.Errorf("invalid: Location = %s, want UTC fallback", got)
	}

	sqlDB, _ := db.DB()
	sqlDB.Close()
	if got := Location(db); got != time.UTC {
		t.Errorf("db error: Location = %s, want UTC fallback", got)
	}
}
