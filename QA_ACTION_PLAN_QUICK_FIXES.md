# QA Action Plan - Quick Fixes Checklist

## 🎯 This Week's Focus: Frontend Cleanup (6-8 hours)

Goal: Get from 119 errors → 0 errors, enabling Phase 4 (frontend tests)

---

## Task 1: Fix Fast Refresh Violations (30 minutes)

**Files:** 
- `src/contexts/AuthContext.tsx`
- `src/contexts/ToastContext.tsx`

**Action:**
```bash
# Step 1: Extract useAuth hook
cat > src/hooks/useAuth.ts << 'EOF'
import { useContext } from 'react';
import { AuthContext } from '../contexts/AuthContext';

export const useAuth = () => {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error('useAuth must be used within AuthProvider');
  }
  return context;
};
EOF

# Step 2: Extract useToast hook  
cat > src/hooks/useToast.ts << 'EOF'
import { useContext } from 'react';
import { ToastContext } from '../contexts/ToastContext';

export const useToast = () => {
  const context = useContext(ToastContext);
  if (!context) {
    throw new Error('useToast must be used within ToastProvider');
  }
  return context;
};
EOF

# Step 3: Update AuthContext.tsx to only export component
# Step 4: Update ToastContext.tsx to only export component
# Step 5: Update all imports in project
# Step 6: Verify linting passes
npm run lint
```

**Checklist:**
- [ ] Create `src/hooks/useAuth.ts`
- [ ] Create `src/hooks/useToast.ts`
- [ ] Update `src/contexts/AuthContext.tsx` (remove hook export)
- [ ] Update `src/contexts/ToastContext.tsx` (remove hook export)
- [ ] Find all `import { useAuth }` and update to `import useAuth from '../hooks/useAuth'`
- [ ] Find all `import { useToast }` and update to `import useToast from '../hooks/useToast'`
- [ ] Run `npm run lint` - should remove 2 errors

---

## Task 2: Fix Case Declaration Issues (30 minutes)

**Files with issues:**
- `src/pages/GroupsPage.tsx` (6 errors)
- `src/pages/UsersPage.tsx` (multiple errors)

**Pattern:**
```typescript
// ❌ Before
case 'EDIT':
  const formData = { ... }
  break;

// ✅ After
case 'EDIT': {
  const formData = { ... }
  break;
}
```

**Checklist:**
- [ ] Find all `case` statements with `const` or `let` declarations
- [ ] Wrap each case block with `{ }` braces
- [ ] Test that logic still works
- [ ] Run `npm run lint` - should remove ~8 errors

---

## Task 3: Remove Unused Test Parameters (30 minutes)

**Files affected:** `tests/*.spec.ts` (~50 instances)

**Pattern:**
```typescript
// ❌ Before
test('should do something', async ({ page }) => {
  // page is never used
});

// ✅ After  
test('should do something', async () => {
  // no unused params
});
```

**Command:**
```bash
# Find all unused 'page' in tests
grep -r "async ({ page })" tests/ | head -20

# Manual fix: Remove unused parameters from test functions
```

**Checklist:**
- [ ] Remove unused `page` parameter from ~50 tests
- [ ] Remove unused `expect` from `tests/tag-selection-ux.spec.ts`
- [ ] Ensure tests still have valid assertions
- [ ] Run `npm run lint` - should remove ~50 errors

---

## Task 4: Define TypeScript Types (2 hours)

**Create:** `src/types/api.ts`

```typescript
// Generic API response wrapper
export interface APIResponse<T> {
  data?: T;
  error?: string;
  details?: Record<string, string[]>;
}

// User types
export interface User {
  id: number;
  username: string;
  email: string;
  is_admin: boolean;
  created_at: string;
  updated_at: string;
}

// Animal types
export interface Animal {
  id: number;
  name: string;
  species: string;
  breed?: string;
  age?: number;
  status: 'Available' | 'Adopted' | 'Pending' | 'Intake';
  description?: string;
  image_url?: string;
  group_id?: number;
  created_at: string;
  updated_at: string;
}

// Group types
export interface Group {
  id: number;
  name: string;
  description?: string;
  created_at: string;
  updated_at: string;
}

// Comment types
export interface Comment {
  id: number;
  content: string;
  author_id: number;
  animal_id: number;
  created_at: string;
  updated_at: string;
}

// Tag types
export interface Tag {
  id: number;
  name: string;
  created_at: string;
}

// Add other types as needed...
```

**Checklist:**
- [ ] Create `src/types/api.ts`
- [ ] Define all API response types
- [ ] Export generic `APIResponse<T>` wrapper
- [ ] Document each type
- [ ] Verify imports work from other files

---

## Task 5: Fix `any` Types in client.ts (30 minutes)

**File:** `src/api/client.ts`

**Lines with issues:** 254, 275, 295, 302, 320

**Before:**
```typescript
254:  (response: any) => response
275:  (response: any) => response
295:  (error: any) => Promise.reject(error)
302:  (config: any) => config
320:  (error: any) => Promise.reject(error)
```

**After:**
```typescript
254:  (response: AxiosResponse) => response
275:  (response: AxiosResponse) => response
295:  (error: AxiosError) => Promise.reject(error)
302:  (config: AxiosRequestConfig) => config
320:  (error: AxiosError) => Promise.reject(error)
```

**Checklist:**
- [ ] Import proper types: `import { AxiosResponse, AxiosError, AxiosRequestConfig } from 'axios'`
- [ ] Replace line 254: `(response: any)` → `(response: AxiosResponse)`
- [ ] Replace line 275: `(response: any)` → `(response: AxiosResponse)`
- [ ] Replace line 295: `(error: any)` → `(error: AxiosError)`
- [ ] Replace line 302: `(config: any)` → `(config: AxiosRequestConfig)`
- [ ] Replace line 320: `(error: any)` → `(error: AxiosError)`
- [ ] Run `npm run lint` - should remove 5 errors

---

## Task 6: Fix React Hook Dependencies (3-4 hours)

**Pattern:** Use `useCallback` to stabilize function references

**Before:**
```typescript
useEffect(() => {
  loadItems();
}, []); // Missing 'loadItems' dependency
```

**After:**
```typescript
const loadItems = useCallback(async () => {
  try {
    const response = await api.get('/items');
    setItems(response.data);
  } catch (error) {
    console.error('Error loading items:', error);
  }
}, []);

useEffect(() => {
  loadItems();
}, [loadItems]);
```

**Files to fix (12 warnings):**
- [ ] `src/components/ActivityFeed.tsx` - Line 70
- [ ] `src/components/ProtocolsList.tsx` - Line 31
- [ ] `src/pages/AdminAnimalTagsPage.tsx` - Line 24
- [ ] `src/pages/AdminDashboard.tsx` - Line 15
- [ ] `src/pages/AnimalDetailPage.tsx` - Line 55
- [ ] `src/pages/AnimalForm.tsx` - Line 42
- [ ] `src/pages/BulkEditAnimalsPage.tsx` - Line 60
- [ ] `src/pages/Dashboard.tsx` - Line 18
- [ ] `src/pages/GroupPage.tsx` - Lines 42, 49
- [ ] `src/pages/SettingsPage.tsx` - Line 18
- [ ] `src/pages/UserProfilePage.tsx` - Line 17

**Checklist:**
- [ ] Wrap data-fetching functions with `useCallback`
- [ ] Add function to dependency array of useEffect
- [ ] Test that data still loads correctly
- [ ] Verify no infinite loops
- [ ] Run `npm run lint` - should remove ~12 warnings

---

## Task 7: Fix Remaining `any` Types in Page Components (3-4 hours)

**Files with `any` instances:**

```
✗ AdminAnimalTagsPage.tsx (2)      → Use Animal type
✗ AdminDashboard.tsx (1)            → Use Stats type
✗ AnimalDetailPage.tsx (3)          → Use Animal type
✗ AnimalForm.tsx (4)                → Use Animal type
✗ BulkEditAnimalsPage.tsx (7)       → Use Animal[] type
✗ GroupsPage.tsx (4)                → Use Group type
✗ Login.tsx (2)                     → Use LoginRequest type
✗ ResetPassword.tsx (1)             → Use PasswordReset type
✗ Settings.tsx (2)                  → Use Settings type
✗ SettingsPage.tsx (1)              → Use Settings type
✗ UpdateForm.tsx (2)                → Use Update type
✗ UserProfilePage.tsx (1)           → Use User type
Total: ~30+ instances
```

**Strategy:**
1. Import types from `src/types/api.ts`
2. Replace `any` with proper type
3. Example:
   ```typescript
   // ❌ Before
   const [animal, setAnimal] = useState<any>(null);
   
   // ✅ After
   import { Animal } from '../types/api';
   const [animal, setAnimal] = useState<Animal | null>(null);
   ```

**Checklist:**
- [ ] AnimalForm.tsx: Replace 4 `any` → use `Animal` type
- [ ] AnimalDetailPage.tsx: Replace 3 `any` → use `Animal` type
- [ ] BulkEditAnimalsPage.tsx: Replace 7 `any` → use `Animal[]` type
- [ ] GroupsPage.tsx: Replace 4 `any` → use `Group` type
- [ ] AdminDashboard.tsx: Replace 1 `any` → define Stats type
- [ ] And so on for remaining files
- [ ] Run `npm run lint` - should have 0 errors!

---

## 📋 Final Checklist

When all tasks complete, run:

```bash
cd frontend

# Verify no errors
npm run lint

# Should output: ✖ 0 problems

# Run tests to ensure nothing broke
npm test

# Build should work
npm run build
```

**Success Criteria:**
- [ ] `npm run lint` outputs 0 errors, 0 warnings
- [ ] `npm test` passes
- [ ] `npm run build` completes without errors
- [ ] No TypeScript compilation errors
- [ ] All imports work correctly
- [ ] Application runs without console errors

---

## 🚀 After This Checklist

Once linting is clean (0 errors):
1. Create PR: `feature/frontend-type-safety`
2. Get code review
3. Merge to main
4. Begin Phase 4: Frontend Unit Tests
5. Expand backend handler tests to 60%+

**Total Estimated Time:** 6-8 hours  
**Impact:** Unlocks all future frontend development

---

*Checklist Created: QA Action Plan Quick Fixes*  
*Target Completion: This Week*  
*Priority: CRITICAL BLOCKER*
