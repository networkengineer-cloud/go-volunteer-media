# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Go Volunteer Media is a full-stack social media application for animal shelter volunteers. It allows volunteers to manage animals, share updates, post photos/comments, and receive announcements through various channels including GroupMe integration.

**Tech Stack:**
- Backend: Go 1.24+ with Gin framework, GORM ORM, PostgreSQL
- Frontend: React 18+ with TypeScript, Vite, React Router
- Deployment: Azure Container Apps with Terraform (HCP managed state)
- External Integrations: SendGrid (email), GroupMe (group messaging)

## Development Commands

### Backend Development
```bash
# Start backend server (auto-migrates database)
make dev-backend
# or directly:
go run cmd/api/main.go

# Run tests with coverage
go test ./... -cover

# Run tests with detailed coverage report
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html

# Run specific package tests
go test ./internal/auth -v
go test ./internal/handlers -v

# Run tests with race detector
go test ./... -race

# Lint code
make lint
# or:
golangci-lint run

# Format code
make fmt
```

### Frontend Development
```bash
cd frontend

# Start dev server (proxies /api to backend)
npm run dev

# Run unit tests (Vitest)
npm run test:unit

# Run unit tests with UI
npm run test:unit:ui

# Run unit tests with coverage
npm run test:unit:coverage

# Run E2E tests (Playwright)
npm run test:e2e

# Run E2E tests with UI
npm run test:e2e:ui

# View E2E test report
npm run test:e2e:report

# Lint code
npm run lint

# Build for production
npm run build
```

### Database Commands
```bash
# Start PostgreSQL (Docker)
make db-start

# Stop PostgreSQL and remove volumes
make db-stop

# Connect to PostgreSQL shell
make db-shell

# Seed database with demo data
make seed

# Force seed (even if data exists)
make seed-force

# Fresh database with seed data
make db-reseed
```

### Running Single Tests
```bash
# Backend: Run a specific test function
go test -v -run TestHashPassword ./internal/auth

# Frontend: Run a specific test file
cd frontend
npm run test:e2e tests/authentication.spec.ts
```

## Architecture Overview

### Backend Structure
```
cmd/
  api/main.go           # Application entry point, middleware setup
  seed/main.go          # Database seeding utility

internal/
  auth/                 # JWT token generation/validation, password hashing
  database/             # DB connection, migrations, seeding logic
  email/                # SendGrid email service wrapper
  groupme/              # GroupMe API integration for group messaging
  handlers/             # HTTP request handlers (business logic)
    auth.go             # Login, register, password reset
    group*.go           # Group CRUD and membership
    animal*.go          # Animal CRUD, comments, tags
    user_admin.go       # User management (admin)
    announcement.go     # Site-wide announcements
  logging/              # Structured logging with logrus
  middleware/           # HTTP middleware (auth, CORS, rate limiting, security headers)
  models/               # GORM models and database schema
  upload/               # Image upload, validation, and optimization
```

### Frontend Structure
```
frontend/src/
  api/                  # Axios API client with auth interceptors
  components/           # Reusable React components
  contexts/             # React contexts (AuthContext for global auth state)
  pages/                # Page-level components
    Dashboard.tsx       # Group cards overview
    GroupPage.tsx       # Animals and updates for a group
    AnimalDetail.tsx    # Animal details with comments
    Login.tsx           # Auth pages
  App.tsx               # Router setup and protected routes
```

### Key Architectural Patterns

**Authentication Flow:**
- JWT tokens stored in localStorage
- Axios interceptor adds `Authorization: Bearer <token>` header
- Backend middleware validates JWT on protected routes
- User context (user_id, is_admin) extracted from token and passed to handlers via Gin context

**Authorization:**
- Role-based: Admin flag checked for admin-only endpoints
- Group membership: Users must be members of a group to access its resources
- Resource ownership: Some operations require being the creator (e.g., editing own updates)

**Middleware Pipeline (order matters):**
1. Recovery (panic handler)
2. SecurityHeaders (X-Frame-Options, CSP, etc.)
3. RequestID (unique ID for request tracing)
4. LoggingMiddleware (structured logging with request/response details)
5. CORS (cross-origin resource sharing)
6. AuthRequired / AdminRequired (route-specific, validates JWT)
7. Rate limiting (on auth endpoints)

**Database Design:**
- Soft deletes: All models have `DeletedAt` field (GORM convention)
- Many-to-many: `user_groups` join table for User ↔ Group relationship
- Foreign keys: Animals → Group, Updates → Group + User, Comments → Animal + User
- Indexes: On foreign keys, `deleted_at`, and frequently queried fields

**Image Upload Flow:**
1. Client uploads to `/api/animals/upload-image` (multipart/form-data)
2. Server validates: type (jpg/png/gif), size (<10MB), content (decode to verify)
3. Server optimizes: resize if >1200px, encode as JPEG quality 85
4. Server saves to `public/uploads/` with unique filename (timestamp_uuid.jpg)
5. Server returns URL: `/uploads/filename.jpg`
6. Client includes URL in animal/comment form submission

## Testing Strategy

**Current Coverage:**
- Backend: ~11.6% (target: 80%)
- Frontend E2E: 95% coverage with Playwright
- Frontend Unit: Planned (target: 70% with Vitest + React Testing Library)

**Running Tests:**
- See commands above for backend, frontend unit, and E2E tests
- CI runs on push to main/develop/copilot/** branches and PRs
- See TESTING.md for detailed testing guide

**Writing Tests:**
- Backend: Use table-driven tests, test both success and error cases
- Frontend: E2E tests for user workflows, unit tests for components/utils
- Always clean up resources (e.g., test database fixtures)

## Important Development Notes

### Environment Variables
Required for backend (see `.env.example`):
- `JWT_SECRET`: Generate with `openssl rand -base64 32`
- `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`: PostgreSQL connection
- `DB_SSLMODE`: Use `disable` for dev, `require` for prod
- `SENDGRID_API_KEY`, `SENDGRID_FROM_EMAIL`: Optional for email features
- `GROUPME_ACCESS_TOKEN`: Optional for GroupMe integration
- `ENV`: Set to `production` for production mode

### Database Migrations
- Migrations run automatically on startup via `database.RunMigrations()`
- GORM AutoMigrate handles schema changes
- No separate migration files—models define schema
- For destructive changes, manually back up data first

### Demo Data
- Seed command creates demo users, groups, animals, updates
- Admin user: `admin` / `demo1234`
- See SEED_DATA.md for full list of demo accounts
- Use `make db-reseed` for fresh database with demo data

### GroupMe Integration
- Service located in `internal/groupme/`
- Sends announcements to GroupMe groups via bot integration
- Groups must have `groupme_bot_id` configured in database
- Requires `GROUPME_ACCESS_TOKEN` environment variable
- See `internal/groupme/groupme_test.go` for comprehensive test examples (94.3% coverage)

### Security Considerations
- Never commit secrets to git (use .env, Key Vault for production)
- All passwords hashed with bcrypt
- JWT tokens have expiration
- Account locking after 5 failed login attempts
- Rate limiting on auth endpoints (5 req/min)
- Input validation on all user inputs
- File upload validation (type, size, content)
- Parameterized queries prevent SQL injection
- Security headers prevent XSS, clickjacking

### Common Pitfalls
- **CORS issues:** Backend must be running on 8080, frontend on 5173 for dev proxy to work
- **JWT validation:** Ensure `JWT_SECRET` matches between backend instances
- **Group access:** Users must be added to a group before they can see it
- **Soft deletes:** GORM adds `deleted_at IS NULL` automatically; use `Unscoped()` to see deleted records
- **Image paths:** Images stored in `public/uploads/`, served as `/uploads/filename.jpg`
- **Middleware order:** Security headers must come before CORS

## Deployment

**Production Deployment (Azure):**
- Infrastructure managed via Terraform (HCP Terraform for state)
- OIDC federation for passwordless authentication (no secrets)
- Container Apps for serverless hosting
- PostgreSQL Flexible Server for managed database
- Blob Storage for image uploads
- Key Vault for secrets management
- Application Insights for monitoring
- See DEPLOYMENT.md and ARCHITECTURE.md for details

**CI/CD:**
- GitHub Actions workflow: `.github/workflows/test.yml`
- Runs backend tests, frontend tests, linting, and security scans
- See ARCHITECTURE.md for deployment architecture diagrams

## Code Style and Conventions

**Go:**
- Follow standard Go conventions (`gofmt`, `go vet`)
- Use structured logging with contextual fields: `logger.WithFields(...).Info(...)`
- Table-driven tests for multiple scenarios
- Error handling: Always check and log errors
- Handlers return JSON with consistent structure: `{"error": "message"}` or `{"data": {...}}`

**TypeScript/React:**
- Functional components with hooks
- TypeScript strict mode
- Avoid `any` types (technical debt being addressed)
- Use React Router v6 patterns
- Context for global state (AuthContext)
- Axios interceptors for adding auth headers

## Resources

- **ARCHITECTURE.md**: Comprehensive architecture diagrams (request flow, DB schema, auth flow, deployment)
- **TESTING.md**: Testing strategy, coverage goals, best practices
- **DEPLOYMENT.md**: Production deployment guide
- **SECURITY.md**: Security features and best practices
- **API.md**: API endpoint documentation
- **SEED_DATA.md**: Demo data and test accounts
- **README.md**: Getting started, setup instructions
