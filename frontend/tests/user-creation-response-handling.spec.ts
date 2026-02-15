import { test, expect } from '@playwright/test';
import { loginAsAdmin } from './helpers/auth';

/**
 * Test for User Creation Response Handling Fix
 * 
 * This test verifies that the Users page correctly handles both response formats
 * from the backend user creation endpoints:
 * 1. Direct User object (when password is provided)
 * 2. { user: User, message?: string, warning?: string } (when send_setup_email is used)
 * 
 * Addresses the bug: "Uncaught TypeError: d is not iterable"
 */

test.describe('User Creation Response Handling', () => {
  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto('/users');
    await page.waitForLoadState('networkidle');
  });

  test('handles user creation with direct password (returns User object)', async ({ page }) => {
    // Open the add user form
    await page.click('button:has-text("Add User")');
    await expect(page.locator('.users-create-form')).toBeVisible();

    const timestamp = Date.now();
    const username = `testuser_${timestamp}`;
    const email = `test_${timestamp}@example.com`;

    // Fill in the form with a direct password (not using setup email)
    await page.fill('input[name="username"]', username);
    await page.fill('input[name="email"]', email);
    
    // Uncheck "Send setup email" to provide password directly
    const setupEmailCheckbox = page.locator('input[name="send_setup_email"]');
    if (await setupEmailCheckbox.isChecked()) {
      await setupEmailCheckbox.click();
    }
    
    // Wait for password field to be visible
    await expect(page.locator('input[name="password"]')).toBeVisible();
    await page.fill('input[name="password"]', 'SecurePassword123!');

    // Submit the form
    await page.click('button[type="submit"]:has-text("Create User")');

    // Should show success message (not throw an error)
    await expect(page.locator('.alert-success')).toBeVisible({ timeout: 10000 });
    await expect(page.locator('.alert-success')).toContainText('successfully');

    // Form should close after success
    await expect(page.locator('.users-create-form')).not.toBeVisible({ timeout: 5000 });

    // User should appear in the users list
    await expect(page.locator(`text=${username}`)).toBeVisible();
  });

  test('handles user creation with setup email (returns { user, message } object)', async ({ page }) => {
    // Open the add user form
    await page.click('button:has-text("Add User")');
    await expect(page.locator('.users-create-form')).toBeVisible();

    const timestamp = Date.now();
    const username = `setupuser_${timestamp}`;
    const email = `setup_${timestamp}@example.com`;

    // Fill in the form with setup email option
    await page.fill('input[name="username"]', username);
    await page.fill('input[name="email"]', email);
    
    // Ensure "Send setup email" is checked (should be default)
    const setupEmailCheckbox = page.locator('input[name="send_setup_email"]');
    if (!await setupEmailCheckbox.isChecked()) {
      await setupEmailCheckbox.click();
    }

    // Password field should be hidden when setup email is checked
    await expect(page.locator('input[name="password"]')).not.toBeVisible();

    // Submit the form
    await page.click('button[type="submit"]:has-text("Create User")');

    // Should show success message or warning (if email fails)
    // Either is acceptable - we just want to verify no TypeError occurs
    const hasSuccess = await page.locator('.alert-success').isVisible({ timeout: 10000 });
    const hasError = await page.locator('.alert-error').isVisible({ timeout: 1000 });
    
    expect(hasSuccess || hasError).toBe(true);

    // If successful, form should close
    if (hasSuccess) {
      await expect(page.locator('.users-create-form')).not.toBeVisible({ timeout: 5000 });
      await expect(page.locator(`text=${username}`)).toBeVisible();
    } else if (hasError) {
      // If email failed to send, we should see a warning but user should be created
      await expect(page.locator('.alert-error')).toContainText(/setup email|could not be sent|email/i);
      // Form might stay open in this case to show the warning
    }
  });

  test('page does not throw "d is not iterable" error during user creation', async ({ page }) => {
    // Set up console error listener
    const consoleErrors: string[] = [];
    page.on('console', msg => {
      if (msg.type() === 'error') {
        consoleErrors.push(msg.text());
      }
    });

    // Set up page error listener  
    const pageErrors: Error[] = [];
    page.on('pageerror', error => {
      pageErrors.push(error);
    });

    // Open the add user form
    await page.click('button:has-text("Add User")');

    const timestamp = Date.now();
    const username = `noerroruser_${timestamp}`;
    const email = `noerror_${timestamp}@example.com`;

    // Fill and submit form (with direct password)
    await page.fill('input[name="username"]', username);
    await page.fill('input[name="email"]', email);
    
    const setupEmailCheckbox = page.locator('input[name="send_setup_email"]');
    if (await setupEmailCheckbox.isChecked()) {
      await setupEmailCheckbox.click();
    }
    
    await page.fill('input[name="password"]', 'SecurePassword123!');
    await page.click('button[type="submit"]:has-text("Create User")');

    // Wait for response
    await page.waitForTimeout(3000);

    // Check that no "d is not iterable" errors occurred
    const hasIterableError = consoleErrors.some(err => err.includes('is not iterable')) ||
                             pageErrors.some(err => err.message.includes('is not iterable'));
    
    expect(hasIterableError).toBe(false);

    // Check that no TypeError occurred
    const hasTypeError = pageErrors.some(err => err.name === 'TypeError');
    
    if (hasTypeError) {
      console.error('TypeError found:', pageErrors.filter(err => err.name === 'TypeError'));
    }
    
    expect(hasTypeError).toBe(false);
  });
});
