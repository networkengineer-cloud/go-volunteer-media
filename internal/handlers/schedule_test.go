package handlers

import (
	"testing"

	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
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
