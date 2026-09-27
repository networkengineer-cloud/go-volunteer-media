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
| A | [Architecture prerequisites](#a-architecture-prerequisites) | Not started | AR-Q1, AR-Q2 |
| S | [Agent skills](#s-agent-skills) | In progress (#325) | — |
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
| Replicas | Prod scales 1–3 container replicas (`max_replicas = 3`) | `terraform/environments/prod/` |
| Schema changes | GORM AutoMigrate + ~39 raw SQL statements on startup. No versions, no rename/drop, no rollback | `internal/database/database.go` |
| Authorization | ~66 inline group-membership / group-admin checks across handlers; no central policy | `internal/handlers/`, `group-auth-pattern` skill |
| Background jobs | In-process goroutine tickers per replica (embedding sweep, coverage digest). The digest claims rows atomically, so it is replica-safe | `internal/embedding/sweep.go`, `handlers/schedule_coverage_digest.go` |
| Backend structure | One flat `handlers` package (83 files, ~42k lines) holding business logic; one `models.go`; one 638-line route table | `internal/handlers/`, `cmd/api/main.go` |
| Frontend data | Hand-rolled axios calls in `useEffect`; no query cache. Largest pages: `GroupPage.tsx` 1,934 lines, `UsersPage.tsx` 1,839, `AnimalForm.tsx` 1,612 | `frontend/src/api/client.ts`, `pages/` |
| Frontend styling | Global, unscoped CSS per page; known cascade hazards | `frontend-styling` skill |
| Sessions | JWT in `localStorage`, 24h expiry | `internal/auth/auth.go`, `api/client.ts` |
| Tests | Handler tests mostly SQLite; `_postgres_test.go` suffix for real-Postgres tests (31 today; they skip when no DB is reachable) | `internal/handlers/` |
| CI | `test.yml` (full suite) is `workflow_dispatch` only. Until #325 its backend job had no Postgres, so the Postgres tests always skipped. #325 adds `pr-checks.yml`, the first checks that run automatically on PRs | `.github/workflows/` |
| Code/security review tooling | CodeQL (GitHub default setup) and GitGuardian run on PRs; Renovate for updates. #325 adds golangci-lint (new issues), ESLint + `tsc -b`, Semgrep (custom + registry rules), govulncheck, OSV-Scanner, Trivy, actionlint, zizmor | `.github/workflows/pr-checks.yml`, `tools/semgrep/`, `.golangci.yml` |
| Agent skills | 9 skills in `.claude/skills/`; 4 corrected against the code in #325 — see [workstream S](#s-agent-skills) | `.claude/skills/` |

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

## A. Architecture prerequisites

The stack (Go/Gin, Postgres/GORM, React/TS/Vite, single container on Azure
Container Apps) is suitable for this scale — no rewrite or language change.
These are targeted changes that make the expansion safe to build. Items
AR-1 – AR-5 should land **before** the feature workstreams that depend on
them; AR-6 – AR-9 are conventions for *new* code, not rewrites of old code.

### Before feature work

- [ ] **AR-1** Versioned schema migrations. Adopt a migration tool with
  numbered up/down files, baseline it from the current schema, and keep
  AutoMigrate only until the baseline is in place. Required for the
  status backfill (FD-1) and the `UserSkillTag` → levels conversion (ON-4).
  **(blocked: AR-Q1)**
- [ ] **AR-2** Central authorization policy: one helper (e.g.
  `authz.Can(user, action, group)`) that replaces the inline checks, with
  tests per role. Do this *before* adding the coordinator role (FD-2), kiosk
  scope (TT-2), applicants (ON-1) or mentors (ON-6). Update the
  `group-auth-pattern` skill to match.
- [ ] **AR-3** Replica-safe background jobs: a Postgres-backed job runner (or
  a shared locking helper) with retries, used by every new job — auto-close
  check-outs (TT-5), expiry reminders (ON-10), shift reminders (SC-5), SMS
  (CM-2). Follow the coverage digest's atomic-claim pattern at minimum.
  **(blocked: AR-Q2)**
- [ ] **AR-4** Login rate limiting that works behind a shared shelter IP and
  across replicas: key on username + IP, store limiter state in Postgres or
  another shared store; kiosk traffic on its own path.
  *(moved from FD-7)*
- [ ] **AR-5** Shelter time zone setting; all shift / check-in / "late" /
  "no-show" logic evaluates in that zone. *(moved from FD-3)*

### Conventions for new code

- [ ] **AR-6** Backend: new domains get their own package (e.g.
  `internal/timeclock`, `internal/training`) with business logic in a
  service layer and thin handlers. New models go in per-domain files inside
  `internal/models` (same package). Existing handlers are not moved.
- [ ] **AR-7** Split the route table in `cmd/api/main.go` into per-domain
  `RegisterXRoutes(router, deps)` functions. Mechanical, low risk; do it
  before the route count grows further.
- [ ] **AR-8** Frontend data layer: adopt a query/cache library (e.g.
  TanStack Query) for new pages, wrapping the existing `client.ts` methods
  rather than replacing them. Needed for the kiosk, live roster (TT-6) and
  dashboards (RP-8).
- [ ] **AR-9** Frontend styling: CSS Modules for new pages to avoid the
  global cascade. Update the `frontend-styling` skill.
- [ ] **AR-10** New volunteer features ship as new pages (Volunteer Hub,
  Coordinator, Kiosk), not tabs in `GroupPage.tsx`. Split the large existing
  pages only when they are touched; `UsersPage.tsx` gets rebuilt under FD-9.
- [ ] **AR-11** Concurrency-sensitive features (capacity sign-ups,
  simultaneous check-ins) are tested against real Postgres
  (`_postgres_test.go`), not SQLite.
- [ ] **AR-12** Record AR-6 – AR-11 in `CLAUDE.md` and
  `.github/copilot-instructions.md` so every contributor follows them.
  Skill updates for each convention are tracked in
  [workstream S](#s-agent-skills) (SK-5 – SK-9).
- [~] **AR-15** Run the `_postgres_test.go` tests in CI: Postgres (pgvector)
  service added to the backend test job (#325). Verified locally: all 31
  pass in ~25s.
- [ ] **AR-16** Run `test.yml` automatically on pull requests (it is manual
  only today). **(blocked: AR-Q5)**

### PR review tooling

`pr-checks.yml` runs on every PR, on pushes to `main` (so code scanning has a
baseline to compare PRs against) and weekly (new CVEs in unchanged
dependencies). Every tool is free and open source, installed at a pinned
version; the only actions used are GitHub's own, pinned to commit SHAs.
Diff-aware jobs report only what the PR introduces; security scanners
upload SARIF to code scanning (free: the repo is public).

| Job | Tools | Reports | Existing findings on `main` |
|---|---|---|---|
| Go lint | golangci-lint v2 (`.golangci.yml`: standard + gosec, errorlint, bodyclose, rowserrcheck, sqlclosecheck, noctx) | New issues only (`--new-from-rev`) | 105 (hidden) |
| Frontend lint & types | ESLint on changed files; `tsc -b` | Changed files; tsc report-only | ESLint 69 errors repo-wide; `tsc -b` 8 errors |
| Semgrep | Custom rules in `tools/semgrep/` + `p/golang`, `p/react`, `p/typescript` | New findings only (`--baseline-commit`) | Custom rules: 3 (pre-login axios calls) |
| Dependency vulns | govulncheck, OSV-Scanner (go.mod + package-lock) | Code scanning | Unknown — couldn't reach vuln DBs from the dev sandbox; first CI run will tell |
| IaC & Dockerfile | Trivy config (HIGH/CRITICAL) | Code scanning | 5 (see AR-29, AR-30) |
| GitHub Actions | actionlint (fails on errors), zizmor | actionlint in log; zizmor to code scanning | actionlint 0 (after #325 fixes); zizmor 103 |

- [~] **AR-24** Add `pr-checks.yml`, `.golangci.yml`, `.semgrepignore` and
  the Semgrep rules (#325). Verified locally: actionlint clean, zero zizmor
  findings on the new workflow, `--new-from-rev` and `--baseline-commit`
  each report only a deliberately introduced issue, rule tests 5/5, zero
  false positives on current code.
- [~] **AR-25** Custom Semgrep rules (`tools/semgrep/`, with tests):
  `handler-reassigns-shared-db`, `handler-missing-request-scoped-db`,
  `internal-error-leaks-to-client`, `handler-writes-local-filesystem`,
  `raw-axios-http-call`. Add a rule whenever a review comment would
  otherwise repeat a CLAUDE.md convention.
- [ ] **AR-26** Fix the 8 `tsc -b` errors (4 in `AnimalForm.tsx`, 1 in
  `PhotoGallery.tsx`, 3 in tests), then remove `continue-on-error` from the
  type-check step. (CLAUDE.md and `frontend-styling` said `npx tsc --noEmit`,
  which checks nothing because the root `tsconfig.json` only holds project
  references; corrected to `tsc -b` in #325.)
- [ ] **AR-27** After a trial period, mark the checks that should block as
  required status checks in branch protection. **(blocked: AR-Q6)**
- [ ] **AR-28** Decide on the 3 pre-login `axios.post` calls
  (`Login.tsx`, `ResetPassword.tsx`, `SetupPassword.tsx`): route through
  `client.ts` or annotate with `nosemgrep` and the reason.
- [ ] **AR-29** Trivy: Azure storage account and Key Vault have no network
  rules / default-allow ACLs in both dev and prod (AZU-0012, AZU-0013).
  Decide whether to restrict them (private endpoints or IP rules) or
  accept and suppress with a reason.
- [ ] **AR-30** zizmor: pin the ~57 unpinned action references in existing
  workflows to SHAs (Renovate can keep them updated), add
  `persist-credentials: false` to checkouts, and review the
  `template-injection` findings in `build-image.yml` and the Terraform
  workflows.
- [x] **AR-31** actionlint: the Terraform deploy workflows referenced step
  IDs (`fmt`, `init`, `validate`) that didn't exist, so their plan
  summaries always showed empty outcomes. Fixed in #325.
- [ ] **AR-32** Add `eslint-plugin-jsx-a11y` (the page skills require
  accessibility; nothing checks it). Needs a `package.json` change and a
  baseline.
- [ ] **AR-33** Later additions: Squawk (migration linting) with AR-1; an
  OWASP ZAP baseline scan against dev once public pages exist (ON-1,
  TT-2); OpenSSF Scorecard; Gitleaks as a pre-commit hook.

### New agent skills

*Moved to [workstream S](#s-agent-skills): AR-17 → SK-10, AR-18 → SK-11,
AR-19 → SK-12, AR-20 → SK-13, AR-21 → SK-14, AR-22 → SK-15,
AR-23 → SK-16.*

### Worth deciding (not blocking)

- [ ] **AR-13** Session storage: evaluate httpOnly cookie sessions instead
  of `localStorage` JWTs once the app holds DOB, emergency contacts and
  background-check status. **(AR-Q3)**
- [ ] **AR-14** Kiosk device authentication: a scoped, revocable device
  credential, separate from user sessions (prerequisite for TT-2).

### Open questions

- **AR-Q1** Which migration tool (e.g. goose vs golang-migrate), and should
  migrations run on startup or as a separate deploy step?
- **AR-Q2** Adopt a Postgres job queue library (e.g. River) or hand-roll
  advisory-lock-based jobs?
- **AR-Q3** Move to httpOnly cookie sessions, or keep bearer tokens and
  harden (shorter expiry, refresh tokens)?
- **AR-Q4** Should production stay at up to 3 replicas, or is a single
  replica acceptable (simplifies jobs and rate limiting, reduces
  availability)?
- **AR-Q5** Should `test.yml` run on every pull request? (Costs Actions
  minutes; it was presumably made manual deliberately.)
- **AR-Q6** Which PR checks should be required (blocking)? Suggested after a
  two-week trial: Go lint, Semgrep, actionlint, and code scanning at
  HIGH/CRITICAL for new alerts. Who owns triaging code scanning alerts?
- **AR-Q7** Keep ESLint scoped to changed files (touching a file means
  fixing its existing errors), or fix the 69 existing errors once and lint
  everything?

---

## S. Agent skills

Skills in `.claude/skills/` are how Claude Code (and, via
`.github/copilot-instructions.md`, Copilot) learn this codebase's patterns.
Every feature in this roadmap will be built through them, so a wrong skill
spreads a wrong pattern into every new endpoint. This workstream keeps the
existing skills accurate and adds new ones as the architecture changes.

**Rule:** write or update a skill in the same PR as the pattern it
documents — not before, or it describes code that doesn't exist.

### Skill inventory

| Skill | Reviewed against code | Status | Needs updating when |
|---|---|---|---|
| `add-api-endpoint` (+ `handler-template.go`) | 2026-09-27 | Corrected in #325 | AR-1 (migrations), AR-6 (domain packages), AR-7 (route split) |
| `group-auth-pattern` | 2026-09-27 | Corrected in #325 | AR-2 (central policy) — rewrite |
| `image-upload` | 2026-09-27 | Rewritten in #325 | New private file types (waivers, incident attachments) |
| `run-tests` | 2026-09-27 | Extended in #325 | AR-16 (CI on PRs) |
| `add-frontend-page` | 2026-09-27 | Accurate | AR-8 (data layer), AR-9 (CSS Modules), AR-10 (new pages) |
| `frontend-styling` | 2026-09-27 | Type-check command corrected in #325 | AR-9 (CSS Modules) |
| `playwright-e2e-test` | 2026-09-27 | Accurate | Kiosk flows (TT-2) |
| `dev-environment` | 2026-09-27 | Accurate | AR-1 (migration commands), new env vars |
| `roadmap-update` | 2026-09-27 | Accurate | SK-9 |

### Corrections to existing skills

- [~] **SK-1** `add-api-endpoint` (#325): models declare base fields
  explicitly (none embed `gorm.Model`); handlers start with
  `db := middleware.GetDB(c, db)`; use `middleware.GetUserID`; log real
  errors and return generic messages; fixed `respondUnauthorized` arity;
  guidance on `_postgres_test.go`. Template compile-checked.
- [~] **SK-2** `group-auth-pattern` (#325): fixed `respondUnauthorized`
  arity, added `/videos/:uuid` to public routes, aligned tier count,
  pointer to AR-2.
- [~] **SK-3** `image-upload` (#325): rewritten around the real providers
  (`postgres` default, `azure`). The Postgres provider stores no bytes, so
  handlers must persist them in the row; `group_document.go` is the
  reference; auth requirements for serve routes.
- [~] **SK-4** `run-tests` (#325): running Postgres-backed tests, `-race`
  timeout, known-failure baseline. Also fixed the
  `user-invokable` → `user-invocable` frontmatter key on two skills.

### Updates to existing skills (as architecture lands)

- [ ] **SK-5** `add-api-endpoint`: replace the AutoMigrate step with
  versioned migrations (AR-1); per-domain packages and `RegisterXRoutes`
  (AR-6, AR-7) — or hand off to `add-domain` (SK-14).
- [ ] **SK-6** `group-auth-pattern`: rewrite around the central policy
  helper and the new roles (AR-2, FD-2, AR-14).
- [ ] **SK-7** `add-frontend-page` + `frontend-styling`: query/cache library
  (AR-8), CSS Modules (AR-9), new-pages-not-tabs (AR-10).
- [ ] **SK-8** `dev-environment` + `run-tests`: migration commands (AR-1),
  CI-on-PR behaviour (AR-16), any new env vars (SMS, kiosk).
- [ ] **SK-9** `roadmap-update`: also tick items and update the decision log
  in this doc when a roadmap item ships.

### New skills

- [ ] **SK-10** `schema-migration` — with AR-1: writing a versioned
  migration, backfills (FD-1 statuses, ON-4 levels), Postgres testing, what
  is safe on a live database.
- [ ] **SK-11** `background-job` — with AR-3: replica-safe jobs modelled on
  the coverage digest (atomic claim, heartbeat metric, stop func that
  waits, tests). Used by TT-5, ON-10, SC-5, CM-2.
- [ ] **SK-12** `time-and-timezones` — with AR-5: shelter zone, wall-clock
  shift hours vs UTC storage, date-only fields
  (`ShiftCoverageRequest.Date`), DST tests.
- [ ] **SK-13** `sensitive-data` — before FD-5: which fields are sensitive,
  per-audience response DTOs (the `adminGroupResponse` pattern), audit
  logging (FD-4), keeping PII out of logs and telemetry.
- [ ] **SK-14** `add-domain` — with AR-6/AR-7: scaffold an
  `internal/<domain>` package with service, thin handlers,
  `RegisterXRoutes`, and a models file.
- [ ] **SK-15** `notifications` — before CM work: email / GroupMe / SMS
  sends, digest coalescing (#323), per-type preferences (CM-1), no-op when
  unconfigured.
- [ ] **SK-16** `feature-flag-rollout` — before the first large feature:
  backend flag, Terraform wiring, and the frontend gate (avoids the #315
  second-gate bug).

### Keeping skills accurate

- [ ] **SK-17** Compile-check `add-api-endpoint/handler-template.go` in CI
  so the template can't drift from the real helpers again (it is
  `//go:build ignore` today). **(blocked: SK-Q2)**
- [ ] **SK-18** Re-review every skill against the code at each workstream
  boundary; update the "Reviewed against code" column above.

### Found during the skills review

- [ ] **SK-F1** `animal_image.go` (`UploadAnimalImage`) sets
  `ImageData = nil` whenever the provider call succeeds — including with the
  `postgres` provider, which never fails — so gallery uploads under
  `STORAGE_PROVIDER=postgres` (the local-dev default) appear to be served as
  404. Needs a `fix/` PR; `group_document.go` shows the correct branch.

### Open questions

- **SK-Q1** Should `.github/copilot-instructions.md` and
  `.github/instructions/` be kept in step with the skills, or should the
  skills become the single source?
- **SK-Q2** Is a CI compile check for skill templates worth it, given
  `test.yml` is manual-only (AR-Q5)?

---

## 1. Foundations

Cross-cutting changes most later workstreams depend on.

### Work

- [ ] **FD-1** Volunteer status lifecycle on `User`: applicant → onboarding →
  active → inactive → do-not-return, with start/end dates and status history.
- [ ] **FD-2** Volunteer coordinator role: cross-group visibility and
  management without site-admin powers (API tokens, site settings).
  Depends on AR-2. **(blocked: FD-Q1)**
- **FD-3** *Moved to AR-5.*
- [ ] **FD-4** General audit log (who changed what, when, old → new) for
  hours, qualifications, status, and waiver records. Generalise the
  `CommentHistory` pattern.
- [ ] **FD-5** Extended volunteer profile: emergency contact, date of birth or
  minor flag, guardian, pronouns, preferred contact method, address (if
  needed). **(blocked: FD-Q2)**
- [ ] **FD-6** Field-level visibility for sensitive data (DOB, emergency
  contact, background-check status) — coordinator/admin only.
- **FD-7** *Moved to AR-4.*
- [ ] **FD-8** Bulk account creation (CSV → invites) for onboarding cohorts.
- [ ] **FD-9** Users page scaled for 600 users: server-side pagination, search,
  filter by status/program/level.

### Open questions

- **FD-Q1** What roles does the shelter actually have? (e.g. staff
  coordinator, program lead, shift lead, mentor, volunteer.) What can each do?
- **FD-Q2** Which profile fields are required, and who may see each one?
- ~~**FD-Q3** Does the app run more than one container replica in prod?~~
  Answered: yes, up to 3 (see decision log). Follow-up is AR-Q4.
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
  last initial, PIN or QR confirmation, program picker. Depends on AR-4,
  AR-14. **(blocked: TT-Q1)**
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
| 2026-09-27 | Is the current stack suitable, or does it need a rewrite? | Keep the stack; do targeted prerequisites (workstream A) | Project owner |
| 2026-09-27 | FD-Q3: more than one prod replica? | Yes — prod `max_replicas = 3` (from Terraform) | Code |
| 2026-09-27 | Integrate with or replace Galaxy Digital? | Replace | Project owner |
