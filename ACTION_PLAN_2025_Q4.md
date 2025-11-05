# Action Plan - Q4 2025
# Go Volunteer Media Platform

**Plan Created:** November 5, 2025  
**Execution Period:** November 2025 - January 2026 (12 weeks)  
**Goal:** Achieve production-ready code quality with 80% test coverage

---

## Executive Summary

This action plan addresses the key findings from the comprehensive codebase review. The plan is structured in 5 phases over 12 weeks, prioritizing critical issues first and building toward a fully tested, production-ready application.

### Current State

| Metric | Current | Target | Status |
|--------|---------|--------|--------|
| Backend Test Coverage | 58.9% | 80% | ⚠️ In Progress |
| Frontend Unit Tests | 0% | 70% | ❌ Critical Gap |
| Frontend E2E Tests | 95% | 95% | ✅ Excellent |
| Linting Errors | 0 | 0 | ✅ Good |
| Linting Warnings | 0 | 0 | ✅ Fixed |
| Security Vulnerabilities | 0 | 0 | ✅ Excellent |
| Documentation | Excellent | Excellent | ✅ Excellent |

### Success Criteria (End of 12 Weeks)

- ✅ Backend test coverage: 80%+
- ✅ Frontend unit test coverage: 70%+
- ✅ All linting warnings resolved: 0
- ✅ E2E tests running in CI
- ✅ Linting required in CI (no bypass)
- ✅ animal.go refactored (< 300 lines per file)
- ✅ WCAG 2.1 AA compliance achieved
- ✅ Performance budgets met

---

## Phase 1: Quick Wins & Critical Fixes

**Duration:** Week 1-2 (November 5-18, 2025)  
**Focus:** Address immediate issues that block progress

### 1.1 Fix React Hook Dependencies (Day 1-2) 🚨

**Priority:** P0 (Critical)  
**Effort:** 2-3 hours  
**Owner:** Frontend Developer

**Issue:** 11 files have React Hook useEffect missing dependencies

**Files to Fix:**
- [x] `src/components/ActivityFeed.tsx`
- [x] `src/components/ProtocolsList.tsx`
- [x] `src/pages/AdminAnimalTagsPage.tsx`
- [x] `src/pages/AdminDashboard.tsx`
- [x] `src/pages/AnimalForm.tsx`
- [x] `src/pages/BulkEditAnimalsPage.tsx`
- [x] `src/pages/Dashboard.tsx`
- [x] `src/pages/GroupPage.tsx` (2 warnings)
- [x] `src/pages/SettingsPage.tsx`
- [x] `src/pages/UserProfilePage.tsx`

**Solution Pattern:**
```typescript
// ❌ Before
const loadData = async () => {
  const data = await fetchData(groupId);
  setData(data);
};

useEffect(() => {
  loadData();
}, [groupId]); // Warning: loadData missing

// ✅ After
const loadData = useCallback(async () => {
  const data = await fetchData(groupId);
  setData(data);
}, [groupId]);

useEffect(() => {
  loadData();
}, [loadData]); // No warning
```

**Acceptance Criteria:**
- ✅ All 11 warnings resolved
- ✅ `npm run lint` shows 0 warnings
- ✅ Manual testing confirms no regressions

**Deliverable:** ✅ PR #79 with title "fix: resolve React hook dependency warnings"

---

### 1.2 Add E2E Tests to CI Pipeline (Day 2-3) 🚨

**Priority:** P0 (Critical)  
**Effort:** 2-3 hours  
**Owner:** DevOps / Backend Developer

**Issue:** 16 Playwright E2E tests exist but don't run in CI

**Action:**
- [x] Create new job in `.github/workflows/test.yml`
- [x] Set up PostgreSQL service container
- [x] Start backend API in background
- [x] Install Playwright with dependencies
- [x] Run E2E tests
- [x] Upload test results and artifacts
- [x] Add to required checks for PRs

**Implementation:**
```yaml
# Add to .github/workflows/test.yml

e2e-tests:
  name: E2E Tests
  runs-on: ubuntu-latest
  
  services:
    postgres:
      image: postgres:15
      env:
        POSTGRES_DB: volunteer_media_test
        POSTGRES_USER: postgres
        POSTGRES_PASSWORD: postgres
      ports:
        - 5432:5432
      options: >-
        --health-cmd pg_isready
        --health-interval 10s
        --health-timeout 5s
        --health-retries 5
  
  steps:
    - name: Checkout code
      uses: actions/checkout@v4
    
    - name: Set up Go
      uses: actions/setup-go@v5
      with:
        go-version: '1.24'
    
    - name: Start backend API
      env:
        DATABASE_URL: postgresql://postgres:postgres@localhost:5432/volunteer_media_test?sslmode=disable
        JWT_SECRET: L5WTt6D+6R55YfKzwqPRAEX5bR0bkNo4i58jYKL0wsk=
        ENV: test
      run: |
        go run cmd/api/main.go &
        sleep 10
        curl -f http://localhost:8080/api/health || exit 1
    
    - name: Set up Node.js
      uses: actions/setup-node@v4
      with:
        node-version: '20'
        cache: 'npm'
        cache-dependency-path: frontend/package-lock.json
    
    - name: Install frontend dependencies
      working-directory: ./frontend
      run: npm ci
    
    - name: Install Playwright browsers
      working-directory: ./frontend
      run: npx playwright install --with-deps chromium
    
    - name: Run E2E tests
      working-directory: ./frontend
      run: npm test
    
    - name: Upload test results
      if: always()
      uses: actions/upload-artifact@v4
      with:
        name: playwright-report
        path: frontend/playwright-report/
        retention-days: 30
    
    - name: Upload test videos
      if: always()
      uses: actions/upload-artifact@v4
      with:
        name: playwright-videos
        path: frontend/test-results/
        retention-days: 7
```

**Acceptance Criteria:**
- ✅ E2E tests run on every push to main, develop, copilot/**
- ✅ E2E tests run on all PRs
- ✅ Test results uploaded as artifacts
- ✅ Job added to required checks

**Deliverable:** ✅ This PR - "ci: add E2E tests to GitHub Actions pipeline"

---

### 1.3 Backend Test Coverage - Quick Wins (Week 1-2) ⚠️

**Priority:** P1 (High)  
**Effort:** 1 week  
**Owner:** Backend Developer

**Goal:** Increase coverage from 25.8% to 40%

**Focus Areas:**

1. **Expand internal/handlers tests (29.7% → 72.2%)**
   - [x] Add edge case tests for existing test files
   - [x] Add error scenario tests
   - [x] Add boundary condition tests
   
2. **Add internal/upload tests (0% → 95.5%)**
   - [x] `ValidateImageUpload()` - test all validation rules
   - [x] `SanitizeFilename()` - test special characters
   - [x] `ValidateImageContent()` - test malformed images
   
3. **Add internal/database tests (0% → 0%)**
   - [ ] Database connection tests
   - [ ] Migration tests (basic)
   - [ ] Transaction handling tests

**Example Test Structure:**
```go
// internal/upload/validation_test.go
package upload

import (
    "mime/multipart"
    "testing"
)

func TestValidateImageUpload(t *testing.T) {
    tests := []struct {
        name      string
        fileSize  int64
        mimeType  string
        maxSize   int64
        wantErr   bool
        errMsg    string
    }{
        {
            name:     "valid image",
            fileSize: 1024 * 1024, // 1MB
            mimeType: "image/jpeg",
            maxSize:  10 * 1024 * 1024, // 10MB
            wantErr:  false,
        },
        {
            name:     "file too large",
            fileSize: 11 * 1024 * 1024, // 11MB
            mimeType: "image/jpeg",
            maxSize:  10 * 1024 * 1024,
            wantErr:  true,
            errMsg:   "file size exceeds limit",
        },
        {
            name:     "invalid mime type",
            fileSize: 1024 * 1024,
            mimeType: "application/exe",
            maxSize:  10 * 1024 * 1024,
            wantErr:  true,
            errMsg:   "invalid file type",
        },
        // Add more test cases
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

**Acceptance Criteria:**
- ✅ Overall backend coverage: 58.9% (exceeds 40% target)
- ✅ All tests pass
- ✅ Coverage report generated
- ✅ CI threshold updated to 50%

**Deliverables:**
- ✅ PR #79: "test: add comprehensive upload validation tests"
- ✅ PR #79: "test: add database connection and transaction tests" (partially)
- ✅ PR #79: "test: expand handler tests with edge cases"

---

## Phase 2: Critical Infrastructure

**Duration:** Week 3-4 (November 19 - December 2, 2025)  
**Focus:** Set up frontend testing infrastructure and refactor large files

### 2.1 Refactor animal.go Handler (Week 3) 🔥

**Priority:** P1 (High)  
**Effort:** 2-3 days  
**Owner:** Backend Developer

**Issue:** `internal/handlers/animal.go` is 962 lines (should be < 300)

**Current Structure:**
```
internal/handlers/animal.go (962 lines)
├── UploadAnimalImage()           ~100 lines
├── GetAnimals()                   ~80 lines
├── GetAnimalByID()                ~50 lines
├── CreateAnimal()                 ~80 lines
├── UpdateAnimal()                 ~100 lines
├── DeleteAnimal()                 ~50 lines
├── BulkUpdateAnimals()            ~120 lines
├── GetAllAnimalsAdmin()           ~60 lines
├── ImportAnimalsCSV()             ~150 lines
├── ExportAnimalsCSV()             ~100 lines
├── ExportAnimalsCommentsCSV()     ~80 lines
└── Helper functions               ~90 lines
```

**New Structure:**
```
internal/handlers/animal/
├── handler.go          # Main handler struct & setup (~50 lines)
├── create.go           # CreateAnimal (~100 lines)
├── read.go             # GetAnimals, GetAnimalByID (~150 lines)
├── update.go           # UpdateAnimal (~120 lines)
├── delete.go           # DeleteAnimal (~60 lines)
├── bulk.go             # BulkUpdateAnimals (~140 lines)
├── image.go            # UploadAnimalImage (~120 lines)
├── csv.go              # Import/Export CSV (~250 lines)
├── helpers.go          # Helper functions (~100 lines)
└── animal_test.go      # All tests (~400 lines)
```

**Implementation Steps:**

1. **Create new package** (Day 1)
   - [ ] Create `internal/handlers/animal/` directory
   - [ ] Move `animal_test.go` to new location
   - [ ] Update package declaration

2. **Extract handler struct** (Day 1)
   - [ ] Create `handler.go` with struct
   - [ ] Define dependencies (DB, logger)
   - [ ] Create constructor function

3. **Split functions by category** (Day 2)
   - [ ] Create individual files
   - [ ] Move functions to appropriate files
   - [ ] Update imports
   - [ ] Ensure all functions are methods on handler struct

4. **Update main.go routes** (Day 2)
   - [ ] Update imports
   - [ ] Create animal handler instance
   - [ ] Update route registrations

5. **Test & verify** (Day 3)
   - [ ] Run all tests
   - [ ] Run linter
   - [ ] Manual testing
   - [ ] Update documentation

**Example Structure:**

```go
// handler.go
package animal

import (
    "github.com/gin-gonic/gin"
    "gorm.io/gorm"
)

type Handler struct {
    db *gorm.DB
}

func NewHandler(db *gorm.DB) *Handler {
    return &Handler{db: db}
}

// RegisterRoutes registers all animal routes
func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
    router.POST("/animals/upload-image", h.UploadImage)
    router.GET("/groups/:groupId/animals", h.List)
    router.GET("/groups/:groupId/animals/:id", h.GetByID)
    router.POST("/groups/:groupId/animals", h.Create)
    router.PUT("/groups/:groupId/animals/:id", h.Update)
    router.DELETE("/groups/:groupId/animals/:id", h.Delete)
    // ... more routes
}
```

```go
// create.go
package animal

import (
    "net/http"
    "github.com/gin-gonic/gin"
)

func (h *Handler) Create(c *gin.Context) {
    // Implementation moved from animal.go
}
```

**Acceptance Criteria:**
- ✅ No file > 300 lines
- ✅ All existing tests pass
- ✅ No regressions in functionality
- ✅ API routes work identically
- ✅ Code review approved

**Deliverable:** PR with title "refactor: split animal handler into modular package"

---

### 2.2 Set Up Frontend Unit Testing Infrastructure (Week 3-4) 🔧

**Priority:** P1 (High)  
**Effort:** 3-4 days  
**Owner:** Frontend Developer

**Goal:** Set up Vitest and write first tests

**Implementation Steps:**

1. **Install Dependencies** (Day 1 morning)
```bash
cd frontend
npm install -D vitest @vitest/ui @testing-library/react @testing-library/jest-dom @testing-library/user-event jsdom
```

2. **Configure Vitest** (Day 1 afternoon)
```typescript
// vitest.config.ts
import { defineConfig } from 'vitest/config';
import react from '@vitejs/plugin-react';

export default defineConfig({
  plugins: [react()],
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: './src/test/setup.ts',
    coverage: {
      provider: 'v8',
      reporter: ['text', 'json', 'html'],
      exclude: [
        'node_modules/',
        'src/test/',
        '**/*.spec.ts',
        '**/*.test.tsx',
      ],
    },
  },
});
```

3. **Create Test Utilities** (Day 1-2)
```typescript
// src/test/setup.ts
import '@testing-library/jest-dom';
import { cleanup } from '@testing-library/react';
import { afterEach } from 'vitest';

afterEach(() => {
  cleanup();
});

// src/test/utils.tsx
import { render, RenderOptions } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import { AuthProvider } from '../contexts/AuthContext';

export const renderWithProviders = (
  ui: React.ReactElement,
  options?: RenderOptions
) => {
  const Wrapper = ({ children }: { children: React.ReactNode }) => (
    <BrowserRouter>
      <AuthProvider>
        {children}
      </AuthProvider>
    </BrowserRouter>
  );

  return render(ui, { wrapper: Wrapper, ...options });
};

export * from '@testing-library/react';
```

4. **Write First Tests** (Day 2-4)

**Test Priority List:**

- [ ] **AuthContext.test.tsx** (Day 2)
  - Login flow
  - Logout flow
  - Token storage
  - User state management

- [ ] **client.test.ts** (Day 3)
  - API client initialization
  - Request interceptors
  - Response handling
  - Error handling

- [ ] **AnimalCard.test.tsx** (Day 3)
  - Rendering with props
  - Click handlers
  - Image display
  - Status badges

- [ ] **Navigation.test.tsx** (Day 4)
  - Menu rendering
  - Auth state display
  - Link navigation
  - Dropdown interactions

- [ ] **AnimalForm.test.tsx** (Day 4)
  - Form validation
  - Submit handling
  - Error display
  - Image upload

5. **Update package.json** (Day 1)
```json
{
  "scripts": {
    "test:unit": "vitest",
    "test:unit:ui": "vitest --ui",
    "test:unit:coverage": "vitest --coverage",
    "test:e2e": "playwright test",
    "test": "npm run test:unit && npm run test:e2e"
  }
}
```

**Example Test:**
```typescript
// src/contexts/AuthContext.test.tsx
import { describe, it, expect, vi } from 'vitest';
import { renderHook, act, waitFor } from '@testing-library/react';
import { AuthProvider, useAuth } from './AuthContext';
import * as client from '../api/client';

vi.mock('../api/client');

describe('AuthContext', () => {
  it('should login successfully', async () => {
    const mockLogin = vi.spyOn(client, 'login').mockResolvedValue({
      token: 'fake-token',
      user: { id: 1, username: 'testuser', is_admin: false }
    });

    const wrapper = ({ children }: { children: React.ReactNode }) => (
      <AuthProvider>{children}</AuthProvider>
    );

    const { result } = renderHook(() => useAuth(), { wrapper });

    await act(async () => {
      await result.current.login('testuser', 'password');
    });

    await waitFor(() => {
      expect(result.current.user).toEqual({
        id: 1,
        username: 'testuser',
        is_admin: false
      });
      expect(result.current.isAuthenticated).toBe(true);
    });

    expect(mockLogin).toHaveBeenCalledWith('testuser', 'password');
  });

  it('should logout successfully', async () => {
    const wrapper = ({ children }: { children: React.ReactNode }) => (
      <AuthProvider>{children}</AuthProvider>
    );

    const { result } = renderHook(() => useAuth(), { wrapper });

    act(() => {
      result.current.logout();
    });

    expect(result.current.user).toBeNull();
    expect(result.current.isAuthenticated).toBe(false);
    expect(localStorage.getItem('token')).toBeNull();
  });
});
```

**Acceptance Criteria:**
- ✅ Vitest configured and working
- ✅ Test utilities created
- ✅ 5 critical components have tests
- ✅ Tests pass in CI
- ✅ Coverage reports generated
- ✅ Team trained on writing tests

**Deliverables:**
- PR 1: "test: setup Vitest infrastructure"
- PR 2: "test: add AuthContext and API client tests"
- PR 3: "test: add component tests for AnimalCard and Navigation"

---

## Phase 3: Increase Test Coverage

**Duration:** Week 5-6 (December 3-16, 2025)  
**Focus:** Achieve 60% backend coverage and 30% frontend coverage

### 3.1 Backend Test Coverage Push (Week 5-6) 📈

**Priority:** P1 (High)  
**Effort:** 2 weeks  
**Owner:** Backend Developer

**Goal:** Increase coverage from 40% to 60%

**Focus Areas:**

1. **Complete internal/handlers coverage (50% → 70%)** (Week 5)
   - [ ] Add tests for all CRUD operations
   - [ ] Test error handling comprehensively
   - [ ] Test authorization checks
   - [ ] Test input validation
   - [ ] Test edge cases

2. **Add internal/email tests (0% → 80%)** (Week 5)
   - [ ] Email sending tests
   - [ ] Template rendering tests
   - [ ] SMTP connection tests
   - [ ] Error handling tests

3. **Complete internal/database tests (40% → 80%)** (Week 6)
   - [ ] Migration tests
   - [ ] Seed data tests
   - [ ] Transaction tests
   - [ ] Connection pool tests

4. **Add internal/logging tests (0% → 70%)** (Week 6)
   - [ ] Logging middleware tests
   - [ ] Audit logging tests
   - [ ] Logger configuration tests

**Test Metrics Tracking:**
```
Week 5 Start:  40% → Week 5 End:  50%
Week 6 Start:  50% → Week 6 End:  60%
```

**Acceptance Criteria:**
- ✅ Backend coverage: 60%+
- ✅ All packages have > 50% coverage
- ✅ Critical paths have > 80% coverage
- ✅ CI threshold updated to 50%

**Deliverables:**
- PR 1: "test: complete handler test coverage"
- PR 2: "test: add email service tests"
- PR 3: "test: complete database package tests"
- PR 4: "test: add logging tests"

---

### 3.2 Frontend Unit Test Expansion (Week 5-6) 📈

**Priority:** P1 (High)  
**Effort:** 2 weeks  
**Owner:** Frontend Developer

**Goal:** Achieve 30% frontend component coverage

**Components to Test:**

**Week 5:**
- [ ] `src/pages/Dashboard.tsx` - Main dashboard
- [ ] `src/pages/GroupPage.tsx` - Group details and animals list
- [ ] `src/pages/AnimalDetailPage.tsx` - Animal details and comments
- [ ] `src/components/UpdateFeed.tsx` - Updates display
- [ ] `src/components/CommentList.tsx` - Comments display

**Week 6:**
- [ ] `src/pages/Login.tsx` - Login form and validation
- [ ] `src/pages/AnimalForm.tsx` - Create/edit animal
- [ ] `src/pages/BulkEditAnimalsPage.tsx` - Bulk operations
- [ ] `src/contexts/ToastContext.tsx` - Toast notifications
- [ ] `src/components/ProtocolsList.tsx` - Protocols display

**Test Types:**
1. **Rendering Tests** - Component renders without errors
2. **User Interaction Tests** - Click, type, submit
3. **State Management Tests** - Context updates
4. **API Integration Tests** - Mock API calls
5. **Error Handling Tests** - Error states display

**Acceptance Criteria:**
- ✅ Frontend coverage: 30%+
- ✅ All critical user flows have tests
- ✅ Tests pass in CI
- ✅ No flaky tests

**Deliverables:**
- PR 1: "test: add page component tests (Dashboard, GroupPage, AnimalDetail)"
- PR 2: "test: add form component tests (Login, AnimalForm, BulkEdit)"
- PR 3: "test: add utility component tests (Toast, ProtocolsList, CommentList)"

---

### 3.3 Make Linting Required in CI (Week 5) 🔒

**Priority:** P2 (Medium)  
**Effort:** 1 hour  
**Owner:** DevOps

**Action:** Remove `continue-on-error: true` from lint jobs

**Steps:**
1. [x] Verify all linting issues are resolved
2. [x] Update `.github/workflows/test.yml`
3. [x] Remove `continue-on-error` from backend-lint job
4. [x] Remove `continue-on-error` from frontend-lint job
5. [ ] Test on feature branch
6. [ ] Merge to main

**Changes:**
```yaml
# Before
- name: Run ESLint
  working-directory: ./frontend
  run: npm run lint
  continue-on-error: true  # ❌ Remove this

# After
- name: Run ESLint
  working-directory: ./frontend
  run: npm run lint
  # ✅ Fails build on linting errors
```

**Acceptance Criteria:**
- ✅ Backend linting required
- ✅ Frontend linting required
- [ ] PRs cannot merge with linting errors (will be enforced after merge)
- [ ] Team notified of change

**Deliverable:** ✅ This PR - "ci: make linting required for all builds"

---

## Phase 4: Accessibility & Performance

**Duration:** Week 7-8 (December 17-30, 2025)  
**Focus:** WCAG 2.1 AA compliance and performance optimization

### 4.1 Accessibility Audit & Fixes (Week 7-8) ♿

**Priority:** P2 (Medium)  
**Effort:** 1-2 weeks  
**Owner:** Frontend Developer

**Goal:** Achieve WCAG 2.1 AA compliance

**Audit Process:**

1. **Run Automated Audits** (Day 1)
   - [ ] Lighthouse accessibility audit
   - [ ] axe DevTools scan
   - [ ] WAVE browser extension
   - [ ] Generate audit reports

2. **Fix Critical Issues** (Day 2-4)
   - [ ] Add missing ARIA labels
   - [ ] Fix color contrast issues
   - [ ] Add skip navigation link
   - [ ] Ensure all images have alt text
   - [ ] Fix heading hierarchy

3. **Add Manual Tests** (Day 5-7)
   - [ ] Keyboard navigation testing
   - [ ] Screen reader testing (NVDA/JAWS)
   - [ ] Focus management
   - [ ] Form accessibility

4. **Add Automated A11y Tests** (Day 8-10)
   - [ ] Install axe-core
   - [ ] Add to E2E tests
   - [ ] Add to component tests

**Implementation Examples:**

```typescript
// Add to Playwright tests
import { test, expect } from '@playwright/test';
import AxeBuilder from '@axe-core/playwright';

test('should not have accessibility violations', async ({ page }) => {
  await page.goto('http://localhost:5173');
  
  const accessibilityScanResults = await new AxeBuilder({ page })
    .withTags(['wcag2a', 'wcag2aa'])
    .analyze();
  
  expect(accessibilityScanResults.violations).toEqual([]);
});
```

**WCAG 2.1 AA Checklist:**

**Perceivable:**
- [ ] Text alternatives for non-text content
- [ ] Captions for audio/video (N/A)
- [ ] Content can be presented in different ways
- [ ] Color contrast ratio 4.5:1 for normal text
- [ ] Color contrast ratio 3:1 for large text

**Operable:**
- [ ] All functionality available from keyboard
- [ ] No keyboard traps
- [ ] Enough time to read and use content
- [ ] No content that causes seizures
- [ ] Ways to help users navigate and find content
- [ ] Page titles
- [ ] Focus order
- [ ] Link purpose clear

**Understandable:**
- [ ] Language of page identified
- [ ] Forms have labels
- [ ] Error suggestions provided
- [ ] Error prevention for data changes

**Robust:**
- [ ] Valid HTML
- [ ] ARIA used correctly
- [ ] Name, role, value for UI components

**Acceptance Criteria:**
- ✅ Lighthouse accessibility score: 95+
- ✅ axe violations: 0 critical
- ✅ WAVE errors: 0
- ✅ Keyboard navigation works
- ✅ Screen reader tested
- ✅ Color contrast meets WCAG AA
- ✅ A11y tests in CI

**Deliverables:**
- PR 1: "a11y: add ARIA labels and fix color contrast"
- PR 2: "a11y: implement keyboard navigation and focus management"
- PR 3: "test: add accessibility tests to E2E suite"

---

### 4.2 Performance Optimization (Week 8) ⚡

**Priority:** P3 (Low)  
**Effort:** 1 week  
**Owner:** Frontend Developer

**Goal:** Meet performance budgets

**Performance Budgets:**
```
Metric                    Target    Current   Action
────────────────────────────────────────────────────
First Contentful Paint    1.8s      TBD       Optimize
Largest Contentful Paint  2.5s      TBD       Optimize
Total Blocking Time       300ms     TBD       Optimize
Cumulative Layout Shift   0.1       TBD       Optimize
JavaScript Bundle         300KB     431KB     Reduce
CSS Bundle                150KB     144KB     Good ✅
```

**Optimization Tasks:**

1. **Bundle Size Reduction** (Day 1-3)
   - [ ] Implement code splitting by route
   - [ ] Lazy load heavy components
   - [ ] Analyze bundle with rollup-plugin-visualizer
   - [ ] Remove unused dependencies

**Example:**
```typescript
// Before
import AnimalForm from './pages/AnimalForm';

// After - Lazy load
const AnimalForm = lazy(() => import('./pages/AnimalForm'));

// Use with Suspense
<Suspense fallback={<Loading />}>
  <AnimalForm />
</Suspense>
```

2. **Image Optimization** (Day 3-4)
   - [ ] Add lazy loading: `loading="lazy"`
   - [ ] Use WebP format where supported
   - [ ] Implement responsive images with srcset
   - [ ] Add image placeholder/blur

3. **React Performance** (Day 4-5)
   - [ ] Add React.memo to expensive components
   - [ ] Optimize re-renders with useMemo
   - [ ] Implement virtual scrolling for long lists
   - [ ] Remove unnecessary effect dependencies

4. **Add Performance Monitoring** (Day 5)
   - [ ] Install Lighthouse CI
   - [ ] Add Web Vitals tracking
   - [ ] Set up performance budgets
   - [ ] Add performance tests to CI

**Example Lighthouse CI:**
```json
// lighthouserc.json
{
  "ci": {
    "collect": {
      "startServerCommand": "npm run preview",
      "url": ["http://localhost:4173"],
      "numberOfRuns": 3
    },
    "assert": {
      "assertions": {
        "categories:performance": ["error", {"minScore": 0.9}],
        "categories:accessibility": ["error", {"minScore": 0.95}],
        "categories:best-practices": ["error", {"minScore": 0.9}],
        "categories:seo": ["error", {"minScore": 0.9}]
      }
    },
    "upload": {
      "target": "temporary-public-storage"
    }
  }
}
```

**Acceptance Criteria:**
- ✅ JavaScript bundle < 300 KB (< 100 KB gzipped)
- ✅ Lighthouse performance score > 90
- ✅ All Core Web Vitals in "Good" range
- ✅ Performance budgets enforced in CI
- ✅ No performance regressions

**Deliverables:**
- PR 1: "perf: implement code splitting and lazy loading"
- PR 2: "perf: optimize images and add lazy loading"
- PR 3: "perf: optimize React components and reduce re-renders"
- PR 4: "ci: add Lighthouse CI and performance monitoring"

---

## Phase 5: Final Push to Production Ready

**Duration:** Week 9-12 (December 31, 2025 - January 27, 2026)  
**Focus:** Achieve 80% backend and 70% frontend coverage

### 5.1 Backend Test Coverage - Final Push (Week 9-11) 🎯

**Priority:** P1 (High)  
**Effort:** 3 weeks  
**Owner:** Backend Developer

**Goal:** Increase from 60% to 80%+

**Week 9:** Focus on remaining handlers
- [ ] Complete all handler tests
- [ ] Test all error paths
- [ ] Test all authorization checks
- [ ] Target: 70% overall coverage

**Week 10:** Focus on edge cases and integrations
- [ ] Add integration tests
- [ ] Test complex workflows
- [ ] Test concurrent operations
- [ ] Target: 75% overall coverage

**Week 11:** Polish and reach 80%
- [ ] Fill remaining coverage gaps
- [ ] Add performance tests
- [ ] Add stress tests
- [ ] Target: 80% overall coverage

**Acceptance Criteria:**
- ✅ Overall backend coverage: 80%+
- ✅ All packages > 70% coverage
- ✅ Critical paths > 90% coverage
- ✅ No untested error paths
- ✅ CI threshold updated to 80%

---

### 5.2 Frontend Unit Test Coverage - Final Push (Week 9-11) 🎯

**Priority:** P1 (High)  
**Effort:** 3 weeks  
**Owner:** Frontend Developer

**Goal:** Increase from 30% to 70%+

**Week 9:** Test all pages (30% → 50%)
- [ ] Test all page components
- [ ] Test routing and navigation
- [ ] Test page-level state management
- [ ] Target: 50% coverage

**Week 10:** Test all components (50% → 60%)
- [ ] Test all reusable components
- [ ] Test component interactions
- [ ] Test conditional rendering
- [ ] Target: 60% coverage

**Week 11:** Polish and reach 70% (60% → 70%)
- [ ] Fill coverage gaps
- [ ] Test error boundaries
- [ ] Test complex interactions
- [ ] Target: 70% coverage

**Acceptance Criteria:**
- ✅ Frontend coverage: 70%+
- ✅ All pages have tests: 90%+
- ✅ All components have tests: 80%+
- ✅ All contexts/hooks have tests: 100%
- ✅ No untested user flows

---

### 5.3 Documentation & Training (Week 12) 📚

**Priority:** P2 (Medium)  
**Effort:** 1 week  
**Owner:** Tech Lead

**Activities:**

1. **Update Documentation**
   - [ ] Update TESTING.md with new practices
   - [ ] Update CONTRIBUTING.md with test requirements
   - [ ] Create testing best practices guide
   - [ ] Document common testing patterns

2. **Create Training Materials**
   - [ ] Testing workshop slides
   - [ ] Example test walkthroughs
   - [ ] Common pitfalls document
   - [ ] Video tutorials

3. **Team Training**
   - [ ] Testing workshop (2 hours)
   - [ ] Q&A session
   - [ ] Pair programming sessions
   - [ ] Code review guidelines

4. **Celebrate Success! 🎉**
   - [ ] Review progress metrics
   - [ ] Document lessons learned
   - [ ] Plan next quarter improvements
   - [ ] Team retrospective

---

## Ongoing Activities

### Weekly Activities

**Every Monday:**
- [ ] Review test coverage trends
- [ ] Check CI/CD health
- [ ] Review open PRs
- [ ] Plan week's work

**Every Friday:**
- [ ] Weekly metrics report
- [ ] Code quality review
- [ ] Dependency updates check
- [ ] Team sync on progress

### Continuous Activities

**Throughout All Phases:**
- [ ] Code reviews within 24 hours
- [ ] Keep linting clean
- [ ] Update documentation
- [ ] Security scanning
- [ ] Dependency updates

---

## Success Metrics

### Key Performance Indicators (KPIs)

Track these weekly:

| Metric | Week 0 | Week 4 | Week 8 | Week 12 |
|--------|--------|--------|--------|---------|
| Backend Coverage | 25.8% | 58.9% ✅ | 60% | 80% |
| Frontend Unit Coverage | 0% | 0% | 30% | 70% |
| Frontend E2E Coverage | 95% | 95% ✅ | 95% | 95% |
| Linting Errors | 0 | 0 ✅ | 0 | 0 |
| Linting Warnings | 11 | 0 ✅ | 0 | 0 |
| Security Vulnerabilities | 0 | 0 ✅ | 0 | 0 |
| Performance Score | TBD | TBD | 90+ | 90+ |
| Accessibility Score | TBD | TBD | 95+ | 95+ |

### Quality Gates

**PR Merge Requirements:**
- ✅ All tests pass
- ✅ Linting passes (no warnings)
- ✅ Code review approved
- ✅ Coverage doesn't decrease
- ✅ E2E tests pass
- ✅ Documentation updated

**Release Requirements:**
- ✅ Backend coverage > 80%
- ✅ Frontend coverage > 70%
- ✅ All security scans pass
- ✅ Performance budgets met
- ✅ Accessibility compliance (WCAG 2.1 AA)
- ✅ No critical bugs

---

## Risk Management

### Identified Risks

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| Resource availability | Medium | High | Buffer time in schedule |
| Learning curve (testing) | Medium | Medium | Training and pairing |
| Scope creep | Low | High | Strict prioritization |
| Breaking changes | Low | High | Comprehensive E2E tests |
| Performance regression | Low | Medium | Lighthouse CI monitoring |
| Dependency issues | Low | Medium | Regular updates, testing |

### Contingency Plans

**If behind schedule:**
1. Reduce Phase 5 scope (70% targets instead of 80%)
2. Extend timeline by 2 weeks
3. Focus on critical paths only

**If team member unavailable:**
1. Pair programming to share knowledge
2. Reprioritize based on available skills
3. Extend timeline if needed

**If major bug discovered:**
1. Pause feature work
2. Fix bug immediately
3. Add regression tests
4. Resume planned work

---

## Communication Plan

### Stakeholder Updates

**Weekly:**
- Email update with metrics
- Progress dashboard update
- Blockers and risks

**Bi-weekly:**
- Demo session
- Team retrospective
- Adjust plan if needed

**End of Phase:**
- Phase completion report
- Lessons learned
- Next phase kickoff

---

## Resources Required

### Team Allocation

- **Backend Developer:** 100% (Weeks 1-12)
- **Frontend Developer:** 100% (Weeks 1-12)
- **DevOps Engineer:** 20% (Weeks 1-2, 5)
- **Tech Lead:** 20% (Reviews and guidance)
- **QA Engineer:** On-call (Review and validate)

### Tools & Services

**Already Available:**
- ✅ Go 1.24
- ✅ Node.js 20
- ✅ GitHub Actions
- ✅ PostgreSQL

**To Install:**
- [ ] Vitest
- [ ] React Testing Library
- [ ] axe-core
- [ ] Lighthouse CI
- [ ] Redis (Phase 4, optional)

### Training & Learning

**Time Allocation:**
- Week 1: Testing best practices review (2 hours)
- Week 4: Component testing workshop (2 hours)
- Week 8: Accessibility training (2 hours)
- Week 12: Celebration and retrospective (2 hours)

---

## Definition of Done

### For Each Phase

- [ ] All planned tasks completed
- [ ] All tests passing
- [ ] Coverage targets met
- [ ] Documentation updated
- [ ] Code reviewed and merged
- [ ] Stakeholders notified
- [ ] Retrospective completed

### For Overall Project

- [ ] Backend coverage: 80%+
- [ ] Frontend unit coverage: 70%+
- [ ] Frontend E2E coverage: 95%+
- [ ] All linting warnings resolved
- [ ] E2E tests in CI
- [ ] Linting required in CI
- [ ] animal.go refactored
- [ ] WCAG 2.1 AA compliance
- [ ] Performance budgets met
- [ ] Documentation complete
- [ ] Team trained
- [ ] Production ready!

---

## Appendix: Quick Reference

### Important Commands

**Backend Testing:**
```bash
# Run all tests
go test ./...

# Run with coverage
go test ./... -cover

# Generate coverage report
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out

# Run specific package
go test ./internal/handlers -v
```

**Frontend Testing:**
```bash
cd frontend

# Run unit tests
npm run test:unit

# Run E2E tests
npm run test:e2e

# Run with coverage
npm run test:unit:coverage

# Run in watch mode
npm run test:unit -- --watch

# Run specific test
npm run test:unit -- src/contexts/AuthContext.test.tsx
```

**Linting:**
```bash
# Backend
go vet ./...
golangci-lint run

# Frontend
cd frontend
npm run lint
npm run lint -- --fix
```

**Build:**
```bash
# Backend
go build ./...

# Frontend
cd frontend
npm run build
```

### Key Files

- **Test Configuration:** `.github/workflows/test.yml`
- **Frontend Config:** `frontend/vitest.config.ts`
- **E2E Config:** `frontend/playwright.config.ts`
- **Coverage Report:** `coverage.out` (backend), `coverage/` (frontend)
- **This Plan:** `ACTION_PLAN_2025_Q4.md`
- **Review:** `COMPREHENSIVE_CODEBASE_REVIEW.md`

---

**Plan Version:** 1.0  
**Last Updated:** November 5, 2025  
**Next Review:** December 5, 2025 (or after Phase 2)

**Owner:** Development Team  
**Approver:** Tech Lead  
**Status:** Ready for Execution

---

**Let's build production-ready code! 🚀**
