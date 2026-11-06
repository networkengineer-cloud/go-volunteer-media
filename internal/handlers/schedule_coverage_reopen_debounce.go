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
// creation digest (which is durable in the rows themselves and survives a
// restart), this is a best-effort in-memory window - losing it on a crash
// just means those reopens fall back to no combined announcement, not to
// no announcement at all, since each one's row still ends up open and
// visible on the schedule regardless.
type reopenDebouncer struct {
	mu      sync.Mutex
	pending map[uint]map[uint]map[uint]struct{} // groupID -> requesterID -> reopened request IDs
	timers  map[uint]*time.Timer
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

	coverageNotifyWG.Add(1)
	d.timers[groupID] = time.AfterFunc(window, func() {
		d.mu.Lock()
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
func (d *reopenDebouncer) reset() {
	d.mu.Lock()
	defer d.mu.Unlock()
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
