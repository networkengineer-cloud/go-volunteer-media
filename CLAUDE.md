# go-volunteer-media

Go (Gin) + Postgres (GORM) API serving a React/TS (Vite) SPA. Deployed to Azure
Container Apps via Terraform.

For architecture, routing, auth context keys, and env vars, read
`.github/copilot-instructions.md` — it is accurate and current. This file covers
only what that one doesn't: the traps that have actually cost rework.

## Skills

Project skills are symlinked into `.claude/skills/` from `.github/skills/`
(shared with GitHub Copilot — edit the `.github/skills/` originals, not the
links). `frontend-styling` is Claude-only and lives directly in `.claude/skills/`.

**Read `frontend-styling` before touching any `.css` file.** This app has no CSS
modules and 70+ unscoped global button rules; styling changes that look trivial
have repeatedly needed a follow-up PR (#304→#305, #308→#309, #312→#313→#314).

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

## Commands

```bash
make docker-run                      # full stack (see dev-environment skill)
make dev-backend / make dev-frontend # local-only iteration
go build ./... && go vet ./... && gofmt -l internal/
cd frontend && npx tsc --noEmit && npx vitest run
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

## Feature flags

Backend env-var flags (`SCHEDULE_EMAIL_NOTIFICATIONS_ENABLED`,
`COVERAGE_REQUESTS_FEED_ENABLED`, `SEMANTIC_SEARCH_ENABLED`) are wired through
`terraform/` to the container app. When a feature "doesn't appear in prod
despite the flag being on", check for a *second* gate on the frontend — that
class of bug cost a prod debugging session in #315.

## Code layout traps

- `frontend/src/pages/GroupPage.tsx` is ~1,900 lines and renders the group
  activity feed **inline**. Cards for a new activity type are added there, in
  the `activity.type` switch — not in a component. An unrendered type silently
  falls through to `<p>{activity.content}</p>`, which is empty (#318).
- `frontend/src/pages/GroupPage.css` is ~1,800 lines.

## Workflow

- Never commit directly to `main`. Branch `feature/`, `fix/`, `refactor/`,
  `docs/`, `perf/`, `test/`. Conventional Commits.
- `ROADMAP.md` is Product-Owner-owned — never edit it. Report completions via
  the `roadmap-update` skill.
- Do not create summary/status/assessment `.md` files. See
  `.github/instructions/development-workflow.instructions.md`.
- `docs/superpowers/` is gitignored — never force-add plan or spec files.
