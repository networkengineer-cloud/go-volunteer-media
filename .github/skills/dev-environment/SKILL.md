---
name: dev-environment
description: Set up, run, and manage the go-volunteer-media development environment. Covers all make commands, dev server URLs, Docker vs local-only workflows, database operations, seed data credentials, and environment variable configuration. Use when starting development, resetting the DB, or troubleshooting the dev stack.
argument-hint: [task: start | seed | reset-db | docker | env-vars]
---

# Development Environment

## Quick Start

```bash
# 1. Full Docker stack (API + Postgres + frontend build)
make docker-build   # builds image (includes frontend)
make docker-run     # starts docker compose (postgres_dev + API container)
make docker-logs    # tail logs

# 2. Local-only (faster iteration — Vite proxies /api to :8080)
make db-start       # start only postgres_dev container
make dev-backend    # go run cmd/api/main.go  (port 8080)
make dev-frontend   # cd frontend && npm run dev  (port 5173, proxy to 8080)
```

Open `http://localhost:5173` in local mode or `http://localhost:8080` in Docker mode.

## All Make Targets

| Command | What it does |
|---|---|
| `make setup` | `go mod download` + `cd frontend && npm install` |
| `make dev-backend` | Start Go API server (hot-reload not built-in; use `air` for live reload) |
| `make dev-frontend` | Start Vite dev server with proxy |
| `make build` | Build backend binary + `frontend/dist` |
| `make build-frontend` | Build frontend only (outputs to `frontend/dist`) |
| `make docker-build` | Build Docker image (runs `make build-frontend` first) |
| `make docker-run` | `docker compose up -d` |
| `make docker-stop` | `docker compose down` |
| `make docker-logs` | `docker compose logs -f` |
| `make db-start` | `docker compose up -d postgres_dev` |
| `make db-stop` | `docker compose down postgres_dev` |
| `make db-shell` | Open `psql` inside dev postgres container |
| `make seed` | Run seed script (`cmd/seed/main.go`) |
| `make seed-force` | Seed with `--force` flag (overwrites existing data) |
| `make db-reseed` | Full reset: db-stop → db-start → seed |
| `make test` | `go test -v ./...` |
| `make lint` | Run `golangci-lint run` |
| `make fmt` | `go fmt ./...` + `cd frontend && npm run lint -- --fix` |

## Seed Data Credentials

After `make seed` or `make db-reseed`, these users are available:

| Username | Role | Password |
|---|---|---|
| `admin` | Site Admin | `demo1234` |
| `merry` | Group Admin (ModSquad) | `demo1234` |
| `sophia` | Group Admin (ModSquad) | `demo1234` |
| `terry` | Volunteer | `volunteer2026!` |
| `alex`, `jordan`, `casey`, `taylor` | Volunteer | `volunteer2026!` |

Source of truth: `cmd/seed/main.go`

## Environment Variables

Copy `.env.example` to `.env` before first run.

| Variable | Required | Default (dev) | Description |
|---|---|---|---|
| `PORT` | No | `8080` | API listen port |
| `DB_HOST` | No | `localhost` | Postgres host |
| `DB_PORT` | No | `5432` | Postgres port |
| `DB_USER` | No | `postgres` | Postgres user |
| `DB_PASSWORD` | No | `postgres` | Postgres password |
| `DB_NAME` | No | `volunteer_media_dev` | Database name |
| `DB_SSLMODE` | No | `disable` | SSL mode |
| `JWT_SECRET` | **Yes** | — | Min 32 chars; validated on startup |
| `ALLOWED_ORIGINS` | No | `http://localhost:5173` | CORS allowed origins |
| `AUTH_RATE_LIMIT_PER_MINUTE` | No | `5` | Auth endpoint rate limit |
| `FRONTEND_URL` | No | `http://localhost:5173` | Used in emails |
| `SMTP_HOST/PORT/USERNAME/PASSWORD/FROM_EMAIL/FROM_NAME` | No | — | Email sending (optional in dev) |

## Vite Proxy (local-only mode)

When running `make dev-frontend`, Vite (`frontend/vite.config.ts`) proxies:
- `/api` → `http://localhost:8080`
- `/uploads` → `http://localhost:8080`

This means the React app can call `/api/...` without CORS issues in dev.

## Common Troubleshooting

```bash
# Port already in use
lsof -ti:8080 | xargs kill -9   # kill backend
lsof -ti:5173 | xargs kill -9   # kill frontend

# Database connection failed
pg_isready                        # check if postgres is up
make db-start                     # start the dev container

# Stale Go test cache
go clean -testcache

# Frontend dependencies out of sync
cd frontend && rm -rf node_modules package-lock.json && npm install

# Full environment reset
make db-reseed
```
