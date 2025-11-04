# QA Action Plan - Current Status Report

**Last Updated:** November 3, 2025  
**Overall Progress:** Phase 2 Underway - Significant Backend Progress, Frontend Cleanup Needed

---

## 📊 Executive Summary

| Phase | Status | Progress | Target |
|-------|--------|----------|--------|
| **Phase 1: Quick Wins** | ✅ COMPLETE | 100% | Week 1 |
| **Phase 2: Backend Tests** | 🔄 IN PROGRESS | ~65% | Weeks 2-4 |
| **Phase 3: Frontend Types** | ⏳ NOT STARTED | 0% | Week 5 |
| **Phase 4: Frontend Tests** | ⏳ NOT STARTED | 0% | Weeks 6-9 |
| **Phase 5: E2E Tests** | ⏳ NOT STARTED | 0% | Weeks 10-12 |

**Overall Timeline Status:** On pace for Weeks 2-4, need to accelerate frontend work (Weeks 5+)

---

## ✅ Phase 1: Quick Wins (COMPLETE)

### Backend Test Infrastructure - EXCELLENT PROGRESS ✅

**Tests Created:** 8 test files with comprehensive coverage

| Test File | Location | Status | Coverage |
|-----------|----------|--------|----------|
| auth_test.go | `internal/auth/` | ✅ COMPLETE | 84.0% |
| auth_test.go | `internal/handlers/` | ✅ COMPLETE | (included in handlers) |
| animal_test.go | `internal/handlers/` | ✅ COMPLETE | (included in handlers) |
| group_test.go | `internal/handlers/` | ✅ COMPLETE | (included in handlers) |
| user_admin_test.go | `internal/handlers/` | ✅ COMPLETE | (included in handlers) |
| announcement_test.go | `internal/handlers/` | ✅ COMPLETE | (included in handlers) |
| password_reset_test.go | `internal/handlers/` | ✅ COMPLETE | (included in handlers) |
| middleware_test.go | `internal/middleware/` | ✅ COMPLETE | 71.8% |
| models_test.go | `internal/models/` | ✅ COMPLETE | 100.0% |

### Backend Coverage Summary

```
internal/auth           → 84.0%  ✅ EXCELLENT (exceeds 80% target)
internal/middleware     → 71.8%  ✅ GOOD (near target)
internal/models         → 100.0% ✅ PERFECT
internal/handlers       → 29.7%  ⚠️  NEEDS WORK (target: 80%)
internal/database       → 0.0%   ❌ NOT COVERED
internal/email          → 0.0%   ❌ NOT COVERED
internal/upload         → 0.0%   ❌ NOT COVERED
internal/logging        → 0.0%   ❌ NOT COVERED
cmd/api                 → 0.0%   ❌ NOT TESTED
cmd/seed                → 0.0%   ❌ NOT TESTED
```

**Overall Backend Coverage:** ~30% (estimated) - **FAR FROM 70% TARGET**

---

## 🔄 Phase 2: Critical Backend Tests (IN PROGRESS)

### ✅ Completed Tests

- [x] **Authentication & Security (Week 2)**
  - [x] `internal/auth/auth_test.go` - 84% coverage
  - [x] `internal/handlers/auth_test.go` - Password hashing, token validation
  - [x] Account lockout logic
  - [x] JWT validation

- [x] **Models & Validation**
  - [x] `internal/models/models_test.go` - 100% coverage
  - [x] All model methods tested

- [x] **Middleware**
  - [x] `internal/middleware/middleware_test.go` - 71.8% coverage
  - [x] Auth middleware tests
  - [x] Admin middleware tests
  - [x] Rate limiting tests

- [x] **Animal Operations (Partial)**
  - [x] `internal/handlers/animal_test.go` - Created, but **29.7% coverage**
  - [x] Create/Update/Delete operations
  - [ ] ⚠️ Missing: Complete filtering, pagination, edge cases

- [x] **Admin Functions**
  - [x] `internal/handlers/user_admin_test.go` - User management tests

- [x] **Other Handlers**
  - [x] `internal/handlers/group_test.go`
  - [x] `internal/handlers/announcement_test.go`
  - [x] `internal/handlers/password_reset_test.go`

### ⚠️ Coverage Gaps - HANDLERS ONLY AT 29.7%

**Problem:** Handlers coverage is only 29.7%, far below the 80% target.

**Root Cause:** Tests exist but many edge cases and error scenarios are not fully tested.

**Action Items:**
- [ ] Expand animal_test.go: filtering, pagination, sorting
- [ ] Add edge case tests: boundary conditions, invalid inputs
- [ ] Add error scenario tests: database failures, unauthorized access
- [ ] Increase handler coverage to minimum 60% this week

---

## ⏳ Phase 3: Frontend Type Safety (NOT STARTED) - URGENT!

### Current Linting Status

```
Total Errors:   119 errors
Total Warnings: 12 warnings
Total Issues:   131 problems
```

### Error Breakdown

| Error Type | Count | Examples |
|-----------|-------|----------|
| **`any` types** | ~35 errors | client.ts (5), BulkEditAnimalsPage (7), AnimalForm (4), etc. |
| **unused variables** | ~50 errors | Unused `page` params in Playwright tests |
| **React hooks deps** | ~12 warnings | ActivityFeed, ProtocolsList, Dashboard, etc. |
| **Fast refresh** | 2 errors | AuthContext.tsx, ToastContext.tsx |
| **Case declarations** | ~8 errors | GroupsPage.tsx, UsersPage.tsx |
| **Lexical scope** | ~3 errors | Various switch statements |

### Files with `any` Type Issues (35+ instances)

```
✗ client.ts (5 instances)
✗ AdminAnimalTagsPage.tsx (2)
✗ AdminDashboard.tsx (1)
✗ AnimalDetailPage.tsx (3)
✗ AnimalForm.tsx (4)
✗ BulkEditAnimalsPage.tsx (7)
✗ GroupsPage.tsx (4)
✗ Login.tsx (2)
✗ ResetPassword.tsx (1)
✗ Settings.tsx (2)
✗ SettingsPage.tsx (1)
✗ UpdateForm.tsx (2)
✗ UserProfilePage.tsx (1)
Total: ~35+ instances
```

### React Hook Exhaustive Dependencies (12 warnings)

Pages with missing dependencies:
- ActivityFeed.tsx
- ProtocolsList.tsx
- AdminAnimalTagsPage.tsx
- AdminDashboard.tsx
- AnimalDetailPage.tsx
- AnimalForm.tsx
- BulkEditAnimalsPage.tsx
- Dashboard.tsx
- GroupPage.tsx (2 warnings)
- SettingsPage.tsx
- UserProfilePage.tsx

### Fast Refresh Violations (2 errors)

These files export both components AND hooks/functions, which breaks fast refresh:
- `src/contexts/AuthContext.tsx` - exports `useAuth` hook
- `src/contexts/ToastContext.tsx` - exports `useToast` hook

**Fix:** Move hooks to separate files (e.g., `useAuth.ts`, `useToast.ts`)

### Case Declaration Issues (8 errors)

Switch statements with const/let declarations need wrapping in blocks:
```typescript
// ❌ Bad
case 'EDIT':
  const formData = { ... }

// ✅ Good
case 'EDIT': {
  const formData = { ... }
}
```

---

## 🎯 Priority Actions (Next 2 Weeks)

### URGENT: Frontend Cleanup (Days 1-3)

These are quick wins that must be completed immediately:

#### 1. Fix Fast Refresh (30 minutes)
```bash
# Move hooks to separate files
mv src/contexts/useAuth.ts  # Extract from AuthContext.tsx
mv src/contexts/useToast.ts # Extract from ToastContext.tsx
```
- [ ] Extract `useAuth` from `src/contexts/AuthContext.tsx`
- [ ] Extract `useToast` from `src/contexts/ToastContext.tsx`
- [ ] Update imports in affected files
- [ ] Verify linting passes

#### 2. Fix Case Declarations (30 minutes)
```bash
# Wrap all switch case blocks with const/let
```
- [ ] GroupsPage.tsx (6 cases)
- [ ] UsersPage.tsx (multiple cases)
- [ ] Other switch statements

#### 3. Fix Unused Test Parameters (30 minutes)
```bash
# In tests/**, remove unused `page` and `expect` parameters
```
- [ ] Remove `page` from ~50 test functions
- [ ] Remove `expect` from tag-selection-ux.spec.ts
- [ ] Verify all tests still work

**Expected Result:** Reduce errors from 119 to ~50

#### 4. Define TypeScript Types (2 hours)
```typescript
// Create src/types/api.ts with all response types
interface APIResponse<T> {
  data?: T;
  error?: string;
}
interface User { ... }
interface Animal { ... }
interface Group { ... }
interface Comment { ... }
```
- [ ] Create `src/types/api.ts`
- [ ] Define User, Animal, Group, Comment, Tag, etc. interfaces
- [ ] Export APIResponse<T> generic type

#### 5. Fix `any` Types in client.ts (30 minutes)
```typescript
// Replace all any with proper types
// Line 254, 275, 295, 302, 320
```
- [ ] Line 254: Replace with proper interceptor type
- [ ] Line 275: Replace with proper response type
- [ ] Line 295: Replace with proper error type
- [ ] Line 302: Replace with proper config type
- [ ] Line 320: Replace with proper type

**Expected Result:** client.ts goes from 5 errors to 0

#### 6. Fix React Hook Dependencies (3 hours)
Use `useCallback` to wrap functions:
```typescript
const loadItems = useCallback(async () => {
  // fetch logic
}, [dependencies]);

useEffect(() => {
  loadItems();
}, [loadItems]);
```

Files to fix:
- [ ] ActivityFeed.tsx
- [ ] ProtocolsList.tsx
- [ ] AdminAnimalTagsPage.tsx
- [ ] AdminDashboard.tsx
- [ ] AnimalDetailPage.tsx
- [ ] AnimalForm.tsx
- [ ] BulkEditAnimalsPage.tsx
- [ ] Dashboard.tsx
- [ ] GroupPage.tsx
- [ ] SettingsPage.tsx
- [ ] UserProfilePage.tsx

**Expected Result:** Reduce warnings from 12 to 0

#### 7. Fix Remaining `any` Types (4 hours)
Apply the TypeScript interfaces from step 4 to all page components

**Expected Result:** 119 errors → 0 errors ✅

---

## ⏳ Phase 4: Frontend Unit Tests (NOT STARTED)

### Required Setup
- [ ] Install Vitest + dependencies
- [ ] Create vitest.config.ts
- [ ] Create test utilities (custom render, mocks)
- [ ] Update package.json scripts

### Test Infrastructure Status
- ❌ Vitest not installed
- ❌ @testing-library/react not installed
- ❌ Test utilities not created
- ❌ No component tests written

---

## 📈 Estimated Timelines

### If You Focus on Frontend First (RECOMMENDED)
```
Week 1 (This Week):
  Mon-Tue: Quick wins (fast refresh, case declarations, unused params)
  Wed-Thu: TypeScript types + any type fixes
  Fri:     React hooks dependencies

Week 2:
  Frontend type safety complete (0 errors)
  Backend tests expanded (handlers 40%+)

Week 3-4:
  Backend coverage 60-70%
  Frontend unit tests started
```

### If You Ignore Frontend
```
Backend will reach 60-70% by Week 4
But frontend will still have 119 errors and 0% unit test coverage
```

---

## 🎯 Success Criteria for This Week

- [ ] Frontend linting errors: 119 → 0
- [ ] Frontend linting warnings: 12 → 0
- [ ] Backend handler coverage: 29.7% → 40%+
- [ ] All tests passing locally
- [ ] No new linting issues in PRs

---

## 🚨 Blocker Alert

**FRONTEND IS A BLOCKER**: No component tests can be written until linting is clean and types are fixed. Frontend work is currently impossible to progress on tests.

---

## 💡 Recommendations

1. **THIS WEEK:** Fix frontend linting (6-8 hours of focused work)
2. **NEXT WEEK:** Expand backend handler tests (catch-up on 40%+ coverage)
3. **WEEK 3:** Setup frontend unit test infrastructure
4. **WEEKS 4-9:** Write frontend component tests (70% target)

---

## 📝 Next Steps

1. Create a branch: `feature/frontend-type-safety`
2. Fix issues in order:
   - Fast refresh violations
   - Case declarations  
   - Unused test parameters
   - Define TypeScript types
   - Fix `any` types
   - Fix React hooks dependencies
3. Create PR with all fixes
4. Verify `npm run lint` = 0 errors
5. Then proceed to expand backend tests

**Estimated Time:** 6-8 hours for frontend cleanup → unlocks Phase 4

---

*Report Generated: QA Expert Review of Codebase*  
*Branch Status: copilot/fix-100809204...*  
*Last Commit: Latest from main branch*
