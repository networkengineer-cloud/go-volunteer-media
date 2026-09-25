package handlers

import (
	"sync"
	"testing"
	"time"

	"github.com/networkengineer-cloud/go-volunteer-media/internal/email"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
	"gorm.io/gorm"
)

// notifyRecorder captures the (group, user) pairs a sweep decided to
// announce, standing in for the real email/GroupMe fan-out so these tests
// exercise the claim-and-batch logic without any notification services.
type notifyRecorder struct {
	mu    sync.Mutex
	calls []coverageDigestTarget
}

func (r *notifyRecorder) record(target coverageDigestTarget) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, target)
}

func (r *notifyRecorder) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.calls)
}

// createDigestRequest inserts an open, unnotified coverage request whose
// created_at is backdated by age, so a test can place it inside or outside
// the quiet period without sleeping.
func createDigestRequest(t *testing.T, db *gorm.DB, groupID, userID uint, hour int, age time.Duration) *models.ShiftCoverageRequest {
	t.Helper()
	req := &models.ShiftCoverageRequest{
		GroupID:           groupID,
		RequestedByUserID: userID,
		Date:              time.Date(2026, 8, 18, 0, 0, 0, 0, time.UTC),
		Hour:              hour,
		Status:            models.CoverageRequestOpen,
	}
	if err := db.Create(req).Error; err != nil {
		t.Fatalf("Failed to create coverage request: %v", err)
	}
	// GORM sets created_at on insert, so backdate it in a second write.
	if err := db.Model(req).UpdateColumn("created_at", time.Now().Add(-age)).Error; err != nil {
		t.Fatalf("Failed to backdate coverage request: %v", err)
	}
	return req
}

func TestSweepCoverageDigests_SendsOneNotificationForABurst(t *testing.T) {
	db := SetupTestDB(t)
	requester, _, group := setupCoverageTestGroup(t, db)

	// Three consecutive one-hour shifts requested separately - the exact
	// shape that used to produce three emails.
	for _, hour := range []int{10, 11, 12} {
		createDigestRequest(t, db, group.ID, requester.ID, hour, 10*time.Minute)
	}

	rec := &notifyRecorder{}
	sweepCoverageDigests(db, rec.record)

	if got := rec.count(); got != 1 {
		t.Errorf("expected exactly 1 notification for a 3-shift burst, got %d", got)
	}

	var unstamped int64
	db.Model(&models.ShiftCoverageRequest{}).Where("notified_at IS NULL").Count(&unstamped)
	if unstamped != 0 {
		t.Errorf("expected all 3 requests stamped as notified, %d still unstamped", unstamped)
	}
}

func TestSweepCoverageDigests_LeavesRequestsInsideQuietPeriod(t *testing.T) {
	db := SetupTestDB(t)
	requester, _, group := setupCoverageTestGroup(t, db)

	// Just created - the volunteer may still be adding more shifts.
	createDigestRequest(t, db, group.ID, requester.ID, 10, 0)

	rec := &notifyRecorder{}
	sweepCoverageDigests(db, rec.record)

	if got := rec.count(); got != 0 {
		t.Errorf("expected no notification inside the quiet period, got %d", got)
	}

	var unstamped int64
	db.Model(&models.ShiftCoverageRequest{}).Where("notified_at IS NULL").Count(&unstamped)
	if unstamped != 1 {
		t.Errorf("expected the request to stay unstamped for a later tick, got %d unstamped", unstamped)
	}
}

func TestSweepCoverageDigests_WaitsForAStraddlingBurstToFinish(t *testing.T) {
	// The quiet period means "this volunteer has stopped adding shifts",
	// not "this row is old enough". A burst that straddles the cutoff - one
	// shift flagged well before it, another still inside it - must wait, or
	// the old shift is announced alone now and the new one announced again
	// a minute later: the same two-email split the digest exists to stop.
	db := SetupTestDB(t)
	requester, _, group := setupCoverageTestGroup(t, db)

	createDigestRequest(t, db, group.ID, requester.ID, 10, 10*time.Minute)
	stillTyping := createDigestRequest(t, db, group.ID, requester.ID, 11, 30*time.Second)

	rec := &notifyRecorder{}
	sweepCoverageDigests(db, rec.record)

	if got := rec.count(); got != 0 {
		t.Fatalf("expected the digest to wait for the in-progress burst, got %d notification(s)", got)
	}
	var unstamped int64
	db.Model(&models.ShiftCoverageRequest{}).Where("notified_at IS NULL").Count(&unstamped)
	if unstamped != 2 {
		t.Errorf("expected both requests left for a later tick, got %d unstamped", unstamped)
	}

	// The volunteer stops; the whole burst now clears in one announcement.
	if err := db.Model(stillTyping).UpdateColumn("created_at", time.Now().Add(-10*time.Minute)).Error; err != nil {
		t.Fatalf("Failed to age the second request: %v", err)
	}
	sweepCoverageDigests(db, rec.record)

	if got := rec.count(); got != 1 {
		t.Errorf("expected exactly 1 notification covering the finished burst, got %d", got)
	}
	db.Model(&models.ShiftCoverageRequest{}).Where("notified_at IS NULL").Count(&unstamped)
	if unstamped != 0 {
		t.Errorf("expected both requests stamped, %d still unstamped", unstamped)
	}
}

func TestSweepCoverageDigests_SecondSweepSendsNothing(t *testing.T) {
	db := SetupTestDB(t)
	requester, _, group := setupCoverageTestGroup(t, db)
	createDigestRequest(t, db, group.ID, requester.ID, 10, 10*time.Minute)

	rec := &notifyRecorder{}
	sweepCoverageDigests(db, rec.record)
	sweepCoverageDigests(db, rec.record)

	if got := rec.count(); got != 1 {
		t.Errorf("expected the claimed request to notify exactly once across two ticks, got %d", got)
	}
}

func TestSweepCoverageDigests_IgnoresNonOpenRequests(t *testing.T) {
	db := SetupTestDB(t)
	requester, other, group := setupCoverageTestGroup(t, db)

	// Cancelled and claimed during the quiet window - neither should
	// trigger a "coverage needed" announcement.
	cancelled := createDigestRequest(t, db, group.ID, requester.ID, 10, 10*time.Minute)
	if err := db.Model(cancelled).Update("status", models.CoverageRequestCancelled).Error; err != nil {
		t.Fatalf("Failed to cancel request: %v", err)
	}
	claimed := createDigestRequest(t, db, group.ID, requester.ID, 11, 10*time.Minute)
	if err := db.Model(claimed).Updates(map[string]interface{}{
		"status":             models.CoverageRequestClaimed,
		"claimed_by_user_id": other.ID,
	}).Error; err != nil {
		t.Fatalf("Failed to claim request: %v", err)
	}

	rec := &notifyRecorder{}
	sweepCoverageDigests(db, rec.record)

	if got := rec.count(); got != 0 {
		t.Errorf("expected no notification when nothing is still open, got %d", got)
	}
}

func TestSweepCoverageDigests_SeparatesRequestersAndGroups(t *testing.T) {
	db := SetupTestDB(t)
	requester, other, group := setupCoverageTestGroup(t, db)
	secondGroup := CreateTestGroup(t, db, "Cats", "Cat volunteers")

	createDigestRequest(t, db, group.ID, requester.ID, 10, 10*time.Minute)
	createDigestRequest(t, db, group.ID, requester.ID, 11, 10*time.Minute)
	createDigestRequest(t, db, group.ID, other.ID, 14, 10*time.Minute)
	createDigestRequest(t, db, secondGroup.ID, requester.ID, 9, 10*time.Minute)

	rec := &notifyRecorder{}
	sweepCoverageDigests(db, rec.record)

	// One per (group, requester) pair: requester+Dogs, other+Dogs,
	// requester+Cats. The two requester+Dogs rows coalesce.
	if got := rec.count(); got != 3 {
		t.Errorf("expected 3 notifications (one per group/requester pair), got %d", got)
	}
}

func TestStartCoverageDigestSweep_StopDoesNotLeakGoroutine(t *testing.T) {
	db := SetupTestDB(t)

	stop := StartCoverageDigestSweep(db, nil, nil, 10*time.Millisecond)

	done := make(chan struct{})
	go func() {
		stop()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(coverageDigestStopTimeout + 2*time.Second):
		t.Fatal("stop() did not return - sweep goroutine leaked")
	}

	// Calling stop() twice must not panic on a double close.
	stop()
}

// TestCoverageDigest_EmailGatedByScheduleFlag exercises the actual SendEmail
// call through the sweep's real notifier, not scheduleEmailNotificationsEnabled()
// in isolation, so a regression that gates the wrong condition is caught
// here. This assertion used to live on CreateCoverageRequest; announcement
// moved to the sweep, so the gate has to be verified where the send now
// happens.
func TestCoverageDigest_EmailGatedByScheduleFlag(t *testing.T) {
	setup := func(t *testing.T) (*gorm.DB, *mockEmailProvider, coverageDigestTarget) {
		t.Helper()
		db := SetupTestDB(t)
		requester, other, group := setupCoverageTestGroup(t, db)
		if err := db.Model(other).Update("email_notifications_enabled", true).Error; err != nil {
			t.Fatalf("Failed to enable other's email notifications: %v", err)
		}
		createDigestRequest(t, db, group.ID, requester.ID, 10, 10*time.Minute)
		return db, &mockEmailProvider{}, coverageDigestTarget{GroupID: group.ID, RequestedByUserID: requester.ID}
	}

	t.Run("no email is sent while the flag is unset (default)", func(t *testing.T) {
		t.Setenv("SCHEDULE_EMAIL_NOTIFICATIONS_ENABLED", "")
		db, provider, _ := setup(t)
		emailSvc := email.NewServiceWithProvider(provider, db)

		sweepCoverageDigests(db, coverageDigestNotifier(db, emailSvc, nil))

		time.Sleep(50 * time.Millisecond)
		if got := provider.sendCount(); got != 0 {
			t.Fatalf("Expected no coverage digest email while the flag is off, got %d", got)
		}
	})

	t.Run("an email is sent once the flag is enabled", func(t *testing.T) {
		t.Setenv("SCHEDULE_EMAIL_NOTIFICATIONS_ENABLED", "true")
		db, provider, _ := setup(t)
		emailSvc := email.NewServiceWithProvider(provider, db)

		sweepCoverageDigests(db, coverageDigestNotifier(db, emailSvc, nil))

		time.Sleep(50 * time.Millisecond)
		if got := provider.sendCount(); got == 0 {
			t.Fatal("Expected a coverage digest email to be sent while the flag is on, got none")
		}
	})
}
