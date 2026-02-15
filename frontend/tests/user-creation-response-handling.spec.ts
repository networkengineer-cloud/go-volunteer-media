import { test, expect } from '@playwright/test';
import { loginAsAdmin } from './helpers/auth';

/**
 * Test for User Creation Response Handling Fix
 * 
 * This test verifies that the Users page correctly handles both response formats
 * from the backend user creation endpoints using mocked responses for deterministic testing:
 * 1. Direct User object (when password is provided)
 * 2. { user: User, message?: string, warning?: string } (when send_setup_email is used)
 * 
 * Addresses the bug: "Uncaught TypeError: d is not iterable"
 */

test.describe('User Creation Response Handling with Mocked Responses', () => {
  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto('/users');
    await page.waitForLoadState('networkidle');
  });

  test('handles direct User object response (password provided)', async ({ page }) => {
    const timestamp = Date.now();
    const username = `testuser_${timestamp}`;
    const email = `test_${timestamp}@example.com`;

    // Mock the API response - direct User object
    await page.route('**/api/admin/users', async (route) => {
      if (route.request().method() === 'POST') {
        await route.fulfill({
          status: 201,
          contentType: 'application/json',
          body: JSON.stringify({
            id: 999,
            username: username,
            email: email,
            is_admin: false,
            groups: []
          })
        });
      } else {
        await route.continue();
      }
    });

    // Open the add user form
    await page.click('button:has-text("Add User")');
    await expect(page.locator('.users-create-form')).toBeVisible();

    // Fill in the form with a direct password
    await page.fill('input[name="username"]', username);
    await page.fill('input[name="email"]', email);
    
    // Uncheck "Send setup email" to provide password directly
    const setupEmailCheckbox = page.locator('input[name="send_setup_email"]');
    if (await setupEmailCheckbox.isChecked()) {
      await setupEmailCheckbox.click();
    }
    
    await expect(page.locator('input[name="password"]')).toBeVisible();
    await page.fill('input[name="password"]', 'SecurePassword123!');

    // Submit the form
    await page.click('button[type="submit"]:has-text("Create User")');

    // Should show success message with no errors
    await expect(page.locator('.alert-success')).toBeVisible({ timeout: 5000 });
    await expect(page.locator('.alert-success')).toContainText('successfully');
    await expect(page.locator('.alert-error')).not.toBeVisible();
    await expect(page.locator('.alert-warning')).not.toBeVisible();
  });

  test('handles wrapped response with success message (setup email sent)', async ({ page }) => {
    const timestamp = Date.now();
    const username = `setupuser_${timestamp}`;
    const email = `setup_${timestamp}@example.com`;

    // Mock the API response - wrapped format with message
    await page.route('**/api/admin/users', async (route) => {
      if (route.request().method() === 'POST') {
        await route.fulfill({
          status: 201,
          contentType: 'application/json',
          body: JSON.stringify({
            user: {
              id: 998,
              username: username,
              email: email,
              is_admin: false,
              groups: []
            },
            message: `User created successfully. Password setup email sent to ${email}`
          })
        });
      } else {
        await route.continue();
      }
    });

    // Open the add user form
    await page.click('button:has-text("Add User")');
    await expect(page.locator('.users-create-form')).toBeVisible();

    // Fill in the form with setup email option
    await page.fill('input[name="username"]', username);
    await page.fill('input[name="email"]', email);
    
    // Ensure "Send setup email" is checked
    const setupEmailCheckbox = page.locator('input[name="send_setup_email"]');
    if (!await setupEmailCheckbox.isChecked()) {
      await setupEmailCheckbox.click();
    }

    // Password field should be hidden
    await expect(page.locator('input[name="password"]')).not.toBeVisible();

    // Submit the form
    await page.click('button[type="submit"]:has-text("Create User")');

    // Should show success message with email notification
    await expect(page.locator('.alert-success')).toBeVisible({ timeout: 5000 });
    await expect(page.locator('.alert-success')).toContainText(/setup email sent|successfully/i);
    await expect(page.locator('.alert-error')).not.toBeVisible();
  });

  test('handles wrapped response with warning (email failed)', async ({ page }) => {
    const timestamp = Date.now();
    const username = `warnuser_${timestamp}`;
    const email = `warn_${timestamp}@example.com`;

    // Mock the API response - wrapped format with warning
    await page.route('**/api/admin/users', async (route) => {
      if (route.request().method() === 'POST') {
        await route.fulfill({
          status: 201,
          contentType: 'application/json',
          body: JSON.stringify({
            user: {
              id: 997,
              username: username,
              email: email,
              is_admin: false,
              groups: []
            },
            warning: "User created successfully, but the setup email could not be sent. You can use the 'Reset Password' button on the user's profile to send a new setup email."
          })
        });
      } else {
        await route.continue();
      }
    });

    // Open the add user form
    await page.click('button:has-text("Add User")');
    await expect(page.locator('.users-create-form')).toBeVisible();

    // Fill in the form
    await page.fill('input[name="username"]', username);
    await page.fill('input[name="email"]', email);
    
    // Ensure "Send setup email" is checked
    const setupEmailCheckbox = page.locator('input[name="send_setup_email"]');
    if (!await setupEmailCheckbox.isChecked()) {
      await setupEmailCheckbox.click();
    }

    // Submit the form
    await page.click('button[type="submit"]:has-text("Create User")');

    // Should show BOTH success AND warning (user created, but email failed)
    await expect(page.locator('.alert-success')).toBeVisible({ timeout: 5000 });
    await expect(page.locator('.alert-success')).toContainText('User created successfully');
    await expect(page.locator('.alert-warning')).toBeVisible();
    await expect(page.locator('.alert-warning')).toContainText(/could not be sent|email/i);
    await expect(page.locator('.alert-error')).not.toBeVisible();
  });

  test('page does not throw "d is not iterable" error during user creation', async ({ page }) => {
    // Set up error listeners
    const consoleErrors: string[] = [];
    page.on('console', msg => {
      if (msg.type() === 'error') {
        consoleErrors.push(msg.text());
      }
    });

    const pageErrors: Error[] = [];
    page.on('pageerror', error => {
      pageErrors.push(error);
    });

    const timestamp = Date.now();
    const username = `noerroruser_${timestamp}`;
    const email = `noerror_${timestamp}@example.com`;

    // Mock direct User response
    await page.route('**/api/admin/users', async (route) => {
      if (route.request().method() === 'POST') {
        await route.fulfill({
          status: 201,
          contentType: 'application/json',
          body: JSON.stringify({
            id: 996,
            username: username,
            email: email,
            is_admin: false,
            groups: []
          })
        });
      } else {
        await route.continue();
      }
    });

    // Open and submit form
    await page.click('button:has-text("Add User")');
    await page.fill('input[name="username"]', username);
    await page.fill('input[name="email"]', email);
    
    const setupEmailCheckbox = page.locator('input[name="send_setup_email"]');
    if (await setupEmailCheckbox.isChecked()) {
      await setupEmailCheckbox.click();
    }
    
    await page.fill('input[name="password"]', 'SecurePassword123!');
    await page.click('button[type="submit"]:has-text("Create User")');

    // Wait for response processing
    await page.waitForTimeout(2000);

    // Verify no "d is not iterable" errors
    const hasIterableError = consoleErrors.some(err => err.includes('is not iterable')) ||
                             pageErrors.some(err => err.message.includes('is not iterable'));
    expect(hasIterableError).toBe(false);

    // Verify no TypeErrors
    const hasTypeError = pageErrors.some(err => err.name === 'TypeError');
    if (hasTypeError) {
      console.error('TypeError found:', pageErrors.filter(err => err.name === 'TypeError'));
    }
    expect(hasTypeError).toBe(false);
  });
});
