package handlers

import (
	"fmt"
	"testing"
	"time"

	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
)

// The SQLite-backed digest tests in schedule_coverage_digest_test.go run
// ticks one after another, which cannot exercise the property the sweep
// actually depends on in production: prod scales to several container
// replicas (terraform/environments/prod/main.tf), so two pods can tick at
// the same instant against the same rows. The claim UPDATE is what stops
// that from sending the group two identical announcements, and only a real
// Postgres can demonstrate it - row-level locking and READ COMMITTED
// re-evaluation are exactly what SQLite's serialized writes paper over.
//
// The test asserts that contention actually occurred (replica B's claim must
// still be blocked when A commits), so it cannot quietly degrade into two
// sequential claims that would pass without ever consulting the guard.
func TestSweepCoverageDigests_ConcurrentReplicasClaimOnce(t *testing.T) {
	db := openSearchTestPostgres(t)

	unique := time.Now().UnixNano()
	requester := &models.User{
		Username: fmt.Sprintf("digest-requester-%d", unique),
		Email:    fmt.Sprintf("digest-requester-%d@example.com", unique),
		Password: "x",
	}
	if err := db.Create(requester).Error; err != nil {
		t.Fatalf("create requester: %v", err)
	}
	group := &models.Group{Name: fmt.Sprintf("DigestTest-%d", unique)}
	if err := db.Create(group).Error; err != nil {
		t.Fatalf("create group: %v", err)
	}
	t.Cleanup(func() {
		db.Where("group_id = ?", group.ID).Delete(&models.ShiftCoverageRequest{})
		db.Delete(group)
		db.Unscoped().Delete(requester)
	})

	// Three shifts, all past the quiet period - one digest is owed.
	for _, hour := range []int{10, 11, 12} {
		req := &models.ShiftCoverageRequest{
			GroupID:           group.ID,
			RequestedByUserID: requester.ID,
			Date:              time.Date(2026, time.August, 18, 0, 0, 0, 0, time.UTC),
			Hour:              hour,
			Status:            models.CoverageRequestOpen,
		}
		if err := db.Create(req).Error; err != nil {
			t.Fatalf("create coverage request: %v", err)
		}
		if err := db.Model(req).UpdateColumn("created_at", time.Now().Add(-10*time.Minute)).Error; err != nil {
			t.Fatalf("backdate coverage request: %v", err)
		}
	}

	// Two replicas, genuinely concurrent: separate connections, each in its
	// own explicit transaction, both holding the same target before either
	// writes. This is the interleaving that matters in prod and the one a
	// naive two-goroutine test misses - if one sweep simply finishes first,
	// the other's read returns nothing and the guard is never consulted.
	cutoff := time.Now().Add(-coverageDigestQuietPeriod)
	hardCutoff := time.Now().Add(-coverageDigestMaxDelay)
	target := coverageDigestTarget{GroupID: group.ID, RequestedByUserID: requester.ID}

	txA := db.Begin()
	txB := db.Begin()
	defer func() { txA.Rollback(); txB.Rollback() }()

	// Both read first - neither has written, so both see the work as pending.
	targetsA, err := pendingCoverageDigestTargets(txA, cutoff, hardCutoff)
	if err != nil {
		t.Fatalf("replica A read: %v", err)
	}
	targetsB, err := pendingCoverageDigestTargets(txB, cutoff, hardCutoff)
	if err != nil {
		t.Fatalf("replica B read: %v", err)
	}
	if len(targetsA) == 0 || len(targetsB) == 0 {
		t.Fatalf("both replicas should see the pending target, got A=%d B=%d", len(targetsA), len(targetsB))
	}

	// A claims and commits. B's claim is issued while A still holds the row
	// locks, so it blocks until A commits, then re-evaluates its WHERE under
	// READ COMMITTED against the now-stamped rows.
	claimedA, err := claimCoverageDigest(txA, cutoff, hardCutoff, target)
	if err != nil {
		t.Fatalf("replica A claim: %v", err)
	}

	type claimResult struct {
		claimed bool
		err     error
	}
	bDone := make(chan claimResult, 1)
	go func() {
		claimed, err := claimCoverageDigest(txB, cutoff, hardCutoff, target)
		bDone <- claimResult{claimed, err}
	}()

	// B must still be blocked on A's locks - if it isn't, no contention
	// happened and the test would prove nothing.
	select {
	case r := <-bDone:
		t.Fatalf("replica B's claim completed before A committed (claimed=%v, err=%v) - no lock contention occurred, so this test is not exercising the guard", r.claimed, r.err)
	case <-time.After(300 * time.Millisecond):
	}

	if err := txA.Commit().Error; err != nil {
		t.Fatalf("replica A commit: %v", err)
	}

	var rb claimResult
	select {
	case rb = <-bDone:
	case <-time.After(10 * time.Second):
		t.Fatal("replica B's claim never returned after A committed - possible deadlock")
	}
	if rb.err != nil {
		t.Fatalf("replica B claim: %v", rb.err)
	}
	if err := txB.Commit().Error; err != nil {
		t.Fatalf("replica B commit: %v", err)
	}

	if !claimedA {
		t.Error("replica A should have won the claim")
	}
	if rb.claimed {
		t.Error("replica B claimed rows A had already stamped - the group would be told twice")
	}

	var unstamped int64
	db.Model(&models.ShiftCoverageRequest{}).
		Where("group_id = ? AND notified_at IS NULL", group.ID).
		Count(&unstamped)
	if unstamped != 0 {
		t.Errorf("expected all requests stamped after the claims, %d still unstamped", unstamped)
	}
}
