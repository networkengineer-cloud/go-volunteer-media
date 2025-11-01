---
description: Development workflow and Git branching strategy for the go-volunteer-media project
applyTo: '**'
---

# Development Workflow Instructions

## Git Branching Strategy

### Branch Protection Rules

⚠️ **CRITICAL**: Never commit directly to `main` branch. All changes must go through pull requests.

### Branch Naming Conventions

Use descriptive branch names following these patterns:

- **Features**: `feature/short-description`
  - Example: `feature/user-profile-management`
  - Example: `feature/terraform-azure-infrastructure`

- **Bug Fixes**: `fix/short-description`
  - Example: `fix/login-validation-error`
  - Example: `fix/light-mode-text-and-modsquad-description`

- **Copilot/Agent Work**: `copilot/short-description`
  - Example: `copilot/add-announcements-to-group-page`
  - Example: `copilot/improve-user-experience-animals`

- **Refactoring**: `refactor/short-description`
  - Example: `refactor/extract-email-service`

- **Documentation**: `docs/short-description`
  - Example: `docs/update-api-documentation`

- **Performance**: `perf/short-description`
  - Example: `perf/optimize-database-queries`

- **Tests**: `test/short-description`
  - Example: `test/add-authentication-e2e-tests`

### Workflow Steps

1. **Always create a new branch** from `main` before starting work:
   ```bash
   git checkout main
   git pull origin main
   git checkout -b feature/your-feature-name
   ```

2. **Make your changes** following project coding standards

3. **Commit regularly** with clear, descriptive messages:
   ```bash
   git add .
   git commit -m "feat: Add user profile management functionality"
   ```

4. **Push your branch** to remote:
   ```bash
   git push -u origin feature/your-feature-name
   ```

5. **Create a Pull Request** on GitHub:
   - Provide a clear title and description
   - Reference any related issues
   - Request reviews from team members
   - Ensure CI checks pass

6. **Address review feedback** by pushing additional commits

7. **Merge after approval** (typically done by maintainers)

8. **Delete the branch** after merging (automated on GitHub)

9. **Update your local main**:
   ```bash
   git checkout main
   git pull origin main
   git branch -d feature/your-feature-name
   ```

## Commit Message Conventions

Follow [Conventional Commits](https://www.conventionalcommits.org/) format:

```
<type>(<scope>): <description>

[optional body]

[optional footer]
```

### Commit Types

- **feat**: New feature
  - Example: `feat: Add password reset functionality`
  
- **fix**: Bug fix
  - Example: `fix: Resolve JWT token expiration issue`
  
- **docs**: Documentation changes
  - Example: `docs: Update API documentation for auth endpoints`
  
- **style**: Code style changes (formatting, semicolons, etc.)
  - Example: `style: Format code with prettier`
  
- **refactor**: Code refactoring without feature changes
  - Example: `refactor: Extract email service to separate package`
  
- **perf**: Performance improvements
  - Example: `perf: Add database indexes for user queries`
  
- **test**: Adding or updating tests
  - Example: `test: Add E2E tests for authentication flow`
  
- **chore**: Maintenance tasks
  - Example: `chore: Update dependencies to latest versions`
  
- **ci**: CI/CD changes
  - Example: `ci: Add GitHub Actions workflow for tests`

### Commit Message Examples

Good commit messages:
```bash
feat: Add animal image upload with validation

- Support JPG, PNG, and GIF formats
- Validate file size (max 5MB)
- Generate unique filenames
- Store in /public/uploads directory

Closes #42
```

```bash
fix: Prevent race condition in rate limiter

The rate limiter was not thread-safe, causing incorrect
limit calculations under concurrent requests. Added mutex
locking to synchronize access.

Fixes #127
```

```bash
refactor: Migrate to HCP Terraform with federated credentials

- Replace Azure Blob Storage backend with HCP Terraform Cloud
- Configure OIDC federated credentials for passwordless auth
- Update GitHub Actions workflow for HCP Terraform integration
- Add comprehensive FEDERATED_CREDENTIALS.md guide

Benefits:
- Zero cost state management (free tier: 500 resources)
- Enhanced collaboration with web UI and run history
- Better security with federated credentials (no secrets)

Breaking Changes: None - additive infrastructure change
```

## Code Review Process

### Before Requesting Review

- [ ] Code follows project style guides (`.github/instructions/`)
- [ ] All tests pass (`go test ./...` and `npm test`)
- [ ] New features have tests (unit, integration, E2E with Playwright)
- [ ] Documentation is updated (README, API.md, inline comments)
- [ ] No console.log statements left in code
- [ ] No commented-out code
- [ ] No secrets or credentials in code
- [ ] Branch is up to date with main

### Review Checklist

Reviewers should check:

- [ ] Code quality and maintainability
- [ ] Security considerations
- [ ] Performance implications
- [ ] Test coverage
- [ ] Documentation completeness
- [ ] Accessibility compliance (for UI changes)
- [ ] Error handling
- [ ] Edge cases handled

## Testing Requirements

### Backend Testing

All new features require tests:

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific package
go test ./internal/handlers/...
```

### Frontend Testing

#### Unit/Integration Tests
```bash
cd frontend
npm test
```

#### End-to-End Tests (Playwright)

**REQUIRED** for all new features and UI changes:

```bash
cd frontend

# Install Playwright browsers (first time only)
npx playwright install

# Run all tests
npx playwright test

# Run tests in UI mode (interactive)
npx playwright test --ui

# Run specific test file
npx playwright test tests/auth.spec.ts

# Generate test report
npx playwright show-report
```

**When to write Playwright tests:**
- ✅ New user-facing features
- ✅ Authentication/authorization flows
- ✅ Form submissions
- ✅ Navigation changes
- ✅ Admin functionality
- ✅ Complex user interactions
- ✅ Responsive design changes

### Security Testing

Run security scans before creating PR:

```bash
# Go vulnerability check
govulncheck ./...

# Frontend dependency audit
cd frontend
npm audit

# Docker image scanning (if applicable)
trivy image your-image:tag
```

## Environment Management

### Development Environment

Copy example environment file:
```bash
cp .env.example .env
```

Required environment variables:
```bash
# Database
DATABASE_URL=postgresql://user:password@localhost:5432/volunteer_media_dev

# JWT
JWT_SECRET=your-secret-key-minimum-32-chars-generate-with-openssl-rand

# SMTP (optional for development)
SMTP_HOST=smtp.example.com
SMTP_PORT=587
SMTP_USERNAME=user@example.com
SMTP_PASSWORD=password
SMTP_FROM_EMAIL=noreply@example.com
SMTP_FROM_NAME=HAWS Volunteers

# Frontend URL
FRONTEND_URL=http://localhost:5173
```

### Production Environment

- **Never commit `.env` files**
- Use environment variables or secret management services
- Rotate secrets regularly
- Use strong, randomly generated secrets

## CI/CD Pipeline

### GitHub Actions

Pull requests automatically trigger:
1. Code linting (Go and TypeScript)
2. Unit tests (backend and frontend)
3. Build verification
4. Security scans (if configured)

### Deployment

Deployment to production happens automatically on merge to `main`:
1. Run all tests
2. Build Docker image
3. Push to container registry
4. Deploy via Terraform (Azure infrastructure)
5. Run smoke tests

## Common Tasks

### Starting Development

```bash
# 1. Update main branch
git checkout main
git pull origin main

# 2. Create feature branch
git checkout -b feature/my-feature

# 3. Start development servers
# Terminal 1: Backend
go run cmd/api/main.go

# Terminal 2: Frontend
cd frontend
npm run dev

# Terminal 3: Watch for changes (optional)
# Use air for live reload: air
```

### Adding a New API Endpoint

1. Define model in `internal/models/models.go`
2. Create handler in `internal/handlers/`
3. Register route in `cmd/api/main.go`
4. Add frontend API method in `frontend/src/api/client.ts`
5. Write tests (unit + E2E)
6. Update API documentation in `API.md`

### Adding a New Page

1. Create component in `frontend/src/pages/PageName.tsx`
2. Create styles in `frontend/src/pages/PageName.css`
3. Add route in `frontend/src/App.tsx`
4. Add navigation link in `frontend/src/components/Navigation.tsx`
5. Write Playwright tests
6. Ensure accessibility compliance

### Database Migrations

GORM auto-migrates on startup. For production:

1. Create migration script in `internal/database/migrations/`
2. Test on staging environment first
3. Document migration in PR
4. Consider data migration needs

## Troubleshooting

### Common Issues

**Port already in use:**
```bash
# Kill process on port 8080
lsof -ti:8080 | xargs kill -9

# Kill process on port 5173
lsof -ti:5173 | xargs kill -9
```

**Database connection failed:**
```bash
# Check if PostgreSQL is running
pg_isready

# Start PostgreSQL (Docker)
docker compose up -d postgres_dev

# Check database exists
psql -l
```

**Dependencies out of sync:**
```bash
# Backend
go mod tidy

# Frontend
cd frontend
rm -rf node_modules package-lock.json
npm install
```

**Tests failing locally:**
```bash
# Clear test cache
go clean -testcache

# Rebuild frontend
cd frontend
npm run build
```

## Project-Specific Guidelines

### Technology Stack

**Backend:**
- Go 1.24+ with Gin framework
- GORM for PostgreSQL
- JWT authentication (golang-jwt/jwt/v5)
- bcrypt for password hashing

**Frontend:**
- React 18 with TypeScript
- Vite build tool
- React Router v6
- Axios for API calls

**Infrastructure:**
- Docker for development
- Azure Container Apps for production
- PostgreSQL Flexible Server
- HCP Terraform for state management

### Code Style

Follow project-specific instructions:
- Go: `.github/instructions/go.instructions.md`
- React: `.github/instructions/reactjs.instructions.md`
- Docker: `.github/instructions/containerization-docker-best-practices.instructions.md`

### Security Requirements

- ✅ All secrets in environment variables
- ✅ JWT_SECRET minimum 32 characters
- ✅ Input validation on both client and server
- ✅ SQL injection prevention (parameterized queries)
- ✅ XSS prevention (proper escaping)
- ✅ CORS properly configured
- ✅ Rate limiting on auth endpoints
- ✅ No sensitive data in logs
- ✅ HTTPS in production

### Accessibility Requirements

- ✅ WCAG 2.1 AA compliance
- ✅ Semantic HTML
- ✅ Keyboard navigation support
- ✅ ARIA labels where needed
- ✅ Alt text for images
- ✅ Minimum 4.5:1 contrast ratio
- ✅ Focus indicators visible
- ✅ Form labels properly associated

## Getting Help

1. **Check documentation:**
   - `README.md` - Project overview
   - `SETUP.md` - Setup instructions
   - `API.md` - API documentation
   - `CONTRIBUTING.md` - Contribution guidelines
   - `ARCHITECTURE.md` - System architecture

2. **Review existing code** for patterns

3. **Check GitHub Issues** for similar questions

4. **Create a new issue** with the `question` label

5. **Consult agent documentation** in `.github/agents/`

## Remember

- 🚫 **Never commit directly to main**
- ✅ **Always create a feature branch**
- ✅ **Write tests for new features**
- ✅ **Follow commit message conventions**
- ✅ **Request code review before merging**
- ✅ **Keep PRs focused and reasonably sized**
- ✅ **Update documentation**
- ✅ **Run tests locally before pushing**
- ✅ **Check for security vulnerabilities**
- ✅ **Ensure accessibility compliance**

---

*Last updated: October 31, 2025*
