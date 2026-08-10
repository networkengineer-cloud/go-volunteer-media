package handlers

import (
	"testing"
	"time"

	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
)

func TestShiftCoverageRequestMigration(t *testing.T) {
	db := SetupTestDB(t)
	user := CreateTestUser(t, db, "volunteer1", "volunteer1@example.com", "password123", false)
	group := CreateTestGroup(t, db, "Dogs", "Dog volunteers")

	req := &models.ShiftCoverageRequest{
		GroupID:           group.ID,
		RequestedByUserID: user.ID,
		Date:              time.Date(2026, 8, 18, 0, 0, 0, 0, time.UTC),
		Hour:              10,
		Status:            models.CoverageRequestOpen,
	}
	if err := db.Create(req).Error; err != nil {
		t.Fatalf("Failed to create ShiftCoverageRequest: %v", err)
	}
	if req.ID == 0 {
		t.Fatal("Expected ShiftCoverageRequest to get an assigned ID")
	}
}
