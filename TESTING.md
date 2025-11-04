# Testing Guide

## Overview

This guide describes the testing strategy, coverage goals, and how to run tests for the Go Volunteer Media platform.

## Test Coverage Status

### Backend (Go)

| Package | Coverage | Target | Status |
|---------|----------|--------|--------|
| internal/auth | 84.0% | 90% | ✅ Exceeds target |
| internal/middleware | 33.1% | 80% | 🔄 In Progress |
| internal/handlers | 0.0% | 80% | ⚪ Planned |
| internal/models | 0.0% | 75% | ⚪ Planned |
| internal/upload | 0.0% | 75% | ⚪ Planned |
| **Overall** | **~11.6%** | **80%** | 🔄 **In Progress** |

### Frontend (React/TypeScript)

| Category | Coverage | Target | Status |
|----------|----------|--------|--------|
| Unit Tests | 0% | 70% | ⚪ Planned |
| E2E Tests | 95% | 95% | ✅ Excellent |

## Running Tests

### Backend Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test ./... -cover

# Run tests with detailed coverage report
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html

# Run specific package tests
go test ./internal/auth -v
go test ./internal/middleware -v

# Run tests with race detector
go test ./... -race

# Run tests with verbose output
go test ./... -v
```

### Frontend Tests

```bash
cd frontend

# Run E2E tests
npm test

# Run E2E tests in UI mode
npm test -- --ui

# Run E2E tests for specific file
npm test tests/authentication.spec.ts

# Generate test report
npx playwright show-report

# Lint frontend code
npm run lint

# Build frontend
npm run build
```

## Test Structure

### Backend Test Organization

```
internal/
├── auth/
│   ├── auth.go
│   └── auth_test.go          # 84% coverage
├── middleware/
│   ├── middleware.go
│   └── middleware_test.go    # 33% coverage
├── handlers/
│   ├── auth.go
│   └── auth_test.go          # Planned
└── models/
    ├── models.go
    └── models_test.go        # Planned
```

### Frontend Test Organization

```
frontend/
├── tests/                    # E2E tests (Playwright)
│   ├── authentication.spec.ts
│   ├── animal-form-ux.spec.ts
│   └── ...
└── src/
    ├── api/
    │   └── client.test.ts    # Planned (Vitest)
    ├── pages/
    │   └── Dashboard.test.tsx # Planned (React Testing Library)
    └── components/
        └── Navigation.test.tsx # Planned
```

## Writing Tests

### Backend Test Example (Table-Driven)

```go
func TestHashPassword(t *testing.T) {
    tests := []struct {
        name     string
        password string
        wantErr  bool
    }{
        {
            name:     "valid password",
            password: "ValidPassword123",
            wantErr:  false,
        },
        {
            name:     "empty password",
            password: "",
            wantErr:  false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            hash, err := HashPassword(tt.password)
            if (err != nil) != tt.wantErr {
                t.Errorf("HashPassword() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if !tt.wantErr && len(hash) == 0 {
                t.Errorf("HashPassword() returned empty hash")
            }
        })
    }
}
```

### Frontend E2E Test Example

```typescript
test('user can log in successfully', async ({ page }) => {
    await page.goto('http://localhost:5173/login');
    
    await page.fill('input[name="username"]', 'admin');
    await page.fill('input[name="password"]', 'admin123');
    await page.click('button[type="submit"]');
    
    await expect(page).toHaveURL(/.*dashboard/);
    await expect(page.locator('h1')).toContainText('Dashboard');
});
```

## Test Coverage Goals

### Phase 1: Quick Wins (Week 1) ✅
- [x] Fix build warnings
- [x] Set up CI/CD testing
- [x] Auto-fix linting errors

### Phase 2: Critical Backend Tests (Weeks 2-4) 🔄
- [x] Auth package: 84% coverage (Target: 90%) ✅
- [x] Middleware: 33% coverage (Target: 80%) 🔄
- [ ] Auth handlers (Target: 80%)
- [ ] Animal CRUD handlers (Target: 80%)
- [ ] Models validation (Target: 75%)

### Phase 3: Frontend Type Safety (Week 5)
- [ ] Replace all `any` types
- [ ] Fix React hook dependencies
- [ ] Zero linting errors

### Phase 4: Frontend Unit Tests (Weeks 6-9)
- [ ] API client tests
- [ ] Component tests
- [ ] Context tests
- [ ] 70% coverage target

## CI/CD Integration

Tests run automatically on:
- Push to main, develop, or copilot/** branches
- Pull requests to main or develop

### GitHub Actions Workflow

The test workflow includes:
1. **Backend Tests** - Go tests with race detector and coverage
2. **Backend Lint** - go vet and golangci-lint
3. **Frontend Lint** - ESLint and TypeScript checks
4. **Frontend Build** - Production build verification
5. **Security Scan** - govulncheck and npm audit
6. **Test Summary** - Combined results report

See `.github/workflows/test.yml` for full configuration.

## Coverage Thresholds

Current thresholds enforced in CI:
- Backend: 10% (temporary, goal: 80%)
- Frontend: Not enforced yet (goal: 70%)

Thresholds will be increased as coverage improves:
- Week 4: 40% backend coverage
- Week 8: 60% backend coverage
- Week 12: 80% backend coverage

## Best Practices

### Backend Testing

✅ **DO:**
- Write table-driven tests for multiple scenarios
- Test both success and error cases
- Use meaningful test names that describe the scenario
- Clean up resources after tests
- Use proper test fixtures and helpers
- Test edge cases and boundary conditions

❌ **DON'T:**
- Use real database connections (use mocks/stubs)
- Leave test data in shared state
- Write flaky tests that depend on timing
- Ignore race conditions (`-race` flag)

### Frontend Testing

✅ **DO:**
- Write E2E tests for user workflows
- Test accessibility with proper ARIA labels
- Use data-testid attributes for reliable selectors
- Test responsive design with multiple viewports
- Mock external API calls in unit tests

❌ **DON'T:**
- Use brittle CSS selectors
- Write tests that depend on external services
- Test implementation details instead of behavior
- Ignore mobile viewports

## Debugging Tests

### Backend

```bash
# Run specific test with verbose output
go test -v -run TestHashPassword ./internal/auth

# Run with race detector for concurrency issues
go test -race ./internal/auth

# Run with coverage and open report
go test -coverprofile=coverage.out ./internal/auth
go tool cover -html=coverage.out
```

### Frontend

```bash
cd frontend

# Run tests in headed mode (see browser)
npm test -- --headed

# Run tests in debug mode
npm test -- --debug

# Run specific test file
npm test tests/authentication.spec.ts

# Generate trace for debugging
npm test -- --trace on
```

## Test Data

### Backend Test Secrets

Use the following test JWT secret in your tests:
```go
os.Setenv("JWT_SECRET", "L5WTt6D+6R55YfKzwqPRAEX5bR0bkNo4i58jYKL0wsk=")
```

**Never use this in production!** Generate a new secret with:
```bash
openssl rand -base64 32
```

### Frontend Test Users

E2E tests use these demo accounts:
- Admin: `admin` / `admin123`
- User: `john` / `password123`

See `SEED_DATA.md` for full list of test accounts.

## Continuous Improvement

### Coverage Tracking

Monitor coverage trends:
```bash
# Generate coverage badge
go test ./... -coverprofile=coverage.out
go tool cover -func=coverage.out | grep total

# Track coverage over time
git log --all --oneline --grep="coverage"
```

### Quality Metrics

Track these metrics weekly:
- Total test count
- Coverage percentage (backend and frontend)
- Test execution time
- Flaky test rate
- Linting errors count

## Resources

- [Go Testing Guide](https://go.dev/doc/tutorial/add-a-test)
- [Playwright Documentation](https://playwright.dev/)
- [React Testing Library](https://testing-library.com/react)
- [QA Assessment Report](./QA_ASSESSMENT_REPORT.md)
- [QA Action Plan](./QA_ACTION_PLAN.md)

## Getting Help

- Review existing tests for patterns and examples
- Check the QA assessment reports for guidance
- Create an issue with the `testing` label
- See `CONTRIBUTING.md` for contribution guidelines

---

**Last Updated:** November 4, 2025  
**Status:** Phase 2 in progress (11.6% backend coverage)  
**Next Milestone:** 40% backend coverage by Week 4
