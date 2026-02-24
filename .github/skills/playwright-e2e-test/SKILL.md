---
name: playwright-e2e-test
description: Write Playwright end-to-end tests for go-volunteer-media features. Covers the test file location convention, how to authenticate in tests, helper patterns used in the existing specs, what scenarios require E2E vs unit tests, and accessibility testing. Use when adding E2E coverage for a new page, feature, or user workflow.
argument-hint: [feature or page to test]
---

# Writing Playwright E2E Tests

## Where Tests Live

```
frontend/tests/               ← primary location for feature E2E specs
tests/                        ← root-level specs (currently: users-page.spec.ts)
frontend/playwright.config.ts ← Playwright config (baseURL, browser list, webserver)
frontend/scripts/e2e-webserver.mjs  ← dev server startup script for tests
```

Naming convention: `frontend/tests/<feature-name>.spec.ts`

## Test File Scaffold

```typescript
import { test, expect } from '@playwright/test';

test.describe('Foo Feature', () => {
  // Authenticate before each test (if the page requires login)
  test.beforeEach(async ({ page }) => {
    await page.goto('/login');
    await page.getByLabel('Username').fill('admin');
    await page.getByLabel('Password').fill('demo1234');
    await page.getByRole('button', { name: 'Sign in' }).click();
    await page.waitForURL('**/dashboard');
  });

  test('displays list of foos', async ({ page }) => {
    await page.goto('/groups/1/foos');
    await expect(page.getByRole('heading', { name: 'Foo' })).toBeVisible();
    await expect(page.getByRole('list')).toBeVisible();
  });

  test('can create a new foo', async ({ page }) => {
    await page.goto('/groups/1/foos');
    await page.getByRole('button', { name: 'Add Foo' }).click();
    await page.getByLabel('Name').fill('Test Foo');
    await page.getByRole('button', { name: 'Save' }).click();
    await expect(page.getByText('Test Foo')).toBeVisible();
  });

  test('shows error when API fails', async ({ page }) => {
    await page.route('**/api/groups/*/foos', route => route.fulfill({ status: 500 }));
    await page.goto('/groups/1/foos');
    await expect(page.getByRole('alert')).toBeVisible();
  });
});
```

## Seed Credentials

Tests must use the seeded demo accounts. Do not hardcode other credentials:

| Use case | Username | Password |
|---|---|---|
| Site admin actions | `admin` | `demo1234` |
| Group admin actions (ModSquad) | `merry` or `sophia` | `demo1234` |
| Regular volunteer | `terry` | `volunteer2026!` |

## Selector Priority (Playwright best practice)

Use selectors in this order of preference:
1. `getByRole()` — semantic and accessible
2. `getByLabel()` — form fields
3. `getByText()` — visible text content
4. `getByTestId()` — add `data-testid` attributes only when above options aren't possible
5. CSS selectors — last resort

**Avoid** `page.$('.my-class')` — brittle and not accessibility-aware.

## Accessibility Checks

Every new spec should verify:
- Headings are semantically correct (`getByRole('heading', { level: 1 })`)
- Interactive elements are keyboard-accessible (`page.keyboard.press('Tab')`)
- Error messages use `role="alert"` (automatically announced to screen readers)
- Form labels are associated with inputs (`getByLabel(...)` succeeds)

## When to Write E2E vs Unit Tests

| Scenario | Test type |
|---|---|
| New page/screen | E2E (Playwright) |
| User login / auth flow | E2E |
| Form submission + backend interaction | E2E |
| Navigation / routing | E2E |
| UI component in isolation (no network) | Unit (Vitest) |
| Utility function / hook | Unit (Vitest) |
| Pure logic / calculation | Unit (Vitest) |

## Running Your Tests

```bash
cd frontend

# All E2E tests
npx playwright test

# Your new spec only
npx playwright test tests/foo-page.spec.ts

# Debug mode (interactive)
npx playwright test tests/foo-page.spec.ts --ui

# View report after failure
npx playwright show-report
```

## Looking at Existing Tests for Patterns

Before writing a new spec, read a similar existing one:
- `frontend/tests/auth.spec.ts` — login/logout flow
- `frontend/tests/activity-feed.spec.ts` — group-scoped page with data
- `frontend/tests/add-user-form-ux.spec.ts` — admin form with validation
- `frontend/tests/activity-feed-animal-links.spec.ts` — navigation between pages
