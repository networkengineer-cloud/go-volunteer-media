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
func TestSweepCoverageDigests_ConcurrentRepicasClaimOnce(t *testing.T) {
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

	// Force the interleaving that a naive concurrent-goroutine test misses:
	// BOTH replicas must complete their read before EITHER writes. That is
	// the only window in which the claim guard does any work - if one sweep
	// finishes entirely first, the other's read simply returns nothing and
	// the guard is never consulted.
	cutoff := time.Now().Add(-coverageDigestQuietPeriod)
	hardCutoff := time.Now().Add(-coverageDigestMaxDelay)

	targetsA, err := pendingCoverageDigestTargets(db, cutoff, hardCutoff)
	if err != nil {
		t.Fatalf("replica A read: %v", err)
	}
	targetsB, err := pendingCoverageDigestTargets(db, cutoff, hardCutoff)
	if err != nil {
		t.Fatalf("replica B read: %v", err)
	}
	if len(targetsA) == 0 || len(targetsB) == 0 {
		t.Fatalf("both replicas should see the pending target, got A=%d B=%d", len(targetsA), len(targetsB))
	}

	target := coverageDigestTarget{GroupID: group.ID, RequestedByUserID: requester.ID}
	claimedA, err := claimCoverageDigest(db, cutoff, hardCutoff, target)
	if err != nil {
		t.Fatalf("replica A claim: %v", err)
	}
	claimedB, err := claimCoverageDigest(db, cutoff, hardCutoff, target)
	if err != nil {
		t.Fatalf("replica B claim: %v", err)
	}

	if claimedA == claimedB {
		t.Errorf("exactly one replica must win the claim; got A=%v B=%v (the group would be told %s)",
			claimedA, claimedB, map[bool]string{true: "twice", false: "never"}[claimedA])
	}

	var unstamped int64
	db.Model(&models.ShiftCoverageRequest{}).
		Where("group_id = ? AND notified_at IS NULL", group.ID).
		Count(&unstamped)
	if unstamped != 0 {
		t.Errorf("expected all requests stamped after the claim, %d still unstamped", unstamped)
	}
}
