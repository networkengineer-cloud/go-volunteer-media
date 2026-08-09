package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
	"gorm.io/gorm"
)

// TestShiftSlotUniqueConstraint verifies the model's composite unique index
// rejects a duplicate (user_id, group_id, day_of_week, hour) row, and that a
// distinct slot (different hour) for the same user/group succeeds.
func TestShiftSlotUniqueConstraint(t *testing.T) {
	db := SetupTestDB(t)
	user := CreateTestUser(t, db, "vol1", "vol1@test.com", "password123", false)
	group := CreateTestGroup(t, db, "Dogs", "Dog volunteers")

	first := models.ShiftSlot{UserID: user.ID, GroupID: group.ID, DayOfWeek: 2, Hour: 9}
	if err := db.Create(&first).Error; err != nil {
		t.Fatalf("expected first slot to be created, got error: %v", err)
	}

	duplicate := models.ShiftSlot{UserID: user.ID, GroupID: group.ID, DayOfWeek: 2, Hour: 9}
	if err := db.Create(&duplicate).Error; err == nil {
		t.Fatal("expected duplicate (user_id, group_id, day_of_week, hour) to be rejected, got no error")
	}

	distinctHour := models.ShiftSlot{UserID: user.ID, GroupID: group.ID, DayOfWeek: 2, Hour: 10}
	if err := db.Create(&distinctHour).Error; err != nil {
		t.Fatalf("expected a slot with a different hour to be created, got error: %v", err)
	}
}

func TestGetMySchedule(t *testing.T) {
	tests := []struct {
		name           string
		setupFunc      func(*gorm.DB) (*models.User, *models.Group)
		isAdmin        bool
		expectedStatus int
		expectedCount  int
	}{
		{
			name: "member with no slots yet gets an empty list",
			setupFunc: func(db *gorm.DB) (*models.User, *models.Group) {
				user := CreateTestUser(t, db, "vol1", "vol1@test.com", "password123", false)
				group := CreateTestGroup(t, db, "Dogs", "Dog volunteers")
				AddUserToGroupWithAdmin(t, db, user.ID, group.ID, false)
				return user, group
			},
			isAdmin:        false,
			expectedStatus: http.StatusOK,
			expectedCount:  0,
		},
		{
			name: "member sees their own slots for this group only",
			setupFunc: func(db *gorm.DB) (*models.User, *models.Group) {
				user := CreateTestUser(t, db, "vol1", "vol1@test.com", "password123", false)
				group := CreateTestGroup(t, db, "Dogs", "Dog volunteers")
				otherGroup := CreateTestGroup(t, db, "Cats", "Cat volunteers")
				AddUserToGroupWithAdmin(t, db, user.ID, group.ID, false)
				AddUserToGroupWithAdmin(t, db, user.ID, otherGroup.ID, false)
				db.Create(&models.ShiftSlot{UserID: user.ID, GroupID: group.ID, DayOfWeek: 2, Hour: 9})
				db.Create(&models.ShiftSlot{UserID: user.ID, GroupID: otherGroup.ID, DayOfWeek: 3, Hour: 10})
				return user, group
			},
			isAdmin:        false,
			expectedStatus: http.StatusOK,
			expectedCount:  1,
		},
		{
			name: "non-member is denied",
			setupFunc: func(db *gorm.DB) (*models.User, *models.Group) {
				user := CreateTestUser(t, db, "vol1", "vol1@test.com", "password123", false)
				group := CreateTestGroup(t, db, "Dogs", "Dog volunteers")
				return user, group
			},
			isAdmin:        false,
			expectedStatus: http.StatusForbidden,
		},
		{
			name: "site admin (non-member) is allowed",
			setupFunc: func(db *gorm.DB) (*models.User, *models.Group) {
				admin := CreateTestUser(t, db, "admin", "admin@test.com", "password123", true)
				group := CreateTestGroup(t, db, "Dogs", "Dog volunteers")
				return admin, group
			},
			isAdmin:        true,
			expectedStatus: http.StatusOK,
			expectedCount:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := SetupTestDB(t)
			user, group := tt.setupFunc(db)

			c, w := setupGroupTestContext(user.ID, tt.isAdmin)
			c.Params = gin.Params{{Key: "id", Value: fmt.Sprintf("%d", group.ID)}}
			c.Request = httptest.NewRequest("GET", fmt.Sprintf("/api/groups/%d/schedule/me", group.ID), nil)

			handler := GetMySchedule(db)
			handler(c)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d: %s", tt.expectedStatus, w.Code, w.Body.String())
			}
			if tt.expectedStatus == http.StatusOK {
				var resp struct {
					Slots []scheduleSlotResponse `json:"slots"`
				}
				if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if len(resp.Slots) != tt.expectedCount {
					t.Errorf("expected %d slots, got %d", tt.expectedCount, len(resp.Slots))
				}
			}
		})
	}
}

func TestUpdateMySchedule(t *testing.T) {
	tests := []struct {
		name           string
		setupFunc      func(*gorm.DB) (*models.User, *models.Group)
		isAdmin        bool
		body           string
		expectedStatus int
		expectedError  string
	}{
		{
			name: "member can set their own schedule",
			setupFunc: func(db *gorm.DB) (*models.User, *models.Group) {
				user := CreateTestUser(t, db, "vol1", "vol1@test.com", "password123", false)
				group := CreateTestGroup(t, db, "Dogs", "Dog volunteers")
				AddUserToGroupWithAdmin(t, db, user.ID, group.ID, false)
				return user, group
			},
			isAdmin:        false,
			body:           `{"slots":[{"day_of_week":2,"hour":9},{"day_of_week":2,"hour":10}]}`,
			expectedStatus: http.StatusOK,
		},
		{
			name: "replace-all semantics: a second update replaces the first set",
			setupFunc: func(db *gorm.DB) (*models.User, *models.Group) {
				user := CreateTestUser(t, db, "vol1", "vol1@test.com", "password123", false)
				group := CreateTestGroup(t, db, "Dogs", "Dog volunteers")
				AddUserToGroupWithAdmin(t, db, user.ID, group.ID, false)
				db.Create(&models.ShiftSlot{UserID: user.ID, GroupID: group.ID, DayOfWeek: 0, Hour: 8})
				return user, group
			},
			isAdmin:        false,
			body:           `{"slots":[{"day_of_week":5,"hour":16}]}`,
			expectedStatus: http.StatusOK,
		},
		{
			name: "invalid day_of_week is rejected",
			setupFunc: func(db *gorm.DB) (*models.User, *models.Group) {
				user := CreateTestUser(t, db, "vol1", "vol1@test.com", "password123", false)
				group := CreateTestGroup(t, db, "Dogs", "Dog volunteers")
				AddUserToGroupWithAdmin(t, db, user.ID, group.ID, false)
				return user, group
			},
			isAdmin:        false,
			body:           `{"slots":[{"day_of_week":7,"hour":9}]}`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "day_of_week",
		},
		{
			name: "invalid hour is rejected",
			setupFunc: func(db *gorm.DB) (*models.User, *models.Group) {
				user := CreateTestUser(t, db, "vol1", "vol1@test.com", "password123", false)
				group := CreateTestGroup(t, db, "Dogs", "Dog volunteers")
				AddUserToGroupWithAdmin(t, db, user.ID, group.ID, false)
				return user, group
			},
			isAdmin:        false,
			body:           `{"slots":[{"day_of_week":2,"hour":18}]}`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "hour",
		},
		{
			name: "duplicate slot in payload is rejected",
			setupFunc: func(db *gorm.DB) (*models.User, *models.Group) {
				user := CreateTestUser(t, db, "vol1", "vol1@test.com", "password123", false)
				group := CreateTestGroup(t, db, "Dogs", "Dog volunteers")
				AddUserToGroupWithAdmin(t, db, user.ID, group.ID, false)
				return user, group
			},
			isAdmin:        false,
			body:           `{"slots":[{"day_of_week":2,"hour":9},{"day_of_week":2,"hour":9}]}`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "duplicate",
		},
		{
			name: "non-member is denied",
			setupFunc: func(db *gorm.DB) (*models.User, *models.Group) {
				user := CreateTestUser(t, db, "vol1", "vol1@test.com", "password123", false)
				group := CreateTestGroup(t, db, "Dogs", "Dog volunteers")
				return user, group
			},
			isAdmin:        false,
			body:           `{"slots":[]}`,
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := SetupTestDB(t)
			user, group := tt.setupFunc(db)

			c, w := setupGroupTestContext(user.ID, tt.isAdmin)
			c.Params = gin.Params{{Key: "id", Value: fmt.Sprintf("%d", group.ID)}}
			c.Request = httptest.NewRequest("PUT", fmt.Sprintf("/api/groups/%d/schedule/me", group.ID), bytes.NewBufferString(tt.body))
			c.Request.Header.Set("Content-Type", "application/json")

			handler := UpdateMySchedule(db)
			handler(c)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d: %s", tt.expectedStatus, w.Code, w.Body.String())
			}
			if tt.expectedError != "" && !strings.Contains(w.Body.String(), tt.expectedError) {
				t.Errorf("expected error containing %q, got %q", tt.expectedError, w.Body.String())
			}
			if tt.name == "replace-all semantics: a second update replaces the first set" {
				var count int64
				db.Model(&models.ShiftSlot{}).Where("user_id = ? AND group_id = ?", user.ID, group.ID).Count(&count)
				if count != 1 {
					t.Errorf("expected exactly 1 slot after replace, got %d", count)
				}
				var slot models.ShiftSlot
				if err := db.Where("user_id = ? AND group_id = ?", user.ID, group.ID).First(&slot).Error; err != nil {
					t.Fatalf("failed to load remaining slot: %v", err)
				}
				if slot.DayOfWeek != 5 || slot.Hour != 16 {
					t.Errorf("expected remaining slot to be day 5 hour 16, got day %d hour %d", slot.DayOfWeek, slot.Hour)
				}
			}
		})
	}
}
