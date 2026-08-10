package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
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
		if !date.After(today) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "date must be in the future"})
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

		title := "Coverage needed"
		content := fmt.Sprintf("A volunteer needs coverage for their %s shift on %s.",
			formatHourAMPM(req.Hour), date.Format("Monday, January 2"))

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
