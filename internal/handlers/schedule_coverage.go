package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/email"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/groupme"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/logging"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/middleware"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
	"gorm.io/gorm"
)

var (
	errNoMatchingSlot    = errors.New("no matching recurring shift for that date and hour")
	errDuplicateRequest  = errors.New("a coverage request already exists for that date and hour")
	errPastDate          = errors.New("date must not be in the past")
	errRequestNotFound   = errors.New("coverage request not found")
	errRequestNotOpen    = errors.New("coverage request is no longer open")
	errSelfClaim         = errors.New("cannot claim your own coverage request")
	errClaimConflict     = errors.New("claimant already has a conflicting shift at that time")
	errReassignSameUser  = errors.New("cannot reassign a shift to the same person")
	errNotGroupMember    = errors.New("user is not a member of this group")
	errRequestNotClaimed = errors.New("coverage request is not currently claimed")
)

// hasConflictingCommitment reports whether userID already has a real-world
// scheduling conflict at date/hour: either an active recurring ShiftSlot (in
// any group - a time conflict doesn't respect group boundaries) or an
// already-claimed ShiftCoverageRequest at that exact date/hour. Shared by
// ClaimCoverageRequest and ReassignShiftsBatch, which both need to guarantee the
// person ending up on the shift isn't double-booked.
func hasConflictingCommitment(tx *gorm.DB, userID uint, date time.Time, hour int) (bool, error) {
	var conflictingSlots []models.ShiftSlot
	if err := tx.Where("user_id = ? AND day_of_week = ? AND hour = ?", userID, int(date.Weekday()), hour).
		Find(&conflictingSlots).Error; err != nil {
		return false, err
	}
	for _, s := range conflictingSlots {
		if slotActiveForWeek(s.Cadence, weekStartOf(date)) {
			return true, nil
		}
	}
	var conflictCount int64
	if err := tx.Model(&models.ShiftCoverageRequest{}).
		Where("claimed_by_user_id = ? AND date = ? AND hour = ? AND status = ?",
			userID, date, hour, models.CoverageRequestClaimed).
		Count(&conflictCount).Error; err != nil {
		return false, err
	}
	return conflictCount > 0, nil
}

// scheduleEmailNotificationsEnabled gates coverage-request and claim emails
// only - GroupMe posts are unaffected. Deliberately opt-in - unset or any
// value other than "true"/"1" means disabled - so beta testers can use the
// Schedule tab without the rest of the group being emailed before the
// feature is ready for everyone.
func scheduleEmailNotificationsEnabled() bool {
	v := os.Getenv("SCHEDULE_EMAIL_NOTIFICATIONS_ENABLED")
	return v == "true" || v == "1"
}

type createCoverageRequestRequest struct {
	Date     string `json:"date"`
	Hour     int    `json:"hour"`
	UserID   *uint  `json:"user_id"`
	Priority string `json:"priority"`
}

type coverageRequestResponse struct {
	ID                uint   `json:"id"`
	GroupID           uint   `json:"group_id"`
	RequestedByUserID uint   `json:"requested_by_user_id"`
	Date              string `json:"date"`
	Hour              int    `json:"hour"`
	Status            string `json:"status"`
	Priority          string `json:"priority"`
	ClaimedByUserID   *uint  `json:"claimed_by_user_id"`
}

func toCoverageRequestResponse(r models.ShiftCoverageRequest) coverageRequestResponse {
	return coverageRequestResponse{
		ID:                r.ID,
		GroupID:           r.GroupID,
		RequestedByUserID: r.RequestedByUserID,
		Date:              r.Date.Format("2006-01-02"),
		Hour:              r.Hour,
		Status:            string(r.Status),
		Priority:          r.Priority,
		ClaimedByUserID:   r.ClaimedByUserID,
	}
}

// normalizeCoveragePriority defaults an empty priority to "normal" and
// rejects anything other than the two allowed values, mirroring how
// validateScheduleSlots normalizes/validates Cadence.
func normalizeCoveragePriority(priority string) (string, error) {
	switch priority {
	case "":
		return "normal", nil
	case "normal", "optional":
		return priority, nil
	default:
		return "", fmt.Errorf("priority must be \"normal\" or \"optional\", got %q", priority)
	}
}

type coverageRequestListItem struct {
	ID                uint   `json:"id"`
	GroupID           uint   `json:"group_id"`
	RequestedByUserID uint   `json:"requested_by_user_id"`
	RequestedByName   string `json:"requested_by_name"`
	Date              string `json:"date"`
	Hour              int    `json:"hour"`
	Priority          string `json:"priority"`
	Claimable         bool   `json:"claimable"`
}

// displayName mirrors the first/last-name-with-username-fallback logic used
// for member display names elsewhere, for use in notification text.
func displayName(u models.User) string {
	if u.FirstName != "" || u.LastName != "" {
		name := u.FirstName
		if u.LastName != "" {
			if name != "" {
				name += " "
			}
			name += u.LastName
		}
		return name
	}
	return u.Username
}

// buildCoverageRequestSummary renders one requester's currently-open
// coverage requests as a single bulk notification body, so a member with
// several shifts needing coverage produces one accurate email/GroupMe post
// listing all of them instead of a separate, fragmented message per shift.
// A request's Priority softens the wording - "optional" requests read as a
// nice-to-have rather than urgent, since the whole point of the flag is to
// stop an over-staffed shift from broadcasting to the group as if it needs
// help. When every request in the list is optional, that's said once in
// the header rather than repeated on every line.
func buildCoverageRequestSummary(requesterName string, requests []models.ShiftCoverageRequest) string {
	if len(requests) == 1 {
		r := requests[0]
		if r.Priority == "optional" {
			return fmt.Sprintf("%s could use coverage for their %s shift on %s, if anyone's available.",
				requesterName, formatSlotRangeLabel(int(r.Date.Weekday()), r.Hour), r.Date.Format("Monday, January 2"))
		}
		return fmt.Sprintf("%s needs coverage for their %s shift on %s.",
			requesterName, formatSlotRangeLabel(int(r.Date.Weekday()), r.Hour), r.Date.Format("Monday, January 2"))
	}

	allOptional := true
	for _, r := range requests {
		if r.Priority != "optional" {
			allOptional = false
			break
		}
	}

	lines := make([]string, 0, len(requests))
	for _, r := range requests {
		line := fmt.Sprintf("- %s at %s", r.Date.Format("Monday, January 2"), formatSlotRangeLabel(int(r.Date.Weekday()), r.Hour))
		if r.Priority == "optional" && !allOptional {
			line += " (optional)"
		}
		lines = append(lines, line)
	}
	if allOptional {
		return fmt.Sprintf("%s could use coverage for %d shifts, if anyone's available:\n%s", requesterName, len(requests), strings.Join(lines, "\n"))
	}
	return fmt.Sprintf("%s needs coverage for %d shifts:\n%s", requesterName, len(requests), strings.Join(lines, "\n"))
}

// createOneCoverageRequest validates and creates a single coverage
// request: the date must not be in the past, the target user must have a
// matching ShiftSlot for the date's weekday and the given hour that is
// actually active that week (a biweekly slot on its off-week does not match),
// and there must not already be an active (non-cancelled) request for that
// exact date/hour. Returns the created row, or one of the sentinel errors
// errPastDate / errNoMatchingSlot / errDuplicateRequest. Runs its own
// transaction - a caller creating several requests (see
// CreateCoverageRequestsBatch) calls this once per item rather than
// wrapping the whole batch in one transaction, so one item's failure
// doesn't roll back the others.
func createOneCoverageRequest(db *gorm.DB, groupIDUint, targetUserID uint, date time.Time, hour int, priority string) (models.ShiftCoverageRequest, error) {
	today := time.Now().UTC().Truncate(24 * time.Hour)
	if date.Before(today) {
		return models.ShiftCoverageRequest{}, errPastDate
	}

	var created models.ShiftCoverageRequest
	err := db.Transaction(func(tx *gorm.DB) error {
		var slot models.ShiftSlot
		if err := tx.Where("user_id = ? AND group_id = ? AND day_of_week = ? AND hour = ?",
			targetUserID, groupIDUint, int(date.Weekday()), hour).First(&slot).Error; err != nil {
			return errNoMatchingSlot
		}
		if !slotActiveForWeek(slot.Cadence, weekStartOf(date)) {
			return errNoMatchingSlot
		}

		var existing models.ShiftCoverageRequest
		err := tx.Where("group_id = ? AND requested_by_user_id = ? AND date = ? AND hour = ? AND status != ?",
			groupIDUint, targetUserID, date, hour, models.CoverageRequestCancelled).
			First(&existing).Error
		if err == nil {
			return errDuplicateRequest
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		created = models.ShiftCoverageRequest{
			GroupID:           groupIDUint,
			RequestedByUserID: targetUserID,
			Date:              date,
			Hour:              hour,
			Status:            models.CoverageRequestOpen,
			Priority:          priority,
		}
		return tx.Create(&created).Error
	})
	return created, err
}

// notifyGroupOfOpenCoverageRequests sends one bulk notification listing
// every currently-open coverage request the given user has in the group -
// not just whatever was just created - so the group always sees the
// complete, accurate picture in a single email/GroupMe post rather than
// one fragment per request. No-ops if the user currently has no open
// requests. Runs on rawDB (the unscoped db captured before the handler's
// middleware.GetDB shadow) since callers invoke this after their own
// create(s) have already committed.
func notifyGroupOfOpenCoverageRequests(rawDB *gorm.DB, emailService *email.Service, groupMeService *groupme.Service, groupIDUint, targetUserID uint) {
	var requester models.User
	var grp models.Group
	var openRequests []models.ShiftCoverageRequest
	rawDB.First(&requester, targetUserID)
	rawDB.Select("name").First(&grp, groupIDUint)
	rawDB.Where("group_id = ? AND requested_by_user_id = ? AND status = ?",
		groupIDUint, targetUserID, models.CoverageRequestOpen).
		Order("date, hour").Find(&openRequests)
	if len(openRequests) == 0 {
		return
	}

	title := fmt.Sprintf("Coverage needed in %s", grp.Name)
	content := buildCoverageRequestSummary(displayName(requester), openRequests)

	if emailService != nil && emailService.IsConfigured() && scheduleEmailNotificationsEnabled() {
		go func() {
			bgCtx := context.Background()
			if err := sendGroupAnnouncementEmails(bgCtx, rawDB, emailService, groupIDUint, title, content); err != nil {
				logging.WithContext(bgCtx).Error("Error sending coverage request emails", err)
			}
		}()
	}
	if groupMeService != nil {
		go func() {
			bgCtx := context.Background()
			if err := sendUpdateToGroupMe(bgCtx, rawDB, groupMeService, groupIDUint, title, content); err != nil {
				logging.WithContext(bgCtx).Error("Error sending coverage request to GroupMe", err)
			}
		}()
	}
}

// CreateCoverageRequest flags a specific future occurrence of the caller's
// (or, for a group admin, another member's) recurring shift as needing
// coverage. Requires group membership (or site admin) for self-requests;
// requires group admin (or site admin) to request on behalf of someone else.
func CreateCoverageRequest(db *gorm.DB, emailService *email.Service, groupMeService *groupme.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		// rawDB is captured before the shadow below so the notification
		// goroutines get the unscoped db, not one bound to this request's
		// context (which is canceled the instant this handler returns). See
		// the same pattern in update.go's CreateUpdate.
		rawDB := db
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

		var req createCoverageRequestRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
			return
		}

		targetUserID := callerUserID
		if req.UserID != nil && *req.UserID != callerUserID {
			if !checkGroupAdminAccess(db, userID, isAdmin, groupIDParam) {
				c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required to request coverage for another member"})
				return
			}
			targetUserID = *req.UserID
		}

		date, err := time.Parse("2006-01-02", req.Date)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "date must be in YYYY-MM-DD format"})
			return
		}
		maxHour := maxHourFor(int(date.Weekday()))
		if req.Hour < 8 || req.Hour > maxHour {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("hour must be between 8 and %d for that date's weekday", maxHour)})
			return
		}

		priority, err := normalizeCoveragePriority(req.Priority)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		groupIDUint64, err := strconv.ParseUint(groupIDParam, 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid group ID"})
			return
		}
		groupIDUint := uint(groupIDUint64)

		created, err := createOneCoverageRequest(db, groupIDUint, targetUserID, date, req.Hour, priority)
		switch {
		case errors.Is(err, errPastDate):
			c.JSON(http.StatusBadRequest, gin.H{"error": errPastDate.Error()})
			return
		case errors.Is(err, errNoMatchingSlot):
			c.JSON(http.StatusBadRequest, gin.H{"error": errNoMatchingSlot.Error()})
			return
		case errors.Is(err, errDuplicateRequest):
			c.JSON(http.StatusConflict, gin.H{"error": errDuplicateRequest.Error()})
			return
		case err != nil:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create coverage request"})
			return
		}

		c.JSON(http.StatusCreated, toCoverageRequestResponse(created))

		// Cooldown: throttle (not silence) the group-wide broadcast if this
		// same user created another coverage request in this group within
		// the last few seconds, so a rapid double-submission (e.g. a network
		// retry on a slow connection) doesn't fire two near-identical emails/
		// GroupMe posts back to back. Deliberately short: a real, separate
		// request even a few seconds later should still notify normally.
		//
		// KNOWN LIMITATION: this checks "was another row created recently,"
		// not "was a notification actually sent recently." If a batch create
		// (CreateCoverageRequestsBatch, which always notifies unconditionally)
		// is immediately followed by a single create within this window, the
		// single create's notification is suppressed even though the group
		// was never told about that specific new item. A fully correct fix
		// needs a persisted "last notified at" timestamp per (user, group),
		// which is out of scope here - this short window just keeps the
		// practical blast radius small. Runs on the outer (non-transaction)
		// db since the transaction has already committed by this point,
		// matching how the notification helper below already runs post-commit.
		const coverageNotificationCooldown = 5 * time.Second
		var recentCount int64
		if err := rawDB.Model(&models.ShiftCoverageRequest{}).
			Where("group_id = ? AND requested_by_user_id = ? AND id != ? AND created_at > ?",
				groupIDUint, targetUserID, created.ID, time.Now().Add(-coverageNotificationCooldown)).
			Count(&recentCount).Error; err == nil && recentCount == 0 {
			notifyGroupOfOpenCoverageRequests(rawDB, emailService, groupMeService, groupIDUint, targetUserID)
		}
	}
}

// ClaimCoverageRequest lets any group member (other than the original
// requester) take an open coverage request, provided they don't already
// have a conflicting shift at that exact date/hour in any group. Requires
// group membership (or site admin).
func ClaimCoverageRequest(db *gorm.DB, emailService *email.Service, groupMeService *groupme.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		rawDB := db
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

		requestID, err := strconv.ParseUint(c.Param("requestId"), 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request ID"})
			return
		}

		var claimed models.ShiftCoverageRequest
		err = db.Transaction(func(tx *gorm.DB) error {
			var reqRow models.ShiftCoverageRequest
			if err := tx.Where("id = ? AND group_id = ?", requestID, groupIDParam).First(&reqRow).Error; err != nil {
				return errRequestNotFound
			}
			if reqRow.Status != models.CoverageRequestOpen {
				return errRequestNotOpen
			}
			if reqRow.RequestedByUserID == callerUserID {
				return errSelfClaim
			}

			conflict, err := hasConflictingCommitment(tx, callerUserID, reqRow.Date, reqRow.Hour)
			if err != nil {
				return err
			}
			if conflict {
				return errClaimConflict
			}

			// Conditional update, gated on status still being open, is what
			// actually closes the race between two concurrent claimants who
			// both passed the checks above under READ COMMITTED (Postgres in
			// production) - the earlier status check is just a cheap
			// fast-path rejection, not the correctness guarantee.
			now := time.Now().UTC()
			result := tx.Model(&models.ShiftCoverageRequest{}).
				Where("id = ? AND status = ?", reqRow.ID, models.CoverageRequestOpen).
				Updates(map[string]interface{}{
					"status":             models.CoverageRequestClaimed,
					"claimed_by_user_id": callerUserID,
					"claimed_at":         now,
				})
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected == 0 {
				return errRequestNotOpen
			}

			reqRow.Status = models.CoverageRequestClaimed
			reqRow.ClaimedByUserID = &callerUserID
			reqRow.ClaimedAt = &now
			claimed = reqRow
			return nil
		})

		switch {
		case errors.Is(err, errRequestNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": errRequestNotFound.Error()})
			return
		case errors.Is(err, errRequestNotOpen):
			c.JSON(http.StatusConflict, gin.H{"error": errRequestNotOpen.Error()})
			return
		case errors.Is(err, errSelfClaim):
			c.JSON(http.StatusBadRequest, gin.H{"error": errSelfClaim.Error()})
			return
		case errors.Is(err, errClaimConflict):
			c.JSON(http.StatusConflict, gin.H{"error": errClaimConflict.Error()})
			return
		case err != nil:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to claim coverage request"})
			return
		}

		c.JSON(http.StatusOK, toCoverageRequestResponse(claimed))

		notifyRequesterOfClaim(rawDB, emailService, groupMeService, claimed, callerUserID)
	}
}

// notifyRequesterOfClaim tells the original requester their shift is
// covered. Runs in the background so it never delays the HTTP response.
func notifyRequesterOfClaim(db *gorm.DB, emailService *email.Service, groupMeService *groupme.Service, req models.ShiftCoverageRequest, claimantID uint) {
	if emailService == nil && groupMeService == nil {
		return
	}
	go func() {
		bgCtx := context.Background()
		logger := logging.WithContext(bgCtx)

		var requester, claimant models.User
		if err := db.WithContext(bgCtx).First(&requester, req.RequestedByUserID).Error; err != nil {
			logger.Error("Failed to load requester for coverage claim notification", err)
			return
		}
		if err := db.WithContext(bgCtx).First(&claimant, claimantID).Error; err != nil {
			logger.Error("Failed to load claimant for coverage claim notification", err)
			return
		}

		title := "Your shift is covered"
		content := fmt.Sprintf("%s will cover your %s shift on %s.",
			displayName(claimant), formatSlotRangeLabel(int(req.Date.Weekday()), req.Hour), req.Date.Format("Monday, January 2"))

		if emailService != nil && emailService.IsConfigured() && requester.EmailNotificationsEnabled && scheduleEmailNotificationsEnabled() {
			if err := emailService.SendAnnouncementEmail(bgCtx, requester.Email, title, content); err != nil {
				logger.Error("Failed to send coverage claim email", err)
			}
		}
		if groupMeService != nil {
			if err := sendUpdateToGroupMe(bgCtx, db, groupMeService, req.GroupID, title, content); err != nil {
				logger.Error("Failed to send coverage claim GroupMe message", err)
			}
		}
	}()
}

const maxReassignHours = 20

type reassignShiftsBatchRequest struct {
	FromUserID uint   `json:"from_user_id"`
	ToUserID   uint   `json:"to_user_id"`
	Date       string `json:"date"`
	Hours      []int  `json:"hours"`
	// Notify lets the admin skip the notification emails/GroupMe post for a
	// change already agreed in person - a *bool (not bool) so an omitted
	// field defaults to true (notify) rather than false, which a plain
	// bool's zero value would silently do.
	Notify *bool `json:"notify"`
}

type reassignShiftsBatchSkipped struct {
	Hour   int    `json:"hour"`
	Reason string `json:"reason"`
}

type reassignShiftsBatchResponse struct {
	Created []coverageRequestResponse    `json:"created"`
	Skipped []reassignShiftsBatchSkipped `json:"skipped"`
}

// ReassignShiftsBatch lets a group admin directly swap who's covering
// several of a specific date's shifts (e.g. a whole morning) in one step -
// changes already agreed in person, which would otherwise need the
// original volunteer to request coverage and the replacement to
// separately claim it, once per hour. Each hour is validated and created
// independently in the claimed state directly (never a visible "open" one,
// so GetGroupScheduleOverview reflects the swap immediately with no
// additional wiring) - a conflict on one hour doesn't block the others,
// mirroring CreateCoverageRequestsBatch - but exactly one notification per
// person covers every successfully-reassigned hour. Requires group admin
// (or site admin) access.
func ReassignShiftsBatch(db *gorm.DB, emailService *email.Service, groupMeService *groupme.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		rawDB := db
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

		var req reassignShiftsBatchRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
			return
		}
		if req.ToUserID == req.FromUserID {
			c.JSON(http.StatusBadRequest, gin.H{"error": errReassignSameUser.Error()})
			return
		}
		if len(req.Hours) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "hours must not be empty"})
			return
		}
		if len(req.Hours) > maxReassignHours {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("hours must not exceed %d items", maxReassignHours)})
			return
		}

		date, err := time.Parse("2006-01-02", req.Date)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "date must be in YYYY-MM-DD format"})
			return
		}
		today := time.Now().UTC().Truncate(24 * time.Hour)
		if date.Before(today) {
			c.JSON(http.StatusBadRequest, gin.H{"error": errPastDate.Error()})
			return
		}
		maxHour := maxHourFor(int(date.Weekday()))
		seenHours := make(map[int]struct{}, len(req.Hours))
		for _, h := range req.Hours {
			if h < 8 || h > maxHour {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("hour must be between 8 and %d for that date's weekday", maxHour)})
				return
			}
			if _, dup := seenHours[h]; dup {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("duplicate hour %d in request", h)})
				return
			}
			seenHours[h] = struct{}{}
		}

		groupIDUint64, err := strconv.ParseUint(groupIDParam, 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid group ID"})
			return
		}
		groupIDUint := uint(groupIDUint64)

		var membership models.UserGroup
		if err := db.Where("user_id = ? AND group_id = ?", req.ToUserID, groupIDUint).First(&membership).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": errNotGroupMember.Error()})
			return
		}

		response := reassignShiftsBatchResponse{
			Created: make([]coverageRequestResponse, 0, len(req.Hours)),
			Skipped: make([]reassignShiftsBatchSkipped, 0),
		}
		successfulHours := make([]int, 0, len(req.Hours))

		for _, hour := range req.Hours {
			var created models.ShiftCoverageRequest
			err := db.Transaction(func(tx *gorm.DB) error {
				var slot models.ShiftSlot
				if err := tx.Where("user_id = ? AND group_id = ? AND day_of_week = ? AND hour = ?",
					req.FromUserID, groupIDUint, int(date.Weekday()), hour).First(&slot).Error; err != nil {
					return errNoMatchingSlot
				}
				if !slotActiveForWeek(slot.Cadence, weekStartOf(date)) {
					return errNoMatchingSlot
				}

				var existing models.ShiftCoverageRequest
				err := tx.Where("group_id = ? AND requested_by_user_id = ? AND date = ? AND hour = ? AND status != ?",
					groupIDUint, req.FromUserID, date, hour, models.CoverageRequestCancelled).
					First(&existing).Error
				if err == nil {
					return errDuplicateRequest
				}
				if !errors.Is(err, gorm.ErrRecordNotFound) {
					return err
				}

				conflict, err := hasConflictingCommitment(tx, req.ToUserID, date, hour)
				if err != nil {
					return err
				}
				if conflict {
					return errClaimConflict
				}

				now := time.Now().UTC()
				toUserID := req.ToUserID
				created = models.ShiftCoverageRequest{
					GroupID:           groupIDUint,
					RequestedByUserID: req.FromUserID,
					Date:              date,
					Hour:              hour,
					Status:            models.CoverageRequestClaimed,
					Priority:          "normal",
					ClaimedByUserID:   &toUserID,
					ClaimedAt:         &now,
				}
				return tx.Create(&created).Error
			})

			switch {
			case errors.Is(err, errNoMatchingSlot), errors.Is(err, errDuplicateRequest), errors.Is(err, errClaimConflict):
				response.Skipped = append(response.Skipped, reassignShiftsBatchSkipped{Hour: hour, Reason: err.Error()})
			case err != nil:
				logging.WithContext(c.Request.Context()).WithFields(map[string]interface{}{
					"group_id": groupIDUint,
					"date":     date.Format("2006-01-02"),
					"hour":     hour,
				}).Error("Failed to reassign shift in batch", err)
				response.Skipped = append(response.Skipped, reassignShiftsBatchSkipped{Hour: hour, Reason: "internal error, please try again"})
			default:
				response.Created = append(response.Created, toCoverageRequestResponse(created))
				successfulHours = append(successfulHours, hour)
			}
		}

		c.JSON(http.StatusOK, response)

		notify := req.Notify == nil || *req.Notify
		if notify && len(successfulHours) > 0 {
			notifyOfReassignmentBatch(rawDB, emailService, groupMeService, groupIDUint, req.FromUserID, req.ToUserID, date, successfulHours)
		}
	}
}

// notifyOfReassignmentBatch tells both volunteers involved in an
// admin-arranged reassignment about every hour that was successfully
// reassigned in one message, so a multi-hour swap (e.g. a whole morning)
// sends exactly one email per person instead of one per hour. Posts one
// GroupMe message to the group (not one per recipient, unlike the two
// emails) since GroupMe is a shared channel. Runs in the background so it
// never delays the HTTP response.
func notifyOfReassignmentBatch(db *gorm.DB, emailService *email.Service, groupMeService *groupme.Service, groupID, fromUserID, toUserID uint, date time.Time, hours []int) {
	if emailService == nil && groupMeService == nil {
		return
	}
	if len(hours) == 0 {
		return
	}
	go func() {
		bgCtx := context.Background()
		logger := logging.WithContext(bgCtx)

		var from, to models.User
		if err := db.WithContext(bgCtx).First(&from, fromUserID).Error; err != nil {
			logger.Error("Failed to load original volunteer for reassignment notification", err)
			return
		}
		if err := db.WithContext(bgCtx).First(&to, toUserID).Error; err != nil {
			logger.Error("Failed to load new volunteer for reassignment notification", err)
			return
		}

		sortedHours := append([]int(nil), hours...)
		sort.Ints(sortedHours)
		hoursLabel := formatReassignedHoursSummary(int(date.Weekday()), sortedHours)
		shiftLabel := fmt.Sprintf("%s shift on %s", hoursLabel, date.Format("Monday, January 2"))
		fromTitle := "Your shift was reassigned"
		fromContent := fmt.Sprintf("%s will now cover your %s.", displayName(to), shiftLabel)
		toTitle := "You've been assigned a shift"
		toContent := fmt.Sprintf("You're now covering %s's %s.", displayName(from), shiftLabel)

		if emailService != nil && emailService.IsConfigured() && scheduleEmailNotificationsEnabled() {
			if from.EmailNotificationsEnabled {
				if err := emailService.SendAnnouncementEmail(bgCtx, from.Email, fromTitle, fromContent); err != nil {
					logger.Error("Failed to send reassignment email to original volunteer", err)
				}
			}
			if to.EmailNotificationsEnabled {
				if err := emailService.SendAnnouncementEmail(bgCtx, to.Email, toTitle, toContent); err != nil {
					logger.Error("Failed to send reassignment email to new volunteer", err)
				}
			}
		}
		if groupMeService != nil {
			groupContent := fmt.Sprintf("%s's %s has been reassigned to %s.", displayName(from), shiftLabel, displayName(to))
			if err := sendUpdateToGroupMe(bgCtx, db, groupMeService, groupID, "Shift reassigned", groupContent); err != nil {
				logger.Error("Failed to send reassignment GroupMe message", err)
			}
		}
	}()
}

// claimConflictKey keys the claimKeys map loadUserConflictKeys returns:
// date+hour for an already-claimed ShiftCoverageRequest.
func claimConflictKey(date time.Time, hour int) string {
	return fmt.Sprintf("%s-%d", date.Format("2006-01-02"), hour)
}

// loadUserConflictKeys loads every recurring ShiftSlot and already-claimed
// ShiftCoverageRequest userID has (in any group - a real-world time
// conflict doesn't respect group boundaries). Slots are returned as-is
// (not pre-keyed) since whether a slot actually conflicts with a specific
// coverage request depends on that request's date (biweekly parity), not
// just its weekday/hour. claimKeys stays a flat set since a claimed
// ShiftCoverageRequest is inherently tied to one exact date already.
func loadUserConflictKeys(db *gorm.DB, userID uint) (slots []models.ShiftSlot, claimKeys map[string]struct{}, err error) {
	if err := db.Where("user_id = ?", userID).Find(&slots).Error; err != nil {
		return nil, nil, err
	}

	var claimed []models.ShiftCoverageRequest
	if err := db.Where("claimed_by_user_id = ? AND status = ?", userID, models.CoverageRequestClaimed).Find(&claimed).Error; err != nil {
		return nil, nil, err
	}
	claimKeys = make(map[string]struct{}, len(claimed))
	for _, r := range claimed {
		claimKeys[claimConflictKey(r.Date, r.Hour)] = struct{}{}
	}
	return slots, claimKeys, nil
}

// isRequestClaimableGiven reports whether userID could claim req right
// now, given userID's own conflict data (see loadUserConflictKeys): not
// their own request, and no conflicting ShiftSlot (active on req's date's
// week) or already-claimed coverage request at that exact date/hour.
func isRequestClaimableGiven(req models.ShiftCoverageRequest, userID uint, slots []models.ShiftSlot, claimKeys map[string]struct{}) bool {
	if req.RequestedByUserID == userID {
		return false
	}
	weekStart := weekStartOf(req.Date)
	for _, s := range slots {
		if s.DayOfWeek == int(req.Date.Weekday()) && s.Hour == req.Hour && slotActiveForWeek(s.Cadence, weekStart) {
			return false
		}
	}
	_, conflict := claimKeys[claimConflictKey(req.Date, req.Hour)]
	return !conflict
}

// ListCoverageRequests returns every currently-open, not-yet-past coverage
// request in the group, soonest first, so members can see at a glance what
// still needs a volunteer without having to spot it in the schedule
// overview heatmap. Each item is annotated with whether the viewer could
// claim it, so the frontend can disable the Claim button without a second
// round trip. Requires group membership (or site admin).
func ListCoverageRequests(db *gorm.DB) gin.HandlerFunc {
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

		today := time.Now().UTC().Truncate(24 * time.Hour)
		var requests []models.ShiftCoverageRequest
		if err := db.Preload("RequestedByUser").
			Where("group_id = ? AND status = ? AND date >= ?", groupIDParam, models.CoverageRequestOpen, today).
			Order("date, hour").
			Find(&requests).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load coverage requests"})
			return
		}

		slots, claimKeys, err := loadUserConflictKeys(db, callerUserID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load coverage requests"})
			return
		}

		items := make([]coverageRequestListItem, 0, len(requests))
		for _, r := range requests {
			claimable := isRequestClaimableGiven(r, callerUserID, slots, claimKeys)
			items = append(items, coverageRequestListItem{
				ID:                r.ID,
				GroupID:           r.GroupID,
				RequestedByUserID: r.RequestedByUserID,
				RequestedByName:   displayName(r.RequestedByUser),
				Date:              r.Date.Format("2006-01-02"),
				Hour:              r.Hour,
				Priority:          r.Priority,
				Claimable:         claimable,
			})
		}

		c.JSON(http.StatusOK, items)
	}
}

// CancelCoverageRequest withdraws a coverage request. The original
// requester can cancel it only while still open; a group admin (or site
// admin) can cancel it at any status.
func CancelCoverageRequest(db *gorm.DB) gin.HandlerFunc {
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

		requestID, err := strconv.ParseUint(c.Param("requestId"), 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request ID"})
			return
		}

		var reqRow models.ShiftCoverageRequest
		if err := db.Where("id = ? AND group_id = ?", requestID, groupIDParam).First(&reqRow).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Coverage request not found"})
			return
		}

		// State check first: an already-cancelled request is a 409 conflict
		// for anyone, not an authorization failure for the owner. Checking
		// this before the ownership/admin gate matches the state-before-
		// identity ordering ClaimCoverageRequest uses above (errRequestNotOpen
		// before errSelfClaim) - otherwise a non-admin owner whose own
		// already-cancelled request they try to cancel again would fail the
		// "is it still open" authorization check and get a misleading 403.
		if reqRow.Status == models.CoverageRequestCancelled {
			c.JSON(http.StatusConflict, gin.H{"error": "Coverage request is already cancelled"})
			return
		}

		isAdminCaller := checkGroupAdminAccess(db, userID, isAdmin, groupIDParam)
		isOwnOpenRequest := reqRow.RequestedByUserID == callerUserID && reqRow.Status == models.CoverageRequestOpen
		// isOwnClaim: the volunteer currently covering this shift backing out
		// of their own claim (e.g. something came up and they can no longer
		// make it) - distinct from isOwnOpenRequest, which is the original
		// requester withdrawing their own still-open ask.
		isOwnClaim := reqRow.Status == models.CoverageRequestClaimed &&
			reqRow.ClaimedByUserID != nil && *reqRow.ClaimedByUserID == callerUserID
		if !isOwnOpenRequest && !isOwnClaim && !isAdminCaller {
			c.JSON(http.StatusForbidden, gin.H{"error": "Cannot cancel this coverage request"})
			return
		}

		// Conditional update, gated on the state actually authorized above,
		// closes the race with a concurrent claim (or cancel) landing between
		// the read above and this write - same technique as the conditional
		// update in ClaimCoverageRequest. A non-admin owner may only flip
		// open -> cancelled; a non-admin claimant may only flip their own
		// claimed -> cancelled; an admin may flip open or claimed ->
		// cancelled regardless of who owns/claimed it.
		var result *gorm.DB
		switch {
		case isAdminCaller:
			result = db.Model(&models.ShiftCoverageRequest{}).
				Where("id = ? AND status IN ?", reqRow.ID, []models.CoverageRequestStatus{models.CoverageRequestOpen, models.CoverageRequestClaimed}).
				Update("status", models.CoverageRequestCancelled)
		case isOwnClaim:
			result = db.Model(&models.ShiftCoverageRequest{}).
				Where("id = ? AND claimed_by_user_id = ? AND status = ?", reqRow.ID, callerUserID, models.CoverageRequestClaimed).
				Update("status", models.CoverageRequestCancelled)
		default:
			result = db.Model(&models.ShiftCoverageRequest{}).
				Where("id = ? AND requested_by_user_id = ? AND status = ?", reqRow.ID, callerUserID, models.CoverageRequestOpen).
				Update("status", models.CoverageRequestCancelled)
		}
		if result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to cancel coverage request"})
			return
		}
		if result.RowsAffected == 0 {
			// Someone else changed the request's state between our read and
			// this write. For an admin, the only remaining state given the
			// (open, claimed) guard is "already cancelled". For a non-admin
			// owner or claimant, it most likely means someone else (an admin,
			// or - for the owner's case - a new claimant) changed it first -
			// either way it's no longer cancellable by them.
			if isAdminCaller {
				c.JSON(http.StatusConflict, gin.H{"error": "Coverage request is already cancelled"})
			} else {
				c.JSON(http.StatusConflict, gin.H{"error": "Coverage request is no longer open"})
			}
			return
		}

		reqRow.Status = models.CoverageRequestCancelled
		c.JSON(http.StatusOK, toCoverageRequestResponse(reqRow))
	}
}

type updateCoverageRequestPriorityRequest struct {
	Priority string `json:"priority"`
}

// UpdateCoverageRequestPriority lets a group admin (or site admin) override
// a coverage request's priority after creation - e.g. downgrading a
// request to "optional" once a shift turns out to already have enough
// coverage. Requires group admin access; a regular member, including the
// original requester, cannot change it themselves.
func UpdateCoverageRequestPriority(db *gorm.DB) gin.HandlerFunc {
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

		requestID, err := strconv.ParseUint(c.Param("requestId"), 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request ID"})
			return
		}

		var req updateCoverageRequestPriorityRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
			return
		}
		// Unlike creation, an explicit override with no priority is
		// rejected rather than defaulting to "normal" - normalizeCoveragePriority
		// alone would silently default an omitted field, which is the wrong
		// contract for a call whose entire purpose is to set the value.
		if req.Priority == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "priority is required"})
			return
		}
		priority, err := normalizeCoveragePriority(req.Priority)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var reqRow models.ShiftCoverageRequest
		if err := db.Where("id = ? AND group_id = ?", requestID, groupIDParam).First(&reqRow).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Coverage request not found"})
			return
		}

		if err := db.Model(&reqRow).Update("priority", priority).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update coverage request priority"})
			return
		}
		reqRow.Priority = priority

		c.JSON(http.StatusOK, toCoverageRequestResponse(reqRow))
	}
}

type createCoverageRequestBatchItem struct {
	Date string `json:"date"`
	Hour int    `json:"hour"`
	// Priority, when set, overrides the batch-level default for this item
	// alone - e.g. "I need help Sat, Sun and Tue, but Sat is optional".
	// Empty falls back to the batch's top-level Priority.
	Priority string `json:"priority"`
}

type createCoverageRequestBatchRequest struct {
	Requests []createCoverageRequestBatchItem `json:"requests"`
	Priority string                           `json:"priority"`
}

type coverageRequestBatchSkipped struct {
	Date   string `json:"date"`
	Hour   int    `json:"hour"`
	Reason string `json:"reason"`
}

type coverageRequestBatchResponse struct {
	Created []coverageRequestResponse     `json:"created"`
	Skipped []coverageRequestBatchSkipped `json:"skipped"`
}

// CreateCoverageRequestsBatch flags several future occurrences of the
// caller's own recurring shifts as needing coverage in one call, so a
// volunteer requesting coverage for multiple shifts (e.g. "I'm out all of
// next week") triggers exactly one group notification instead of one per
// shift. Self-service only - always creates on behalf of the caller, no
// on-behalf-of-another-member support (unlike CreateCoverageRequest).
// Each item may set its own Priority (e.g. "Sat is optional, Sun and Tue
// aren't") - an item that omits it falls back to the request's top-level
// Priority.
// Items that fail per-item validation (no matching slot, already an
// active request, past date) are skipped and reported rather than failing
// the whole batch; a structurally invalid item (bad date format,
// out-of-range hour) fails the whole request with 400, since that's a
// payload error, not a per-item business-rule rejection. Requires group
// membership (or site admin).
func CreateCoverageRequestsBatch(db *gorm.DB, emailService *email.Service, groupMeService *groupme.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		rawDB := db
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

		var req createCoverageRequestBatchRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
			return
		}
		if len(req.Requests) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "requests must not be empty"})
			return
		}
		const maxBatchItems = 200
		if len(req.Requests) > maxBatchItems {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("requests must not exceed %d items", maxBatchItems)})
			return
		}
		priority, err := normalizeCoveragePriority(req.Priority)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		groupIDUint64, err := strconv.ParseUint(groupIDParam, 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid group ID"})
			return
		}
		groupIDUint := uint(groupIDUint64)

		type parsedItem struct {
			date     time.Time
			hour     int
			priority string
		}
		parsedItems := make([]parsedItem, 0, len(req.Requests))
		for _, item := range req.Requests {
			date, err := time.Parse("2006-01-02", item.Date)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid date %q: must be in YYYY-MM-DD format", item.Date)})
				return
			}
			maxHour := maxHourFor(int(date.Weekday()))
			if item.Hour < 8 || item.Hour > maxHour {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid hour %d: must be between 8 and %d for %s", item.Hour, maxHour, item.Date)})
				return
			}
			itemPriority := priority
			if item.Priority != "" {
				itemPriority, err = normalizeCoveragePriority(item.Priority)
				if err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
			}
			parsedItems = append(parsedItems, parsedItem{date: date, hour: item.Hour, priority: itemPriority})
		}

		response := coverageRequestBatchResponse{
			Created: make([]coverageRequestResponse, 0, len(parsedItems)),
			Skipped: make([]coverageRequestBatchSkipped, 0),
		}
		for _, item := range parsedItems {
			created, err := createOneCoverageRequest(db, groupIDUint, callerUserID, item.date, item.hour, item.priority)
			switch {
			case errors.Is(err, errPastDate), errors.Is(err, errNoMatchingSlot), errors.Is(err, errDuplicateRequest):
				response.Skipped = append(response.Skipped, coverageRequestBatchSkipped{
					Date: item.date.Format("2006-01-02"), Hour: item.hour, Reason: err.Error(),
				})
			case err != nil:
				logging.WithContext(c.Request.Context()).WithFields(map[string]interface{}{
					"group_id": groupIDUint,
					"date":     item.date.Format("2006-01-02"),
					"hour":     item.hour,
				}).Error("Failed to create coverage request in batch", err)
				response.Skipped = append(response.Skipped, coverageRequestBatchSkipped{
					Date: item.date.Format("2006-01-02"), Hour: item.hour, Reason: "internal error, please try again",
				})
			default:
				response.Created = append(response.Created, toCoverageRequestResponse(created))
			}
		}

		c.JSON(http.StatusOK, response)

		if len(response.Created) > 0 {
			notifyGroupOfOpenCoverageRequests(rawDB, emailService, groupMeService, groupIDUint, callerUserID)
		}
	}
}

type cancelCoverageRequestsBatchRequest struct {
	RequestIDs []uint `json:"request_ids"`
}

type coverageRequestCancelBatchSkipped struct {
	ID     uint   `json:"id"`
	Reason string `json:"reason"`
}

type coverageRequestCancelBatchResponse struct {
	Cancelled []coverageRequestResponse           `json:"cancelled"`
	Skipped   []coverageRequestCancelBatchSkipped `json:"skipped"`
}

// CancelCoverageRequestsBatch cancels several of the caller's own open
// coverage requests in one call, so withdrawing from a whole date range
// (e.g. "I requested coverage for two weeks but I'm back early") doesn't
// require cancelling one shift at a time. A group admin may also bulk-
// cancel other members' open requests (e.g. clearing stale requests for a
// member who left). Deliberately open-status only, for both self and
// admin - once a request is claimed, withdrawing it is a one-at-a-time
// action elsewhere (CancelCoverageRequest via the schedule overview), not
// a bulk one, since it means un-committing another volunteer who already
// stepped up. Per-item failures (not found, not open, not authorized) are
// skipped and reported rather than failing the whole batch, matching
// CreateCoverageRequestsBatch's partial-success shape.
func CancelCoverageRequestsBatch(db *gorm.DB) gin.HandlerFunc {
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

		var req cancelCoverageRequestsBatchRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
			return
		}
		if len(req.RequestIDs) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "request_ids must not be empty"})
			return
		}
		const maxBatchItems = 200
		if len(req.RequestIDs) > maxBatchItems {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("request_ids must not exceed %d items", maxBatchItems)})
			return
		}

		isAdminCaller := checkGroupAdminAccess(db, userID, isAdmin, groupIDParam)

		response := coverageRequestCancelBatchResponse{
			Cancelled: make([]coverageRequestResponse, 0, len(req.RequestIDs)),
			Skipped:   make([]coverageRequestCancelBatchSkipped, 0),
		}
		for _, requestID := range req.RequestIDs {
			var reqRow models.ShiftCoverageRequest
			if err := db.Where("id = ? AND group_id = ?", requestID, groupIDParam).First(&reqRow).Error; err != nil {
				response.Skipped = append(response.Skipped, coverageRequestCancelBatchSkipped{ID: requestID, Reason: "coverage request not found"})
				continue
			}

			isOwnRequest := reqRow.RequestedByUserID == callerUserID
			if !isOwnRequest && !isAdminCaller {
				response.Skipped = append(response.Skipped, coverageRequestCancelBatchSkipped{ID: requestID, Reason: "not authorized to cancel this request"})
				continue
			}
			if reqRow.Status != models.CoverageRequestOpen {
				reason := "coverage request is no longer open"
				switch reqRow.Status {
				case models.CoverageRequestClaimed:
					reason = "coverage request has already been claimed"
				case models.CoverageRequestCancelled:
					reason = "coverage request is already cancelled"
				}
				response.Skipped = append(response.Skipped, coverageRequestCancelBatchSkipped{ID: requestID, Reason: reason})
				continue
			}

			// Conditional update gated on status = open closes the race with a
			// concurrent claim/cancel landing between the read above and this
			// write, same technique as the single-item CancelCoverageRequest.
			result := db.Model(&models.ShiftCoverageRequest{}).
				Where("id = ? AND status = ?", reqRow.ID, models.CoverageRequestOpen).
				Update("status", models.CoverageRequestCancelled)
			if result.Error != nil {
				logging.WithContext(c.Request.Context()).WithFields(map[string]interface{}{
					"group_id":   groupIDParam,
					"request_id": requestID,
				}).Error("Failed to cancel coverage request in batch", result.Error)
				response.Skipped = append(response.Skipped, coverageRequestCancelBatchSkipped{ID: requestID, Reason: "internal error, please try again"})
				continue
			}
			if result.RowsAffected == 0 {
				// Someone else changed the request's state between our read
				// above and this write (a concurrent claim or cancel) - same
				// race window CancelCoverageRequest closes for the single-item
				// case.
				response.Skipped = append(response.Skipped, coverageRequestCancelBatchSkipped{ID: requestID, Reason: "coverage request is no longer open"})
				continue
			}

			reqRow.Status = models.CoverageRequestCancelled
			response.Cancelled = append(response.Cancelled, toCoverageRequestResponse(reqRow))
		}

		c.JSON(http.StatusOK, response)
	}
}

type claimCoverageRequestsBatchRequest struct {
	RequestIDs []uint `json:"request_ids"`
}

type coverageRequestClaimBatchSkipped struct {
	ID     uint   `json:"id"`
	Reason string `json:"reason"`
}

type coverageRequestClaimBatchResponse struct {
	Claimed []coverageRequestResponse          `json:"claimed"`
	Skipped []coverageRequestClaimBatchSkipped `json:"skipped"`
}

// ClaimCoverageRequestsBatch lets a member take several open coverage
// requests in one call, so covering a whole run of open shifts (e.g. a
// stretch someone dropped) doesn't require claiming them one at a time.
// Each item is validated and claimed independently - not found, not open,
// self-claim, or a conflicting commitment are skipped and reported rather
// than failing the whole batch, matching CancelCoverageRequestsBatch's
// partial-success shape - and every successful claim still goes through
// the same race-safe conditional update as the single-item
// ClaimCoverageRequest. Requires group membership (or site admin).
func ClaimCoverageRequestsBatch(db *gorm.DB, emailService *email.Service, groupMeService *groupme.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		rawDB := db
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

		var req claimCoverageRequestsBatchRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
			return
		}
		if len(req.RequestIDs) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "request_ids must not be empty"})
			return
		}
		const maxBatchItems = 200
		if len(req.RequestIDs) > maxBatchItems {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("request_ids must not exceed %d items", maxBatchItems)})
			return
		}

		response := coverageRequestClaimBatchResponse{
			Claimed: make([]coverageRequestResponse, 0, len(req.RequestIDs)),
			Skipped: make([]coverageRequestClaimBatchSkipped, 0),
		}
		claimedByRequester := make(map[uint][]models.ShiftCoverageRequest)

		for _, requestID := range req.RequestIDs {
			var claimed models.ShiftCoverageRequest
			err := db.Transaction(func(tx *gorm.DB) error {
				var reqRow models.ShiftCoverageRequest
				if err := tx.Where("id = ? AND group_id = ?", requestID, groupIDParam).First(&reqRow).Error; err != nil {
					return errRequestNotFound
				}
				if reqRow.Status != models.CoverageRequestOpen {
					return errRequestNotOpen
				}
				if reqRow.RequestedByUserID == callerUserID {
					return errSelfClaim
				}

				conflict, err := hasConflictingCommitment(tx, callerUserID, reqRow.Date, reqRow.Hour)
				if err != nil {
					return err
				}
				if conflict {
					return errClaimConflict
				}

				now := time.Now().UTC()
				result := tx.Model(&models.ShiftCoverageRequest{}).
					Where("id = ? AND status = ?", reqRow.ID, models.CoverageRequestOpen).
					Updates(map[string]interface{}{
						"status":             models.CoverageRequestClaimed,
						"claimed_by_user_id": callerUserID,
						"claimed_at":         now,
					})
				if result.Error != nil {
					return result.Error
				}
				if result.RowsAffected == 0 {
					return errRequestNotOpen
				}

				reqRow.Status = models.CoverageRequestClaimed
				reqRow.ClaimedByUserID = &callerUserID
				reqRow.ClaimedAt = &now
				claimed = reqRow
				return nil
			})

			switch {
			case errors.Is(err, errRequestNotFound):
				response.Skipped = append(response.Skipped, coverageRequestClaimBatchSkipped{ID: requestID, Reason: errRequestNotFound.Error()})
			case errors.Is(err, errRequestNotOpen):
				response.Skipped = append(response.Skipped, coverageRequestClaimBatchSkipped{ID: requestID, Reason: errRequestNotOpen.Error()})
			case errors.Is(err, errSelfClaim):
				response.Skipped = append(response.Skipped, coverageRequestClaimBatchSkipped{ID: requestID, Reason: errSelfClaim.Error()})
			case errors.Is(err, errClaimConflict):
				response.Skipped = append(response.Skipped, coverageRequestClaimBatchSkipped{ID: requestID, Reason: errClaimConflict.Error()})
			case err != nil:
				logging.WithContext(c.Request.Context()).WithFields(map[string]interface{}{
					"group_id":   groupIDParam,
					"request_id": requestID,
				}).Error("Failed to claim coverage request in batch", err)
				response.Skipped = append(response.Skipped, coverageRequestClaimBatchSkipped{ID: requestID, Reason: "internal error, please try again"})
			default:
				response.Claimed = append(response.Claimed, toCoverageRequestResponse(claimed))
				claimedByRequester[claimed.RequestedByUserID] = append(claimedByRequester[claimed.RequestedByUserID], claimed)
			}
		}

		c.JSON(http.StatusOK, response)

		for requesterID, claims := range claimedByRequester {
			notifyRequesterOfClaimBatch(rawDB, emailService, groupMeService, requesterID, callerUserID, claims)
		}
	}
}

// notifyRequesterOfClaimBatch tells one original requester their shifts are
// covered, summarizing every shift claimed from them in this batch in a
// single message - so a claimant covering several of the same person's
// shifts in one bulk action sends exactly one notification to that person,
// not one per shift, mirroring notifyOfReassignmentBatch's per-recipient
// batching for ReassignShiftsBatch. Runs in the background so it never
// delays the HTTP response.
func notifyRequesterOfClaimBatch(db *gorm.DB, emailService *email.Service, groupMeService *groupme.Service, requesterID, claimantID uint, claims []models.ShiftCoverageRequest) {
	if emailService == nil && groupMeService == nil {
		return
	}
	if len(claims) == 0 {
		return
	}
	go func() {
		bgCtx := context.Background()
		logger := logging.WithContext(bgCtx)

		var requester, claimant models.User
		if err := db.WithContext(bgCtx).First(&requester, requesterID).Error; err != nil {
			logger.Error("Failed to load requester for coverage claim batch notification", err)
			return
		}
		if err := db.WithContext(bgCtx).First(&claimant, claimantID).Error; err != nil {
			logger.Error("Failed to load claimant for coverage claim batch notification", err)
			return
		}

		sorted := append([]models.ShiftCoverageRequest(nil), claims...)
		sort.Slice(sorted, func(i, j int) bool {
			if !sorted[i].Date.Equal(sorted[j].Date) {
				return sorted[i].Date.Before(sorted[j].Date)
			}
			return sorted[i].Hour < sorted[j].Hour
		})
		shiftLabels := make([]string, 0, len(sorted))
		for _, claim := range sorted {
			shiftLabels = append(shiftLabels, fmt.Sprintf("%s on %s",
				formatSlotRangeLabel(int(claim.Date.Weekday()), claim.Hour), claim.Date.Format("Monday, January 2")))
		}

		title := "Your shifts are covered"
		verb := "shift is"
		if len(shiftLabels) > 1 {
			verb = "shifts are"
		}
		content := fmt.Sprintf("%s will cover your %s: %s.", displayName(claimant), verb, strings.Join(shiftLabels, "; "))

		groupID := sorted[0].GroupID
		if emailService != nil && emailService.IsConfigured() && requester.EmailNotificationsEnabled && scheduleEmailNotificationsEnabled() {
			if err := emailService.SendAnnouncementEmail(bgCtx, requester.Email, title, content); err != nil {
				logger.Error("Failed to send coverage claim batch email", err)
			}
		}
		if groupMeService != nil {
			if err := sendUpdateToGroupMe(bgCtx, db, groupMeService, groupID, title, content); err != nil {
				logger.Error("Failed to send coverage claim batch GroupMe message", err)
			}
		}
	}()
}

// ReopenCoverageRequest lets the current claimant of a claimed coverage
// request put it back into the open pool - e.g. they claimed a shift but
// something came up and they can no longer cover it, and rather than
// silently going back to the original requester (that's CancelCoverageRequest's
// job), they want ANY other member to be able to pick it up. The
// requested_by_user_id is deliberately left unchanged: the original
// requester still can't self-claim it (same as before it was ever
// claimed), but it reappears in their name on the Needs Coverage list and
// the schedule overview exactly like a freshly-created request would, so
// no other code path needs to know the difference. A group admin may also
// reopen a claim on the claimant's behalf (e.g. an unresponsive volunteer).
func ReopenCoverageRequest(db *gorm.DB, emailService *email.Service, groupMeService *groupme.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		rawDB := db
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

		requestID, err := strconv.ParseUint(c.Param("requestId"), 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request ID"})
			return
		}

		var reqRow models.ShiftCoverageRequest
		if err := db.Where("id = ? AND group_id = ?", requestID, groupIDParam).First(&reqRow).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": errRequestNotFound.Error()})
			return
		}

		if reqRow.Status == models.CoverageRequestCancelled {
			c.JSON(http.StatusConflict, gin.H{"error": "Coverage request is already cancelled"})
			return
		}
		if reqRow.Status == models.CoverageRequestOpen {
			c.JSON(http.StatusConflict, gin.H{"error": errRequestNotClaimed.Error()})
			return
		}

		isAdminCaller := checkGroupAdminAccess(db, userID, isAdmin, groupIDParam)
		isOwnClaim := reqRow.ClaimedByUserID != nil && *reqRow.ClaimedByUserID == callerUserID
		if !isOwnClaim && !isAdminCaller {
			c.JSON(http.StatusForbidden, gin.H{"error": "Cannot reopen this coverage request"})
			return
		}

		// Conditional update, gated on the state actually authorized above,
		// closes the race with a concurrent cancel/reopen landing between the
		// read above and this write, same technique used throughout this
		// file. An admin may reopen any claimed request; a non-admin
		// claimant may only reopen their own.
		var result *gorm.DB
		if isAdminCaller {
			result = db.Model(&models.ShiftCoverageRequest{}).
				Where("id = ? AND status = ?", reqRow.ID, models.CoverageRequestClaimed).
				Updates(map[string]interface{}{
					"status":             models.CoverageRequestOpen,
					"claimed_by_user_id": nil,
					"claimed_at":         nil,
				})
		} else {
			result = db.Model(&models.ShiftCoverageRequest{}).
				Where("id = ? AND claimed_by_user_id = ? AND status = ?", reqRow.ID, callerUserID, models.CoverageRequestClaimed).
				Updates(map[string]interface{}{
					"status":             models.CoverageRequestOpen,
					"claimed_by_user_id": nil,
					"claimed_at":         nil,
				})
		}
		if result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reopen coverage request"})
			return
		}
		if result.RowsAffected == 0 {
			c.JSON(http.StatusConflict, gin.H{"error": "Coverage request is no longer claimed"})
			return
		}

		reqRow.Status = models.CoverageRequestOpen
		reqRow.ClaimedByUserID = nil
		reqRow.ClaimedAt = nil
		c.JSON(http.StatusOK, toCoverageRequestResponse(reqRow))

		notifyGroupOfOpenCoverageRequests(rawDB, emailService, groupMeService, reqRow.GroupID, reqRow.RequestedByUserID)
	}
}
