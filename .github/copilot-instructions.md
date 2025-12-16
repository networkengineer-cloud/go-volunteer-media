# Copilot instructions (go-volunteer-media)

## Big picture
- Full-stack app: Go (Gin) API + Postgres (GORM) + React/TS (Vite).
- Backend entrypoint + routing/middleware is in [cmd/api/main.go](../cmd/api/main.go); in production the Go server also serves the SPA from `frontend/dist`.
- Core backend packages live under `internal/` (handlers, middleware, auth, database, models, upload, integrations).

## Dev workflows (container-first)
- Build image (includes frontend build): `make docker-build` (runs `make build-frontend` then `docker build`).
- Run stack locally via compose: `make docker-run` (see [docker-compose.yml](../docker-compose.yml)); tail logs with `make docker-logs`.
- Start/stop only dev Postgres: `make db-start` / `make db-stop`.
- Seed demo data (including demo creds): `make seed` (or `make seed-force`). Reference the seeded users/passwords in [cmd/seed/main.go](../cmd/seed/main.go) rather than hardcoding.
- The production container image includes `/frontend/dist` (built in [Dockerfile](../Dockerfile)); the API serves it via [cmd/api/main.go](../cmd/api/main.go).
- Optional local-only (non-container) iteration: `make dev-backend` + `make dev-frontend` (Vite proxies `/api` + `/uploads` to `localhost:8080`, see [frontend/vite.config.ts](../frontend/vite.config.ts)).

## Auth + permissions conventions
- JWT auth is required for most `/api/**` routes; clients send `Authorization: Bearer <token>`.
- Middleware stores auth context keys: `user_id` (uint) and `is_admin` (bool) (see [internal/middleware/middleware.go](../internal/middleware/middleware.go)).
- Site-admin-only routes live under `/api/admin/**` and use `AdminRequired()`.
- “Group admin” authorization is commonly enforced inside handlers (not a separate middleware), so check handler logic before changing route protection.
- Registration is intentionally disabled (invite-only); admins create users via `POST /api/admin/users` (see comment in [cmd/api/main.go](../cmd/api/main.go)).

## Data + migrations
- DB connection defaults to local dev values if env vars are missing (DB_HOST/PORT/USER/PASSWORD/NAME/SSLMODE) (see [internal/database/database.go](../internal/database/database.go)).
- Migrations are run on startup via `database.RunMigrations(db)`; it AutoMigrates models in [internal/models/models.go](../internal/models/models.go) and creates default groups/tags/settings.

## Runtime config (common env vars)
- API listens on `PORT` (default 8080).
- DB: `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `DB_SSLMODE`.
- Auth/security: `JWT_SECRET` (required; validated for length/entropy in [internal/auth/auth.go](../internal/auth/auth.go)), `ALLOWED_ORIGINS` (CORS), `AUTH_RATE_LIMIT_PER_MINUTE`.

## Media/document handling (important integration points)
- Images can be served from DB via `GET /api/images/:uuid`; legacy/static uploads are also served from `/uploads` mapped to `public/uploads` (see [cmd/api/main.go](../cmd/api/main.go)).
- The frontend API client is centralized in [frontend/src/api/client.ts](../frontend/src/api/client.ts): axios instance uses `localStorage['token']` and redirects to `/login` on 401.

## Tests (what’s real today)
- Backend: `go test ./...` (most coverage is in `internal/auth` and `internal/middleware` per [TESTING.md](../TESTING.md)).
- Frontend E2E (Playwright): `cd frontend && npm run test:e2e` (tests live in `frontend/tests/`).
- Frontend unit (Vitest): `cd frontend && npm run test:unit`.

## When adding/changing API behavior
- Typical flow: add/modify handler in `internal/handlers/` → wire route in [cmd/api/main.go](../cmd/api/main.go) → update typed frontend calls/interfaces in [frontend/src/api/client.ts](../frontend/src/api/client.ts).

## Repo-specific guardrails
- Follow the project’s workflow rules in `.github/instructions/*` (notably: don’t commit directly to `main`, avoid creating new documentation files unless asked).
