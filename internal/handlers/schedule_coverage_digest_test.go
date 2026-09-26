package handlers

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/networkengineer-cloud/go-volunteer-media/internal/database"
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

// Fix 6. Waiting for the requester to go quiet has no upper bound on its
// own: someone who adds a shift every couple of minutes keeps resetting
// MAX(created_at), so the target never becomes eligible and the group is
// never told at all. A hard cap makes the burst ship regardless once its
// OLDEST pending request passes coverageDigestMaxDelay.
func TestSweepCoverageDigests_HardCapDefeatsAnEndlessBurst(t *testing.T) {
	db := SetupTestDB(t)
	requester, _, group := setupCoverageTestGroup(t, db)

	// Oldest pending request is past the hard cap...
	createDigestRequest(t, db, group.ID, requester.ID, 10, coverageDigestMaxDelay+time.Minute)
	// ...but the volunteer is still adding, so the quiet period never elapses.
	createDigestRequest(t, db, group.ID, requester.ID, 11, 10*time.Second)

	rec := &notifyRecorder{}
	sweepCoverageDigests(db, rec.record)

	if got := rec.count(); got != 1 {
		t.Errorf("expected the hard cap to force one announcement, got %d", got)
	}

	// The whole burst ships together, the still-fresh shift included, so the
	// next tick has nothing left to announce separately.
	var unstamped int64
	db.Model(&models.ShiftCoverageRequest{}).Where("notified_at IS NULL").Count(&unstamped)
	if unstamped != 0 {
		t.Errorf("expected the hard cap to sweep the whole burst, %d left unstamped", unstamped)
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

// Fix 1. RunMigrations runs on every process start - each API pod boot, each
// scale-up, each `make seed` - so its notified_at backfill must not treat
// "notified_at IS NULL" as "legacy row". That condition IS the sweep's work
// queue: stamping it unconditionally means any restart inside the quiet
// period silently discards every pending announcement, with nothing logged.
func TestCoverageDigestBackfill_DoesNotDiscardPendingAnnouncements(t *testing.T) {
	db := SetupTestDB(t)
	requester, _, group := setupCoverageTestGroup(t, db)

	// The deploy that introduced the column: its one-time backfill runs here.
	if err := database.RunMigrations(db); err != nil {
		t.Fatalf("Failed to run initial migrations: %v", err)
	}

	createDigestRequest(t, db, group.ID, requester.ID, 10, 10*time.Minute)

	// A second process boots (autoscale, restart, redeploy) and migrates
	// again. The pending request above must survive it.
	if err := database.RunMigrations(db); err != nil {
		t.Fatalf("Failed to re-run migrations: %v", err)
	}

	rec := &notifyRecorder{}
	sweepCoverageDigests(db, rec.record)

	if got := rec.count(); got != 1 {
		t.Errorf("expected the pending request to still be announced after a restart, got %d notification(s)", got)
	}
}

// Fix 2. The NOT EXISTS subquery in claimCoverageDigest is the only thing
// distinguishing it from a plain UPDATE, and the only hand-written SQL here.
// It re-checks the quiet period at claim time so a request that lands
// between the read and the write makes the claim find nothing, keeping the
// burst together instead of announcing it without its newest shift. Deleting
// the subquery must fail this test.
func TestClaimCoverageDigest_RefusesWhenANewerRequestLandedAfterTheRead(t *testing.T) {
	db := SetupTestDB(t)
	requester, _, group := setupCoverageTestGroup(t, db)
	createDigestRequest(t, db, group.ID, requester.ID, 10, 10*time.Minute)

	cutoff := time.Now().Add(-coverageDigestQuietPeriod)
	hardCutoff := time.Now().Add(-coverageDigestMaxDelay)
	targets, err := pendingCoverageDigestTargets(db, cutoff, hardCutoff)
	if err != nil {
		t.Fatalf("read targets: %v", err)
	}
	if len(targets) != 1 {
		t.Fatalf("expected 1 pending target, got %d", len(targets))
	}

	// The volunteer adds another shift in the gap between read and claim.
	createDigestRequest(t, db, group.ID, requester.ID, 11, 0)

	claimed, err := claimCoverageDigest(db, cutoff, hardCutoff, targets[0])
	if err != nil {
		t.Fatalf("claim: %v", err)
	}
	if claimed {
		t.Error("expected the claim to be refused so the whole burst waits for the next tick")
	}

	var unstamped int64
	db.Model(&models.ShiftCoverageRequest{}).Where("notified_at IS NULL").Count(&unstamped)
	if unstamped != 2 {
		t.Errorf("expected both requests left unstamped, got %d unstamped", unstamped)
	}
}

// blockingEmailProvider signals when a send begins and holds it open until
// released, so a test can observe whether the caller is still inside the
// send or has detached it to a goroutine.
type blockingEmailProvider struct {
	started chan struct{}
	release chan struct{}
	once    sync.Once
	mu      sync.Mutex
	sends   int
}

func (m *blockingEmailProvider) SendEmail(_ context.Context, _, _, _ string) error {
	m.once.Do(func() { close(m.started) })
	<-m.release
	m.mu.Lock()
	m.sends++
	m.mu.Unlock()
	return nil
}
func (m *blockingEmailProvider) IsConfigured() bool      { return true }
func (m *blockingEmailProvider) GetProviderName() string { return "blocking-mock" }

// Fix 3. stop() promises that nothing is still writing to the database once
// it returns, so cmd/api/main.go can close the connection pool immediately
// afterwards. That promise rests on notifyGroupOfOpenCoverageRequests
// sending synchronously - if it detaches the send to a goroutine, a SIGTERM
// just after a claim closes the pool underneath it and the announcement is
// lost permanently, because its rows are already stamped.
//
// This drives the REAL notifier rather than an injected stub: stubbing it
// would assert only that the sweep waits for whatever notify does, which
// stays true however the notifier behaves internally.
func TestNotifyGroupOfOpenCoverageRequests_SendsSynchronously(t *testing.T) {
	t.Setenv("SCHEDULE_EMAIL_NOTIFICATIONS_ENABLED", "true")
	db := SetupTestDB(t)
	requester, other, group := setupCoverageTestGroup(t, db)
	if err := db.Model(other).Update("email_notifications_enabled", true).Error; err != nil {
		t.Fatalf("Failed to enable recipient email: %v", err)
	}
	createDigestRequest(t, db, group.ID, requester.ID, 10, 10*time.Minute)

	provider := &blockingEmailProvider{started: make(chan struct{}), release: make(chan struct{})}
	emailSvc := email.NewServiceWithProvider(provider, db)

	returned := make(chan struct{})
	go func() {
		notifyGroupOfOpenCoverageRequests(db, emailSvc, nil, group.ID, requester.ID)
		close(returned)
	}()

	select {
	case <-provider.started:
	case <-time.After(5 * time.Second):
		t.Fatal("the notifier never attempted a send")
	}

	// The send is open. A synchronous notifier is still inside it.
	select {
	case <-returned:
		t.Fatal("notifyGroupOfOpenCoverageRequests returned while its send was still in flight - it detached the send, so the sweep's stop() can no longer guarantee the DB pool is safe to close")
	case <-time.After(100 * time.Millisecond):
	}

	close(provider.release)
	select {
	case <-returned:
	case <-time.After(5 * time.Second):
		t.Fatal("notifier did not return after its send completed")
	}
}

// Fix 7. ReopenCoverageRequest announces inline, so the row it reopens must
// be stamped as announced too. Otherwise it goes back to status=open with
// notified_at NULL, the sweep sees pending work, and the group is told a
// second time about a shift it was just told about - the exact double-send
// this sweep exists to prevent.
func TestReopenCoverageRequest_DoesNotLeaveWorkForTheDigestSweep(t *testing.T) {
	db := SetupTestDB(t)
	requester, claimant, group := setupCoverageTestGroup(t, db)
	date, _ := time.Parse("2006-01-02", nextWeekday(time.Tuesday))
	reqRow := createOpenCoverageRequest(t, db, group.ID, requester.ID, 2, 10, date)

	// Claimed INSIDE the quiet period, i.e. before the digest ever announced
	// it - so notified_at is still NULL. That is the only state in which the
	// reopen can leave pending work behind; a request claimed after its
	// digest went out is already stamped and cannot regress.
	if err := db.Model(reqRow).Updates(map[string]interface{}{
		"status":             models.CoverageRequestClaimed,
		"claimed_by_user_id": claimant.ID,
		"claimed_at":         time.Now(),
	}).Error; err != nil {
		t.Fatalf("Failed to claim request: %v", err)
	}
	var stamped int64
	db.Model(&models.ShiftCoverageRequest{}).Where("id = ? AND notified_at IS NOT NULL", reqRow.ID).Count(&stamped)
	if stamped != 0 {
		t.Fatalf("test setup is wrong: the request must be unannounced before the reopen")
	}
	if w := performReopenCoverageRequest(db, claimant.ID, false, group.ID, reqRow.ID); w.Code != 200 {
		t.Fatalf("Expected reopen to succeed, got %d: %s", w.Code, w.Body.String())
	}

	// Backdate so the row is past the quiet period and would be picked up.
	if err := db.Model(&models.ShiftCoverageRequest{}).Where("id = ?", reqRow.ID).
		UpdateColumn("created_at", time.Now().Add(-10*time.Minute)).Error; err != nil {
		t.Fatalf("Failed to backdate: %v", err)
	}

	rec := &notifyRecorder{}
	sweepCoverageDigests(db, rec.record)

	if got := rec.count(); got != 0 {
		t.Errorf("reopen already announced this request; expected the sweep to stay quiet, got %d notification(s)", got)
	}
}

// Round-2 finding 1. The reopen stamps rows so the digest sweep won't
// re-announce what it just announced - but the announcement it fires sends
// the requester's COMPLETE open list, not just the reopened shift. Stamping
// only the reopened row leaves any sibling still in the digest queue to be
// announced a second time, byte-identical, a few minutes later. The stamp
// has to cover what the announcement actually covered.
func TestReopenCoverageRequest_StampsEveryShiftItsAnnouncementCovered(t *testing.T) {
	db := SetupTestDB(t)
	requester, claimant, group := setupCoverageTestGroup(t, db)
	date, _ := time.Parse("2006-01-02", nextWeekday(time.Tuesday))

	// A sibling shift the requester flagged a minute ago: still unannounced,
	// still sitting in the digest queue.
	sibling := createDigestRequest(t, db, group.ID, requester.ID, 11, time.Minute)

	// A different shift of theirs, previously claimed, now handed back.
	reopened := createOpenCoverageRequest(t, db, group.ID, requester.ID, 2, 10, date)
	if err := db.Model(reopened).Updates(map[string]interface{}{
		"status":             models.CoverageRequestClaimed,
		"claimed_by_user_id": claimant.ID,
		"claimed_at":         time.Now(),
	}).Error; err != nil {
		t.Fatalf("Failed to claim: %v", err)
	}
	if w := performReopenCoverageRequest(db, claimant.ID, false, group.ID, reopened.ID); w.Code != 200 {
		t.Fatalf("Expected reopen to succeed, got %d: %s", w.Code, w.Body.String())
	}

	// The reopen's announcement listed both shifts, so both are now told.
	var stillPending int64
	db.Model(&models.ShiftCoverageRequest{}).
		Where("group_id = ? AND requested_by_user_id = ? AND notified_at IS NULL", group.ID, requester.ID).
		Count(&stillPending)
	if stillPending != 0 {
		t.Errorf("expected the reopen to stamp every shift its announcement listed, %d left pending", stillPending)
	}

	// Age the sibling past the quiet period: the sweep must stay quiet.
	if err := db.Model(&models.ShiftCoverageRequest{}).Where("id = ?", sibling.ID).
		UpdateColumn("created_at", time.Now().Add(-10*time.Minute)).Error; err != nil {
		t.Fatalf("Failed to backdate sibling: %v", err)
	}
	rec := &notifyRecorder{}
	sweepCoverageDigests(db, rec.record)

	if got := rec.count(); got != 0 {
		t.Errorf("the reopen already announced these shifts; expected no second announcement, got %d", got)
	}
}

// Round-2 finding 2. The reopen announces off-request so the caller isn't
// blocked on email delivery, but that goroutine writes to the database after
// the handler returns. Shutdown must drain it, exactly as it drains the
// write-path embed goroutines - otherwise the pool closes mid-send and the
// announcement is lost with its rows already stamped.
func TestWaitForPendingCoverageNotifications_DrainsInFlightSends(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	finished := make(chan struct{})

	trackCoverageNotification(func() {
		close(started)
		<-release
		close(finished)
	})

	<-started

	drained := make(chan struct{})
	go func() {
		WaitForPendingCoverageNotifications()
		close(drained)
	}()

	time.Sleep(100 * time.Millisecond)
	select {
	case <-drained:
		t.Fatal("WaitForPendingCoverageNotifications returned while a send was still in flight")
	default:
	}

	close(release)
	<-finished
	select {
	case <-drained:
	case <-time.After(coverageNotifyDrainTimeout + 2*time.Second):
		t.Fatal("WaitForPendingCoverageNotifications never returned")
	}
}

// Round-3 finding 1. The reminder broadcasts every open request in the
// group, which includes rows still waiting in the digest queue. If it
// doesn't stamp them, the sweep announces the same shifts again minutes
// later. Same rule as the reopen path: whatever an announcement covered
// must be marked as told.
func TestSendCoverageReminder_StampsThePendingShiftsItAnnounced(t *testing.T) {
	t.Setenv("SCHEDULE_EMAIL_NOTIFICATIONS_ENABLED", "true")
	db := SetupTestDB(t)
	requester, _, group := setupCoverageTestGroup(t, db)
	admin := CreateTestUser(t, db, "reminder-admin", "reminder-admin@example.com", "password123", false)
	AddUserToGroupWithAdmin(t, db, admin.ID, group.ID, true)

	// Two shifts the volunteer just flagged: announced by nobody yet.
	future := time.Now().UTC().AddDate(0, 0, 7).Truncate(24 * time.Hour)
	for _, hour := range []int{10, 11} {
		req := &models.ShiftCoverageRequest{
			GroupID: group.ID, RequestedByUserID: requester.ID,
			Date: future, Hour: hour, Status: models.CoverageRequestOpen,
		}
		if err := db.Create(req).Error; err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
	}

	provider := &mockEmailProvider{}
	emailSvc := email.NewServiceWithProvider(provider, db)
	if w := performSendCoverageReminder(db, emailSvc, admin.ID, false, group.ID); w.Code != 200 {
		t.Fatalf("Expected reminder to succeed, got %d: %s", w.Code, w.Body.String())
	}
	WaitForPendingCoverageNotifications()

	var pending int64
	db.Model(&models.ShiftCoverageRequest{}).Where("notified_at IS NULL").Count(&pending)
	if pending != 0 {
		t.Errorf("the reminder announced these shifts; expected them stamped, %d left pending", pending)
	}

	// Age them past the quiet period: the sweep must stay quiet.
	if err := db.Model(&models.ShiftCoverageRequest{}).Where("1 = 1").
		UpdateColumn("created_at", time.Now().Add(-10*time.Minute)).Error; err != nil {
		t.Fatalf("Failed to backdate: %v", err)
	}
	rec := &notifyRecorder{}
	sweepCoverageDigests(db, rec.record)
	if got := rec.count(); got != 0 {
		t.Errorf("the reminder already told the group; expected no digest announcement, got %d", got)
	}
}

// Round-3 finding 3. TestWaitForPendingCoverageNotifications_DrainsInFlightSends
// calls trackCoverageNotification directly, which proves the helper works but
// not that the reopen path actually uses it - swapping that call for a bare
// `go` left the whole suite green. This drives the real handler.
func TestReopenCoverageRequest_ItsAnnouncementIsDrainedAtShutdown(t *testing.T) {
	t.Setenv("SCHEDULE_EMAIL_NOTIFICATIONS_ENABLED", "true")
	db := SetupTestDB(t)
	requester, claimant, group := setupCoverageTestGroup(t, db)
	if err := db.Model(claimant).Update("email_notifications_enabled", true).Error; err != nil {
		t.Fatalf("Failed to enable recipient email: %v", err)
	}
	date, _ := time.Parse("2006-01-02", nextWeekday(time.Tuesday))
	reqRow := createOpenCoverageRequest(t, db, group.ID, requester.ID, 2, 10, date)
	if err := db.Model(reqRow).Updates(map[string]interface{}{
		"status":             models.CoverageRequestClaimed,
		"claimed_by_user_id": claimant.ID,
		"claimed_at":         time.Now(),
	}).Error; err != nil {
		t.Fatalf("Failed to claim: %v", err)
	}

	provider := &blockingEmailProvider{started: make(chan struct{}), release: make(chan struct{})}
	emailSvc := email.NewServiceWithProvider(provider, db)

	if w := performReopenCoverageRequestWithEmail(db, emailSvc, claimant.ID, false, group.ID, reqRow.ID); w.Code != 200 {
		t.Fatalf("Expected reopen to succeed, got %d: %s", w.Code, w.Body.String())
	}

	select {
	case <-provider.started:
	case <-time.After(5 * time.Second):
		t.Fatal("the reopen never attempted its announcement")
	}

	drained := make(chan struct{})
	go func() {
		WaitForPendingCoverageNotifications()
		close(drained)
	}()

	time.Sleep(100 * time.Millisecond)
	select {
	case <-drained:
		t.Fatal("shutdown drain returned while the reopen's announcement was still in flight - the send is untracked, so the DB pool could close underneath it")
	default:
	}

	close(provider.release)
	select {
	case <-drained:
	case <-time.After(coverageNotifyDrainTimeout + 2*time.Second):
		t.Fatal("drain never returned")
	}
}

// Round-3 finding 4. startCoverageDigestSweepWithNotify exists so a test can
// assert stop() waits for an in-flight ANNOUNCEMENT, not merely for the tick
// loop - removing the wait entirely left the suite green, because the only
// other stop() test runs with nil services and no pending rows, so no tick
// ever has a body to be in flight.
func TestStartCoverageDigestSweep_StopWaitsForTheAnnouncement(t *testing.T) {
	db := SetupTestDB(t)
	requester, _, group := setupCoverageTestGroup(t, db)
	createDigestRequest(t, db, group.ID, requester.ID, 10, 10*time.Minute)

	started := make(chan struct{})
	release := make(chan struct{})
	notifyDone := make(chan struct{})
	slowNotify := func(coverageDigestTarget) {
		close(started)
		<-release
		close(notifyDone)
	}

	stop := startCoverageDigestSweepWithNotify(db, slowNotify, 5*time.Millisecond)

	// Only call stop() once an announcement is genuinely in flight, or stop()
	// can win the race against the first tick and the test proves nothing.
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("the sweep never started an announcement")
	}

	stopReturned := make(chan struct{})
	go func() {
		stop()
		close(stopReturned)
	}()

	time.Sleep(100 * time.Millisecond)
	select {
	case <-stopReturned:
		t.Fatal("stop() returned while an announcement was still in flight - the DB pool could be closed underneath it")
	default:
	}

	close(release)
	<-notifyDone
	select {
	case <-stopReturned:
	case <-time.After(coverageDigestStopTimeout + 2*time.Second):
		t.Fatal("stop() did not return after the announcement finished")
	}
}
