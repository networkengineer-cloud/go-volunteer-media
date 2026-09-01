package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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

// TestRemoveMemberFromGroupDeletesShiftSlots verifies that removing a member
// from a group also deletes their ShiftSlot rows for that group, so a stale
// schedule doesn't silently resurrect if they rejoin later.
func TestRemoveMemberFromGroupDeletesShiftSlots(t *testing.T) {
	db := SetupTestDB(t)
	admin := CreateTestUser(t, db, "admin", "admin@test.com", "password123", true)
	member := CreateTestUser(t, db, "vol1", "vol1@test.com", "password123", false)
	group := createSchedulingEnabledGroup(t, db, "Dogs", "Dog volunteers")
	AddUserToGroupWithAdmin(t, db, member.ID, group.ID, false)
	db.Create(&models.ShiftSlot{UserID: member.ID, GroupID: group.ID, DayOfWeek: 2, Hour: 9})
	db.Create(&models.ShiftSlot{UserID: member.ID, GroupID: group.ID, DayOfWeek: 3, Hour: 10})

	c, w := setupGroupTestContext(admin.ID, true)
	c.Params = gin.Params{
		{Key: "id", Value: fmt.Sprintf("%d", group.ID)},
		{Key: "userId", Value: fmt.Sprintf("%d", member.ID)},
	}
	c.Request = httptest.NewRequest("DELETE", fmt.Sprintf("/api/groups/%d/members/%d", group.ID, member.ID), nil)

	handler := RemoveMemberFromGroup(db)
	handler(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var remaining []models.ShiftSlot
	if err := db.Where("user_id = ? AND group_id = ?", member.ID, group.ID).Find(&remaining).Error; err != nil {
		t.Fatalf("failed to query remaining slots: %v", err)
	}
	if len(remaining) != 0 {
		t.Errorf("expected all ShiftSlot rows for the removed member to be deleted, found %d", len(remaining))
	}
}

// TestRemoveMemberFromGroupCancelsCoverageRequests verifies that removing a
// member from a group also cancels any non-cancelled ShiftCoverageRequest in
// that group where the removed member is either the original requester or
// the claimant. Without this, a removed member who had claimed someone
// else's open request would keep showing up in the schedule overview
// roster tagged "covering" even though they're no longer in the group.
func TestRemoveMemberFromGroupCancelsCoverageRequests(t *testing.T) {
	db := SetupTestDB(t)
	admin := CreateTestUser(t, db, "admin", "admin@test.com", "password123", true)
	requester := CreateTestUser(t, db, "requester", "requester@test.com", "password123", false)
	claimant := CreateTestUser(t, db, "claimant", "claimant@test.com", "password123", false)
	group := createSchedulingEnabledGroup(t, db, "Dogs", "Dog volunteers")
	AddUserToGroupWithAdmin(t, db, requester.ID, group.ID, false)
	AddUserToGroupWithAdmin(t, db, claimant.ID, group.ID, false)

	date, _ := time.Parse("2006-01-02", nextWeekday(time.Tuesday))
	reqRow := createOpenCoverageRequest(t, db, group.ID, requester.ID, 2, 10, date)
	if claim := performClaimCoverageRequest(db, claimant.ID, group.ID, reqRow.ID); claim.Code != http.StatusOK {
		t.Fatalf("Expected claim to succeed, got %d: %s", claim.Code, claim.Body.String())
	}

	c, w := setupGroupTestContext(admin.ID, true)
	c.Params = gin.Params{
		{Key: "id", Value: fmt.Sprintf("%d", group.ID)},
		{Key: "userId", Value: fmt.Sprintf("%d", claimant.ID)},
	}
	c.Request = httptest.NewRequest("DELETE", fmt.Sprintf("/api/groups/%d/members/%d", group.ID, claimant.ID), nil)

	handler := RemoveMemberFromGroup(db)
	handler(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var updated models.ShiftCoverageRequest
	if err := db.First(&updated, reqRow.ID).Error; err != nil {
		t.Fatalf("failed to reload coverage request: %v", err)
	}
	if updated.Status != models.CoverageRequestCancelled {
		t.Fatalf("expected coverage request status to be cancelled after claimant left the group, got %s", updated.Status)
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

func TestValidateScheduleSlots_DayAwareHourRange(t *testing.T) {
	tests := []struct {
		name    string
		slots   []scheduleSlotInput
		wantErr bool
	}{
		{"weekday hour 17 is valid", []scheduleSlotInput{{DayOfWeek: 3, Hour: 17}}, false},
		{"weekday hour 18 is rejected", []scheduleSlotInput{{DayOfWeek: 3, Hour: 18}}, true},
		{"weekend hour 15 is valid", []scheduleSlotInput{{DayOfWeek: 0, Hour: 15}}, false},
		{"weekend hour 16 is rejected", []scheduleSlotInput{{DayOfWeek: 6, Hour: 16}}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateScheduleSlots(tt.slots)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateScheduleSlots(%+v) error = %v, wantErr %v", tt.slots, err, tt.wantErr)
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

// TestUpdateMySchedule_CancelsOrphanedCoverageRequests verifies that
// editing a recurring schedule cancels any open/claimed
// ShiftCoverageRequest the user made as the original requester whose
// weekday/hour no longer has a matching ShiftSlot row after the edit -
// otherwise GetGroupScheduleOverview can never surface that request again
// (it can only show a coverage request tied to a weekday/hour that still
// has a ShiftSlot), silently hiding even an already-claimed request from
// the only view that shows it. A request for a weekday/hour the user KEPT
// must NOT be cancelled.
func TestUpdateMySchedule_CancelsOrphanedCoverageRequests(t *testing.T) {
	db := SetupTestDB(t)
	user := CreateTestUser(t, db, "vol1", "vol1@test.com", "password123", false)
	group := createSchedulingEnabledGroup(t, db, "Dogs", "Dog volunteers")
	AddUserToGroupWithAdmin(t, db, user.ID, group.ID, false)

	// Two recurring slots: Tuesday 10am and Wednesday 9am.
	db.Create(&models.ShiftSlot{UserID: user.ID, GroupID: group.ID, DayOfWeek: 2, Hour: 10})
	db.Create(&models.ShiftSlot{UserID: user.ID, GroupID: group.ID, DayOfWeek: 3, Hour: 9})

	tuesdayDate, _ := time.Parse("2006-01-02", nextWeekday(time.Tuesday))
	wednesdayDate, _ := time.Parse("2006-01-02", nextWeekday(time.Wednesday))
	orphaned := createOpenCoverageRequest(t, db, group.ID, user.ID, 2, 10, tuesdayDate)
	kept := createOpenCoverageRequest(t, db, group.ID, user.ID, 3, 9, wednesdayDate)

	// New schedule drops Tuesday 10am but keeps Wednesday 9am.
	c, w := setupGroupTestContext(user.ID, false)
	c.Params = gin.Params{{Key: "id", Value: fmt.Sprintf("%d", group.ID)}}
	c.Request = httptest.NewRequest("PUT", fmt.Sprintf("/api/groups/%d/schedule/me", group.ID),
		bytes.NewBufferString(`{"slots":[{"day_of_week":3,"hour":9}]}`))
	c.Request.Header.Set("Content-Type", "application/json")

	handler := UpdateMySchedule(db)
	handler(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var updatedOrphaned models.ShiftCoverageRequest
	if err := db.First(&updatedOrphaned, orphaned.ID).Error; err != nil {
		t.Fatalf("failed to reload orphaned request: %v", err)
	}
	if updatedOrphaned.Status != models.CoverageRequestCancelled {
		t.Errorf("expected orphaned coverage request (dropped weekday/hour) to be cancelled, got %s", updatedOrphaned.Status)
	}

	var updatedKept models.ShiftCoverageRequest
	if err := db.First(&updatedKept, kept.ID).Error; err != nil {
		t.Fatalf("failed to reload kept request: %v", err)
	}
	if updatedKept.Status != models.CoverageRequestOpen {
		t.Errorf("expected coverage request for a retained weekday/hour to remain open, got %s", updatedKept.Status)
	}
}

// TestUpdateMySchedule_CancelsOrphanedRequestOnCadenceChange verifies that
// editing a schedule to change a slot's cadence cancels any open/claimed
// ShiftCoverageRequest the user made as the original requester for a date
// that now falls outside the new cadence's active weeks. Example: a user
// changes their Tuesday 10am slot from weekly to biweekly_a; an existing
// open coverage request for a Tuesday that falls on a "B" week should now
// be cancelled, even though the (day_of_week, hour) pair itself still exists
// as a ShiftSlot row.
func TestUpdateMySchedule_CancelsOrphanedRequestOnCadenceChange(t *testing.T) {
	db := SetupTestDB(t)
	user := CreateTestUser(t, db, "vol1", "vol1@test.com", "password123", false)
	group := createSchedulingEnabledGroup(t, db, "Dogs", "Dog volunteers")
	AddUserToGroupWithAdmin(t, db, user.ID, group.ID, false)

	// Weekly Tuesday 10am to start.
	db.Create(&models.ShiftSlot{UserID: user.ID, GroupID: group.ID, DayOfWeek: 2, Hour: 10, Cadence: "weekly"})

	offWeekDate, err := time.Parse("2006-01-02", nextWeekdayWithParity(t, time.Tuesday, "b"))
	if err != nil {
		t.Fatalf("failed to parse date: %v", err)
	}
	orphanedByParity := createOpenCoverageRequest(t, db, group.ID, user.ID, 2, 10, offWeekDate)

	// User switches their Tuesday 10am slot from weekly to biweekly_a - the
	// existing open request's date is a "b" week, so it's now orphaned even
	// though the (day_of_week, hour) pair itself is unchanged.
	c, w := setupGroupTestContext(user.ID, false)
	c.Params = gin.Params{{Key: "id", Value: fmt.Sprintf("%d", group.ID)}}
	c.Request = httptest.NewRequest("PUT", fmt.Sprintf("/api/groups/%d/schedule/me", group.ID),
		bytes.NewBufferString(`{"slots":[{"day_of_week":2,"hour":10,"cadence":"biweekly_a"}]}`))
	c.Request.Header.Set("Content-Type", "application/json")

	UpdateMySchedule(db)(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var updated models.ShiftCoverageRequest
	if err := db.First(&updated, orphanedByParity.ID).Error; err != nil {
		t.Fatalf("failed to reload request: %v", err)
	}
	if updated.Status != models.CoverageRequestCancelled {
		t.Errorf("expected the request to be cancelled once its date fell outside the new cadence's active weeks, got %s", updated.Status)
	}
}

// TestUpdateMemberSchedule_CancelsRedundantClaimsForNewSlots verifies the
// root-cause fix for a claimant later gaining a conflicting ShiftSlot:
// Alice requests coverage for her Tuesday 10am shift, Bob claims it (Bob
// has no Tuesday 10am ShiftSlot of his own at that point). An admin then
// edits Bob's schedule to add a Tuesday 10am slot. Without cancelling the
// redundant claim, Bob would render twice in GetGroupScheduleOverview for
// that slot - once as a normal member (his new ShiftSlot), once as
// "covering" (his old claim).
func TestUpdateMemberSchedule_CancelsRedundantClaimsForNewSlots(t *testing.T) {
	db := SetupTestDB(t)
	admin := CreateTestUser(t, db, "admin", "admin@test.com", "password123", true)
	alice := CreateTestUser(t, db, "alice", "alice@test.com", "password123", false)
	bob := CreateTestUser(t, db, "bob", "bob@test.com", "password123", false)
	group := createSchedulingEnabledGroup(t, db, "Dogs", "Dog volunteers")
	AddUserToGroupWithAdmin(t, db, alice.ID, group.ID, false)
	AddUserToGroupWithAdmin(t, db, bob.ID, group.ID, false)

	db.Create(&models.ShiftSlot{UserID: alice.ID, GroupID: group.ID, DayOfWeek: 2, Hour: 10})

	date, _ := time.Parse("2006-01-02", nextWeekday(time.Tuesday))
	reqRow := createOpenCoverageRequest(t, db, group.ID, alice.ID, 2, 10, date)
	if claim := performClaimCoverageRequest(db, bob.ID, group.ID, reqRow.ID); claim.Code != http.StatusOK {
		t.Fatalf("Expected claim to succeed, got %d: %s", claim.Code, claim.Body.String())
	}

	// Admin adds Bob to Tuesday 10am, colliding with the claim he already
	// holds for that weekday/hour.
	c, w := setupGroupTestContext(admin.ID, true)
	c.Params = gin.Params{
		{Key: "id", Value: fmt.Sprintf("%d", group.ID)},
		{Key: "userId", Value: fmt.Sprintf("%d", bob.ID)},
	}
	c.Request = httptest.NewRequest("PUT", fmt.Sprintf("/api/groups/%d/schedule/%d", group.ID, bob.ID),
		bytes.NewBufferString(`{"slots":[{"day_of_week":2,"hour":10}]}`))
	c.Request.Header.Set("Content-Type", "application/json")

	handler := UpdateMemberSchedule(db)
	handler(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var updated models.ShiftCoverageRequest
	if err := db.First(&updated, reqRow.ID).Error; err != nil {
		t.Fatalf("failed to reload coverage request: %v", err)
	}
	if updated.Status != models.CoverageRequestCancelled {
		t.Errorf("expected the redundant claim to be cancelled once the claimant gained a colliding ShiftSlot, got %s", updated.Status)
	}
}

// TestUpdateMemberSchedule_KeepsClaimsForDifferentSlots is the companion
// case to TestUpdateMemberSchedule_CancelsRedundantClaimsForNewSlots: a
// claim for a weekday/hour that does NOT collide with the newly-added slot
// must survive the schedule edit.
func TestUpdateMemberSchedule_KeepsClaimsForDifferentSlots(t *testing.T) {
	db := SetupTestDB(t)
	admin := CreateTestUser(t, db, "admin", "admin@test.com", "password123", true)
	alice := CreateTestUser(t, db, "alice", "alice@test.com", "password123", false)
	bob := CreateTestUser(t, db, "bob", "bob@test.com", "password123", false)
	group := createSchedulingEnabledGroup(t, db, "Dogs", "Dog volunteers")
	AddUserToGroupWithAdmin(t, db, alice.ID, group.ID, false)
	AddUserToGroupWithAdmin(t, db, bob.ID, group.ID, false)

	// Alice's shift needing coverage is Wednesday 9am, not Tuesday 10am.
	db.Create(&models.ShiftSlot{UserID: alice.ID, GroupID: group.ID, DayOfWeek: 3, Hour: 9})

	wedDate, _ := time.Parse("2006-01-02", nextWeekday(time.Wednesday))
	reqRow := createOpenCoverageRequest(t, db, group.ID, alice.ID, 3, 9, wedDate)
	if claim := performClaimCoverageRequest(db, bob.ID, group.ID, reqRow.ID); claim.Code != http.StatusOK {
		t.Fatalf("Expected claim to succeed, got %d: %s", claim.Code, claim.Body.String())
	}

	// Admin adds Bob to Tuesday 10am - a different weekday/hour than his
	// Wednesday 9am claim, so the claim must not be cancelled.
	c, w := setupGroupTestContext(admin.ID, true)
	c.Params = gin.Params{
		{Key: "id", Value: fmt.Sprintf("%d", group.ID)},
		{Key: "userId", Value: fmt.Sprintf("%d", bob.ID)},
	}
	c.Request = httptest.NewRequest("PUT", fmt.Sprintf("/api/groups/%d/schedule/%d", group.ID, bob.ID),
		bytes.NewBufferString(`{"slots":[{"day_of_week":2,"hour":10}]}`))
	c.Request.Header.Set("Content-Type", "application/json")

	handler := UpdateMemberSchedule(db)
	handler(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var updated models.ShiftCoverageRequest
	if err := db.First(&updated, reqRow.ID).Error; err != nil {
		t.Fatalf("failed to reload coverage request: %v", err)
	}
	if updated.Status != models.CoverageRequestClaimed {
		t.Errorf("expected the claim for a non-colliding weekday/hour to survive, got %s", updated.Status)
	}
}

// TestUpdateMemberSchedule_KeepsClaimForInactiveBiweeklyNewSlot is the
// parity-aware companion to TestUpdateMemberSchedule_CancelsRedundantClaimsForNewSlots:
// Bob claims Alice's Tuesday 10am coverage request for a date on a "b"
// parity week. An admin then adds a Tuesday 10am biweekly_a slot to Bob's
// own schedule - same day/hour as the claim, but biweekly_a is INACTIVE on
// "b" weeks, so it doesn't actually collide with the claim's specific date.
// The claim must survive; cancelling it here would silently un-cover a
// shift Bob already agreed to take.
func TestUpdateMemberSchedule_KeepsClaimForInactiveBiweeklyNewSlot(t *testing.T) {
	db := SetupTestDB(t)
	admin := CreateTestUser(t, db, "admin", "admin@test.com", "password123", true)
	alice := CreateTestUser(t, db, "alice", "alice@test.com", "password123", false)
	bob := CreateTestUser(t, db, "bob", "bob@test.com", "password123", false)
	group := createSchedulingEnabledGroup(t, db, "Dogs", "Dog volunteers")
	AddUserToGroupWithAdmin(t, db, alice.ID, group.ID, false)
	AddUserToGroupWithAdmin(t, db, bob.ID, group.ID, false)

	db.Create(&models.ShiftSlot{UserID: alice.ID, GroupID: group.ID, DayOfWeek: 2, Hour: 10})

	bWeekDate, err := time.Parse("2006-01-02", nextWeekdayWithParity(t, time.Tuesday, "b"))
	if err != nil {
		t.Fatalf("failed to parse date: %v", err)
	}
	reqRow := createOpenCoverageRequest(t, db, group.ID, alice.ID, 2, 10, bWeekDate)
	if claim := performClaimCoverageRequest(db, bob.ID, group.ID, reqRow.ID); claim.Code != http.StatusOK {
		t.Fatalf("Expected claim to succeed, got %d: %s", claim.Code, claim.Body.String())
	}

	// Admin adds Bob to Tuesday 10am as biweekly_a - same day/hour as his
	// claim, but inactive on the claim's "b"-parity date.
	c, w := setupGroupTestContext(admin.ID, true)
	c.Params = gin.Params{
		{Key: "id", Value: fmt.Sprintf("%d", group.ID)},
		{Key: "userId", Value: fmt.Sprintf("%d", bob.ID)},
	}
	c.Request = httptest.NewRequest("PUT", fmt.Sprintf("/api/groups/%d/schedule/%d", group.ID, bob.ID),
		bytes.NewBufferString(`{"slots":[{"day_of_week":2,"hour":10,"cadence":"biweekly_a"}]}`))
	c.Request.Header.Set("Content-Type", "application/json")

	handler := UpdateMemberSchedule(db)
	handler(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var updated models.ShiftCoverageRequest
	if err := db.First(&updated, reqRow.ID).Error; err != nil {
		t.Fatalf("failed to reload coverage request: %v", err)
	}
	if updated.Status != models.CoverageRequestClaimed {
		t.Errorf("expected the claim to survive since the new biweekly_a slot is inactive on the claim's b-parity date, got %s", updated.Status)
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
			// Enabling scheduling is a one-time setup decision made from Manage
			// Groups, not something group admins toggle for their own group -
			// unlike the rest of a group's schedule (member shifts, coverage
			// requests), which group admins do manage.
			name: "group admin cannot enable scheduling",
			setupFunc: func(db *gorm.DB) (*models.User, *models.Group) {
				groupAdmin := CreateTestUser(t, db, "gadmin", "gadmin@test.com", "password123", false)
				group := CreateTestGroup(t, db, "Dogs", "Dog volunteers")
				AddUserToGroupWithAdmin(t, db, groupAdmin.ID, group.ID, true)
				return groupAdmin, group
			},
			isAdmin:        false,
			body:           `{"enabled":true}`,
			expectedStatus: http.StatusForbidden,
		},
		{
			name: "group admin cannot disable scheduling",
			setupFunc: func(db *gorm.DB) (*models.User, *models.Group) {
				groupAdmin := CreateTestUser(t, db, "gadmin", "gadmin@test.com", "password123", false)
				group := createSchedulingEnabledGroup(t, db, "Dogs", "Dog volunteers")
				AddUserToGroupWithAdmin(t, db, groupAdmin.ID, group.ID, true)
				return groupAdmin, group
			},
			isAdmin:        false,
			body:           `{"enabled":false}`,
			expectedStatus: http.StatusForbidden,
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
			name: "site admin can disable scheduling",
			setupFunc: func(db *gorm.DB) (*models.User, *models.Group) {
				siteAdmin := CreateTestUser(t, db, "admin", "admin@test.com", "password123", true)
				group := createSchedulingEnabledGroup(t, db, "Dogs", "Dog volunteers")
				return siteAdmin, group
			},
			isAdmin:         true,
			body:            `{"enabled":false}`,
			expectedStatus:  http.StatusOK,
			expectedEnabled: false,
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
				siteAdmin := CreateTestUser(t, db, "admin", "admin@test.com", "password123", true)
				group := createSchedulingEnabledGroup(t, db, "Dogs", "Dog volunteers")
				return siteAdmin, group
			},
			isAdmin:        true,
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

func TestGetGroupScheduleOverview(t *testing.T) {
	tests := []struct {
		name           string
		setupFunc      func(*gorm.DB) (contextUser *models.User, group *models.Group)
		isAdmin        bool
		expectedStatus int
	}{
		{
			// Access was widened to any group member (not just group/site
			// admins) - see TestGetGroupScheduleOverview_NonAdminMemberCanAccess
			// for a more targeted test of the same behavior.
			name: "regular member can access",
			setupFunc: func(db *gorm.DB) (*models.User, *models.Group) {
				regular := CreateTestUser(t, db, "regular", "regular@test.com", "password123", false)
				group := createSchedulingEnabledGroup(t, db, "Dogs", "Dog volunteers")
				AddUserToGroupWithAdmin(t, db, regular.ID, group.ID, false)
				return regular, group
			},
			isAdmin:        false,
			expectedStatus: http.StatusOK,
		},
		{
			name: "non-member is denied",
			setupFunc: func(db *gorm.DB) (*models.User, *models.Group) {
				outsider := CreateTestUser(t, db, "outsider", "outsider@test.com", "password123", false)
				group := createSchedulingEnabledGroup(t, db, "Dogs", "Dog volunteers")
				return outsider, group
			},
			isAdmin:        false,
			expectedStatus: http.StatusForbidden,
		},
		{
			name: "scheduling disabled for the group returns 404",
			setupFunc: func(db *gorm.DB) (*models.User, *models.Group) {
				groupAdmin := CreateTestUser(t, db, "gadmin", "gadmin@test.com", "password123", false)
				group := CreateTestGroup(t, db, "Dogs", "Dog volunteers") // scheduling left off
				AddUserToGroupWithAdmin(t, db, groupAdmin.ID, group.ID, true)
				return groupAdmin, group
			},
			isAdmin:        false,
			expectedStatus: http.StatusNotFound,
		},
		{
			name: "group admin sees empty slots for a group with no shifts set",
			setupFunc: func(db *gorm.DB) (*models.User, *models.Group) {
				groupAdmin := CreateTestUser(t, db, "gadmin", "gadmin@test.com", "password123", false)
				group := createSchedulingEnabledGroup(t, db, "Dogs", "Dog volunteers")
				AddUserToGroupWithAdmin(t, db, groupAdmin.ID, group.ID, true)
				return groupAdmin, group
			},
			isAdmin:        false,
			expectedStatus: http.StatusOK,
		},
		{
			name: "site admin sees slots aggregated across multiple members",
			setupFunc: func(db *gorm.DB) (*models.User, *models.Group) {
				siteAdmin := CreateTestUser(t, db, "admin", "admin@test.com", "password123", true)
				group := createSchedulingEnabledGroup(t, db, "Dogs", "Dog volunteers")
				member1 := CreateTestUser(t, db, "vol1", "vol1@test.com", "password123", false)
				member2 := CreateTestUser(t, db, "vol2", "vol2@test.com", "password123", false)
				AddUserToGroupWithAdmin(t, db, member1.ID, group.ID, false)
				AddUserToGroupWithAdmin(t, db, member2.ID, group.ID, false)
				// Both members overlap on (day 2, hour 9); member1 alone on (day 3, hour 10).
				db.Create(&models.ShiftSlot{UserID: member1.ID, GroupID: group.ID, DayOfWeek: 2, Hour: 9})
				db.Create(&models.ShiftSlot{UserID: member2.ID, GroupID: group.ID, DayOfWeek: 2, Hour: 9})
				db.Create(&models.ShiftSlot{UserID: member1.ID, GroupID: group.ID, DayOfWeek: 3, Hour: 10})
				return siteAdmin, group
			},
			isAdmin:        true,
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := SetupTestDB(t)
			contextUser, group := tt.setupFunc(db)

			c, w := setupGroupTestContext(contextUser.ID, tt.isAdmin)
			c.Params = gin.Params{{Key: "id", Value: fmt.Sprintf("%d", group.ID)}}
			c.Request = httptest.NewRequest("GET", fmt.Sprintf("/api/groups/%d/schedule/overview", group.ID), nil)

			handler := GetGroupScheduleOverview(db)
			handler(c)

			if w.Code != tt.expectedStatus {
				t.Fatalf("expected status %d, got %d: %s", tt.expectedStatus, w.Code, w.Body.String())
			}
			if tt.expectedStatus != http.StatusOK {
				return
			}

			var resp struct {
				Slots []scheduleOverviewSlot `json:"slots"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("failed to unmarshal response: %v", err)
			}

			switch tt.name {
			case "group admin sees empty slots for a group with no shifts set":
				if len(resp.Slots) != 0 {
					t.Errorf("expected 0 slots, got %d", len(resp.Slots))
				}
			case "site admin sees slots aggregated across multiple members":
				if len(resp.Slots) != 2 {
					t.Fatalf("expected 2 distinct slots, got %d", len(resp.Slots))
				}
				overlap := resp.Slots[0]
				if overlap.DayOfWeek != 2 || overlap.Hour != 9 {
					t.Errorf("expected first slot to be day 2 hour 9, got day %d hour %d", overlap.DayOfWeek, overlap.Hour)
				}
				if len(overlap.Members) != 2 {
					t.Errorf("expected 2 members on the overlapping slot, got %d", len(overlap.Members))
				}
				solo := resp.Slots[1]
				if solo.DayOfWeek != 3 || solo.Hour != 10 {
					t.Errorf("expected second slot to be day 3 hour 10, got day %d hour %d", solo.DayOfWeek, solo.Hour)
				}
				if len(solo.Members) != 1 || solo.Members[0].Username != "vol1" {
					t.Errorf("expected solo slot to contain only vol1, got %+v", solo.Members)
				}
			}
		})
	}
}

func TestGetGroupScheduleOverview_NonAdminMemberCanAccess(t *testing.T) {
	db := SetupTestDB(t)
	member := CreateTestUser(t, db, "member1", "member1@example.com", "password123", false)
	group := CreateTestGroup(t, db, "Dogs", "Dog volunteers")
	AddUserToGroupWithAdmin(t, db, member.ID, group.ID, false)
	db.Model(group).Update("scheduling_enabled", true)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", member.ID)
		c.Set("is_admin", false)
		c.Next()
	})
	router.GET("/groups/:id/schedule/overview", GetGroupScheduleOverview(db))

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/groups/%d/schedule/overview", group.ID), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 for non-admin member, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetGroupScheduleOverview_EffectiveRoster(t *testing.T) {
	db := SetupTestDB(t)
	requester := CreateTestUser(t, db, "requester", "requester@example.com", "password123", false)
	requester2 := CreateTestUser(t, db, "requester2", "requester2@example.com", "password123", false)
	claimant := CreateTestUser(t, db, "claimant", "claimant@example.com", "password123", false)
	viewer := CreateTestUser(t, db, "viewer", "viewer@example.com", "password123", false)
	group := CreateTestGroup(t, db, "Dogs", "Dog volunteers")
	for _, u := range []*models.User{requester, requester2, claimant, viewer} {
		AddUserToGroupWithAdmin(t, db, u.ID, group.ID, false)
	}
	db.Model(group).Update("scheduling_enabled", true)
	db.Create(&models.ShiftSlot{UserID: requester.ID, GroupID: group.ID, DayOfWeek: 2, Hour: 10})
	// requester2 shares the exact same recurring (day_of_week, hour) bucket
	// as requester - this is what makes it possible for two different
	// members to each file their own coverage request for the same
	// calendar date/hour.
	db.Create(&models.ShiftSlot{UserID: requester2.ID, GroupID: group.ID, DayOfWeek: 2, Hour: 10})

	weekStart := time.Now().UTC().Truncate(24 * time.Hour)
	weekStart = weekStart.AddDate(0, 0, -int(weekStart.Weekday())) // this week's Sunday
	tuesday := weekStart.AddDate(0, 0, 2)

	t.Run("open request keeps requester listed, flagged needs_coverage", func(t *testing.T) {
		reqRow := &models.ShiftCoverageRequest{GroupID: group.ID, RequestedByUserID: requester.ID, Date: tuesday, Hour: 10, Status: models.CoverageRequestOpen}
		db.Create(reqRow)
		defer db.Delete(reqRow)

		gin.SetMode(gin.TestMode)
		router := gin.New()
		router.Use(func(c *gin.Context) {
			c.Set("user_id", viewer.ID)
			c.Set("is_admin", false)
			c.Next()
		})
		router.GET("/groups/:id/schedule/overview", GetGroupScheduleOverview(db))

		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/groups/%d/schedule/overview?week_start=%s", group.ID, weekStart.Format("2006-01-02")), nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var body struct {
			Slots []scheduleOverviewSlot `json:"slots"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}
		found := false
		for _, s := range body.Slots {
			if s.Date != tuesday.Format("2006-01-02") || s.Hour != 10 {
				continue
			}
			for _, m := range s.Members {
				if m.UserID == requester.ID {
					found = true
					if m.Status != "needs_coverage" {
						t.Fatalf("Expected requester status needs_coverage, got %s", m.Status)
					}
					if !m.Claimable {
						t.Fatal("Expected viewer to be able to claim")
					}
				}
			}
		}
		if !found {
			t.Fatal("Expected requester to still appear in the roster")
		}
	})

	t.Run("claimed request swaps requester for claimant", func(t *testing.T) {
		claimedAt := time.Now().UTC()
		reqRow := &models.ShiftCoverageRequest{
			GroupID: group.ID, RequestedByUserID: requester.ID, Date: tuesday, Hour: 10,
			Status: models.CoverageRequestClaimed, ClaimedByUserID: &claimant.ID, ClaimedAt: &claimedAt,
		}
		db.Create(reqRow)
		defer db.Delete(reqRow)

		gin.SetMode(gin.TestMode)
		router := gin.New()
		router.Use(func(c *gin.Context) {
			c.Set("user_id", viewer.ID)
			c.Set("is_admin", false)
			c.Next()
		})
		router.GET("/groups/:id/schedule/overview", GetGroupScheduleOverview(db))

		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/groups/%d/schedule/overview?week_start=%s", group.ID, weekStart.Format("2006-01-02")), nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		var body struct {
			Slots []scheduleOverviewSlot `json:"slots"`
		}
		json.Unmarshal(w.Body.Bytes(), &body)

		var requesterPresent, claimantPresent bool
		var claimantCoverageRequestID *uint
		for _, s := range body.Slots {
			if s.Date != tuesday.Format("2006-01-02") || s.Hour != 10 {
				continue
			}
			for _, m := range s.Members {
				if m.UserID == requester.ID {
					requesterPresent = true
				}
				if m.UserID == claimant.ID && m.Status == "covering" {
					claimantPresent = true
					claimantCoverageRequestID = m.CoverageRequestID
				}
			}
		}
		if requesterPresent {
			t.Fatal("Expected requester to be removed from the roster once claimed")
		}
		if !claimantPresent {
			t.Fatal("Expected claimant to appear in the roster, tagged covering")
		}
		if claimantCoverageRequestID == nil || *claimantCoverageRequestID != reqRow.ID {
			t.Fatalf("Expected the covering member's coverage_request_id to be %d, got %v", reqRow.ID, claimantCoverageRequestID)
		}
	})

	t.Run("two members sharing the same bucket each get their own open request", func(t *testing.T) {
		req1 := &models.ShiftCoverageRequest{GroupID: group.ID, RequestedByUserID: requester.ID, Date: tuesday, Hour: 10, Status: models.CoverageRequestOpen}
		req2 := &models.ShiftCoverageRequest{GroupID: group.ID, RequestedByUserID: requester2.ID, Date: tuesday, Hour: 10, Status: models.CoverageRequestOpen}
		db.Create(req1)
		db.Create(req2)
		defer db.Delete(req1)
		defer db.Delete(req2)

		gin.SetMode(gin.TestMode)
		router := gin.New()
		router.Use(func(c *gin.Context) {
			c.Set("user_id", viewer.ID)
			c.Set("is_admin", false)
			c.Next()
		})
		router.GET("/groups/:id/schedule/overview", GetGroupScheduleOverview(db))

		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/groups/%d/schedule/overview?week_start=%s", group.ID, weekStart.Format("2006-01-02")), nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var body struct {
			Slots []scheduleOverviewSlot `json:"slots"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		var m1, m2 *scheduleOverviewMember
		for i, s := range body.Slots {
			if s.Date != tuesday.Format("2006-01-02") || s.Hour != 10 {
				continue
			}
			for j, m := range s.Members {
				if m.UserID == requester.ID {
					m1 = &body.Slots[i].Members[j]
				}
				if m.UserID == requester2.ID {
					m2 = &body.Slots[i].Members[j]
				}
			}
		}
		if m1 == nil || m2 == nil {
			t.Fatalf("Expected both requester and requester2 to appear in the roster, got requester=%v requester2=%v", m1, m2)
		}
		if m1.Status != "needs_coverage" || m2.Status != "needs_coverage" {
			t.Fatalf("Expected both members tagged needs_coverage, got requester=%s requester2=%s", m1.Status, m2.Status)
		}
		if m1.CoverageRequestID == nil || m2.CoverageRequestID == nil {
			t.Fatal("Expected both members to carry a coverage_request_id")
		}
		if *m1.CoverageRequestID == *m2.CoverageRequestID {
			t.Fatal("Expected requester and requester2 to reference distinct coverage requests")
		}
		if *m1.CoverageRequestID != req1.ID {
			t.Fatalf("Expected requester's coverage_request_id to be %d, got %d", req1.ID, *m1.CoverageRequestID)
		}
		if *m2.CoverageRequestID != req2.ID {
			t.Fatalf("Expected requester2's coverage_request_id to be %d, got %d", req2.ID, *m2.CoverageRequestID)
		}
	})
}

// TestGetGroupScheduleOverview_SurfacesRequestPriority verifies a
// needs_coverage member's roster entry carries the underlying
// ShiftCoverageRequest's Priority, so the frontend can de-emphasize an
// "optional" request instead of rendering it with the same urgency as a
// "normal" one.
func TestGetGroupScheduleOverview_SurfacesRequestPriority(t *testing.T) {
	db := SetupTestDB(t)
	requester := CreateTestUser(t, db, "requester", "requester@example.com", "password123", false)
	viewer := CreateTestUser(t, db, "viewer", "viewer@example.com", "password123", false)
	group := CreateTestGroup(t, db, "Dogs", "Dog volunteers")
	AddUserToGroupWithAdmin(t, db, requester.ID, group.ID, false)
	AddUserToGroupWithAdmin(t, db, viewer.ID, group.ID, false)
	db.Model(group).Update("scheduling_enabled", true)
	db.Create(&models.ShiftSlot{UserID: requester.ID, GroupID: group.ID, DayOfWeek: 2, Hour: 10})

	weekStart := time.Now().UTC().Truncate(24 * time.Hour)
	weekStart = weekStart.AddDate(0, 0, -int(weekStart.Weekday()))
	tuesday := weekStart.AddDate(0, 0, 2)

	reqRow := &models.ShiftCoverageRequest{
		GroupID: group.ID, RequestedByUserID: requester.ID, Date: tuesday, Hour: 10,
		Status: models.CoverageRequestOpen, Priority: "optional",
	}
	if err := db.Create(reqRow).Error; err != nil {
		t.Fatalf("Failed to create coverage request: %v", err)
	}

	c, w := setupGroupTestContext(viewer.ID, false)
	c.Params = gin.Params{{Key: "id", Value: fmt.Sprintf("%d", group.ID)}}
	c.Request = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/groups/%d/schedule/overview?week_start=%s", group.ID, weekStart.Format("2006-01-02")), nil)

	GetGroupScheduleOverview(db)(c)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var body struct {
		Slots []scheduleOverviewSlot `json:"slots"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}
	found := false
	for _, s := range body.Slots {
		if s.Date != tuesday.Format("2006-01-02") || s.Hour != 10 {
			continue
		}
		for _, m := range s.Members {
			if m.UserID == requester.ID {
				found = true
				if m.Priority != "optional" {
					t.Fatalf("Expected member priority %q, got %q", "optional", m.Priority)
				}
			}
		}
	}
	if !found {
		t.Fatal("Expected requester to appear in the roster")
	}
}

// TestParseWeekStart verifies parseWeekStart's Sunday-snapping behavior for
// a non-Sunday input, and that an empty string defaults to the current
// week's Sunday. Every existing GetGroupScheduleOverview test either omits
// week_start or pre-computes a Sunday before passing it, so the snapping
// logic (ref.AddDate(0, 0, -int(ref.Weekday()))) itself was previously
// unverified by any test.
func TestParseWeekStart(t *testing.T) {
	t.Run("a non-Sunday input snaps back to that week's Sunday", func(t *testing.T) {
		// 2026-08-12 is a Wednesday; that week's Sunday is 2026-08-09.
		got, err := parseWeekStart("2026-08-12")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := time.Date(2026, 8, 9, 0, 0, 0, 0, time.UTC)
		if !got.Equal(want) {
			t.Fatalf("expected %s, got %s", want.Format("2006-01-02"), got.Format("2006-01-02"))
		}
		if got.Weekday() != time.Sunday {
			t.Fatalf("expected result to be a Sunday, got %s", got.Weekday())
		}
	})

	t.Run("a Sunday input is returned unchanged", func(t *testing.T) {
		got, err := parseWeekStart("2026-08-09")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := time.Date(2026, 8, 9, 0, 0, 0, 0, time.UTC)
		if !got.Equal(want) {
			t.Fatalf("expected %s, got %s", want.Format("2006-01-02"), got.Format("2006-01-02"))
		}
	})

	t.Run("an empty string defaults to the current week's Sunday", func(t *testing.T) {
		got, err := parseWeekStart("")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Weekday() != time.Sunday {
			t.Fatalf("expected result to be a Sunday, got %s", got.Weekday())
		}
		now := time.Now().UTC()
		wantWeekStart := now.Truncate(24*time.Hour).AddDate(0, 0, -int(now.Weekday()))
		if !got.Equal(wantWeekStart) {
			t.Fatalf("expected %s (this week's Sunday), got %s", wantWeekStart.Format("2006-01-02"), got.Format("2006-01-02"))
		}
	})

	t.Run("an invalid date string is rejected", func(t *testing.T) {
		if _, err := parseWeekStart("not-a-date"); err == nil {
			t.Fatal("expected an error for an invalid date string")
		}
	})
}

// TestShiftSlotCadenceDefaultsToWeekly verifies that new ShiftSlot rows
// without an explicit cadence get the "weekly" default applied by GORM.
func TestShiftSlotCadenceDefaultsToWeekly(t *testing.T) {
	db := SetupTestDB(t)
	user := CreateTestUser(t, db, "vol1", "vol1@test.com", "password123", false)
	group := CreateTestGroup(t, db, "Dogs", "Dog volunteers")

	slot := models.ShiftSlot{UserID: user.ID, GroupID: group.ID, DayOfWeek: 2, Hour: 9}
	if err := db.Create(&slot).Error; err != nil {
		t.Fatalf("expected slot to be created, got error: %v", err)
	}
	if slot.Cadence != "weekly" {
		t.Errorf("expected default Cadence \"weekly\", got %q", slot.Cadence)
	}
}

// TestUpdateMySchedule_Cadence verifies that cadence is round-tripped through
// the schedule API correctly: explicit values are persisted, omitted cadences
// default to "weekly", and invalid cadences are rejected.
func TestUpdateMySchedule_Cadence(t *testing.T) {
	tests := []struct {
		name           string
		body           string
		expectedStatus int
		expectedError  string
		expectCadence  string // checked against the single persisted slot when expectedStatus is 200
	}{
		{
			name:           "explicit biweekly_a is persisted",
			body:           `{"slots":[{"day_of_week":2,"hour":9,"cadence":"biweekly_a"}]}`,
			expectedStatus: http.StatusOK,
			expectCadence:  "biweekly_a",
		},
		{
			name:           "omitted cadence defaults to weekly",
			body:           `{"slots":[{"day_of_week":2,"hour":9}]}`,
			expectedStatus: http.StatusOK,
			expectCadence:  "weekly",
		},
		{
			name:           "unrecognized cadence is rejected",
			body:           `{"slots":[{"day_of_week":2,"hour":9,"cadence":"monthly"}]}`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "cadence",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := SetupTestDB(t)
			user := CreateTestUser(t, db, "vol1", "vol1@test.com", "password123", false)
			group := createSchedulingEnabledGroup(t, db, "Dogs", "Dog volunteers")
			AddUserToGroupWithAdmin(t, db, user.ID, group.ID, false)

			c, w := setupGroupTestContext(user.ID, false)
			c.Params = gin.Params{{Key: "id", Value: fmt.Sprintf("%d", group.ID)}}
			c.Request = httptest.NewRequest("PUT", fmt.Sprintf("/api/groups/%d/schedule/me", group.ID), bytes.NewBufferString(tt.body))
			c.Request.Header.Set("Content-Type", "application/json")

			UpdateMySchedule(db)(c)

			if w.Code != tt.expectedStatus {
				t.Fatalf("expected status %d, got %d: %s", tt.expectedStatus, w.Code, w.Body.String())
			}
			if tt.expectedError != "" && !strings.Contains(w.Body.String(), tt.expectedError) {
				t.Errorf("expected error containing %q, got %q", tt.expectedError, w.Body.String())
			}
			if tt.expectedStatus == http.StatusOK {
				var slot models.ShiftSlot
				if err := db.Where("user_id = ? AND group_id = ?", user.ID, group.ID).First(&slot).Error; err != nil {
					t.Fatalf("failed to load persisted slot: %v", err)
				}
				if slot.Cadence != tt.expectCadence {
					t.Errorf("expected persisted cadence %q, got %q", tt.expectCadence, slot.Cadence)
				}
			}
		})
	}
}

func TestGetGroupScheduleOverview_BiweeklyCadence(t *testing.T) {
	db := SetupTestDB(t)
	alice := CreateTestUser(t, db, "alice", "alice@test.com", "password123", false)
	bob := CreateTestUser(t, db, "bob", "bob@test.com", "password123", false)
	group := createSchedulingEnabledGroup(t, db, "Dogs", "Dog volunteers")
	AddUserToGroupWithAdmin(t, db, alice.ID, group.ID, false)
	AddUserToGroupWithAdmin(t, db, bob.ID, group.ID, false)

	// Alice and Bob alternate the same Tuesday 10am slot.
	db.Create(&models.ShiftSlot{UserID: alice.ID, GroupID: group.ID, DayOfWeek: 2, Hour: 10, Cadence: "biweekly_a"})
	db.Create(&models.ShiftSlot{UserID: bob.ID, GroupID: group.ID, DayOfWeek: 2, Hour: 10, Cadence: "biweekly_b"})

	// 2024-01-07 is an "A" week (the reference Sunday itself).
	aWeekStart := "2024-01-07"
	// 2024-01-14 is a "B" week.
	bWeekStart := "2024-01-14"

	fetchMembers := func(weekStart string) []scheduleOverviewMember {
		c, w := setupGroupTestContext(alice.ID, false)
		c.Params = gin.Params{{Key: "id", Value: fmt.Sprintf("%d", group.ID)}}
		c.Request = httptest.NewRequest("GET", fmt.Sprintf("/api/groups/%d/schedule/overview?week_start=%s", group.ID, weekStart), nil)
		GetGroupScheduleOverview(db)(c)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var resp struct {
			Slots []scheduleOverviewSlot `json:"slots"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}
		for _, s := range resp.Slots {
			if s.DayOfWeek == 2 && s.Hour == 10 {
				return s.Members
			}
		}
		return nil
	}

	aWeekMembers := fetchMembers(aWeekStart)
	if len(aWeekMembers) != 1 || aWeekMembers[0].Username != "alice" {
		t.Errorf("expected only alice on the A week, got %+v", aWeekMembers)
	}

	bWeekMembers := fetchMembers(bWeekStart)
	if len(bWeekMembers) != 1 || bWeekMembers[0].Username != "bob" {
		t.Errorf("expected only bob on the B week, got %+v", bWeekMembers)
	}
}

// TestGetGroupScheduleOverview_BiweeklyViewerConflictParity verifies the
// viewer's own biweekly recurring slot only counts as a scheduling conflict
// (and blocks Claimable on another member's open coverage request) on the
// parity week where that slot is actually active - not on the viewer's
// "off" week. This exercises the viewerSlotCadence/conflict computation in
// GetGroupScheduleOverview, distinct from TestGetGroupScheduleOverview_
// BiweeklyCadence above, which only covers roster membership.
func TestGetGroupScheduleOverview_BiweeklyViewerConflictParity(t *testing.T) {
	db := SetupTestDB(t)
	viewer := CreateTestUser(t, db, "viewer2", "viewer2@test.com", "password123", false)
	otherMember := CreateTestUser(t, db, "othermember", "othermember@test.com", "password123", false)
	group := createSchedulingEnabledGroup(t, db, "Dogs", "Dog volunteers")
	AddUserToGroupWithAdmin(t, db, viewer.ID, group.ID, false)
	AddUserToGroupWithAdmin(t, db, otherMember.ID, group.ID, false)

	// The viewer's own recurring commitment only occurs on "A" parity weeks.
	db.Create(&models.ShiftSlot{UserID: viewer.ID, GroupID: group.ID, DayOfWeek: 2, Hour: 10, Cadence: "biweekly_a"})
	// otherMember has an ordinary weekly slot in the same (day_of_week, hour)
	// bucket, so both members' rows land in the same slotBucket and the
	// viewer's own slot is a plausible source of a false conflict against
	// otherMember's open coverage request.
	db.Create(&models.ShiftSlot{UserID: otherMember.ID, GroupID: group.ID, DayOfWeek: 2, Hour: 10})

	// 2024-01-07 is the "A" week Sunday (the viewer's on-week) - same
	// biweeklyReferenceSunday anchor used by
	// TestGetGroupScheduleOverview_BiweeklyCadence above, so it's safe to
	// hardcode: it can never change without breaking every existing
	// biweekly assignment in production (see schedule_hours.go).
	aWeekStart := "2024-01-07"
	aTuesday, _ := time.Parse("2006-01-02", aWeekStart)
	aTuesday = aTuesday.AddDate(0, 0, 2) // 2024-01-09
	// 2024-01-14 is the following "B" week Sunday - the viewer's off-week.
	bWeekStart := "2024-01-14"
	bTuesday, _ := time.Parse("2006-01-02", bWeekStart)
	bTuesday = bTuesday.AddDate(0, 0, 2) // 2024-01-16

	// findOtherMember creates an open coverage request for otherMember on
	// requestDate, fetches the overview as the viewer for weekStart, and
	// returns otherMember's entry in the (Tuesday, 10am) bucket.
	findOtherMember := func(weekStart string, requestDate time.Time) *scheduleOverviewMember {
		reqRow := &models.ShiftCoverageRequest{
			GroupID: group.ID, RequestedByUserID: otherMember.ID, Date: requestDate, Hour: 10,
			Status: models.CoverageRequestOpen,
		}
		if err := db.Create(reqRow).Error; err != nil {
			t.Fatalf("failed to create coverage request: %v", err)
		}
		defer db.Delete(reqRow)

		c, w := setupGroupTestContext(viewer.ID, false)
		c.Params = gin.Params{{Key: "id", Value: fmt.Sprintf("%d", group.ID)}}
		c.Request = httptest.NewRequest("GET", fmt.Sprintf("/api/groups/%d/schedule/overview?week_start=%s", group.ID, weekStart), nil)
		GetGroupScheduleOverview(db)(c)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var resp struct {
			Slots []scheduleOverviewSlot `json:"slots"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}
		for _, s := range resp.Slots {
			if s.DayOfWeek != 2 || s.Hour != 10 {
				continue
			}
			for i := range s.Members {
				if s.Members[i].UserID == otherMember.ID {
					return &s.Members[i]
				}
			}
		}
		return nil
	}

	t.Run("viewer's on-week: own biweekly slot registers as a conflict", func(t *testing.T) {
		m := findOtherMember(aWeekStart, aTuesday)
		if m == nil {
			t.Fatal("expected otherMember to appear in the roster")
		}
		if m.Status != "needs_coverage" {
			t.Fatalf("expected status needs_coverage, got %s", m.Status)
		}
		if !m.Conflict {
			t.Error("expected Conflict true on the viewer's on-week")
		}
		if m.Claimable {
			t.Error("expected Claimable false on the viewer's on-week (viewer already has a shift then)")
		}
	})

	t.Run("viewer's off-week: own biweekly slot is not active, no false conflict", func(t *testing.T) {
		m := findOtherMember(bWeekStart, bTuesday)
		if m == nil {
			t.Fatal("expected otherMember to appear in the roster")
		}
		if m.Status != "needs_coverage" {
			t.Fatalf("expected status needs_coverage, got %s", m.Status)
		}
		if m.Conflict {
			t.Error("expected Conflict false on the viewer's off-week")
		}
		if !m.Claimable {
			t.Error("expected Claimable true on the viewer's off-week")
		}
	})
}

// TestGetGroupScheduleOverview_ViewerConflictAcrossMultipleGroups verifies
// the fix for viewerSlotCadence collapsing to last-write-wins: viewerSlots
// is queried across ALL of the viewer's groups (no group_id filter), and
// the per-group unique index allows the same (day_of_week, hour) to
// legitimately recur in two different groups. Here the viewer has a WEEKLY
// Tuesday 10am slot in the group whose overview we're fetching, and a
// BIWEEKLY_B Tuesday 10am slot (inactive this "A" week) in a second,
// unrelated group. The weekly slot alone is enough to conflict regardless
// of the biweekly slot's state, so Conflict must be true even though the
// biweekly slot - if it were the only one the map remembered - would report
// no conflict on this particular week.
func TestGetGroupScheduleOverview_ViewerConflictAcrossMultipleGroups(t *testing.T) {
	db := SetupTestDB(t)
	viewer := CreateTestUser(t, db, "viewer3", "viewer3@test.com", "password123", false)
	otherMember := CreateTestUser(t, db, "othermember3", "othermember3@test.com", "password123", false)
	group := createSchedulingEnabledGroup(t, db, "Dogs", "Dog volunteers")
	otherGroup := createSchedulingEnabledGroup(t, db, "Cats", "Cat volunteers")
	AddUserToGroupWithAdmin(t, db, viewer.ID, group.ID, false)
	AddUserToGroupWithAdmin(t, db, otherMember.ID, group.ID, false)
	AddUserToGroupWithAdmin(t, db, viewer.ID, otherGroup.ID, false)

	// The viewer's WEEKLY commitment lives in `group`, the group whose
	// overview we're fetching.
	db.Create(&models.ShiftSlot{UserID: viewer.ID, GroupID: group.ID, DayOfWeek: 2, Hour: 10, Cadence: "weekly"})
	// The viewer's BIWEEKLY_B commitment lives in `otherGroup` - same
	// day_of_week/hour, different group, allowed by the per-group unique
	// index. Inserted after the weekly slot so a last-write-wins map would
	// remember biweekly_b as the (day_of_week, hour) cadence.
	db.Create(&models.ShiftSlot{UserID: viewer.ID, GroupID: otherGroup.ID, DayOfWeek: 2, Hour: 10, Cadence: "biweekly_b"})
	// otherMember has an ordinary weekly slot in `group`'s same bucket, so
	// both land in the same slotBucket and the viewer's own slot(s) are a
	// plausible source of a conflict against otherMember's open request.
	db.Create(&models.ShiftSlot{UserID: otherMember.ID, GroupID: group.ID, DayOfWeek: 2, Hour: 10, Cadence: "weekly"})

	// 2024-01-07 is the "A" week Sunday (biweeklyReferenceSunday) - the
	// viewer's biweekly_b slot in otherGroup is INACTIVE this week, but the
	// weekly slot in `group` is always active.
	aWeekStart := "2024-01-07"
	aTuesday, _ := time.Parse("2006-01-02", aWeekStart)
	aTuesday = aTuesday.AddDate(0, 0, 2) // 2024-01-09

	reqRow := &models.ShiftCoverageRequest{
		GroupID: group.ID, RequestedByUserID: otherMember.ID, Date: aTuesday, Hour: 10,
		Status: models.CoverageRequestOpen,
	}
	if err := db.Create(reqRow).Error; err != nil {
		t.Fatalf("failed to create coverage request: %v", err)
	}

	c, w := setupGroupTestContext(viewer.ID, false)
	c.Params = gin.Params{{Key: "id", Value: fmt.Sprintf("%d", group.ID)}}
	c.Request = httptest.NewRequest("GET", fmt.Sprintf("/api/groups/%d/schedule/overview?week_start=%s", group.ID, aWeekStart), nil)
	GetGroupScheduleOverview(db)(c)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Slots []scheduleOverviewSlot `json:"slots"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	var otherMemberEntry *scheduleOverviewMember
	for _, s := range resp.Slots {
		if s.DayOfWeek != 2 || s.Hour != 10 {
			continue
		}
		for i := range s.Members {
			if s.Members[i].UserID == otherMember.ID {
				otherMemberEntry = &s.Members[i]
			}
		}
	}
	if otherMemberEntry == nil {
		t.Fatal("expected otherMember to appear in the roster")
	}
	if otherMemberEntry.Status != "needs_coverage" {
		t.Fatalf("expected status needs_coverage, got %s", otherMemberEntry.Status)
	}
	if !otherMemberEntry.Conflict {
		t.Error("expected Conflict true: the viewer's weekly slot in a different group is active regardless of the biweekly_b slot's state")
	}
	if otherMemberEntry.Claimable {
		t.Error("expected Claimable false since the viewer's weekly slot conflicts")
	}
}
