# Second-Review Fix Wave — Shift Coverage Requests

Fixes 4 "Important" issues raised by an independent second-pass code review
of the completed Shift Coverage Requests feature branch.

## Fix 1 — Frontend/backend disagreement on "future"

**Backend** (`internal/handlers/schedule_coverage.go`): `CreateCoverageRequest`'s
date check changed from `!date.After(today)` (strictly future) to
`date.Before(today)` (same-day-or-later allowed, only the past rejected).
Error message updated to "date must not be in the past".

**Frontend** (`frontend/src/pages/group/ScheduleOverview.tsx`):
- `todayIso()` simplified from a local-calendar-date-wrapped-in-`Date.UTC()`
  computation (which was actually local time despite the `Date.UTC` call) to
  `new Date().toISOString().slice(0, 10)`, which is genuinely the UTC
  calendar date — now matching the backend's clock exactly.
- The "Request coverage" button visibility condition changed from
  `date > today` to `date >= today` to match the backend now allowing
  same-day requests.
- All three `.catch(...)` handlers (`handleClaim`, `handleCancelRequest`,
  `handleRequestCoverage`) now extract `err.response?.data?.error` with the
  existing fixed string as fallback (matching the `GroupsPage.tsx` /
  `GroupPage.tsx` inline pattern), and each now also calls `loadOverview()`
  inside the catch block so a failed action doesn't leave stale UI state.

**Tests:**
- Go: split the old "rejects a past or same-day date" test in
  `schedule_coverage_test.go` into "rejects a past date" (yesterday, still
  400) and a new "accepts a same-day date" test (creates a fresh group/user
  with a ShiftSlot matching *today's actual* weekday/hour, since the fixed
  Tuesday-10am fixture used elsewhere doesn't track the real clock; asserts
  201).
- Frontend: added "shows a Request coverage button for the current user's
  own name on the same day (today counts as future)" (pins the clock to the
  slot's own date, asserts the button renders), and "surfaces the backend
  error message when a claim fails, and reloads the overview" (asserts
  `mockShowError` gets the backend's `error` string, and `getOverview` is
  called one additional time from inside the catch).

## Fix 2 — Unthrottled group-wide broadcast

`CreateCoverageRequest` now runs a cooldown check on `rawDB` (the
outer/non-transaction db, since it runs post-commit) immediately before the
notification blocks: counts other non-self `ShiftCoverageRequest` rows by
the same `(group_id, requested_by_user_id)` created in the last 5 minutes.
If any exist, both the email and GroupMe notification blocks are skipped
entirely (the request itself is still created and returned normally either
way).

Notification content was also fixed to include names: `title` is now
`"Coverage needed in {group.Name}"` and `content` is now
`"{displayName(requester)} needs coverage for their {hour} shift on
{date}."` — loaded via `rawDB.First(&requester, targetUserID)` and
`rawDB.Select("name").First(&grp, groupIDUint)`, reusing the existing
`displayName` helper (not redefined).

**Test:** added "second request within the cooldown window still succeeds"
to `schedule_coverage_test.go` — creates a request, cancels it, immediately
creates a second one for the same requester/group, and asserts both the
second create returns 201 and both rows (one cancelled, one active) exist in
the DB. This also structurally exercises the cooldown-check code path with
`nil` email/GroupMe services (as `performCreateCoverageRequest` always
passes), confirming it doesn't panic or error when those services are nil.
Directly asserting "no notification was sent" isn't practical without a
mock email/GroupMe service, per the task's own acknowledgment.

## Fix 3 — Symmetric conflict check for claimants gaining a ShiftSlot

**Root-cause fix** (`internal/handlers/schedule.go`): added
`cancelRedundantClaimsForNewSlots(tx, userID, groupID, slots)`, following the
same style as the existing `cancelOrphanedRequesterCoverageRequests` helper.
It loads all `claimed` `ShiftCoverageRequest` rows where this user is the
claimant, and cancels any whose `(weekday, hour)` now collides with a slot
just added to the user's own recurring schedule. Wired into
`replaceGroupScheduleForUser`'s existing transaction, immediately after the
`cancelOrphanedRequesterCoverageRequests` call — both helpers run inside the
same transaction as the slot replace (no second transaction opened).

**Defensive backstop** (`GetGroupScheduleOverview`, same file): the loop
appending `covering` entries from `claimedByDateHour` now builds a
`seenUserIDs` set from the bucket's already-constructed `members` slice
first, and skips (and marks-seen) any claimant already present, so no future
code path can produce a duplicate `user_id` in one slot's member list.

**Tests:** added to `schedule_test.go`:
- `TestUpdateMemberSchedule_CancelsRedundantClaimsForNewSlots` — Alice
  requests Tuesday 10am coverage, Bob claims it, an admin then adds Bob to
  Tuesday 10am via `UpdateMemberSchedule`; asserts the claim's status flips
  to `cancelled`.
- `TestUpdateMemberSchedule_KeepsClaimsForDifferentSlots` — companion case:
  Bob's claim is for Wednesday 9am, the admin adds him to Tuesday 10am
  (a different weekday/hour); asserts the claim stays `claimed` (no
  over-cancellation).

Both pass.

## Fix 4 — Same user claiming two different requests at the same date/hour

**Backend index** (`internal/database/database.go`, `createCustomIndexes`):
added a second partial unique index immediately after the existing
`idx_coverage_request_active_unique` block, matching its exact style (raw
SQL via `db.Exec`, `logging.WithField("error", ...).Warn(...)` on failure,
`logging.Info(...)` on success):

```sql
CREATE UNIQUE INDEX IF NOT EXISTS idx_coverage_request_claimed_unique
ON shift_coverage_requests (claimed_by_user_id, date, hour)
WHERE status = 'claimed'
```

This is on a different column set (`claimed_by_user_id` vs.
`requested_by_user_id` in the existing index) so it doesn't conflict with or
duplicate the existing one — one prevents duplicate *open* requests by the
requester, the other prevents duplicate *claims* by the claimant.

**Frontend hardening** (`ScheduleOverview.tsx`): the Claim button's
`disabled` condition changed from `busyRequestId === member.coverage_request_id`
to `busyRequestId !== null` (combined with the existing `|| member.conflict`),
so any claim in flight disables every Claim button in the popover, not just
the one clicked. (The Cancel-request button's disabled condition was left
unchanged per the task spec — only Claim was in scope.)

**Tests:**
- Go: added `TestCreateCustomIndexes_CoverageRequestClaimedUnique` to
  `internal/database/database_test.go`, mirroring
  `TestCreateCustomIndexes_CoverageRequestActiveUnique`'s exact structure
  (per-test-run unique shared-cache sqlite DSN, `AutoMigrate` +
  `createCustomIndexes`). Creates two different open requests (different
  requesters) at the same `(date, hour)`, claims the first with a shared
  claimant (succeeds), confirms the second, still-open request for a
  different requester is unaffected, then attempts to claim the second
  request with the SAME claimant at the same `(date, hour)` via a direct
  `Updates(...)` call — asserts this is rejected by the unique index.
- Frontend: added "disables every Claim button in the popover while any
  claim is in flight, not just the one clicked" — two `needs_coverage`
  members in one popover, clicks the first Claim button, asserts BOTH
  buttons are disabled while the claim is pending.

## Files changed

- `internal/handlers/schedule_coverage.go` — Fix 1 (backend date check), Fix 2 (cooldown + notification content)
- `internal/handlers/schedule_coverage_test.go` — Fix 1 test split, Fix 2 test
- `internal/handlers/schedule.go` — Fix 3 (new helper + wiring, dedupe backstop)
- `internal/handlers/schedule_test.go` — Fix 3 tests
- `internal/database/database.go` — Fix 4 (new partial unique index)
- `internal/database/database_test.go` — Fix 4 test
- `frontend/src/pages/group/ScheduleOverview.tsx` — Fix 1 (todayIso, button visibility, toast pattern), Fix 4 (Claim button disabled condition)
- `frontend/src/pages/group/ScheduleOverview.test.tsx` — Fix 1 and Fix 4 frontend tests

## Self-review

- Confirmed `date.Before(today)` rejects yesterday and earlier, accepts
  today and later — verified via the new "rejects a past date" /
  "accepts a same-day date" test pair, both passing.
- Confirmed Fix 3's two helpers (`cancelOrphanedRequesterCoverageRequests`,
  `cancelRedundantClaimsForNewSlots`) both run inside the single
  `db.Transaction(...)` closure already present in
  `replaceGroupScheduleForUser` — no second transaction was introduced.
- Confirmed Fix 4's new index (`claimed_by_user_id, date, hour` WHERE
  `status = 'claimed'`) is on a distinct column set from the existing
  `idx_coverage_request_active_unique` (`group_id, requested_by_user_id,
  date, hour` WHERE `status <> 'cancelled'`) — no conflict or duplication.
- All four fixes have a regression test that would have caught the original
  bug, except Fix 2's notification-suppression itself (can't assert "no
  email sent" without mocking the email/GroupMe services, as anticipated in
  the task) — its test instead proves the cooldown code path doesn't break
  normal request creation and exercises the nil-service path explicitly.

## Verification run at the end

- `go test ./internal/handlers/...` — 1 pre-existing unrelated failure only:
  `TestCreateGroup/accepts_valid_GroupMe_bot_id` (all other subtests,
  including every new/modified coverage-request and schedule test, pass).
- `go test ./internal/database/...` — all pass, including the new
  `TestCreateCustomIndexes_CoverageRequestClaimedUnique`.
- `npx vitest run` (frontend) — 411/412 pass; the 1 failure is the
  pre-existing unrelated `AnimalForm.test.tsx` case.
- `npx tsc --noEmit` — clean, no errors.

## Concerns

None outstanding. All four fixes are self-contained, tested, and the full
verification suite matches the expected pre-existing baseline exactly (no
new failures introduced anywhere).
