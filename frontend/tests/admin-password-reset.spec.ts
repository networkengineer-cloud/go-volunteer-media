import { test, expect, type Page } from '@playwright/test';

/**
 * Admin Password Reset E2E Tests
 * 
 * These tests validate the admin password reset functionality:
 * - Admin can access the reset password feature
 * - Admin can reset a user's password
 * - User can login with the new password
 * - Password validation is enforced
 */

// Test configuration
const BASE_URL = 'http://localhost:5173';

// Test credentials
const ADMIN_USER = {
  username: 'testadmin',
  password: 'password123'
};

const TEST_USER = {
  username: 'testuser',
  password: 'password123',
  email: 'testuser@example.com'
};

const NEW_PASSWORD = 'newpassword123';

test.describe('Admin Password Reset', () => {
  // Helper function to login as admin
  async function loginAsAdmin(page: Page) {
    await page.goto(BASE_URL + '/login');
    await page.fill('input[name="username"]', ADMIN_USER.username);
    await page.fill('input[name="password"]', ADMIN_USER.password);
    await page.click('button[type="submit"]');
    await page.waitForURL(/dashboard|\/$/);
  }

  test('admin should see Reset Password button in users list', async () => {
    // Login as admin
    await loginAsAdmin(page);
    
    // Navigate to users page
    await page.goto(BASE_URL + '/admin/users');
    
    // Wait for users to load
    await page.waitForSelector('.users-table, .users-mobile-cards');
    
    // Verify Reset Password button is visible
    const resetPasswordButtons = page.locator('button:has-text("Reset Password")');
    await expect(resetPasswordButtons.first()).toBeVisible();
  });

  test('admin should be able to open password reset modal', async () => {
    // Login as admin
    await loginAsAdmin(page);
    
    // Navigate to users page
    await page.goto(BASE_URL + '/admin/users');
    
    // Wait for users to load
    await page.waitForSelector('.users-table, .users-mobile-cards');
    
    // Click the first Reset Password button
    const resetPasswordButtons = page.locator('button:has-text("Reset Password")');
    await resetPasswordButtons.first().click();
    
    // Verify modal is displayed
    await expect(page.locator('.group-modal-backdrop')).toBeVisible();
    await expect(page.locator('.group-modal h2')).toContainText(/Reset Password for/i);
    
    // Verify password input field is present
    await expect(page.locator('input[type="password"]')).toBeVisible();
    
    // Verify buttons are present
    await expect(page.locator('button:has-text("Reset Password")')).toBeVisible();
    await expect(page.locator('button:has-text("Cancel")')).toBeVisible();
  });

  test('password reset modal should validate minimum password length', async () => {
    // Login as admin
    await loginAsAdmin(page);
    
    // Navigate to users page
    await page.goto(BASE_URL + '/admin/users');
    
    // Wait for users to load
    await page.waitForSelector('.users-table, .users-mobile-cards');
    
    // Click the first Reset Password button
    const resetPasswordButtons = page.locator('button:has-text("Reset Password")');
    await resetPasswordButtons.first().click();
    
    // Try to enter a short password
    const passwordInput = page.locator('input[type="password"]');
    await passwordInput.fill('short');
    
    // Try to submit
    await page.locator('button[type="submit"]:has-text("Reset Password")').click();
    
    // The form should have HTML5 validation preventing submission
    // Check that we're still on the modal (not closed)
    await expect(page.locator('.group-modal-backdrop')).toBeVisible();
  });

  test('admin should be able to reset user password and user can login with new password', async ({ page }) => {
    // Login as admin
    await loginAsAdmin(page);
    
    // Navigate to users page
    await page.goto(BASE_URL + '/admin/users');
    
    // Wait for users to load
    await page.waitForSelector('.users-table, .users-mobile-cards');
    
    // Find the test user row and click Reset Password
    // We need to find the row containing testuser
    const userRow = page.locator(`tr:has-text("${TEST_USER.username}"), .user-card:has-text("${TEST_USER.username}")`);
    await expect(userRow).toBeVisible();
    
    // Click Reset Password button for this user
    await userRow.locator('button:has-text("Reset Password")').click();
    
    // Wait for modal to appear
    await expect(page.locator('.group-modal-backdrop')).toBeVisible();
    await expect(page.locator('.group-modal h2')).toContainText(TEST_USER.username);
    
    // Enter new password
    const passwordInput = page.locator('input[type="password"]');
    await passwordInput.fill(NEW_PASSWORD);
    
    // Submit the form
    await page.locator('button[type="submit"]:has-text("Reset Password")').click();
    
    // Wait for success message
    await expect(page.locator('.users-success')).toContainText(/reset.*successfully/i);
    
    // Wait for modal to close
    await page.waitForTimeout(2000); // Wait for auto-close after success
    
    // Logout as admin
    await page.goto(BASE_URL);
    const logoutButton = page.locator('button:has-text("Logout"), a:has-text("Logout")');
    if (await logoutButton.isVisible()) {
      await logoutButton.click();
    }
    
    // Try to login as the test user with the NEW password
    await page.goto(BASE_URL + '/login');
    await page.fill('input[name="username"]', TEST_USER.username);
    await page.fill('input[name="password"]', NEW_PASSWORD);
    await page.click('button[type="submit"]');
    
    // Verify successful login - should redirect to dashboard
    await page.waitForURL(/dashboard|\/$/);
    await expect(page.locator('body')).toContainText(/(dashboard|groups|welcome)/i);
    
    // Logout
    await page.goto(BASE_URL);
    const logoutButton2 = page.locator('button:has-text("Logout"), a:has-text("Logout")');
    if (await logoutButton2.isVisible()) {
      await logoutButton2.click();
    }
    
    // Now reset the password back to original for other tests
    await loginAsAdmin(page);
    await page.goto(BASE_URL + '/admin/users');
    await page.waitForSelector('.users-table, .users-mobile-cards');
    
    const userRow2 = page.locator(`tr:has-text("${TEST_USER.username}"), .user-card:has-text("${TEST_USER.username}")`);
    await userRow2.locator('button:has-text("Reset Password")').click();
    await page.locator('input[type="password"]').fill(TEST_USER.password);
    await page.locator('button[type="submit"]:has-text("Reset Password")').click();
    await expect(page.locator('.users-success')).toContainText(/reset.*successfully/i);
  });

  test('should close modal when clicking Cancel button', async () => {
    // Login as admin
    await loginAsAdmin(page);
    
    // Navigate to users page
    await page.goto(BASE_URL + '/admin/users');
    
    // Wait for users to load
    await page.waitForSelector('.users-table, .users-mobile-cards');
    
    // Click Reset Password button
    const resetPasswordButtons = page.locator('button:has-text("Reset Password")');
    await resetPasswordButtons.first().click();
    
    // Verify modal is visible
    await expect(page.locator('.group-modal-backdrop')).toBeVisible();
    
    // Click Cancel button
    await page.locator('button:has-text("Cancel")').click();
    
    // Verify modal is closed
    await expect(page.locator('.group-modal-backdrop')).not.toBeVisible();
  });

  test('should close modal when clicking backdrop', async () => {
    // Login as admin
    await loginAsAdmin(page);
    
    // Navigate to users page
    await page.goto(BASE_URL + '/admin/users');
    
    // Wait for users to load
    await page.waitForSelector('.users-table, .users-mobile-cards');
    
    // Click Reset Password button
    const resetPasswordButtons = page.locator('button:has-text("Reset Password")');
    await resetPasswordButtons.first().click();
    
    // Verify modal is visible
    await expect(page.locator('.group-modal-backdrop')).toBeVisible();
    
    // Click backdrop (outside modal)
    await page.locator('.group-modal-backdrop').click({ position: { x: 10, y: 10 } });
    
    // Verify modal is closed
    await expect(page.locator('.group-modal-backdrop')).not.toBeVisible();
  });

  test('Reset Password button should be disabled for deleted users', async () => {
    // Login as admin
    await loginAsAdmin(page);
    
    // Navigate to users page
    await page.goto(BASE_URL + '/admin/users');
    
    // Click "Show Deleted Users" button
    await page.click('button:has-text("Show Deleted Users")');
    
    // Wait for deleted users to load
    await page.waitForTimeout(1000);
    
    // Check if there are any deleted users
    const userRows = page.locator('.users-table tbody tr, .users-mobile-cards .user-card');
    const count = await userRows.count();
    
    if (count > 0) {
      // Verify Reset Password button is NOT present for deleted users (only Restore button)
      const resetPasswordButtons = page.locator('button:has-text("Reset Password")');
      await expect(resetPasswordButtons).toHaveCount(0);
      
      // Verify Restore button IS present
      const restoreButtons = page.locator('button:has-text("Restore")');
      await expect(restoreButtons.first()).toBeVisible();
    }
  });
});
