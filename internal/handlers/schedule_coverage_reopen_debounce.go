package handlers

import (
	"sync"
	"time"
)

// reopenDebounceWindow is how long a reopen's announcement waits for
// sibling reopens in the SAME GROUP - by any requester - before sending, so
// two reopens landing seconds apart (the incident that prompted this: two
// reopens three seconds apart, each firing its own full-group email)
// collapse into one combined announcement instead of two overlapping ones.
//
// Deliberately short relative to coverageDigestQuietPeriod's 3 minutes: a
// volunteer giving back a shift at the last minute still needs the group
// told promptly, unlike a fresh request where a few minutes' delay is an
// accepted tradeoff (see that constant's comment). This only exists to
// catch genuinely back-to-back reopens, not to batch over minutes the way
// the creation digest does.
//
// var, not const, so a test can shrink it instead of waiting out the real
// window.
var reopenDebounceWindow = 5 * time.Second

// reopenDebouncer batches reopen announcements per group: every reopen in
// the same group within reopenDebounceWindow of the first one joins that
// batch's single announcement instead of firing its own. Unlike the
// creation digest (whose pending state lives in the rows themselves, so a
// restart just delays an announcement), this window is in-memory only. A
// hard crash (not a graceful shutdown - see coverageNotifyWG's drain in
// cmd/api/main.go) during the window loses the announcement entirely: the
// reopen handler already stamped notified_at synchronously before
// enqueueing (see ReopenCoverageRequest), so the row no longer looks
// pending to the digest sweep either, and nothing will ever re-announce
// it. The row itself is unaffected - it's still open and visible on the
// schedule - only the proactive "hey, this needs coverage" notice is lost.
// This is the same tradeoff claimCoverageDigest's own comment accepts for
// its cross-replica claim ("a crash in between drops an announcement
// instead of duplicating one"), just with a longer window (seconds,
// not milliseconds) since closing the sweep race in ReopenCoverageRequest
// requires stamping before the window rather than after it.
//
// KNOWN LIMITATION: this batching is per PROCESS, not per group. Prod runs
// up to 3 replicas (terraform/environments/prod/main.tf), so two reopens in
// the same group landing on different replicas within the window still
// produce two separate announcements - each one still correct and
// individually coalesced on its own replica, just not coalesced with each
// other. Closing that fully would need a DB-backed claim the way the
// creation digest sweep does it, which is a lot more machinery for a case
// the incident that prompted this (two reopens three seconds apart, same
// request handled by one replica) didn't actually hit. Left as a known
// edge, same spirit as claimCoverageDigest's own documented limitation.
type reopenDebouncer struct {
	mu      sync.Mutex
	pending map[uint]map[uint]map[uint]struct{} // groupID -> requesterID -> reopened request IDs
	timers  map[uint]*time.Timer
	// generation is bumped by reset(). A timer's callback captures the
	// generation current at enqueue time and compares it against this
	// field before touching pending/timers, so a callback whose timer had
	// already fired (and was merely waiting on d.mu) when reset() ran
	// can't act on state that belongs to whatever enqueued after the
	// reset - see reset()'s comment.
	generation int
}

func newReopenDebouncer() *reopenDebouncer {
	return &reopenDebouncer{
		pending: make(map[uint]map[uint]map[uint]struct{}),
		timers:  make(map[uint]*time.Timer),
	}
}

// coverageReopenDebouncer is the process-wide instance ReopenCoverageRequest
// enqueues into. A package var, like coverageNotifyWG, because reopens from
// different HTTP requests must share one batch per group.
var coverageReopenDebouncer = newReopenDebouncer()

// enqueue adds (requesterID, requestID) to groupID's pending batch and arms
// a timer to flush it after window, unless one is already running for this
// group - a second reopen within the window joins the first's batch
// instead of resetting the clock, so a steady trickle of reopens can't
// hold the group's notification off indefinitely.
//
// The pending announcement is added to coverageNotifyWG here, at enqueue
// time, rather than when the timer fires, so a shutdown landing during the
// debounce window still waits for it - the same guarantee
// trackCoverageNotification gives every other detached coverage send (see
// its comment in schedule_coverage_digest.go).
func (d *reopenDebouncer) enqueue(groupID, requesterID, requestID uint, window time.Duration, flush func(groupID uint, reopenedByRequester map[uint][]uint)) {
	d.mu.Lock()
	if d.pending[groupID] == nil {
		d.pending[groupID] = make(map[uint]map[uint]struct{})
	}
	if d.pending[groupID][requesterID] == nil {
		d.pending[groupID][requesterID] = make(map[uint]struct{})
	}
	d.pending[groupID][requesterID][requestID] = struct{}{}

	if _, running := d.timers[groupID]; running {
		d.mu.Unlock()
		return
	}

	gen := d.generation
	coverageNotifyWG.Add(1)
	d.timers[groupID] = time.AfterFunc(window, func() {
		d.mu.Lock()
		if d.generation != gen {
			// reset() ran between this timer being armed and firing - see
			// its comment. d.pending/d.timers now belong to whatever
			// enqueued after that reset, so this callback must not touch
			// them; it only owes the Add(1) its own enqueue call made.
			d.mu.Unlock()
			coverageNotifyWG.Done()
			return
		}
		batch := d.pending[groupID]
		delete(d.pending, groupID)
		delete(d.timers, groupID)
		d.mu.Unlock()

		defer coverageNotifyWG.Done()
		reopenedByRequester := make(map[uint][]uint, len(batch))
		for rid, ids := range batch {
			idList := make([]uint, 0, len(ids))
			for id := range ids {
				idList = append(idList, id)
			}
			reopenedByRequester[rid] = idList
		}
		flush(groupID, reopenedByRequester)
	})
	d.mu.Unlock()
}

// reset cancels every pending timer and discards all pending state,
// draining the coverageNotifyWG entry each one was holding. Test-only:
// coverageReopenDebouncer is a single process-wide instance, and every test
// that exercises ReopenCoverageRequest shares it, but each test's own
// *gorm.DB is otherwise fully isolated (see SetupTestDB) - without this,
// a group id reused across two unrelated tests' in-memory databases (which
// is the common case, since each starts its own autoincrement from 1)
// could join a batch meant for a different test's db entirely. Called from
// SetupTestDB so every test starts with a clean slate regardless of what a
// previous test left pending.
//
// Bumping generation (not just clearing pending/timers) matters for a
// narrow race: a timer that fired just before reset() runs, but whose
// callback is still waiting on d.mu, has already been dropped from
// d.timers below by the time it gets the lock - timer.Stop() on it would
// return false, so this loop wouldn't double-count its Done(). Without the
// generation check, that callback would then read the FRESH, post-reset
// d.pending[groupID] - which may by then hold a different test's reopen,
// enqueued after this reset - and flush that batch through this callback's
// stale rawDB/emailService closure instead of its own. The generation
// bump makes that callback recognize it's stale and bail out, owing only
// its own Done(), rather than acting on state it no longer owns.
func (d *reopenDebouncer) reset() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.generation++
	for groupID, timer := range d.timers {
		if timer.Stop() {
			// Stopped before it fired, so its own Done() will never run -
			// this owes the one Add(1) its enqueue call made. If Stop()
			// returns false the AfterFunc is already running (or has
			// finished) and will call Done() itself.
			coverageNotifyWG.Done()
		}
		delete(d.timers, groupID)
	}
	d.pending = make(map[uint]map[uint]map[uint]struct{})
}
