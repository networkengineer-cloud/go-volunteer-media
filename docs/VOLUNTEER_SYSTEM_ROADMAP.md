# Volunteer System Roadmap

Planning tracker for growing MyHAWS from an animal-care portal into the
shelter's full volunteer management system, **replacing Galaxy Digital**.

- Target scale: 500–600 active volunteers.
- Volunteer types are groups (dogs, cats, modsquad, small animals, …) and the
  list will keep growing.
- The shelter checks volunteers in on an onsite iPad today (via Galaxy).

This document tracks the *work* and the *open questions*. It is not the
product roadmap — `ROADMAP.md` stays Product-Owner-owned. When an item here
ships, report it through the `roadmap-update` skill as usual.

## How to use this doc

- Each work item has an ID (`TT-3`, `ON-5`, …). Reference it in issue and PR
  titles so work can be traced back here.
- `[ ]` not started · `[~]` in progress (link the issue/PR) · `[x]` done.
- Open questions are numbered per workstream (`TT-Q2`). When one is answered,
  record the answer in the [Decision log](#decision-log) and strike or remove
  it here.
- Items marked **(blocked: Q#)** can't be designed until that question is
  answered.

## Status at a glance

| # | Workstream | Status | Blocking questions |
|---|---|---|---|
| 0 | [Discovery](#0-discovery) | Not started | — |
| 1 | [Foundations](#1-foundations) | Not started | FD-Q1, FD-Q2 |
| 2 | [Programs (volunteer types)](#2-programs-volunteer-types) | Not started | PR-Q1 |
| 3 | [Time tracking & check-in](#3-time-tracking--check-in) | Not started | TT-Q1, TT-Q2 |
| 4 | [Onboarding, training & compliance](#4-onboarding-training--compliance) | Not started | ON-Q1, ON-Q2, ON-Q3 |
| 5 | [Scheduling v2](#5-scheduling-v2) | Not started | SC-Q1 |
| 6 | [Communications](#6-communications) | Not started | CM-Q1 |
| 7 | [Reporting](#7-reporting) | Not started | RP-Q1 |
| 8 | [Volunteer experience](#8-volunteer-experience) | Not started | — |
| 9 | [Safety & incidents](#9-safety--incidents) | Not started | SF-Q1 |
| 10 | [Galaxy Digital replacement](#10-galaxy-digital-replacement) | Not started | GD-Q1, GD-Q2 |

---

## What exists today

A baseline so work items build on the code rather than around it.

| Area | Current state | Where |
|---|---|---|
| Volunteer types | `Group` is a table, not an enum; per-group toggles `SchedulingEnabled`, `HasProtocols`, `GroupMeEnabled` | `internal/models/models.go` |
| Roles | `User.IsAdmin` (site admin) + `UserGroup.IsGroupAdmin` (per group). No cross-group staff/coordinator role | `models.go`, `group-auth-pattern` skill |
| Accounts | Invite-only; setup-token flow; one invite at a time | `handlers/user_admin.go`, `/setup-password` |
| Volunteer lifecycle | None — a user exists or is soft-deleted | `User` |
| Skill levels | `UserSkillTag` per group, assigned to members; animals carry `walker_status` tags. Nothing links them | `handlers/user_skill_tag.go`, `AnimalTag` |
| Hours | No time tracking. Session reports capture `session_date`/`start`/`end` in `SessionMetadata` | `AnimalComment.Metadata` |
| Scheduling | Recurring per-user hourly `ShiftSlot`s (weekly / biweekly A/B) + `ShiftCoverageRequest` claim/reopen/digest. No capacity, no attendance | `handlers/schedule*.go`, `pages/group/` |
| Training docs | Protocols, scripts, group documents (read-only content, no acknowledgment) | `handlers/protocol.go`, `group_document.go` |
| Messaging | Email (Resend/SMTP), GroupMe bot, announcements, coverage digests | `internal/email`, `internal/groupme` |
| Reporting | Admin dashboard counts; group/user/comment-tag statistics | `handlers/admin_dashboard.go`, `statistics.go` |
| Time | App forces `time.Local` to UTC; `ShiftSlot.Hour` is shelter wall-clock | `internal/database` |
| Login rate limit | 5/min **per IP**, in-memory per container | `cmd/api/main.go` (`authLimiter`), `middleware/ratelimit.go` |

---

## 0. Discovery

Answers here unblock design in every other workstream.

### Work

- [ ] **DS-1** Inventory how the shelter uses Galaxy Digital today: which
  features are actually used, which are paid for but unused.
- [ ] **DS-2** Interview the volunteer coordinator(s): daily workflow, top pain
  points, reports they produce and for whom.
- [ ] **DS-3** Collect the current onboarding path end to end (application →
  active volunteer), per program.
- [ ] **DS-4** List every recurring report/export the shelter produces
  (board, grants, city/county, insurance).

### Open questions

- **DS-Q1** Who is the day-to-day owner/decision-maker for this project on
  the shelter side?
- **DS-Q2** When does the Galaxy Digital contract renew? (Sets the overall
  timeline.)
- **DS-Q3** Is anything contractually or legally required of volunteer records
  (retention period, county/city reporting, insurance)?

---

## 1. Foundations

Cross-cutting changes most later workstreams depend on.

### Work

- [ ] **FD-1** Volunteer status lifecycle on `User`: applicant → onboarding →
  active → inactive → do-not-return, with start/end dates and status history.
- [ ] **FD-2** Volunteer coordinator role: cross-group visibility and
  management without site-admin powers (API tokens, site settings).
  **(blocked: FD-Q1)**
- [ ] **FD-3** Shelter time zone setting; all shift/check-in/"late"/"no-show"
  logic evaluates in that zone.
- [ ] **FD-4** General audit log (who changed what, when, old → new) for
  hours, qualifications, status, and waiver records. Generalise the
  `CommentHistory` pattern.
- [ ] **FD-5** Extended volunteer profile: emergency contact, date of birth or
  minor flag, guardian, pronouns, preferred contact method, address (if
  needed). **(blocked: FD-Q2)**
- [ ] **FD-6** Field-level visibility for sensitive data (DOB, emergency
  contact, background-check status) — coordinator/admin only.
- [ ] **FD-7** Login rate limiting that works when many volunteers share the
  shelter's IP: key on username + IP, and move limiter state out of process
  memory if the app runs more than one replica.
- [ ] **FD-8** Bulk account creation (CSV → invites) for onboarding cohorts.
- [ ] **FD-9** Users page scaled for 600 users: server-side pagination, search,
  filter by status/program/level.

### Open questions

- **FD-Q1** What roles does the shelter actually have? (e.g. staff
  coordinator, program lead, shift lead, mentor, volunteer.) What can each do?
- **FD-Q2** Which profile fields are required, and who may see each one?
- **FD-Q3** Does the app run more than one container replica in prod (affects
  in-memory rate limiting and background sweeps)?
- **FD-Q4** Data retention: how long are inactive volunteers' records kept?
  What is deleted vs. anonymised?

---

## 2. Programs (volunteer types)

Groups already model volunteer types. This workstream makes them fit
programs that aren't animal-centric.

### Work

- [ ] **PR-1** Per-program settings: minimum age, required qualifications to
  join, check-in enabled, onsite vs offsite.
- [ ] **PR-2** Allow programs without animals (front desk, laundry, events,
  transport, fundraising) — toggle the animals section like
  `HasProtocols`/`SchedulingEnabled`. **(blocked: PR-Q1)**
- [ ] **PR-3** Program directory for volunteers: what each program is, its
  requirements, how to join.
- [ ] **PR-4** Request-to-join flow for an additional program, approved by the
  program's group admin.
- [ ] **PR-5** Sub-roles within a program if needed (e.g. dog walker vs dog
  enrichment vs dog playgroup). **(blocked: PR-Q2)**

### Open questions

- **PR-Q1** Full current list of volunteer types, and which ones involve direct
  animal handling?
- **PR-Q2** Are there roles within a program that need separate scheduling,
  training, or reporting?
- **PR-Q3** Is fostering a program in this system? If so, should a foster
  volunteer be linked to the animal record (animals already have a `foster`
  status)?

---

## 3. Time tracking & check-in

Replaces Galaxy's kiosk and hours tracking.

### Work

- [ ] **TT-1** `TimeEntry` model: user, program, check-in, check-out, source
  (kiosk / phone / manual / imported), optional link to the scheduled shift,
  edited-by.
- [ ] **TT-2** iPad kiosk mode (`/kiosk`): authenticated with a scoped device
  token (check-in/out and name lookup only), volunteer search by first name +
  last initial, PIN or QR confirmation, program picker.
  **(blocked: TT-Q1)**
- [ ] **TT-3** Kiosk resilience: tolerate brief Wi-Fi loss (queue locally,
  sync on reconnect); works under iPad Guided Access.
- [ ] **TT-4** Self check-in from a phone (fallback / offsite), optionally
  verified by an onsite QR code or location. **(blocked: TT-Q2)**
- [ ] **TT-5** Auto-close forgotten check-outs at closing time and flag them
  for coordinator review.
- [ ] **TT-6** "Who's in the building now" roster (safety/fire roster).
- [ ] **TT-7** Hour corrections: volunteer requests, coordinator approves;
  every change audited (FD-4).
- [ ] **TT-8** Manual hour submission for offsite work (events, fostering,
  transport) with approval.
- [ ] **TT-9** Connect session reports to time entries: prefill start/end from
  the active check-in; prompt at check-out to log animals worked with.
- [ ] **TT-10** Kiosk checks at check-in: expired waiver, missing required
  training, do-not-return status.

### Open questions

- **TT-Q1** How do volunteers identify themselves at the iPad today (name
  search, email, phone, PIN, badge)? What should they do in the new system?
- **TT-Q2** Is phone check-in wanted, or kiosk only? Any concern about
  volunteers checking in from home?
- **TT-Q3** Is there reliable Wi-Fi at the kiosk location? Is the iPad
  shelter-managed (MDM, Guided Access)?
- **TT-Q4** What counts as volunteer time — only onsite, or also offsite,
  training, events, fostering?
- **TT-Q5** Who approves corrections and manual hours — program group admins,
  a coordinator, or either?
- **TT-Q6** Are breaks tracked, or is it simply in/out?
- **TT-Q7** Do volunteers pick a program at check-in, or is it inferred from
  their schedule?

---

## 4. Onboarding, training & compliance

### Work

- [ ] **ON-1** Public application form (`/apply`) creating an *applicant*, not
  a user; coordinator review queue; approval triggers the existing invite
  flow. Needs CAPTCHA and its own rate limit.
- [ ] **ON-2** Configurable onboarding checklist per program (e.g. application
  → background check → waiver → orientation → shadow shifts → sign-off),
  with progress visible to the volunteer and coordinator.
- [ ] **ON-3** `Training` + `TrainingCompletion` models: type (orientation,
  class, shadow shift, online, protocol acknowledgment), verified-by, expiry.
  **(blocked: ON-Q1)**
- [ ] **ON-4** Handler levels: turn `UserSkillTag` into ordered, earned levels
  driven by training completions.
- [ ] **ON-5** Level-gated animal handling: animals carry a required level;
  volunteers see which animals they may handle; session reports warn on a
  mismatch.
- [ ] **ON-6** Mentor/shadow sign-off in the app.
- [ ] **ON-7** Protocol acknowledgment: "read and acknowledged version N",
  re-acknowledge when the protocol changes.
- [ ] **ON-8** E-signed waivers/agreements: document version + hash, typed
  name, timestamp, IP; annual renewal; guardian signature for minors.
  **(blocked: ON-Q3)**
- [ ] **ON-9** Background check tracking (status + date, or vendor
  integration). **(blocked: ON-Q2)**
- [ ] **ON-10** Expiry reminders for trainings, waivers, checks.
- [ ] **ON-11** Offboarding: status change, access revoked, reason recorded,
  do-not-return enforcement at invite and check-in.

### Open questions

- **ON-Q1** What trainings exist today, per program? Which expire, and after
  how long? Who delivers and signs them off?
- **ON-Q2** Are background checks required? For which programs? Done in-house
  or through a vendor (and does Galaxy integrate with it)?
- **ON-Q3** Which waivers/agreements are signed today? Has legal counsel
  reviewed whether a typed-name e-signature is acceptable?
- **ON-Q4** Are minors allowed? Age limits per program? Supervision rules?
- **ON-Q5** What are the current handler levels (e.g. green/yellow/red), and
  how does a volunteer move between them?
- **ON-Q6** Who reviews applications, and what is the expected turnaround?
- **ON-Q7** Is there an orientation session that must happen before a
  volunteer's first shift? Is attendance tracked?

---

## 5. Scheduling v2

Builds on the existing `ShiftSlot` / coverage-request feature.

### Work

- [ ] **SC-1** Shifts with capacity and requirements (e.g. "Sat 9–11 dog
  walking, 4 people, yellow+"). **(blocked: SC-Q1)**
- [ ] **SC-2** Open sign-up for unfilled shifts.
- [ ] **SC-3** One-off events (offsite adoption events, fundraisers) with
  sign-up and capacity.
- [ ] **SC-4** Attendance: compare schedule vs `TimeEntry` for no-shows and
  late arrivals.
- [ ] **SC-5** Shift reminders (email; SMS per CM-2).
- [ ] **SC-6** Late-cancellation tracking as a reliability signal.
- [ ] **SC-7** Calendar export (ICS) of a volunteer's shifts.

### Open questions

- **SC-Q1** How does the shelter schedule today — fixed weekly regulars, open
  sign-up, or both? Does it differ by program?
- **SC-Q2** Are shift lengths always the current 60/90-minute slots, or do
  programs need other shapes?
- **SC-Q3** Minimum commitment expectations (e.g. one shift per week)? Should
  the system flag volunteers below it?
- **SC-Q4** How are no-shows handled today, and should the system act on them
  automatically?

---

## 6. Communications

### Work

- [ ] **CM-1** Per-type notification preferences (replaces the single
  `EmailNotificationsEnabled` switch).
- [ ] **CM-2** SMS for time-sensitive messages (reminders, urgent coverage).
  **(blocked: CM-Q1)**
- [ ] **CM-3** Segmented messaging (by program, level, status, "scheduled this
  week") built on announcements.
- [ ] **CM-4** Email templates for lifecycle events (application received,
  approved, training due, milestone reached).

### Open questions

- **CM-Q1** Is SMS wanted, and is there budget for a provider (per-message
  cost at 600 users)?
- **CM-Q2** Should GroupMe remain a channel long term?
- **CM-Q3** What messages does Galaxy send today that volunteers rely on?

---

## 7. Reporting

### Work

- [ ] **RP-1** Hours by volunteer / program / month, with CSV export.
- [ ] **RP-2** Active vs lapsed volunteers (no check-in in N days); retention
  by start cohort.
- [ ] **RP-3** Coverage health: fill rate, time-to-claim, no-show rate.
- [ ] **RP-4** Onboarding funnel and expiring certifications.
- [ ] **RP-5** Animal-side: animals with no session in N days; sessions per
  animal; volunteers per animal.
- [ ] **RP-6** Board/grant summary: total hours, unique volunteers, estimated
  dollar value of volunteer time.
- [ ] **RP-7** Hours verification letter (PDF) for court-ordered service,
  students, employer matching.
- [ ] **RP-8** Coordinator dashboard pulling the above together.

### Open questions

- **RP-Q1** Which reports are produced today, for whom, and how often? (DS-4)
- **RP-Q2** What "lapsed" threshold should trigger outreach?
- **RP-Q3** Who requests verification letters, and what must they contain?
- **RP-Q4** Should volunteers see their own stats, and program leads their
  program's?

---

## 8. Volunteer experience

### Work

- [ ] **VX-1** Volunteer Hub page: my hours, my schedule, my trainings and
  what's next, my waivers. (New page — not another tab in `GroupPage.tsx`.)
- [ ] **VX-2** Installable PWA (home-screen icon, mobile-first check-in and
  schedule).
- [ ] **VX-3** Milestones and recognition (50 / 100 / 500 hours, anniversaries).
- [ ] **VX-4** Self-service profile updates for the new fields (FD-5).

### Open questions

- **VX-Q1** Does the shelter do volunteer recognition today? What would they
  want automated?
- **VX-Q2** Do volunteers use Galaxy's mobile app? What would they miss?

---

## 9. Safety & incidents

### Work

- [ ] **SF-1** Person incident reports (volunteer injury, bite to a person,
  near miss) — separate from `AnimalBQIncident`.
- [ ] **SF-2** Emergency contact access from the "who's here" roster (TT-6).
- [ ] **SF-3** Do-not-return list enforced at invite, application, and
  check-in (ON-11).

### Open questions

- **SF-Q1** What is the current incident-reporting process, and who must be
  notified?
- **SF-Q2** Are there insurance or regulatory requirements on incident
  records?

---

## 10. Galaxy Digital replacement

Feature parity and cutover readiness. This section tracks *what* must be
true before Galaxy can be switched off — not how the switch happens.

### Parity checklist

Fill in "Used today?" during discovery (DS-1); only used features are
required for cutover.

| Galaxy capability | Used today? | Covered by |
|---|---|---|
| Volunteer applications / registration | ? | ON-1 |
| Volunteer profiles & custom questions | ? | FD-5 |
| Onsite check-in kiosk | Yes (iPad) | TT-2, TT-3 |
| Mobile check-in / app | ? | TT-4, VX-2 |
| Hours tracking & approval | ? | TT-1, TT-7, TT-8 |
| Shift / opportunity scheduling | ? | SC-1 – SC-3 |
| Qualifications & gating | ? | ON-3 – ON-5 |
| E-sign waivers / documents | ? | ON-8 |
| Background check integration | ? | ON-9 |
| Email / text messaging | ? | CM-1 – CM-3 |
| Reports | ? | RP-1 – RP-8 |
| Teams / groups | ? | Programs (§2) |

### Open questions

- **GD-Q1** Which Galaxy features are used today? (Fills the table above.)
- **GD-Q2** Must historical data (hours, qualifications, signed waivers) come
  across, and how far back?
- **GD-Q3** Who has Galaxy admin access, and can they provide API access or a
  full export?
- **GD-Q4** What would make the coordinator confident enough to switch off
  Galaxy (e.g. a period of running both)?

---

## Decision log

Record answers to open questions here, newest first.

| Date | Question | Decision | Decided by |
|---|---|---|---|
| 2026-09-27 | Integrate with or replace Galaxy Digital? | Replace | Project owner |
