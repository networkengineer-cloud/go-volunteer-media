package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
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
	errNoMatchingSlot   = errors.New("no matching recurring shift for that date and hour")
	errDuplicateRequest = errors.New("a coverage request already exists for that date and hour")
	errPastDate         = errors.New("date must not be in the past")
	errRequestNotFound  = errors.New("coverage request not found")
	errRequestNotOpen   = errors.New("coverage request is no longer open")
	errSelfClaim        = errors.New("cannot claim your own coverage request")
	errClaimConflict    = errors.New("claimant already has a conflicting shift at that time")
)

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
	Date   string `json:"date"`
	Hour   int    `json:"hour"`
	UserID *uint  `json:"user_id"`
}

type coverageRequestResponse struct {
	ID                uint   `json:"id"`
	GroupID           uint   `json:"group_id"`
	RequestedByUserID uint   `json:"requested_by_user_id"`
	Date              string `json:"date"`
	Hour              int    `json:"hour"`
	Status            string `json:"status"`
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
		ClaimedByUserID:   r.ClaimedByUserID,
	}
}

type coverageRequestListItem struct {
	ID                uint   `json:"id"`
	GroupID           uint   `json:"group_id"`
	RequestedByUserID uint   `json:"requested_by_user_id"`
	RequestedByName   string `json:"requested_by_name"`
	Date              string `json:"date"`
	Hour              int    `json:"hour"`
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
func buildCoverageRequestSummary(requesterName string, requests []models.ShiftCoverageRequest) string {
	if len(requests) == 1 {
		r := requests[0]
		return fmt.Sprintf("%s needs coverage for their %s shift on %s.",
			requesterName, formatHourAMPM(r.Hour), r.Date.Format("Monday, January 2"))
	}
	lines := make([]string, 0, len(requests))
	for _, r := range requests {
		lines = append(lines, fmt.Sprintf("- %s at %s", r.Date.Format("Monday, January 2"), formatHourAMPM(r.Hour)))
	}
	return fmt.Sprintf("%s needs coverage for %d shifts:\n%s", requesterName, len(requests), strings.Join(lines, "\n"))
}

// formatHourAMPM renders a 24-hour ShiftSlot/CoverageRequest hour (8..17)
// as e.g. "10:00 AM" for notification text.
func formatHourAMPM(hour int) string {
	period := "AM"
	displayHour := hour
	if hour >= 12 {
		period = "PM"
	}
	if displayHour > 12 {
		displayHour -= 12
	}
	return fmt.Sprintf("%d:00 %s", displayHour, period)
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
func createOneCoverageRequest(db *gorm.DB, groupIDUint, targetUserID uint, date time.Time, hour int) (models.ShiftCoverageRequest, error) {
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

		groupIDUint64, err := strconv.ParseUint(groupIDParam, 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid group ID"})
			return
		}
		groupIDUint := uint(groupIDUint64)

		created, err := createOneCoverageRequest(db, groupIDUint, targetUserID, date, req.Hour)
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

			// Conflict check spans every group the claimant belongs to - a
			// real-world time conflict doesn't respect group boundaries.
			var conflictCount int64
			if err := tx.Model(&models.ShiftSlot{}).
				Where("user_id = ? AND day_of_week = ? AND hour = ?", callerUserID, int(reqRow.Date.Weekday()), reqRow.Hour).
				Count(&conflictCount).Error; err != nil {
				return err
			}
			if conflictCount > 0 {
				return errClaimConflict
			}
			if err := tx.Model(&models.ShiftCoverageRequest{}).
				Where("claimed_by_user_id = ? AND date = ? AND hour = ? AND status = ?",
					callerUserID, reqRow.Date, reqRow.Hour, models.CoverageRequestClaimed).
				Count(&conflictCount).Error; err != nil {
				return err
			}
			if conflictCount > 0 {
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
			displayName(claimant), formatHourAMPM(req.Hour), req.Date.Format("Monday, January 2"))

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

// slotConflictKey/claimConflictKey key the maps loadUserConflictKeys
// returns: weekday+hour for a recurring ShiftSlot, and date+hour for an
// already-claimed ShiftCoverageRequest.
func slotConflictKey(dayOfWeek, hour int) string {
	return fmt.Sprintf("%d-%d", dayOfWeek, hour)
}

func claimConflictKey(date time.Time, hour int) string {
	return fmt.Sprintf("%s-%d", date.Format("2006-01-02"), hour)
}

// loadUserConflictKeys loads every recurring ShiftSlot and already-claimed
// ShiftCoverageRequest userID has (in any group - a real-world time
// conflict doesn't respect group boundaries), keyed for O(1) lookup. Two
// queries regardless of how many coverage requests are being checked
// against them, so ListCoverageRequests can annotate every row in the
// group without a per-row round trip.
func loadUserConflictKeys(db *gorm.DB, userID uint) (slotKeys, claimKeys map[string]struct{}, err error) {
	var slots []models.ShiftSlot
	if err := db.Where("user_id = ?", userID).Find(&slots).Error; err != nil {
		return nil, nil, err
	}
	slotKeys = make(map[string]struct{}, len(slots))
	for _, s := range slots {
		slotKeys[slotConflictKey(s.DayOfWeek, s.Hour)] = struct{}{}
	}

	var claimed []models.ShiftCoverageRequest
	if err := db.Where("claimed_by_user_id = ? AND status = ?", userID, models.CoverageRequestClaimed).Find(&claimed).Error; err != nil {
		return nil, nil, err
	}
	claimKeys = make(map[string]struct{}, len(claimed))
	for _, r := range claimed {
		claimKeys[claimConflictKey(r.Date, r.Hour)] = struct{}{}
	}
	return slotKeys, claimKeys, nil
}

// isRequestClaimableGiven reports whether userID could claim req right
// now, given userID's own conflict keys (see loadUserConflictKeys): not
// their own request, and no conflicting ShiftSlot or already-claimed
// coverage request at that exact date/hour. Mirrors the checks
// ClaimCoverageRequest itself makes inside its transaction; this read-only
// version is for annotating list results and is not the source of the
// atomicity guarantee - the conditional update in ClaimCoverageRequest is.
func isRequestClaimableGiven(req models.ShiftCoverageRequest, userID uint, slotKeys, claimKeys map[string]struct{}) bool {
	if req.RequestedByUserID == userID {
		return false
	}
	if _, conflict := slotKeys[slotConflictKey(int(req.Date.Weekday()), req.Hour)]; conflict {
		return false
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

		slotKeys, claimKeys, err := loadUserConflictKeys(db, callerUserID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load coverage requests"})
			return
		}

		items := make([]coverageRequestListItem, 0, len(requests))
		for _, r := range requests {
			claimable := isRequestClaimableGiven(r, callerUserID, slotKeys, claimKeys)
			items = append(items, coverageRequestListItem{
				ID:                r.ID,
				GroupID:           r.GroupID,
				RequestedByUserID: r.RequestedByUserID,
				RequestedByName:   displayName(r.RequestedByUser),
				Date:              r.Date.Format("2006-01-02"),
				Hour:              r.Hour,
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
		if !isOwnOpenRequest && !isAdminCaller {
			c.JSON(http.StatusForbidden, gin.H{"error": "Cannot cancel this coverage request"})
			return
		}

		// Conditional update, gated on the state actually authorized above,
		// closes the race with a concurrent claim (or cancel) landing between
		// the read above and this write - same technique as the conditional
		// update in ClaimCoverageRequest. A non-admin owner may only flip
		// open -> cancelled; an admin may flip open or claimed -> cancelled.
		var result *gorm.DB
		if isAdminCaller {
			result = db.Model(&models.ShiftCoverageRequest{}).
				Where("id = ? AND status IN ?", reqRow.ID, []models.CoverageRequestStatus{models.CoverageRequestOpen, models.CoverageRequestClaimed}).
				Update("status", models.CoverageRequestCancelled)
		} else {
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
			// owner, it most likely means someone claimed it out from under
			// them (or cancelled it) - either way it's no longer cancellable
			// by them.
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

type createCoverageRequestBatchItem struct {
	Date string `json:"date"`
	Hour int    `json:"hour"`
}

type createCoverageRequestBatchRequest struct {
	Requests []createCoverageRequestBatchItem `json:"requests"`
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

		groupIDUint64, err := strconv.ParseUint(groupIDParam, 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid group ID"})
			return
		}
		groupIDUint := uint(groupIDUint64)

		type parsedItem struct {
			date time.Time
			hour int
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
			parsedItems = append(parsedItems, parsedItem{date: date, hour: item.Hour})
		}

		response := coverageRequestBatchResponse{
			Created: make([]coverageRequestResponse, 0, len(parsedItems)),
			Skipped: make([]coverageRequestBatchSkipped, 0),
		}
		for _, item := range parsedItems {
			created, err := createOneCoverageRequest(db, groupIDUint, callerUserID, item.date, item.hour)
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
