---
name: run-tests
description: Run any or all test suites in go-volunteer-media. Covers the three suites (Go unit/integration, Vitest frontend unit, Playwright E2E), how to run a single file or test, how to view coverage, and how to clear stale caches. Use when running tests before a commit, debugging failing tests, or checking coverage.
argument-hint: [suite: all | go | unit | e2e | specific test name or file]
---

# Running Tests

## Run Everything (before a PR)

```bash
# Backend
go test ./...

# Frontend unit
cd frontend && npm run test:unit

# Frontend E2E (requires dev servers — see below)
cd frontend && npm run test:e2e
```

---

## Backend — Go Tests

```bash
# All tests
go test ./...

# Verbose output
go test -v ./...

# Specific package
go test ./internal/handlers/...
go test ./internal/auth/...
go test ./internal/middleware/...

# Specific test function
go test -v -run TestLoginHandler ./internal/handlers/...

# With coverage
go test -cover ./...
go test -coverprofile=coverage.out ./... && go tool cover -html=coverage.out

# Clear stale test cache (if tests pass locally but CI fails)
go clean -testcache

# The handlers package is slow under -race (bcrypt-heavy tests). On a slow
# machine it can exceed go test's 10-minute default — raise it rather than
# dropping -race:
go test -race -timeout 20m ./internal/handlers/...
```

Test files: `internal/handlers/*_test.go`, `internal/auth/*_test.go`, `internal/middleware/*_test.go`

Shared test utilities: `internal/handlers/test_helpers.go`

### Postgres-backed tests (`*_postgres_test.go`)

Most handler tests run on in-memory SQLite. Files ending in
`_postgres_test.go` need a real Postgres with pgvector — they cover JSONB,
full-text search, vector search, and concurrency (e.g. two replicas claiming
the same coverage digest). They connect via `openSearchTestPostgres(t)`
(`search_postgres_test.go`) and **skip themselves** when no database is
reachable, so a green run without Postgres has not exercised them.

```bash
# Start the dev database (pgvector image)
make db-start

# Create the scratch database once (migrations create the extensions)
docker compose exec postgres_dev psql -U postgres -c "CREATE DATABASE volunteer_media_test;"

# Run the handlers package with the DB up (Postgres tests now run instead of skipping)
go test -race -timeout 20m ./internal/handlers/...

# Or just the current Postgres tests (not every one has "Postgres" in its name)
go test -race -v -run 'Postgres|ConcurrentReplicas' ./internal/handlers/...
```

Connection settings come from `DB_HOST` / `DB_PORT` / `DB_USER` /
`DB_PASSWORD` / `DB_NAME` / `DB_SSLMODE`, defaulting to
`localhost:5432`, `postgres`/`postgres`, database `volunteer_media_test`.
Check the output for `skipping: no Postgres reachable` — if you see it, the
tests did not run.

Write a `_postgres_test.go` test whenever behaviour depends on Postgres
semantics (row locking, `ON CONFLICT`, JSONB, ranking) or on two requests
racing — SQLite will pass tests that Postgres would fail.

### Known failures (baseline)

`CLAUDE.md` lists tests that fail on clean `main` (e.g.
`TestCreateGroup/accepts_valid_GroupMe_bot_id` flakes). Confirm the current
baseline on `main` before calling a failure a regression, and name
pre-existing failures in the PR body.

---

## Frontend Unit — Vitest

```bash
cd frontend

# Run all unit tests (single pass)
npm run test:unit

# Watch mode (re-runs on file change)
npx vitest

# Run a specific test file
npx vitest src/components/AgePicker.test.tsx

# With coverage
npx vitest --coverage
```

Test files: `frontend/src/**/*.test.ts`, `frontend/src/**/*.test.tsx`

---

## Frontend E2E — Playwright

```bash
cd frontend

# Install browsers (first time only, or after Playwright version upgrade).
# Skip this in Claude Code on the web — Chromium is preinstalled there.
npx playwright install

# Run all E2E tests
npx playwright test
# or
npm run test:e2e

# Run a specific spec file
npx playwright test tests/auth.spec.ts

# Run a specific test by name
npx playwright test --grep "user can log in"

# Interactive UI mode (great for debugging)
npx playwright test --ui

# Run headed (visible browser window)
npx playwright test --headed

# View the HTML report from the last run
npx playwright show-report
```

Test files: `frontend/tests/*.spec.ts`, `tests/users-page.spec.ts` (root)

Reports:
- HTML report: `frontend/playwright-report/index.html`
- Failure artifacts: `frontend/test-results/*/`

### Dev Server for E2E

The Playwright config (`frontend/playwright.config.ts`) uses `frontend/scripts/e2e-webserver.mjs` to start the necessary servers. You do **not** need to start them manually before running `npx playwright test`.

---

## Pre-Commit Checklist

```bash
go test ./...                     # backend
cd frontend && npm run test:unit  # frontend unit
cd frontend && npm run test:e2e   # E2E (if you changed UI)
go vet ./...                      # static analysis
make lint                         # linters
```

---

## Security Scans

```bash
# Go vulnerability check
govulncheck ./...

# Frontend dependency audit
cd frontend && npm audit
```
