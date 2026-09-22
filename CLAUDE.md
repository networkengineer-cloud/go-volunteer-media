# go-volunteer-media

**MyHAWS Volunteer Portal** — a Go (Gin) + Postgres (GORM) API serving a
React/TS (Vite) SPA. Animal-shelter volunteers use it to read animal profiles,
log session reports, share photos/videos, read care protocols, and coordinate
shift coverage. Invite-only: there is no public registration.

Deployed to Azure Container Apps via Terraform. In production the Go binary
also serves the SPA — `frontend/dist` is embedded into the binary via `go:embed`
(`frontend/` Go package), so there is no separate web server.

`.github/copilot-instructions.md` is the shorter, shared version of this file
(kept accurate for Copilot). This file is the fuller Claude-facing map.

## Domain model

`internal/models/models.go` holds every model (~23 structs, one file). The core
shape:

- **Group** — the primary tenant boundary (dogs, cats, modsquad…). Almost all
  content hangs off a group, and almost all authorization is "is this user in
  this group / an admin of it".
- **User** ↔ **UserGroup** — membership join table, and where *group admin* is
  recorded. Site admin is a bool on `User`.
- **Animal** — the central content object, scoped to a group. Satellites:
  `AnimalImage`, `AnimalVideo`, `AnimalComment` (+ `CommentHistory`,
  `SessionMetadata` for structured session reports), `AnimalTag`,
  `AnimalNameHistory`, `AnimalBQIncident`, protocol documents.
- **Protocol** / **Script** / **GroupDocument** — care documentation, per group.
- **Update** / **Announcement** — feed content; announcements can fan out to
  email and GroupMe.
- **ShiftSlot** / **ShiftCoverageRequest** — the scheduling feature: recurring
  volunteer shifts and requests to have one covered.
- **SiteSetting**, **APIToken**, tag models round it out.

Migrations run on startup: `database.RunMigrations(db)` AutoMigrates these
models and seeds default groups/tags/settings. There are no migration files —
schema changes are struct changes. Adding a column is safe; renaming or
dropping is not automatic.

## Backend layout (`internal/`)

| Package | What lives there |
| --- | --- |
| `handlers` | ~40 handler files, one per feature area, each with a `_test.go`. All business logic. |
| `middleware` | Auth, DB-per-request, CORS, security headers, rate limit, request ID, body-size caps, Cloudflare IP handling. |
| `auth` | JWT issue/verify, bcrypt, `JWT_SECRET` entropy validation. |
| `database` | Connection setup, migrations, pgvector readiness check. Forces `time.Local` to UTC in `init()`. |
| `storage` | `storageProvider` abstraction: `postgres` (default) or `azure` blob. See the `image-upload` skill. |
| `email` | Provider abstraction: Resend or SMTP. No-ops cleanly when unconfigured. |
| `groupme` | Outbound GroupMe bot posts for announcements/coverage requests. |
| `embedding` | Voyage embeddings + a background reconciliation sweep that backfills missing vectors. |
| `convert` | LibreOffice-backed DOCX→PDF conversion for uploaded documents. |
| `telemetry`, `logging` | OpenTelemetry (no-op without an OTLP endpoint) and structured logging. |
| `upload`, `maintenance`, `lifecycle`, `version` | Upload validation, maintenance mode, shutdown hooks, build SHA. |

Handlers are **closures over dependencies** — `handlers.CreateAnimal(db, emailService, embedder)` returns a `gin.HandlerFunc`. Routes are all registered in
`cmd/api/main.go` (~156 routes, 638 lines), which is also where every service
is constructed and wired.

## Request flow

```
request → core middleware → SecurityHeaders → RequestID → Logging
        → DBMiddleware (binds request-scoped *gorm.DB) → MaxRequestBodySize → CORS
        → /api group
             ├── public: /login, /reset-password, /settings, /images/:uuid, /videos/:uuid
             └── AuthRequired(db)  ← JWT bearer token OR API token
                   ├── /admin/**            → AdminRequired() (site admin only)
                   ├── /groups/:id/**       → group checks INSIDE handlers
                   └── everything else      → per-handler checks
```

`AuthRequired` sets context keys `user_id` (uint) and `is_admin` (bool); read
them with `middleware.GetUserID(c)` / `middleware.IsSiteAdmin(c)`.

Three permission tiers — public, group member, group admin/site admin. Only the
first and third are enforced by middleware; group membership and group-admin
checks live in handler bodies. Read the `group-auth-pattern` skill before
touching authorization.

## Frontend layout (`frontend/src/`)

- `App.tsx` — all ~19 routes, each wrapped in a protection component.
- `api/client.ts` (~1,200 lines) — the single typed axios client. Reads
  `localStorage['token']`, redirects to `/login` on 401. Every backend call
  goes through here; add a typed method rather than calling axios directly.
- `pages/` — one component + one CSS file per screen. `pages/group/` holds the
  scheduling sub-components (`ScheduleTab`, `ScheduleOverview`,
  `NeedsCoverageList`, `RequestCoverageRangeForm`, `scheduleGrid.ts`).
- `components/` — shared UI (`Modal`, `Toast`, `ConfirmDialog`, `FormField`,
  `ProtocolViewer`/`ProtocolPdfViewer`/`ProtocolDocxViewer`, `VideoUpload`…).
- `contexts/` — `AuthContext`, `SiteSettingsContext`, `ThemeContext` (light/dark),
  `ToastContext`. Consume via the `hooks/` wrappers (`useAuth`, `useToast`…).
- `utils/`, `types/` — date/animal helpers and shared TS types.

Vite dev server proxies `/api` and `/uploads` to `localhost:8080`.

## Skills

- `dev-environment` — running the stack, seed credentials, DB reset.
- `add-api-endpoint` — the full model → handler → route → client → test loop.
- `add-frontend-page` — page + CSS + route + nav + E2E test.
- `group-auth-pattern` — the three-tier permission model. Read before any
  handler authorization change.
- `image-upload` — the storage-provider abstraction.
- `frontend-styling` — **read before touching any `.css` file.**
- `run-tests`, `playwright-e2e-test` — test suites and E2E conventions.
- `roadmap-update` — how to report completed work (never edit `ROADMAP.md`).

## Commands

```bash
make docker-run                      # full stack (see dev-environment skill)
make dev-backend / make dev-frontend # local-only iteration
make seed / make db-reseed           # demo data (creds in cmd/seed/main.go)
go build ./... && go vet ./... && gofmt -l internal/
cd frontend && npx tsc --noEmit && npx vitest run
cd frontend && npm run test:e2e      # Playwright, specs in frontend/tests/
```

CI (`.github/workflows/test.yml`) runs backend tests + coverage thresholds,
`go vet`, golangci-lint, ESLint, `tsc`, and the frontend build.

## Test baseline — read before claiming a regression

These fail on clean `main`. Do not spend a session proving they are unrelated:

- `frontend/src/pages/AnimalForm.test.tsx` — 1 failing
- `frontend/src/components/DateRangePicker.test.tsx` — 1 failing (asserts a
  hardcoded August 2026 date; fails by calendar date, not by code)
- `go test ./internal/handlers/...` — `TestCreateGroup/accepts_valid_GroupMe_bot_id`
  flakes

Counts drift. Confirm the current baseline on `main` rather than trusting this
list, and say which failures are pre-existing in the PR body:

```bash
git stash && go test ./internal/handlers/... ; cd frontend && npx vitest run ; cd .. && git stash pop
```

## Backend conventions

- **Never reassign a handler's `db` parameter with `=`.** Handlers are closures
  over `db *gorm.DB`; assigning to it mutates shared state across requests. Use
  `db := middleware.GetDB(c, db)` inside the handler body (150+ call sites).
- Group-admin authorization is enforced *inside* handlers, not by middleware.
  Read the handler before changing route protection. See the
  `group-auth-pattern` skill.
- Routes are registered in `cmd/api/main.go`; typed client methods go in
  `frontend/src/api/client.ts`.
- Handler tests live beside handlers; a `_postgres_test.go` suffix means the
  test needs a real Postgres (pgvector/full-text features), not SQLite.

## Feature flags

Backend env-var flags (`SCHEDULE_EMAIL_NOTIFICATIONS_ENABLED`,
`COVERAGE_REQUESTS_FEED_ENABLED`, `SEMANTIC_SEARCH_ENABLED`) are wired through
`terraform/environments/{dev,prod}/main.tf` to the container app. Each must be
exactly `"true"`; a configured API key alone is not enough. When a feature
"doesn't appear in prod despite the flag being on", check for a *second* gate on
the frontend — that class of bug cost a prod debugging session in #315.

Optional integrations degrade rather than fail: no `VOYAGE_API_KEY` → search is
keyword-only; no email config → password reset and notifications are disabled;
no OTLP endpoint → telemetry is a no-op.

## Code layout traps

- `frontend/src/pages/GroupPage.tsx` is ~1,900 lines and renders the group
  activity feed **inline**. Cards for a new activity type are added there, in
  the `activity.type` switch — not in a component. An unrendered type silently
  falls through to `<p>{activity.content}</p>`, which is empty (#318).
- `frontend/src/pages/GroupPage.css` is ~1,800 lines.
- `internal/models/models.go` is one file for every model — grep it, don't
  expect per-model files.
- `cmd/api/main.go` is the only route table; a handler that exists but isn't
  reachable is almost always a missing line there.

## Workflow

- Never commit directly to `main`. Branch `feature/`, `fix/`, `refactor/`,
  `docs/`, `perf/`, `test/`. Conventional Commits.
- `ROADMAP.md` is Product-Owner-owned — never edit it. Report completions via
  the `roadmap-update` skill.
- Do not create summary/status/assessment `.md` files. See
  `.github/instructions/development-workflow.instructions.md`.
- `docs/superpowers/` is gitignored — never force-add plan or spec files.
- Other standing guidance lives in `.github/instructions/` (Go, React, Docker,
  workflow, roadmap communication).

## Reference docs

`README.md` (features/stack) · `SETUP.md` · `DEPLOYMENT.md` · `TESTING.md` ·
`API.md` · `STORAGE.md` · `SECURITY.md` · `docs/EMAIL_CONFIGURATION.md` ·
`terraform/README.md`
