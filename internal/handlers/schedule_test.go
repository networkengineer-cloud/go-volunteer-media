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

// createSchedulingEnabledGroup creates a test group with the scheduling
// feature turned on, since SchedulingEnabled defaults to false and most
// schedule-endpoint tests are exercising auth/validation behavior, not the
// feature-gate itself.
func createSchedulingEnabledGroup(t *testing.T, db *gorm.DB, name, description string) *models.Group {
	group := CreateTestGroup(t, db, name, description)
	if err := db.Model(group).Update("scheduling_enabled", true).Error; err != nil {
		t.Fatalf("failed to enable scheduling for test group: %v", err)
	}
	return group
}

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
				group := createSchedulingEnabledGroup(t, db, "Dogs", "Dog volunteers")
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
				group := createSchedulingEnabledGroup(t, db, "Dogs", "Dog volunteers")
				otherGroup := createSchedulingEnabledGroup(t, db, "Cats", "Cat volunteers")
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
				group := createSchedulingEnabledGroup(t, db, "Dogs", "Dog volunteers")
				return user, group
			},
			isAdmin:        false,
			expectedStatus: http.StatusForbidden,
		},
		{
			name: "site admin (non-member) is allowed",
			setupFunc: func(db *gorm.DB) (*models.User, *models.Group) {
				admin := CreateTestUser(t, db, "admin", "admin@test.com", "password123", true)
				group := createSchedulingEnabledGroup(t, db, "Dogs", "Dog volunteers")
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
				group := createSchedulingEnabledGroup(t, db, "Dogs", "Dog volunteers")
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
				group := createSchedulingEnabledGroup(t, db, "Dogs", "Dog volunteers")
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
				group := createSchedulingEnabledGroup(t, db, "Dogs", "Dog volunteers")
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
				group := createSchedulingEnabledGroup(t, db, "Dogs", "Dog volunteers")
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
				group := createSchedulingEnabledGroup(t, db, "Dogs", "Dog volunteers")
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
				group := createSchedulingEnabledGroup(t, db, "Dogs", "Dog volunteers")
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

func TestGetMemberSchedule(t *testing.T) {
	tests := []struct {
		name           string
		setupFunc      func(*gorm.DB) (contextUser, targetUser *models.User, group *models.Group)
		isAdmin        bool
		expectedStatus int
		expectedCount  int
	}{
		{
			name: "group admin can view another member's schedule",
			setupFunc: func(db *gorm.DB) (*models.User, *models.User, *models.Group) {
				groupAdmin := CreateTestUser(t, db, "gadmin", "gadmin@test.com", "password123", false)
				member := CreateTestUser(t, db, "vol1", "vol1@test.com", "password123", false)
				group := createSchedulingEnabledGroup(t, db, "Dogs", "Dog volunteers")
				AddUserToGroupWithAdmin(t, db, groupAdmin.ID, group.ID, true)
				AddUserToGroupWithAdmin(t, db, member.ID, group.ID, false)
				db.Create(&models.ShiftSlot{UserID: member.ID, GroupID: group.ID, DayOfWeek: 1, Hour: 8})
				return groupAdmin, member, group
			},
			isAdmin:        false,
			expectedStatus: http.StatusOK,
			expectedCount:  1,
		},
		{
			name: "site admin can view any member's schedule",
			setupFunc: func(db *gorm.DB) (*models.User, *models.User, *models.Group) {
				siteAdmin := CreateTestUser(t, db, "admin", "admin@test.com", "password123", true)
				member := CreateTestUser(t, db, "vol1", "vol1@test.com", "password123", false)
				group := createSchedulingEnabledGroup(t, db, "Dogs", "Dog volunteers")
				AddUserToGroupWithAdmin(t, db, member.ID, group.ID, false)
				return siteAdmin, member, group
			},
			isAdmin:        true,
			expectedStatus: http.StatusOK,
			expectedCount:  0,
		},
		{
			name: "regular member is denied viewing another member's schedule",
			setupFunc: func(db *gorm.DB) (*models.User, *models.User, *models.Group) {
				regular := CreateTestUser(t, db, "regular", "regular@test.com", "password123", false)
				member := CreateTestUser(t, db, "vol1", "vol1@test.com", "password123", false)
				group := createSchedulingEnabledGroup(t, db, "Dogs", "Dog volunteers")
				AddUserToGroupWithAdmin(t, db, regular.ID, group.ID, false)
				AddUserToGroupWithAdmin(t, db, member.ID, group.ID, false)
				return regular, member, group
			},
			isAdmin:        false,
			expectedStatus: http.StatusForbidden,
		},
		{
			name: "group admin viewing a non-member's schedule gets 404",
			setupFunc: func(db *gorm.DB) (*models.User, *models.User, *models.Group) {
				groupAdmin := CreateTestUser(t, db, "gadmin", "gadmin@test.com", "password123", false)
				nonMember := CreateTestUser(t, db, "outsider", "outsider@test.com", "password123", false)
				group := createSchedulingEnabledGroup(t, db, "Dogs", "Dog volunteers")
				AddUserToGroupWithAdmin(t, db, groupAdmin.ID, group.ID, true)
				return groupAdmin, nonMember, group
			},
			isAdmin:        false,
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := SetupTestDB(t)
			contextUser, targetUser, group := tt.setupFunc(db)

			c, w := setupGroupTestContext(contextUser.ID, tt.isAdmin)
			c.Params = gin.Params{
				{Key: "id", Value: fmt.Sprintf("%d", group.ID)},
				{Key: "userId", Value: fmt.Sprintf("%d", targetUser.ID)},
			}
			c.Request = httptest.NewRequest("GET", fmt.Sprintf("/api/groups/%d/schedule/%d", group.ID, targetUser.ID), nil)

			handler := GetMemberSchedule(db)
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

func TestUpdateMemberSchedule(t *testing.T) {
	tests := []struct {
		name           string
		setupFunc      func(*gorm.DB) (contextUser, targetUser *models.User, group *models.Group)
		isAdmin        bool
		body           string
		expectedStatus int
		expectedError  string
	}{
		{
			name: "group admin can set another member's schedule",
			setupFunc: func(db *gorm.DB) (*models.User, *models.User, *models.Group) {
				groupAdmin := CreateTestUser(t, db, "gadmin", "gadmin@test.com", "password123", false)
				member := CreateTestUser(t, db, "vol1", "vol1@test.com", "password123", false)
				group := createSchedulingEnabledGroup(t, db, "Dogs", "Dog volunteers")
				AddUserToGroupWithAdmin(t, db, groupAdmin.ID, group.ID, true)
				AddUserToGroupWithAdmin(t, db, member.ID, group.ID, false)
				return groupAdmin, member, group
			},
			isAdmin:        false,
			body:           `{"slots":[{"day_of_week":4,"hour":12}]}`,
			expectedStatus: http.StatusOK,
		},
		{
			name: "regular member is denied editing another member's schedule",
			setupFunc: func(db *gorm.DB) (*models.User, *models.User, *models.Group) {
				regular := CreateTestUser(t, db, "regular", "regular@test.com", "password123", false)
				member := CreateTestUser(t, db, "vol1", "vol1@test.com", "password123", false)
				group := createSchedulingEnabledGroup(t, db, "Dogs", "Dog volunteers")
				AddUserToGroupWithAdmin(t, db, regular.ID, group.ID, false)
				AddUserToGroupWithAdmin(t, db, member.ID, group.ID, false)
				return regular, member, group
			},
			isAdmin:        false,
			body:           `{"slots":[]}`,
			expectedStatus: http.StatusForbidden,
			expectedError:  "Admin access required",
		},
		{
			name: "group admin editing a non-member gets 404",
			setupFunc: func(db *gorm.DB) (*models.User, *models.User, *models.Group) {
				groupAdmin := CreateTestUser(t, db, "gadmin", "gadmin@test.com", "password123", false)
				nonMember := CreateTestUser(t, db, "outsider", "outsider@test.com", "password123", false)
				group := createSchedulingEnabledGroup(t, db, "Dogs", "Dog volunteers")
				AddUserToGroupWithAdmin(t, db, groupAdmin.ID, group.ID, true)
				return groupAdmin, nonMember, group
			},
			isAdmin:        false,
			body:           `{"slots":[]}`,
			expectedStatus: http.StatusNotFound,
		},
		{
			name: "invalid slot payload is rejected",
			setupFunc: func(db *gorm.DB) (*models.User, *models.User, *models.Group) {
				groupAdmin := CreateTestUser(t, db, "gadmin", "gadmin@test.com", "password123", false)
				member := CreateTestUser(t, db, "vol1", "vol1@test.com", "password123", false)
				group := createSchedulingEnabledGroup(t, db, "Dogs", "Dog volunteers")
				AddUserToGroupWithAdmin(t, db, groupAdmin.ID, group.ID, true)
				AddUserToGroupWithAdmin(t, db, member.ID, group.ID, false)
				return groupAdmin, member, group
			},
			isAdmin:        false,
			body:           `{"slots":[{"day_of_week":2,"hour":25}]}`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "hour",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := SetupTestDB(t)
			contextUser, targetUser, group := tt.setupFunc(db)

			c, w := setupGroupTestContext(contextUser.ID, tt.isAdmin)
			c.Params = gin.Params{
				{Key: "id", Value: fmt.Sprintf("%d", group.ID)},
				{Key: "userId", Value: fmt.Sprintf("%d", targetUser.ID)},
			}
			c.Request = httptest.NewRequest("PUT", fmt.Sprintf("/api/groups/%d/schedule/%d", group.ID, targetUser.ID), bytes.NewBufferString(tt.body))
			c.Request.Header.Set("Content-Type", "application/json")

			handler := UpdateMemberSchedule(db)
			handler(c)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d: %s", tt.expectedStatus, w.Code, w.Body.String())
			}
			if tt.expectedError != "" && !strings.Contains(w.Body.String(), tt.expectedError) {
				t.Errorf("expected error containing %q, got %q", tt.expectedError, w.Body.String())
			}
		})
	}
}

// TestScheduleRequiresSchedulingEnabled verifies all four schedule endpoints
// return 404 for a group where SchedulingEnabled is false (the default),
// even for a site admin — this is a feature gate, not an authorization gate.
func TestScheduleRequiresSchedulingEnabled(t *testing.T) {
	t.Run("GetMySchedule returns 404 when scheduling is disabled for the group", func(t *testing.T) {
		db := SetupTestDB(t)
		user := CreateTestUser(t, db, "vol1", "vol1@test.com", "password123", false)
		group := CreateTestGroup(t, db, "Dogs", "Dog volunteers")
		AddUserToGroupWithAdmin(t, db, user.ID, group.ID, false)

		c, w := setupGroupTestContext(user.ID, false)
		c.Params = gin.Params{{Key: "id", Value: fmt.Sprintf("%d", group.ID)}}
		c.Request = httptest.NewRequest("GET", fmt.Sprintf("/api/groups/%d/schedule/me", group.ID), nil)

		GetMySchedule(db)(c)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected 404, got %d: %s", w.Code, w.Body.String())
		}
		if !strings.Contains(w.Body.String(), "Scheduling") {
			t.Errorf("expected error message to mention scheduling, got %q", w.Body.String())
		}
	})

	t.Run("UpdateMySchedule returns 404 when scheduling is disabled for the group", func(t *testing.T) {
		db := SetupTestDB(t)
		user := CreateTestUser(t, db, "vol1", "vol1@test.com", "password123", false)
		group := CreateTestGroup(t, db, "Dogs", "Dog volunteers")
		AddUserToGroupWithAdmin(t, db, user.ID, group.ID, false)

		c, w := setupGroupTestContext(user.ID, false)
		c.Params = gin.Params{{Key: "id", Value: fmt.Sprintf("%d", group.ID)}}
		c.Request = httptest.NewRequest("PUT", fmt.Sprintf("/api/groups/%d/schedule/me", group.ID), bytes.NewBufferString(`{"slots":[]}`))
		c.Request.Header.Set("Content-Type", "application/json")

		UpdateMySchedule(db)(c)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected 404, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("GetMemberSchedule returns 404 when scheduling is disabled for the group", func(t *testing.T) {
		db := SetupTestDB(t)
		groupAdmin := CreateTestUser(t, db, "gadmin", "gadmin@test.com", "password123", false)
		member := CreateTestUser(t, db, "vol1", "vol1@test.com", "password123", false)
		group := CreateTestGroup(t, db, "Dogs", "Dog volunteers")
		AddUserToGroupWithAdmin(t, db, groupAdmin.ID, group.ID, true)
		AddUserToGroupWithAdmin(t, db, member.ID, group.ID, false)

		c, w := setupGroupTestContext(groupAdmin.ID, false)
		c.Params = gin.Params{
			{Key: "id", Value: fmt.Sprintf("%d", group.ID)},
			{Key: "userId", Value: fmt.Sprintf("%d", member.ID)},
		}
		c.Request = httptest.NewRequest("GET", fmt.Sprintf("/api/groups/%d/schedule/%d", group.ID, member.ID), nil)

		GetMemberSchedule(db)(c)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected 404, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("UpdateMemberSchedule returns 404 when scheduling is disabled for the group", func(t *testing.T) {
		db := SetupTestDB(t)
		groupAdmin := CreateTestUser(t, db, "gadmin", "gadmin@test.com", "password123", false)
		member := CreateTestUser(t, db, "vol1", "vol1@test.com", "password123", false)
		group := CreateTestGroup(t, db, "Dogs", "Dog volunteers")
		AddUserToGroupWithAdmin(t, db, groupAdmin.ID, group.ID, true)
		AddUserToGroupWithAdmin(t, db, member.ID, group.ID, false)

		c, w := setupGroupTestContext(groupAdmin.ID, false)
		c.Params = gin.Params{
			{Key: "id", Value: fmt.Sprintf("%d", group.ID)},
			{Key: "userId", Value: fmt.Sprintf("%d", member.ID)},
		}
		c.Request = httptest.NewRequest("PUT", fmt.Sprintf("/api/groups/%d/schedule/%d", group.ID, member.ID), bytes.NewBufferString(`{"slots":[]}`))
		c.Request.Header.Set("Content-Type", "application/json")

		UpdateMemberSchedule(db)(c)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected 404, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("site admin still gets 404 when scheduling is disabled (feature gate applies to everyone)", func(t *testing.T) {
		db := SetupTestDB(t)
		siteAdmin := CreateTestUser(t, db, "admin", "admin@test.com", "password123", true)
		group := CreateTestGroup(t, db, "Dogs", "Dog volunteers")

		c, w := setupGroupTestContext(siteAdmin.ID, true)
		c.Params = gin.Params{{Key: "id", Value: fmt.Sprintf("%d", group.ID)}}
		c.Request = httptest.NewRequest("GET", fmt.Sprintf("/api/groups/%d/schedule/me", group.ID), nil)

		GetMySchedule(db)(c)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected 404, got %d: %s", w.Code, w.Body.String())
		}
	})
}

func TestUpdateGroupScheduling(t *testing.T) {
	tests := []struct {
		name            string
		setupFunc       func(*gorm.DB) (*models.User, *models.Group)
		isAdmin         bool
		body            string
		expectedStatus  int
		expectedEnabled bool
		expectedError   string
	}{
		{
			name: "group admin can enable scheduling",
			setupFunc: func(db *gorm.DB) (*models.User, *models.Group) {
				groupAdmin := CreateTestUser(t, db, "gadmin", "gadmin@test.com", "password123", false)
				group := CreateTestGroup(t, db, "Dogs", "Dog volunteers")
				AddUserToGroupWithAdmin(t, db, groupAdmin.ID, group.ID, true)
				return groupAdmin, group
			},
			isAdmin:         false,
			body:            `{"enabled":true}`,
			expectedStatus:  http.StatusOK,
			expectedEnabled: true,
		},
		{
			name: "group admin can disable scheduling",
			setupFunc: func(db *gorm.DB) (*models.User, *models.Group) {
				groupAdmin := CreateTestUser(t, db, "gadmin", "gadmin@test.com", "password123", false)
				group := createSchedulingEnabledGroup(t, db, "Dogs", "Dog volunteers")
				AddUserToGroupWithAdmin(t, db, groupAdmin.ID, group.ID, true)
				return groupAdmin, group
			},
			isAdmin:         false,
			body:            `{"enabled":false}`,
			expectedStatus:  http.StatusOK,
			expectedEnabled: false,
		},
		{
			name: "site admin can enable scheduling for a group they're not a member of",
			setupFunc: func(db *gorm.DB) (*models.User, *models.Group) {
				siteAdmin := CreateTestUser(t, db, "admin", "admin@test.com", "password123", true)
				group := CreateTestGroup(t, db, "Dogs", "Dog volunteers")
				return siteAdmin, group
			},
			isAdmin:         true,
			body:            `{"enabled":true}`,
			expectedStatus:  http.StatusOK,
			expectedEnabled: true,
		},
		{
			name: "regular member cannot toggle scheduling",
			setupFunc: func(db *gorm.DB) (*models.User, *models.Group) {
				regular := CreateTestUser(t, db, "regular", "regular@test.com", "password123", false)
				group := CreateTestGroup(t, db, "Dogs", "Dog volunteers")
				AddUserToGroupWithAdmin(t, db, regular.ID, group.ID, false)
				return regular, group
			},
			isAdmin:        false,
			body:           `{"enabled":true}`,
			expectedStatus: http.StatusForbidden,
		},
		{
			name: "omitted enabled field is rejected rather than silently disabling",
			setupFunc: func(db *gorm.DB) (*models.User, *models.Group) {
				groupAdmin := CreateTestUser(t, db, "gadmin", "gadmin@test.com", "password123", false)
				group := createSchedulingEnabledGroup(t, db, "Dogs", "Dog volunteers")
				AddUserToGroupWithAdmin(t, db, groupAdmin.ID, group.ID, true)
				return groupAdmin, group
			},
			isAdmin:        false,
			body:           `{}`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "enabled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := SetupTestDB(t)
			user, group := tt.setupFunc(db)

			c, w := setupGroupTestContext(user.ID, tt.isAdmin)
			c.Params = gin.Params{{Key: "id", Value: fmt.Sprintf("%d", group.ID)}}
			c.Request = httptest.NewRequest("PATCH", fmt.Sprintf("/api/groups/%d/scheduling", group.ID), bytes.NewBufferString(tt.body))
			c.Request.Header.Set("Content-Type", "application/json")

			handler := UpdateGroupScheduling(db)
			handler(c)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d: %s", tt.expectedStatus, w.Code, w.Body.String())
			}
			if tt.expectedError != "" && !strings.Contains(w.Body.String(), tt.expectedError) {
				t.Errorf("expected error containing %q, got %q", tt.expectedError, w.Body.String())
			}
			if tt.expectedStatus == http.StatusOK {
				var resp struct {
					SchedulingEnabled bool `json:"scheduling_enabled"`
				}
				if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if resp.SchedulingEnabled != tt.expectedEnabled {
					t.Errorf("expected scheduling_enabled=%v, got %v", tt.expectedEnabled, resp.SchedulingEnabled)
				}
				var reloaded models.Group
				if err := db.First(&reloaded, group.ID).Error; err != nil {
					t.Fatalf("failed to reload group: %v", err)
				}
				if reloaded.SchedulingEnabled != tt.expectedEnabled {
					t.Errorf("expected persisted scheduling_enabled=%v, got %v", tt.expectedEnabled, reloaded.SchedulingEnabled)
				}
			}
			if tt.name == "omitted enabled field is rejected rather than silently disabling" {
				var reloaded models.Group
				if err := db.First(&reloaded, group.ID).Error; err != nil {
					t.Fatalf("failed to reload group: %v", err)
				}
				if !reloaded.SchedulingEnabled {
					t.Error("expected scheduling_enabled to remain true (unchanged) after a rejected request, got false")
				}
			}
		})
	}
}
