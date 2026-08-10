package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/middleware"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
	"gorm.io/gorm"
)

// scheduleSlotInput is one 1-hour slot in an incoming schedule-update request.
type scheduleSlotInput struct {
	DayOfWeek int `json:"day_of_week"`
	Hour      int `json:"hour"`
}

// updateScheduleRequest is the body of PUT .../schedule/me and .../schedule/:userId.
// The full slot set replaces whatever was previously stored.
type updateScheduleRequest struct {
	Slots []scheduleSlotInput `json:"slots"`
}

// scheduleSlotResponse is one 1-hour slot in a schedule GET/PUT response.
type scheduleSlotResponse struct {
	DayOfWeek int `json:"day_of_week"`
	Hour      int `json:"hour"`
}

// requireSchedulingEnabled loads the group's SchedulingEnabled flag and
// writes a 404 response if the feature is off, mirroring the HasProtocols
// gate in protocol.go/script.go. Returns false if the response was already
// written (either because scheduling is disabled or the group lookup
// failed) — callers should return immediately in that case.
func requireSchedulingEnabled(c *gin.Context, db *gorm.DB, groupID string) bool {
	var group models.Group
	if err := db.Select("scheduling_enabled").First(&group, groupID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Group not found"})
		return false
	}
	if !group.SchedulingEnabled {
		c.JSON(http.StatusNotFound, gin.H{"error": "Scheduling is not enabled for this group"})
		return false
	}
	return true
}

func toScheduleSlotResponses(slots []models.ShiftSlot) []scheduleSlotResponse {
	out := make([]scheduleSlotResponse, 0, len(slots))
	for _, s := range slots {
		out = append(out, scheduleSlotResponse{DayOfWeek: s.DayOfWeek, Hour: s.Hour})
	}
	return out
}

// validateScheduleSlots checks each slot's day/hour range (day_of_week 0-6,
// hour 8-17) and rejects duplicate (day_of_week, hour) pairs within the
// payload.
func validateScheduleSlots(slots []scheduleSlotInput) error {
	seen := make(map[[2]int]bool, len(slots))
	for _, s := range slots {
		if s.DayOfWeek < 0 || s.DayOfWeek > 6 {
			return fmt.Errorf("day_of_week must be between 0 and 6, got %d", s.DayOfWeek)
		}
		if s.Hour < 8 || s.Hour > 17 {
			return fmt.Errorf("hour must be between 8 and 17, got %d", s.Hour)
		}
		key := [2]int{s.DayOfWeek, s.Hour}
		if seen[key] {
			return fmt.Errorf("duplicate slot for day_of_week %d, hour %d", s.DayOfWeek, s.Hour)
		}
		seen[key] = true
	}
	return nil
}

// replaceGroupScheduleForUser deletes all existing ShiftSlot rows for the
// given (userID, groupID) pair and inserts the provided slots in a single
// transaction, so a schedule update is all-or-nothing.
func replaceGroupScheduleForUser(db *gorm.DB, userID, groupID uint, slots []scheduleSlotInput) ([]models.ShiftSlot, error) {
	var result []models.ShiftSlot
	err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ? AND group_id = ?", userID, groupID).Delete(&models.ShiftSlot{}).Error; err != nil {
			return err
		}
		for _, s := range slots {
			slot := models.ShiftSlot{UserID: userID, GroupID: groupID, DayOfWeek: s.DayOfWeek, Hour: s.Hour}
			if err := tx.Create(&slot).Error; err != nil {
				return err
			}
			result = append(result, slot)
		}
		return nil
	})
	return result, err
}

// GetMySchedule returns the caller's weekly shift slots for the given group.
// Requires group membership (or site admin).
func GetMySchedule(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		db := middleware.GetDB(c, db)
		groupIDParam := c.Param("id")

		userID, _ := c.Get("user_id")
		isAdmin, _ := c.Get("is_admin")

		if !checkGroupAccess(db, userID, isAdmin, groupIDParam) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			return
		}

		if !requireSchedulingEnabled(c, db, groupIDParam) {
			return
		}

		userIDUint, ok := middleware.GetUserID(c)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "User context not found"})
			return
		}

		var slots []models.ShiftSlot
		if err := db.Where("user_id = ? AND group_id = ?", userIDUint, groupIDParam).
			Order("day_of_week, hour").Find(&slots).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch schedule"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"slots": toScheduleSlotResponses(slots)})
	}
}

// UpdateMySchedule replaces the caller's weekly shift slots for the given
// group. Requires group membership (or site admin).
func UpdateMySchedule(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		db := middleware.GetDB(c, db)
		groupIDParam := c.Param("id")

		userID, _ := c.Get("user_id")
		isAdmin, _ := c.Get("is_admin")

		if !checkGroupAccess(db, userID, isAdmin, groupIDParam) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			return
		}

		if !requireSchedulingEnabled(c, db, groupIDParam) {
			return
		}

		userIDUint, ok := middleware.GetUserID(c)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "User context not found"})
			return
		}

		groupIDUint, err := strconv.ParseUint(groupIDParam, 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid group ID"})
			return
		}

		var req updateScheduleRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
			return
		}

		if err := validateScheduleSlots(req.Slots); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		slots, err := replaceGroupScheduleForUser(db, userIDUint, uint(groupIDUint), req.Slots)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update schedule"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"slots": toScheduleSlotResponses(slots)})
	}
}

// updateSchedulingRequest is the body of PATCH .../scheduling. Enabled is a
// pointer so an omitted field is rejected rather than silently defaulting
// to false (Go's zero value for bool) and disabling the feature.
type updateSchedulingRequest struct {
	Enabled *bool `json:"enabled"`
}

// UpdateGroupScheduling turns the shift-scheduling feature on or off for a
// group. Requires group admin or site admin access.
func UpdateGroupScheduling(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		db := middleware.GetDB(c, db)
		groupIDParam := c.Param("id")

		userID, _ := c.Get("user_id")
		isAdmin, _ := c.Get("is_admin")

		if !checkGroupAdminAccess(db, userID, isAdmin, groupIDParam) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
			return
		}

		var req updateSchedulingRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
			return
		}

		if req.Enabled == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "enabled is required"})
			return
		}

		var group models.Group
		if err := db.First(&group, groupIDParam).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Group not found"})
			return
		}

		if err := db.Model(&group).Update("scheduling_enabled", *req.Enabled).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update group"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"scheduling_enabled": *req.Enabled})
	}
}

// GetMemberSchedule returns a specific group member's weekly shift slots.
// Requires group admin or site admin access. Returns 404 if the target user
// is not a member of the group.
func GetMemberSchedule(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		db := middleware.GetDB(c, db)
		groupIDParam := c.Param("id")

		userID, _ := c.Get("user_id")
		isAdmin, _ := c.Get("is_admin")

		if !checkGroupAdminAccess(db, userID, isAdmin, groupIDParam) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
			return
		}

		if !requireSchedulingEnabled(c, db, groupIDParam) {
			return
		}

		targetUserID, err := strconv.ParseUint(c.Param("userId"), 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
			return
		}

		var membership models.UserGroup
		if err := db.Where("user_id = ? AND group_id = ?", targetUserID, groupIDParam).First(&membership).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "User is not a member of this group"})
			return
		}

		var slots []models.ShiftSlot
		if err := db.Where("user_id = ? AND group_id = ?", targetUserID, groupIDParam).
			Order("day_of_week, hour").Find(&slots).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch schedule"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"slots": toScheduleSlotResponses(slots)})
	}
}

// UpdateMemberSchedule replaces a specific group member's weekly shift
// slots. Requires group admin or site admin access. Returns 404 if the
// target user is not a member of the group.
func UpdateMemberSchedule(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		db := middleware.GetDB(c, db)
		groupIDParam := c.Param("id")

		userID, _ := c.Get("user_id")
		isAdmin, _ := c.Get("is_admin")

		if !checkGroupAdminAccess(db, userID, isAdmin, groupIDParam) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
			return
		}

		if !requireSchedulingEnabled(c, db, groupIDParam) {
			return
		}

		targetUserID, err := strconv.ParseUint(c.Param("userId"), 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
			return
		}

		var membership models.UserGroup
		if err := db.Where("user_id = ? AND group_id = ?", targetUserID, groupIDParam).First(&membership).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "User is not a member of this group"})
			return
		}

		groupIDUint, err := strconv.ParseUint(groupIDParam, 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid group ID"})
			return
		}

		var req updateScheduleRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
			return
		}

		if err := validateScheduleSlots(req.Slots); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		slots, err := replaceGroupScheduleForUser(db, uint(targetUserID), uint(groupIDUint), req.Slots)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update schedule"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"slots": toScheduleSlotResponses(slots)})
	}
}

// scheduleOverviewMember identifies one member available for a given slot in
// a GetGroupScheduleOverview response. Fields mirror GetGroupMembers'
// MemberInfo naming so the frontend can reuse its existing display-name
// fallback logic.
type scheduleOverviewMember struct {
	UserID    uint   `json:"user_id"`
	Username  string `json:"username"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

// scheduleOverviewSlot is one (day_of_week, hour) slot with at least one
// available member.
type scheduleOverviewSlot struct {
	DayOfWeek int                      `json:"day_of_week"`
	Hour      int                      `json:"hour"`
	Members   []scheduleOverviewMember `json:"members"`
}

// GetGroupScheduleOverview returns every group member's weekly shift slots
// aggregated by (day_of_week, hour), so an admin can see who is available at
// a glance instead of paging through members one at a time via
// GetMemberSchedule. Requires group admin or site admin access. Only slots
// with at least one available member are included, ordered by
// (day_of_week, hour) ascending.
func GetGroupScheduleOverview(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		db := middleware.GetDB(c, db)
		groupIDParam := c.Param("id")

		userID, _ := c.Get("user_id")
		isAdmin, _ := c.Get("is_admin")

		if !checkGroupAdminAccess(db, userID, isAdmin, groupIDParam) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
			return
		}

		if !requireSchedulingEnabled(c, db, groupIDParam) {
			return
		}

		var slots []models.ShiftSlot
		if err := db.Preload("User").Where("group_id = ?", groupIDParam).
			Order("day_of_week, hour").Find(&slots).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch schedule overview"})
			return
		}

		type key struct {
			DayOfWeek int
			Hour      int
		}
		order := make([]key, 0)
		bucketed := make(map[key][]scheduleOverviewMember)
		for _, s := range slots {
			k := key{DayOfWeek: s.DayOfWeek, Hour: s.Hour}
			if _, exists := bucketed[k]; !exists {
				order = append(order, k)
			}
			bucketed[k] = append(bucketed[k], scheduleOverviewMember{
				UserID:    s.UserID,
				Username:  s.User.Username,
				FirstName: s.User.FirstName,
				LastName:  s.User.LastName,
			})
		}

		result := make([]scheduleOverviewSlot, 0, len(order))
		for _, k := range order {
			result = append(result, scheduleOverviewSlot{
				DayOfWeek: k.DayOfWeek,
				Hour:      k.Hour,
				Members:   bucketed[k],
			})
		}

		c.JSON(http.StatusOK, gin.H{"slots": result})
	}
}
