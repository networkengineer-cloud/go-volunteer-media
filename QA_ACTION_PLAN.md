# QA Improvement Action Plan - Prioritized Checklist

**Target Audience:** Development Team  
**Purpose:** Step-by-step guide to address QA findings  
**Timeline:** 12 weeks to production-grade quality

---

## 🚀 Phase 1: Quick Wins (Week 1)

**Goal:** Fix low-effort, high-impact issues  
**Time:** 1-2 days  
**Impact:** Immediate code quality improvement

### Day 1: Fix Build Issues & Auto-Fixable Errors

- [ ] **Fix build warning** (5 minutes)
  ```bash
  # File: cmd/seed/main.go:71
  # Change: fmt.Println("...\n") → fmt.Println("...")
  git checkout -b fix/seed-linting-warning
  # Edit file, commit, push
  ```

- [ ] **Fix auto-fixable linting errors** (30 minutes)
  ```bash
  cd frontend
  npm run lint -- --fix
  git add .
  git commit -m "fix: auto-fix linting errors (prefer-const)"
  ```

- [ ] **Remove unused test parameters** (1 hour)
  ```typescript
  // Find all instances of:
  test('...', async ({ page }) => {
    // If 'page' is not used, remove it:
  test('...', async () => {
  ```

### Day 2: Set Up CI/CD & Quick Type Fixes

- [ ] **Create CI/CD testing workflow** (2-3 hours)
  ```yaml
  # File: .github/workflows/test.yml
  # Add:
  # - Go test job
  # - Frontend lint job
  # - Frontend build job
  # - E2E test job (on schedule/manual)
  ```

- [ ] **Create pre-commit hooks** (1 hour)
  ```bash
  # Install husky
  npx husky init
  # Add hooks: go fmt, go test, npm run lint
  ```

- [ ] **Document quick type fixes** (1 hour)
  - Identify the 5 most-used `any` types
  - Create proper TypeScript interfaces
  - Document in TypeScript types file

**✅ Success Metrics:**
- Build passes without warnings
- Linting errors reduced by ~20%
- CI/CD runs on every PR
- Pre-commit hooks prevent future issues

---

## 🧪 Phase 2: Critical Backend Tests (Weeks 2-4)

**Goal:** Establish safety net for backend changes  
**Time:** 15-20 working days  
**Target:** 40% backend coverage (critical paths only)

### Week 2: Authentication & Security Tests

- [ ] **Set up test infrastructure** (Day 1)
  ```bash
  # Create test utilities
  mkdir -p internal/testutil
  # Add: database helpers, JWT test helpers, test fixtures
  ```

- [ ] **Auth package tests** (Days 2-3)
  ```go
  // File: internal/auth/auth_test.go
  - [ ] TestHashPassword
  - [ ] TestCheckPassword
  - [ ] TestGenerateToken
  - [ ] TestValidateToken
  - [ ] TestJWTSecretValidation
  - [ ] TestCheckSecretEntropy
  ```

- [ ] **Auth handler tests** (Days 4-5)
  ```go
  // File: internal/handlers/auth_test.go
  - [ ] TestRegister_Success
  - [ ] TestRegister_DuplicateUsername
  - [ ] TestRegister_DuplicateEmail
  - [ ] TestRegister_WeakPassword
  - [ ] TestRegister_InvalidEmail
  - [ ] TestLogin_Success
  - [ ] TestLogin_InvalidCredentials
  - [ ] TestLogin_AccountLockout
  - [ ] TestLogin_LockedAccount
  ```

**✅ Week 2 Goal:** 90% coverage for auth/security code

### Week 3: Animal CRUD Tests

- [ ] **Animal model tests** (Day 1)
  ```go
  // File: internal/models/models_test.go
  - [ ] TestAnimal_LengthOfStay
  - [ ] TestAnimal_CurrentStatusDuration
  - [ ] TestAnimal_StatusValidation
  ```

- [ ] **Animal handler tests** (Days 2-5)
  ```go
  // File: internal/handlers/animal_test.go
  - [ ] TestCreateAnimal_Success
  - [ ] TestCreateAnimal_ValidationError
  - [ ] TestCreateAnimal_Unauthorized
  - [ ] TestGetAnimal_Success
  - [ ] TestGetAnimal_NotFound
  - [ ] TestUpdateAnimal_Success
  - [ ] TestUpdateAnimal_StatusTransition
  - [ ] TestDeleteAnimal_Success
  - [ ] TestDeleteAnimal_SoftDelete
  - [ ] TestListAnimals_Filtering
  - [ ] TestListAnimals_Pagination
  - [ ] TestListAnimals_ByStatus
  ```

**✅ Week 3 Goal:** 75% coverage for animal handlers

### Week 4: Middleware & Admin Tests

- [ ] **Middleware tests** (Days 1-2)
  ```go
  // File: internal/middleware/middleware_test.go
  - [ ] TestAuthRequired_ValidToken
  - [ ] TestAuthRequired_InvalidToken
  - [ ] TestAuthRequired_MissingToken
  - [ ] TestAdminRequired_AdminUser
  - [ ] TestAdminRequired_RegularUser
  - [ ] TestAdminRequired_NoAuth
  - [ ] TestRateLimit_BelowLimit
  - [ ] TestRateLimit_ExceedsLimit
  - [ ] TestSecurityHeaders
  ```

- [ ] **Admin handler tests** (Days 3-5)
  ```go
  // File: internal/handlers/user_admin_test.go
  - [ ] TestListUsers_AdminOnly
  - [ ] TestCreateUser_AdminOnly
  - [ ] TestUpdateUser_AdminOnly
  - [ ] TestDeleteUser_AdminOnly
  - [ ] TestAddUserToGroup_AdminOnly
  - [ ] TestRemoveUserFromGroup_AdminOnly
  - [ ] TestAdminDashboard_Stats
  ```

**✅ Week 4 Goal:** 40% overall backend coverage achieved

---

## 🎨 Phase 3: Frontend Type Safety (Week 5)

**Goal:** Eliminate type safety issues  
**Time:** 5 working days  
**Target:** Zero TypeScript `any` types, zero linting errors

### Days 1-2: Define TypeScript Interfaces

- [ ] **Create API response types** (Day 1)
  ```typescript
  // File: src/types/api.ts
  interface User { ... }
  interface Group { ... }
  interface Animal { ... }
  interface Comment { ... }
  interface Tag { ... }
  // etc. for all API responses
  ```

- [ ] **Create API error types** (Day 1)
  ```typescript
  interface APIError {
    error: string;
    details?: Record<string, string[]>;
  }
  ```

- [ ] **Update API client** (Day 2)
  ```typescript
  // File: src/api/client.ts
  // Replace all 'any' with proper types
  - [ ] Line 254: Define proper type
  - [ ] Line 275: Define proper type
  - [ ] Line 295: Define proper type
  - [ ] Line 302: Define proper type
  - [ ] Line 320: Define proper type
  ```

### Days 3-4: Fix Page Components

- [ ] **Fix `any` types in pages**
  - [ ] AdminAnimalTagsPage.tsx (2 instances)
  - [ ] AdminDashboard.tsx (1 instance)
  - [ ] AnimalDetailPage.tsx (4 instances)
  - [ ] AnimalForm.tsx (5 instances)
  - [ ] BulkEditAnimalsPage.tsx (6 instances)
  - [ ] GroupPage.tsx (3 instances)
  - [ ] UserProfilePage.tsx (9 instances)
  - [ ] (all other pages with `any` types)

- [ ] **Fix `prefer-const` issues** (auto-fixable)
  - [ ] AnimalDetailPage.tsx:18
  - [ ] AnimalForm.tsx:260
  - [ ] BulkEditAnimalsPage.tsx:30

### Day 5: Fix React Hooks & Context Issues

- [ ] **Fix exhaustive-deps warnings** (12 instances)
  ```typescript
  // Add missing dependencies or use useCallback
  - [ ] ActivityFeed.tsx
  - [ ] ProtocolsList.tsx
  - [ ] AdminAnimalTagsPage.tsx
  - [ ] AdminDashboard.tsx
  - [ ] AnimalDetailPage.tsx
  - [ ] AnimalForm.tsx
  - [ ] BulkEditAnimalsPage.tsx
  - [ ] GroupPage.tsx
  - [ ] UserProfilePage.tsx
  - [ ] (all other components)
  ```

- [ ] **Fix fast-refresh violations**
  ```typescript
  // Separate exports from context providers
  - [ ] AuthContext.tsx - move useAuth to separate file
  - [ ] ToastContext.tsx - move useToast to separate file
  ```

**✅ Week 5 Goal:** Zero linting errors, 100% type safety

---

## 🎭 Phase 4: Frontend Unit Tests (Weeks 6-9)

**Goal:** Test critical UI components in isolation  
**Time:** 20 working days  
**Target:** 70% component coverage

### Week 6: Test Infrastructure Setup

- [ ] **Install test dependencies** (Day 1)
  ```bash
  npm install -D vitest @testing-library/react @testing-library/jest-dom \
    @testing-library/user-event jsdom @vitest/ui
  ```

- [ ] **Configure Vitest** (Day 1)
  ```typescript
  // File: vitest.config.ts
  // Set up JSDOM, globals, test patterns
  ```

- [ ] **Update package.json** (Day 1)
  ```json
  "scripts": {
    "test:unit": "vitest",
    "test:e2e": "playwright test",
    "test": "npm run test:unit && npm run test:e2e",
    "test:coverage": "vitest --coverage"
  }
  ```

- [ ] **Create test utilities** (Days 2-3)
  ```typescript
  // File: src/test/test-utils.tsx
  // Custom render with providers (Auth, Toast)
  // Mock router
  // Test data factories
  ```

- [ ] **Write first component test** (Days 4-5)
  ```typescript
  // File: src/components/Navigation.test.tsx
  - [ ] renders navigation links
  - [ ] shows admin links for admin users
  - [ ] hides admin links for regular users
  - [ ] highlights active route
  ```

**✅ Week 6 Goal:** Test infrastructure ready, first test passing

### Week 7: API Client Tests

- [ ] **Mock Axios** (Day 1)
  ```typescript
  // Set up axios-mock-adapter
  ```

- [ ] **Auth API tests** (Days 2-3)
  ```typescript
  // File: src/api/client.test.ts
  - [ ] login() sends correct credentials
  - [ ] login() stores token on success
  - [ ] login() throws on 401
  - [ ] register() creates user
  - [ ] logout() clears token
  - [ ] authFetch() includes token header
  ```

- [ ] **Animal API tests** (Days 4-5)
  ```typescript
  - [ ] getAnimals() fetches list
  - [ ] getAnimal() fetches single
  - [ ] createAnimal() posts data
  - [ ] updateAnimal() puts data
  - [ ] deleteAnimal() sends delete
  - [ ] Handles error responses
  ```

**✅ Week 7 Goal:** 85% API client coverage

### Week 8: Form Component Tests

- [ ] **AnimalForm tests** (Days 1-3)
  ```typescript
  // File: src/pages/AnimalForm.test.tsx
  - [ ] renders empty form for new animal
  - [ ] loads animal data for edit
  - [ ] validates required fields
  - [ ] validates field formats
  - [ ] submits form with correct data
  - [ ] handles submission errors
  - [ ] disables submit during submission
  ```

- [ ] **UserProfile tests** (Days 4-5)
  ```typescript
  // File: src/pages/UserProfilePage.test.tsx
  - [ ] displays user information
  - [ ] allows editing profile
  - [ ] validates email format
  - [ ] saves changes on submit
  - [ ] handles update errors
  ```

**✅ Week 8 Goal:** Critical forms tested

### Week 9: Context & Page Tests

- [ ] **Context tests** (Days 1-2)
  ```typescript
  // File: src/contexts/AuthContext.test.tsx
  - [ ] provides initial auth state
  - [ ] updates on login
  - [ ] clears on logout
  - [ ] handles token expiration
  
  // File: src/contexts/ToastContext.test.tsx
  - [ ] shows toast messages
  - [ ] auto-dismisses toasts
  - [ ] allows manual dismiss
  ```

- [ ] **Page component tests** (Days 3-5)
  ```typescript
  // Test critical pages:
  - [ ] Login page (authentication flow)
  - [ ] Dashboard (displays stats)
  - [ ] AnimalList (filtering, pagination)
  - [ ] AdminDashboard (admin-only content)
  ```

**✅ Week 9 Goal:** 70% frontend coverage achieved

---

## 🔒 Phase 5: Security & Performance (Weeks 10-11)

**Goal:** Add security tests and performance monitoring  
**Time:** 10 working days

### Week 10: Security Enhancements

- [ ] **Add security scanning to CI** (Day 1)
  ```yaml
  # .github/workflows/security.yml
  - govulncheck ./...
  - npm audit --audit-level=high
  - trivy image scan
  ```

- [ ] **Security tests** (Days 2-3)
  ```go
  // Test security features:
  - [ ] SQL injection prevention
  - [ ] XSS prevention
  - [ ] CSRF token validation
  - [ ] Rate limiting
  - [ ] Account lockout
  ```

- [ ] **Enable HSTS** (Day 4)
  ```go
  // File: internal/middleware/security.go
  // Uncomment HSTS header for production
  ```

- [ ] **Add CSRF protection** (Days 4-5)
  ```go
  // Implement CSRF middleware
  // Add CSRF tokens to forms
  ```

**✅ Week 10 Goal:** Production-grade security

### Week 11: Performance Testing

- [ ] **Add performance tests** (Days 1-2)
  ```bash
  # Install k6 or hey for load testing
  # Create load test scripts
  ```

- [ ] **Bundle size monitoring** (Day 3)
  ```bash
  # Add bundle analyzer to CI
  # Set size budgets
  ```

- [ ] **Lighthouse CI** (Day 4)
  ```yaml
  # Add Lighthouse to CI
  # Set performance budgets
  ```

- [ ] **Database query optimization** (Day 5)
  ```go
  // Add query logging
  // Identify N+1 queries
  // Add indexes where needed
  ```

**✅ Week 11 Goal:** Performance baselines established

---

## 📊 Phase 6: Final Polish (Week 12)

**Goal:** Documentation, cleanup, and final review  
**Time:** 5 working days

### Final Week Checklist

- [ ] **Update documentation** (Days 1-2)
  - [ ] Update README with test commands
  - [ ] Add TESTING.md guide
  - [ ] Document test coverage requirements
  - [ ] Update CONTRIBUTING.md with testing guidelines

- [ ] **Code cleanup** (Day 3)
  - [ ] Remove commented code
  - [ ] Remove console.logs
  - [ ] Remove unused imports
  - [ ] Format all files

- [ ] **Final QA review** (Days 4-5)
  - [ ] Run all tests (unit, integration, E2E)
  - [ ] Run linters
  - [ ] Run security scans
  - [ ] Review code coverage reports
  - [ ] Test deployment process
  - [ ] Manual smoke test of critical features

- [ ] **Prepare for production** (Day 5)
  - [ ] Review security checklist
  - [ ] Update environment configs
  - [ ] Document deployment process
  - [ ] Create runbook for common issues

**✅ Week 12 Goal:** Production-ready codebase

---

## 📈 Success Metrics

Track these metrics weekly:

| Metric | Week 1 | Week 4 | Week 8 | Week 12 |
|--------|--------|--------|--------|---------|
| Backend Test Coverage | 0% | 40% | 60% | 80% |
| Frontend Test Coverage | 0% | 0% | 50% | 70% |
| Linting Errors | 136 | 80 | 20 | 0 |
| Type Safety (`any` types) | 48 | 48 | 10 | 0 |
| CI/CD Tests | 0 | 3 | 5 | 8 |
| Critical Issues | 3 | 1 | 0 | 0 |

---

## 🎯 Definition of Done

**Checklist for Production Readiness:**

- [ ] ✅ Zero critical issues
- [ ] ✅ Backend test coverage ≥ 80%
- [ ] ✅ Frontend test coverage ≥ 70%
- [ ] ✅ E2E tests passing (maintain 95%)
- [ ] ✅ Zero linting errors
- [ ] ✅ Zero TypeScript `any` types
- [ ] ✅ All CI/CD checks passing
- [ ] ✅ Security scans clean
- [ ] ✅ Performance benchmarks met
- [ ] ✅ Documentation up-to-date
- [ ] ✅ Security checklist completed
- [ ] ✅ Load testing passed
- [ ] ✅ Manual QA approved

---

## 💡 Tips for Success

1. **Start Small:** Begin with the easiest tests to build momentum
2. **Test First:** For new features, write tests before code (TDD)
3. **Pair Programming:** Complex tests benefit from pair programming
4. **Code Review:** All test code should be reviewed like production code
5. **Celebrate Wins:** Acknowledge coverage milestones
6. **Automate:** Let CI/CD catch regressions
7. **Document:** Keep testing docs up-to-date
8. **Maintain:** Tests are code - they need maintenance too

---

## 🆘 Getting Help

- Review full report: [QA_ASSESSMENT_REPORT.md](./QA_ASSESSMENT_REPORT.md)
- Review summary: [QA_EXECUTIVE_SUMMARY.md](./QA_EXECUTIVE_SUMMARY.md)
- Create issue with `qa-review` label for questions
- Schedule pair programming sessions for complex tests

---

**Last Updated:** November 3, 2025  
**Status:** Ready for implementation  
**Estimated Completion:** 12 weeks from start
