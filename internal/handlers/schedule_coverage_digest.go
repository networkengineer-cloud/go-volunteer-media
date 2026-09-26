package handlers

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/networkengineer-cloud/go-volunteer-media/internal/email"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/groupme"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/logging"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/telemetry"
	"go.opentelemetry.io/otel/metric"
	"gorm.io/gorm"
)

// coverageDigestQuietPeriod is how long a new coverage request waits before
// the group is told about it. A volunteer flagging several shifts - whether
// through the date-range form in one submission or the schedule popover one
// cell at a time - finishes well inside this window, so the whole burst
// collapses into a single announcement instead of one per shift.
//
// The cost is that every coverage request is announced up to this much
// later. That's deliberate: a genuine last-minute emergency is handled by
// the volunteer posting in the group chat directly, not by the portal
// racing to send email.
const coverageDigestQuietPeriod = 3 * time.Minute

// coverageDigestMaxDelay bounds how long a coverage request can sit
// unannounced. The quiet period alone has no upper bound: a volunteer who
// keeps adding shifts every couple of minutes keeps resetting it, and the
// group would never be told at all. Once the OLDEST pending request in a
// burst passes this age, the burst ships regardless of how recently the
// newest one landed.
const coverageDigestMaxDelay = 15 * time.Minute

// coverageDigestStopTimeout bounds how long stop() waits for an in-flight
// tick, mirroring embedding.StartReconciliationSweep's sweepStopTimeout and
// the other bounded shutdown waits in cmd/api/main.go.
const coverageDigestStopTimeout = 10 * time.Second

// coverageNotifyWG tracks in-flight coverage announcements that were
// deliberately detached from a request (see ReopenCoverageRequest), so
// shutdown can drain them the way WaitForPendingEmbeds drains write-path
// embed goroutines. Without it, the DB pool could close mid-send and lose an
// announcement whose rows are already stamped - unrecoverable, because the
// sweep will never see those rows again.
var coverageNotifyWG sync.WaitGroup

// coverageNotifyDrainTimeout bounds the shutdown wait, mirroring the other
// bounded drains in this package. Generous because an announcement fans out
// one email per group member over SMTP.
const coverageNotifyDrainTimeout = 30 * time.Second

// trackCoverageNotification runs fn in a goroutine that shutdown knows about.
// A detached send that uses a bare `go` instead is invisible to
// WaitForPendingCoverageNotifications and can be cut off by the closing DB
// pool.
//
// That matters most where the send's rows have ALREADY been stamped as
// announced - the reopen and reminder paths - because there a lost send is
// never retried by the sweep. The per-recipient sends elsewhere in
// schedule_coverage.go (claim, claim-batch, reassignment) are still bare
// `go` and predate this; they lose at most one notification with no
// database state claiming otherwise, so they have not been converted.
func trackCoverageNotification(fn func()) {
	coverageNotifyWG.Add(1)
	go func() {
		defer coverageNotifyWG.Done()
		fn()
	}()
}

// WaitForPendingCoverageNotifications blocks (up to
// coverageNotifyDrainTimeout) until every detached coverage announcement has
// finished. Call during graceful shutdown, after the HTTP server has stopped
// accepting requests but before closing the DB pool.
func WaitForPendingCoverageNotifications() {
	done := make(chan struct{})
	go func() {
		coverageNotifyWG.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(coverageNotifyDrainTimeout):
		logging.Warn(fmt.Sprintf("Coverage announcement goroutines did not finish within %s of shutdown signal; proceeding with shutdown anyway", coverageNotifyDrainTimeout))
	}
}

// coverageDigestTarget is one (group, requester) pair with at least one
// coverage request due for announcement. Requests are grouped this way
// because notifyGroupOfOpenCoverageRequests sends one summary per requester
// per group - it lists every open request that person currently has there.
type coverageDigestTarget struct {
	GroupID           uint
	RequestedByUserID uint
}

// StartCoverageDigestSweep runs the periodic pass that announces coverage
// requests once their quiet period has elapsed. Creation no longer notifies
// inline; it just leaves notified_at NULL, and this sweep is what turns a
// burst of rows into one email and one GroupMe post.
//
// Durability comes from the rows themselves, not from any in-process timer
// or cache: an open request with notified_at NULL *is* the pending work
// item. A pod that restarts mid-window loses nothing - the next tick, on
// this replica or another, picks the row up and the announcement is merely
// late.
//
// Returns a stop function; call it during graceful shutdown. stop() blocks
// (up to coverageDigestStopTimeout) until the goroutine has actually
// exited, so a caller that closes the DB pool immediately afterwards does
// not race an in-flight tick's writes against a closed *sql.DB. Best-effort
// rather than absolute: the wait is bounded, and a tick fanning email out to
// a large group over SMTP can exceed coverageDigestStopTimeout, after which
// shutdown proceeds anyway. This holds only
// because notifyGroupOfOpenCoverageRequests sends synchronously: the
// announcement is part of the tick, not detached from it. If it were
// detached, a shutdown landing just after a claim would close the pool
// underneath the send and lose an announcement whose rows are already
// stamped - unrecoverable, since the sweep would never see them again.
func StartCoverageDigestSweep(db *gorm.DB, emailService *email.Service, groupMeService *groupme.Service, interval time.Duration) (stop func()) {
	return startCoverageDigestSweepWithNotify(db, coverageDigestNotifier(db, emailService, groupMeService), interval)
}

// startCoverageDigestSweepWithNotify is StartCoverageDigestSweep with the
// announcement callback injected, so a test can assert that stop() really
// does wait for an in-flight announcement rather than only for the tick.
func startCoverageDigestSweepWithNotify(db *gorm.DB, notify func(coverageDigestTarget), interval time.Duration) (stop func()) {
	meter := telemetry.Meter("internal/handlers")
	heartbeat := telemetry.NewInstrument("coverage.digest.sweep.heartbeat", func() (metric.Int64Counter, error) {
		return meter.Int64Counter(
			"coverage.digest.sweep.heartbeat",
			metric.WithDescription("Incremented once per coverage digest sweep tick, regardless of outcome — absence signals the sweep goroutine died"),
		)
	})

	ticker := time.NewTicker(interval)
	done := make(chan struct{})
	finished := make(chan struct{})

	go func() {
		defer close(finished)
		for {
			select {
			case <-ticker.C:
				heartbeat.Add(context.Background(), 1)
				sweepCoverageDigests(db, notify)
			case <-done:
				ticker.Stop()
				return
			}
		}
	}()

	var once sync.Once
	return func() {
		once.Do(func() { close(done) })
		select {
		case <-finished:
		case <-time.After(coverageDigestStopTimeout):
			logging.Warn(fmt.Sprintf("Coverage digest sweep did not stop within %s of shutdown signal; proceeding with shutdown anyway", coverageDigestStopTimeout))
		}
	}
}

// coverageDigestNotifier builds the announcement callback the sweep runs
// for each claimed target. Separate from StartCoverageDigestSweep so tests
// can drive the real email/GroupMe path — the flag gating inside
// notifyGroupOfOpenCoverageRequests included — without standing up a ticker.
func coverageDigestNotifier(db *gorm.DB, emailService *email.Service, groupMeService *groupme.Service) func(coverageDigestTarget) {
	return func(target coverageDigestTarget) {
		notifyGroupOfOpenCoverageRequests(db, emailService, groupMeService, target.GroupID, target.RequestedByUserID)
	}
}

// pendingCoverageDigestTargets lists the (group, requester) pairs that are
// due: either they have gone quiet - every unannounced request of theirs
// predates the cutoff, which is what HAVING MAX(created_at) tests - or their
// oldest unannounced request has passed the hard cap, which MIN(created_at)
// tests and which ships the burst even though a fresh request is still in
// it.
//
// Keying on the NEWEST pending request rather than each row's own age is
// the whole point. A volunteer flagging shifts over a few minutes would
// otherwise straddle the cutoff: the early ones would be announced now and
// the later ones announced again on a subsequent tick, splitting one burst
// back into the several emails this sweep exists to prevent. Waiting until
// they stop adding means one announcement covers the lot.
//
// Separate from claimCoverageDigest so a test can interleave the read and
// the write the way two replicas do - the read is where sweeps overlap.
func pendingCoverageDigestTargets(db *gorm.DB, cutoff, hardCutoff time.Time) ([]coverageDigestTarget, error) {
	var targets []coverageDigestTarget
	err := db.Model(&models.ShiftCoverageRequest{}).
		Select("group_id, requested_by_user_id").
		Where("status = ? AND notified_at IS NULL", models.CoverageRequestOpen).
		Group("group_id, requested_by_user_id").
		Having("MAX(created_at) < ? OR MIN(created_at) < ?", cutoff, hardCutoff).
		Find(&targets).Error
	return targets, err
}

// claimCoverageDigest stamps notified_at on one target's due requests and
// reports whether this caller is the one that won them. It is the guard
// against a group hearing the same announcement twice: prod scales to
// several container replicas (terraform/environments/prod/main.tf), so two
// pods can hold the same target from pendingCoverageDigestTargets at once.
// Both issue this UPDATE, the second blocks on the first's row locks, then
// re-evaluates its WHERE under READ COMMITTED, sees notified_at already
// set, and updates nothing - so it returns false and stays quiet.
//
// Claiming before announcing rather than after means a crash in between
// drops an announcement instead of duplicating one. That is the better
// failure here: notifyGroupOfOpenCoverageRequests always sends the
// requester's *complete* open list, so their next request re-announces
// whatever a dropped digest missed.
//
// KNOWN LIMITATION: the cutoff re-check closes the read-to-write gap only
// for requests committed before this UPDATE takes its snapshot. Under READ
// COMMITTED a request committed after that instant is invisible to the
// NOT EXISTS (so the claim still succeeds) and outside the UPDATE's scope
// (so it stays unstamped), producing one extra announcement on the next
// tick - the split this sweep exists to avoid, in a window of a few
// milliseconds. Closing it properly needs SERIALIZABLE or an explicit lock
// on the requester's rows, which is a lot of machinery for an outcome no
// worse than the pre-digest behavior. Left as a known edge.
func claimCoverageDigest(db *gorm.DB, cutoff, hardCutoff time.Time, target coverageDigestTarget) (bool, error) {
	// The cutoff is re-checked here, not just in the read above, so a
	// request created in the gap between the two queries makes this claim
	// find nothing and the whole burst waits for the next tick - rather
	// than being announced without its newest shift.
	result := db.Model(&models.ShiftCoverageRequest{}).
		Where(`status = ? AND notified_at IS NULL AND group_id = ? AND requested_by_user_id = ?
			AND (
				NOT EXISTS (
					SELECT 1 FROM shift_coverage_requests newer
					WHERE newer.group_id = shift_coverage_requests.group_id
					  AND newer.requested_by_user_id = shift_coverage_requests.requested_by_user_id
					  AND newer.status = ?
					  AND newer.notified_at IS NULL
					  AND newer.created_at >= ?
				)
				OR EXISTS (
					SELECT 1 FROM shift_coverage_requests overdue
					WHERE overdue.group_id = shift_coverage_requests.group_id
					  AND overdue.requested_by_user_id = shift_coverage_requests.requested_by_user_id
					  AND overdue.status = ?
					  AND overdue.notified_at IS NULL
					  AND overdue.created_at < ?
				)
			)`,
			models.CoverageRequestOpen, target.GroupID, target.RequestedByUserID,
			models.CoverageRequestOpen, cutoff,
			models.CoverageRequestOpen, hardCutoff).
		Update("notified_at", time.Now())
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

// sweepCoverageDigests announces every pair that is due, claiming each
// before it announces. notify is injected so the claim-and-batch logic can
// be tested without email or GroupMe services.
func sweepCoverageDigests(db *gorm.DB, notify func(coverageDigestTarget)) {
	now := time.Now()
	cutoff := now.Add(-coverageDigestQuietPeriod)
	hardCutoff := now.Add(-coverageDigestMaxDelay)

	targets, err := pendingCoverageDigestTargets(db, cutoff, hardCutoff)
	if err != nil {
		logging.Error("Failed to list pending coverage digests", err)
		return
	}

	for _, target := range targets {
		claimed, err := claimCoverageDigest(db, cutoff, hardCutoff, target)
		if err != nil {
			logging.WithFields(map[string]interface{}{
				"group_id": target.GroupID,
			}).Error("Failed to claim coverage requests for digest", err)
			continue
		}
		if !claimed {
			// Another replica got there first; it owns the announcement.
			continue
		}
		notify(target)
	}
}
