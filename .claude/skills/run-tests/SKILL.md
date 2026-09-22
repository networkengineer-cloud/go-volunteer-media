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
```

Test files: `internal/handlers/*_test.go`, `internal/auth/*_test.go`, `internal/middleware/*_test.go`

Shared test utilities: `internal/handlers/test_helpers.go`

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

# Install browsers (first time only, or after Playwright version upgrade)
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
