package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/middleware"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
	"gorm.io/gorm"
)

// scheduleSlotInput is one 1-hour slot in an incoming schedule-update request.
type scheduleSlotInput struct {
	DayOfWeek int    `json:"day_of_week"`
	Hour      int    `json:"hour"`
	Cadence   string `json:"cadence"`
}

// updateScheduleRequest is the body of PUT .../schedule/me and .../schedule/:userId.
// The full slot set replaces whatever was previously stored.
type updateScheduleRequest struct {
	Slots []scheduleSlotInput `json:"slots"`
}

// scheduleSlotResponse is one 1-hour slot in a schedule GET/PUT response.
type scheduleSlotResponse struct {
	DayOfWeek int    `json:"day_of_week"`
	Hour      int    `json:"hour"`
	Cadence   string `json:"cadence"`
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
		out = append(out, scheduleSlotResponse{DayOfWeek: s.DayOfWeek, Hour: s.Hour, Cadence: s.Cadence})
	}
	return out
}

// validateScheduleSlots checks each slot's day/hour range (day_of_week 0-6,
// hour 8-maxHourFor(dayOfWeek), where maxHourFor depends on day of week),
// validates cadence (weekly/biweekly_a/biweekly_b, defaulting to weekly when
// omitted), and rejects duplicate (day_of_week, hour) pairs within the payload.
func validateScheduleSlots(slots []scheduleSlotInput) error {
	seen := make(map[[2]int]bool, len(slots))
	for i, s := range slots {
		if s.DayOfWeek < 0 || s.DayOfWeek > 6 {
			return fmt.Errorf("day_of_week must be between 0 and 6, got %d", s.DayOfWeek)
		}
		if s.Hour < 8 || s.Hour > maxHourFor(s.DayOfWeek) {
			return fmt.Errorf("hour must be between 8 and %d for day_of_week %d, got %d", maxHourFor(s.DayOfWeek), s.DayOfWeek, s.Hour)
		}
		if s.Cadence == "" {
			slots[i].Cadence = "weekly"
		} else if s.Cadence != "weekly" && s.Cadence != "biweekly_a" && s.Cadence != "biweekly_b" {
			return fmt.Errorf("cadence must be one of weekly, biweekly_a, biweekly_b, got %q", s.Cadence)
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
			slot := models.ShiftSlot{UserID: userID, GroupID: groupID, DayOfWeek: s.DayOfWeek, Hour: s.Hour, Cadence: s.Cadence}
			if err := tx.Create(&slot).Error; err != nil {
				return err
			}
			result = append(result, slot)
		}
		if err := cancelOrphanedRequesterCoverageRequests(tx, userID, groupID); err != nil {
			return err
		}
		if err := cancelRedundantClaimsForNewSlots(tx, userID, groupID, slots); err != nil {
			return err
		}
		return nil
	})
	return result, err
}

// cancelOrphanedRequesterCoverageRequests cancels any non-cancelled
// ShiftCoverageRequest this user made (as the original requester, not as
// a claimant - claiming never touches ShiftSlot) whose underlying weekday/
// hour no longer has a matching ShiftSlot row, so GetGroupScheduleOverview
// never surfaces a request tied to a shift the requester no longer has.
// Must run inside the same transaction as the slot replace, after the new
// slots are inserted.
func cancelOrphanedRequesterCoverageRequests(tx *gorm.DB, userID, groupID uint) error {
	var requests []models.ShiftCoverageRequest
	if err := tx.Where("group_id = ? AND requested_by_user_id = ? AND status != ?",
		groupID, userID, models.CoverageRequestCancelled).Find(&requests).Error; err != nil {
		return err
	}
	for _, r := range requests {
		var slot models.ShiftSlot
		err := tx.Where("group_id = ? AND user_id = ? AND day_of_week = ? AND hour = ?",
			groupID, userID, int(r.Date.Weekday()), r.Hour).First(&slot).Error
		stillActive := err == nil && slotActiveForWeek(slot.Cadence, weekStartOf(r.Date))
		if !stillActive {
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			if err := tx.Model(&models.ShiftCoverageRequest{ID: r.ID}).
				Update("status", models.CoverageRequestCancelled).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

// cancelRedundantClaimsForNewSlots cancels any ShiftCoverageRequest this
// user has CLAIMED (covering someone else's shift) whose weekday/hour now
// collides with a slot they just added to their own recurring schedule AND
// is actually active (per slotActiveForWeek) on that claim's specific date.
// Once they're recurring on that slot themselves for that date's week, the
// one-off "covering" arrangement for the same weekday/hour is redundant and
// would otherwise render them twice in the effective roster - once as a
// normal member, once as the covering claimant. A day/hour match alone
// isn't enough: a biweekly new slot that's inactive on the claim's date
// doesn't actually conflict, so cancelling would silently un-cover a shift
// the user already agreed to take.
func cancelRedundantClaimsForNewSlots(tx *gorm.DB, userID, groupID uint, slots []scheduleSlotInput) error {
	if len(slots) == 0 {
		return nil
	}
	newSlotsByKey := make(map[[2]int][]string, len(slots))
	for _, s := range slots {
		key := [2]int{s.DayOfWeek, s.Hour}
		newSlotsByKey[key] = append(newSlotsByKey[key], s.Cadence)
	}

	var claims []models.ShiftCoverageRequest
	if err := tx.Where("group_id = ? AND claimed_by_user_id = ? AND status = ?",
		groupID, userID, models.CoverageRequestClaimed).Find(&claims).Error; err != nil {
		return err
	}
	for _, r := range claims {
		cadences := newSlotsByKey[[2]int{int(r.Date.Weekday()), r.Hour}]
		if cadences == nil {
			continue
		}
		weekStart := weekStartOf(r.Date)
		conflicts := false
		for _, cadence := range cadences {
			if slotActiveForWeek(cadence, weekStart) {
				conflicts = true
				break
			}
		}
		if conflicts {
			if err := tx.Model(&models.ShiftCoverageRequest{ID: r.ID}).
				Update("status", models.CoverageRequestCancelled).Error; err != nil {
				return err
			}
		}
	}
	return nil
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
// group. Requires site admin access - unlike the rest of a group's schedule
// (member shifts, coverage requests), enabling scheduling itself is a
// one-time setup decision made from Manage Groups, not something group
// admins toggle for their own group.
func UpdateGroupScheduling(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		db := middleware.GetDB(c, db)
		groupIDParam := c.Param("id")

		if !middleware.GetIsAdmin(c) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Site admin access required"})
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

// scheduleOverviewMember identifies one member's status for a given
// (date, hour) slot in a GetGroupScheduleOverview response.
type scheduleOverviewMember struct {
	UserID            uint   `json:"user_id"`
	Username          string `json:"username"`
	FirstName         string `json:"first_name"`
	LastName          string `json:"last_name"`
	Cadence           string `json:"cadence"`
	Status            string `json:"status"`             // normal | needs_coverage | covering
	Priority          string `json:"priority,omitempty"` // the underlying request's priority, only set when status is needs_coverage
	CoverageRequestID *uint  `json:"coverage_request_id,omitempty"`
	Claimable         bool   `json:"claimable,omitempty"`
	Conflict          bool   `json:"conflict,omitempty"`
}

// scheduleOverviewSlot is the effective roster for one specific calendar
// date + hour within the viewed week.
type scheduleOverviewSlot struct {
	Date      string                   `json:"date"`
	DayOfWeek int                      `json:"day_of_week"`
	Hour      int                      `json:"hour"`
	Members   []scheduleOverviewMember `json:"members"`
}

type dateHourKey struct {
	Date string
	Hour int
}

type dateHourUserKey struct {
	Date   string
	Hour   int
	UserID uint
}

// parseWeekStart parses an optional "2006-01-02" week_start query param and
// snaps it back to that week's Sunday. An empty string defaults to the
// current week's Sunday (UTC).
func parseWeekStart(raw string) (time.Time, error) {
	ref := time.Now().UTC()
	if raw != "" {
		parsed, err := time.Parse("2006-01-02", raw)
		if err != nil {
			return time.Time{}, err
		}
		ref = parsed
	}
	ref = ref.Truncate(24 * time.Hour)
	return ref.AddDate(0, 0, -int(ref.Weekday())), nil
}

// GetGroupScheduleOverview returns the effective roster for every (date,
// hour) slot in the requested week: the group's recurring ShiftSlot
// assignments for that weekday/hour, overlaid with any open or claimed
// ShiftCoverageRequest for that exact date. Open requests keep the original
// requester listed (tagged needs_coverage); claimed requests swap the
// requester for the claimant (tagged covering). Requires group membership
// (or site admin) - open to every member, not just admins.
func GetGroupScheduleOverview(db *gorm.DB) gin.HandlerFunc {
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

		callerUserID, ok := middleware.GetUserID(c)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "User context not found"})
			return
		}

		weekStart, err := parseWeekStart(c.Query("week_start"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "week_start must be in YYYY-MM-DD format"})
			return
		}
		weekEnd := weekStart.AddDate(0, 0, 6)

		var slots []models.ShiftSlot
		if err := db.Preload("User").Where("group_id = ?", groupIDParam).
			Order("day_of_week, hour").Find(&slots).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch schedule overview"})
			return
		}

		type slotBucket struct {
			DayOfWeek int
			Hour      int
			Members   []models.ShiftSlot
		}
		order := make([]slotBucket, 0)
		bucketIndex := make(map[[2]int]int)
		for _, s := range slots {
			key := [2]int{s.DayOfWeek, s.Hour}
			idx, exists := bucketIndex[key]
			if !exists {
				idx = len(order)
				bucketIndex[key] = idx
				order = append(order, slotBucket{DayOfWeek: s.DayOfWeek, Hour: s.Hour})
			}
			order[idx].Members = append(order[idx].Members, s)
		}

		var coverageRequests []models.ShiftCoverageRequest
		if err := db.Preload("ClaimedByUser").
			Where("group_id = ? AND date >= ? AND date <= ? AND status != ?",
				groupIDParam, weekStart, weekEnd, models.CoverageRequestCancelled).
			Find(&coverageRequests).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch coverage requests"})
			return
		}
		// Keyed by requester so multiple members sharing the same recurring
		// (day_of_week, hour) bucket can each have their own coverage
		// request for the same calendar date without clobbering each
		// other. claimedByDateHour is grouped separately (not keyed by
		// requester) since a "covering" entry is added per-claim, not
		// per-original-member.
		requestsByDateHourUser := make(map[dateHourUserKey]models.ShiftCoverageRequest, len(coverageRequests))
		claimedByDateHour := make(map[dateHourKey][]models.ShiftCoverageRequest)
		for _, r := range coverageRequests {
			dateStr := r.Date.Format("2006-01-02")
			requestsByDateHourUser[dateHourUserKey{Date: dateStr, Hour: r.Hour, UserID: r.RequestedByUserID}] = r
			if r.Status == models.CoverageRequestClaimed {
				key := dateHourKey{Date: dateStr, Hour: r.Hour}
				claimedByDateHour[key] = append(claimedByDateHour[key], r)
			}
		}

		// The viewer's own recurring commitments across every group they're
		// in, and any request they've already claimed this week - both feed
		// the per-slot "conflict" flag.
		var viewerSlots []models.ShiftSlot
		if err := db.Where("user_id = ?", callerUserID).Find(&viewerSlots).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch schedule overview"})
			return
		}
		// viewerSlots is queried across ALL of the viewer's groups (no
		// group_id filter), and the per-group unique index allows the same
		// (day_of_week, hour) to legitimately occur more than once - e.g. a
		// weekly slot in one group and a biweekly slot at the same day/hour
		// in another. Keep every matching slot's cadence per key (not just
		// the last one iterated) so the conflict check below can OR across
		// all of them, matching the old single-boolean viewerBusy semantics
		// where any match at all made the slot busy.
		viewerSlotCadences := make(map[[2]int][]string, len(viewerSlots))
		for _, s := range viewerSlots {
			key := [2]int{s.DayOfWeek, s.Hour}
			viewerSlotCadences[key] = append(viewerSlotCadences[key], s.Cadence)
		}
		viewerClaimedBusy := make(map[dateHourKey]bool)
		for _, r := range coverageRequests {
			if r.Status == models.CoverageRequestClaimed && r.ClaimedByUserID != nil && *r.ClaimedByUserID == callerUserID {
				viewerClaimedBusy[dateHourKey{Date: r.Date.Format("2006-01-02"), Hour: r.Hour}] = true
			}
		}

		result := make([]scheduleOverviewSlot, 0)
		for _, bucket := range order {
			for offset := 0; offset < 7; offset++ {
				date := weekStart.AddDate(0, 0, offset)
				if int(date.Weekday()) != bucket.DayOfWeek {
					continue
				}
				dateStr := date.Format("2006-01-02")
				viewerSlotConflict := false
				for _, cadence := range viewerSlotCadences[[2]int{bucket.DayOfWeek, bucket.Hour}] {
					if slotActiveForWeek(cadence, weekStartOf(date)) {
						viewerSlotConflict = true
						break
					}
				}
				conflict := viewerSlotConflict ||
					viewerClaimedBusy[dateHourKey{Date: dateStr, Hour: bucket.Hour}]

				members := make([]scheduleOverviewMember, 0, len(bucket.Members)+1)
				for _, s := range bucket.Members {
					if !slotActiveForWeek(s.Cadence, weekStartOf(date)) {
						continue
					}
					req, hasRequest := requestsByDateHourUser[dateHourUserKey{Date: dateStr, Hour: bucket.Hour, UserID: s.UserID}]
					if hasRequest && req.Status == models.CoverageRequestClaimed {
						// This member handed their shift off - drop them,
						// the claimant(s) are added below instead.
						continue
					}
					m := scheduleOverviewMember{
						UserID:    s.UserID,
						Username:  s.User.Username,
						FirstName: s.User.FirstName,
						LastName:  s.User.LastName,
						Cadence:   s.Cadence,
						Status:    "normal",
					}
					if hasRequest && req.Status == models.CoverageRequestOpen {
						m.Status = "needs_coverage"
						m.Priority = req.Priority
						reqID := req.ID
						m.CoverageRequestID = &reqID
						if s.UserID != callerUserID {
							m.Claimable = !conflict
							m.Conflict = conflict
						}
					}
					members = append(members, m)
				}
				// Dedupe-by-user_id guard: even with the root-cause fix in
				// replaceGroupScheduleForUser (cancelRedundantClaimsForNewSlots),
				// this backstop ensures no future code path can ever produce a
				// duplicate member in one slot's output - e.g. a "normal" entry
				// from a ShiftSlot and a "covering" entry from a stale claim for
				// the same user would otherwise both render, causing a duplicate
				// React key and double-counted availability tallies.
				seenUserIDs := make(map[uint]bool, len(members))
				for _, m := range members {
					seenUserIDs[m.UserID] = true
				}
				for _, r := range claimedByDateHour[dateHourKey{Date: dateStr, Hour: bucket.Hour}] {
					if r.ClaimedByUser == nil || seenUserIDs[r.ClaimedByUser.ID] {
						continue
					}
					seenUserIDs[r.ClaimedByUser.ID] = true
					members = append(members, scheduleOverviewMember{
						UserID:    r.ClaimedByUser.ID,
						Username:  r.ClaimedByUser.Username,
						FirstName: r.ClaimedByUser.FirstName,
						LastName:  r.ClaimedByUser.LastName,
						Status:    "covering",
					})
				}

				result = append(result, scheduleOverviewSlot{
					Date:      dateStr,
					DayOfWeek: bucket.DayOfWeek,
					Hour:      bucket.Hour,
					Members:   members,
				})
			}
		}

		c.JSON(http.StatusOK, gin.H{"week_start": weekStart.Format("2006-01-02"), "slots": result})
	}
}
