package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
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
	errRequestNotFound  = errors.New("coverage request not found")
	errRequestNotOpen   = errors.New("coverage request is no longer open")
	errSelfClaim        = errors.New("cannot claim your own coverage request")
	errClaimConflict    = errors.New("claimant already has a conflicting shift at that time")
)

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

		// time.Parse("2006-01-02", ...) already yields UTC midnight, matching
		// how ShiftCoverageRequest.Date is stored and compared throughout.
		date, err := time.Parse("2006-01-02", req.Date)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "date must be in YYYY-MM-DD format"})
			return
		}
		today := time.Now().UTC().Truncate(24 * time.Hour)
		if date.Before(today) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "date must not be in the past"})
			return
		}
		if req.Hour < 8 || req.Hour > 17 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "hour must be between 8 and 17"})
			return
		}

		groupIDUint64, err := strconv.ParseUint(groupIDParam, 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid group ID"})
			return
		}
		groupIDUint := uint(groupIDUint64)

		var created models.ShiftCoverageRequest
		err = db.Transaction(func(tx *gorm.DB) error {
			var slot models.ShiftSlot
			if err := tx.Where("user_id = ? AND group_id = ? AND day_of_week = ? AND hour = ?",
				targetUserID, groupIDUint, int(date.Weekday()), req.Hour).First(&slot).Error; err != nil {
				return errNoMatchingSlot
			}

			var existing models.ShiftCoverageRequest
			err := tx.Where("group_id = ? AND requested_by_user_id = ? AND date = ? AND hour = ? AND status != ?",
				groupIDUint, targetUserID, date, req.Hour, models.CoverageRequestCancelled).
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
				Hour:              req.Hour,
				Status:            models.CoverageRequestOpen,
			}
			return tx.Create(&created).Error
		})

		switch {
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
		// the last minute, so a rapid double-submission doesn't fire two
		// near-identical emails/GroupMe posts back to back. Short on purpose:
		// its only job is absorbing accidental double-clicks, not gatekeeping
		// a legitimate second request for a different shift moments later -
		// whenever a notification IS sent (see below), it always lists every
		// currently-open request this user has in the group, not just the
		// one that triggered it, so nothing is ever silently left out of the
		// email because it happened to land inside the cooldown window. Runs
		// on the outer (non-transaction) db since the transaction has
		// already committed by this point, matching how the notification
		// goroutines below already run post-commit.
		const coverageNotificationCooldown = 60 * time.Second
		var recentCount int64
		if err := rawDB.Model(&models.ShiftCoverageRequest{}).
			Where("group_id = ? AND requested_by_user_id = ? AND id != ? AND created_at > ?",
				groupIDUint, targetUserID, created.ID, time.Now().Add(-coverageNotificationCooldown)).
			Count(&recentCount).Error; err == nil && recentCount == 0 {
			var requester models.User
			var grp models.Group
			var openRequests []models.ShiftCoverageRequest
			rawDB.First(&requester, targetUserID)
			rawDB.Select("name").First(&grp, groupIDUint)
			rawDB.Where("group_id = ? AND requested_by_user_id = ? AND status = ?",
				groupIDUint, targetUserID, models.CoverageRequestOpen).
				Order("date, hour").Find(&openRequests)

			title := fmt.Sprintf("Coverage needed in %s", grp.Name)
			content := buildCoverageRequestSummary(displayName(requester), openRequests)

			if emailService != nil && emailService.IsConfigured() {
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

		if emailService != nil && emailService.IsConfigured() && requester.EmailNotificationsEnabled {
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
