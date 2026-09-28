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
	"github.com/networkengineer-cloud/go-volunteer-media/internal/shelterclock"
)

// TestCreateCoverageRequest_UsesShelterDateNotUTC checks that "is this date
// in the past?" is decided on the shelter's calendar (AR-5). Before the
// shelter_timezone setting, a US shelter could not request coverage for
// that evening's shift once UTC had rolled over to tomorrow.
//
// At any instant at least one of these zones is on a different calendar
// date than UTC, so each run exercises at least one direction.
func TestCreateCoverageRequest_UsesShelterDateNotUTC(t *testing.T) {
	ran := false
	for _, zone := range []string{"Pacific/Pago_Pago", "Pacific/Kiritimati"} { // UTC-11, UTC+14
		loc, err := shelterclock.ParseZone(zone)
		if err != nil {
			t.Fatal(err)
		}
		shelterToday := shelterclock.Today(loc)
		utcToday := shelterclock.Today(time.UTC)
		if shelterToday.Equal(utcToday) {
			continue
		}
		ran = true

		t.Run(zone, func(t *testing.T) {
			db := SetupTestDB(t)
			requester := CreateTestUser(t, db, "requester", "requester@example.com", "password123", false)
			group := CreateTestGroup(t, db, "Dogs", "")
			AddUserToGroupWithAdmin(t, db, requester.ID, group.ID, false)
			db.Model(group).Update("scheduling_enabled", true)
			db.Create(&models.SiteSetting{Key: shelterclock.SettingKey, Value: zone})

			if shelterToday.Before(utcToday) {
				// Shelter is still on "yesterday" by UTC: its today is not past.
				db.Create(&models.ShiftSlot{UserID: requester.ID, GroupID: group.ID, DayOfWeek: int(shelterToday.Weekday()), Hour: 10})
				body := fmt.Sprintf(`{"date":"%s","hour":10}`, shelterToday.Format("2006-01-02"))
				if w := performCreateCoverageRequest(db, requester.ID, false, group.ID, body); w.Code != http.StatusCreated {
					t.Fatalf("coverage for the shelter's today: expected 201, got %d: %s", w.Code, w.Body.String())
				}
			} else {
				// Shelter is already on "tomorrow" by UTC: UTC's today is past.
				db.Create(&models.ShiftSlot{UserID: requester.ID, GroupID: group.ID, DayOfWeek: int(utcToday.Weekday()), Hour: 10})
				body := fmt.Sprintf(`{"date":"%s","hour":10}`, utcToday.Format("2006-01-02"))
				if w := performCreateCoverageRequest(db, requester.ID, false, group.ID, body); w.Code != http.StatusBadRequest {
					t.Fatalf("coverage for the shelter's yesterday: expected 400, got %d: %s", w.Code, w.Body.String())
				}
			}
		})
	}
	if !ran {
		t.Fatal("expected at least one zone to be on a different date than UTC")
	}
}

func TestUpdateSiteSetting_ValidatesShelterTimezone(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := SetupTestDB(t)
	cases := []struct {
		value string
		want  int
	}{
		{"America/Chicago", http.StatusOK},
		{"Mars/Olympus", http.StatusBadRequest},
		{"Local", http.StatusBadRequest},
		{"", http.StatusBadRequest},
		{" America/Denver ", http.StatusOK},
	}
	for _, tc := range cases {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodPut, "/", strings.NewReader(fmt.Sprintf(`{"value":%q}`, tc.value)))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Params = gin.Params{{Key: "key", Value: shelterclock.SettingKey}}
		UpdateSiteSetting(db)(c)
		if w.Code != tc.want {
			t.Errorf("value %q: expected %d, got %d: %s", tc.value, tc.want, w.Code, w.Body.String())
		}
	}

	var s models.SiteSetting
	db.Where("key = ?", shelterclock.SettingKey).First(&s)
	if s.Value != "America/Denver" {
		t.Errorf("stored value = %q, want the last valid value, trimmed", s.Value)
	}
}
