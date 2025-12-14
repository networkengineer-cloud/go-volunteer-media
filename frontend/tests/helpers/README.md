# Playwright Test Helpers

This directory contains helper utilities for Playwright E2E tests.

## API Mocks (`api-mocks.ts`)

Mock helpers for external API calls to allow tests to run without a backend server.

### Usage Example

```typescript
import { test, expect } from '@playwright/test';
import { mockLoginSuccess, MockUser } from './helpers/api-mocks';

test('should login successfully', async ({ page }) => {
  // Mock the login API
  const adminUser: MockUser = {
    id: 1,
    username: 'testadmin',
    email: 'admin@test.com',
    is_admin: true,
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString()
  };
  
  await mockLoginSuccess(page, adminUser);
  
  // Now test the login flow
  await page.goto('/login');
  await page.fill('input[name="username"]', 'testadmin');
  await page.fill('input[name="password"]', 'password123');
  await page.click('button[type="submit"]');
  
  // Verify success
  await expect(page).toHaveURL(/dashboard/);
});
```

### Available Mocks

- `mockLoginSuccess(page, user, token?)` - Mock successful login
- `mockLoginFailure(page, message?)` - Mock failed login
- More mocks can be added as needed

## Guidelines

### What to Mock
✅ External API calls (backend endpoints)
✅ Network requests
✅ File uploads
✅ Third-party services

### What NOT to Mock
❌ Application code (components, routes)
❌ Frontend logic and state management
❌ UI interactions and rendering
❌ Browser APIs (unless testing browser compatibility)

## Adding New Mocks

When adding new mock helpers:

1. Keep them simple and focused
2. Use TypeScript interfaces for type safety
3. Document parameters and return values
4. Follow the naming convention: `mock[Feature][Action]`
5. Add usage examples in this README

## Running Tests

```bash
# Run all tests
npm run test:e2e

# Run specific test file
npx playwright test tests/authentication.spec.ts

# Run tests in UI mode
npx playwright test --ui

# Run tests with debugging
npx playwright test --debug
```
