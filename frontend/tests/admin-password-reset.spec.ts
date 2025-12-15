import { test, expect, type Page } from '@playwright/test';
import { loginAsAdmin as loginAsAdminHelper, testUsers } from './helpers/auth';

/**
 * Admin Password Reset E2E Tests
 * 
 * These tests validate the admin password reset functionality:
 * - Admin can access the reset password feature
 * - Admin can reset a user's password
 * - User can login with the new password
 * - Password validation is enforced
 */

// Seeded demo credentials
const ADMIN_USER = testUsers.admin;
const TEST_USER = {
  username: 'sophia',
  password: 'demo1234',
  email: 'sophia@demo.local',
};

const NEW_PASSWORD = 'newpassword123';

test.describe('Admin Password Reset', () => {
  test.describe.configure({ mode: 'serial' });

  // Helper function to login as admin
  async function loginAsAdmin(page: Page) {
    await loginAsAdminHelper(page, { waitForUrl: /\/(dashboard|groups)/i });
  }

  async function goToUsersPage(page: Page) {
    // Wait until the authenticated nav is present (avoids route-guard redirects
    // while auth context is still hydrating).
    await expect(page.getByRole('button', { name: /^logout$/i })).toBeVisible({ timeout: 15000 });

    // Prefer SPA navigation via the nav link (avoids a full reload).
    const mobileMenuToggle = page.locator('.mobile-menu-toggle');
    if (await mobileMenuToggle.isVisible().catch(() => false)) {
      await mobileMenuToggle.click();
    }
    await page.getByRole('link', { name: /^users$/i }).click();

    await expect(page).toHaveURL(/\/users\b/i, { timeout: 15000 });
    await expect(page.locator('.users-page')).toBeVisible({ timeout: 15000 });
    await expect(page.locator('.users-grid, .users-error')).toBeVisible({ timeout: 30000 });
  }

  async function logout(page: Page) {
    const mobileMenuToggle = page.locator('.mobile-menu-toggle');
    if (await mobileMenuToggle.isVisible().catch(() => false)) {
      await mobileMenuToggle.click();
    }
    await page.getByRole('button', { name: /^logout$/i }).click();
    await expect(page.getByRole('link', { name: /^login$/i })).toBeVisible({ timeout: 15000 });
    await expect
      .poll(async () => page.evaluate(() => localStorage.getItem('token')), { timeout: 5000 })
      .toBeNull();
  }

  test('admin should see Reset Password button in users list', async ({ page }) => {
    // Login as admin
    await loginAsAdmin(page);
    
    // Navigate to users page
    await goToUsersPage(page);
    
    // Verify Reset Password button is visible
    const resetPasswordButtons = page.locator('.users-grid').getByRole('button', { name: /^reset password$/i });
    await expect(resetPasswordButtons.first()).toBeVisible();
  });

  test('admin should be able to open password reset modal', async ({ page }) => {
    // Login as admin
    await loginAsAdmin(page);
    
    // Navigate to users page
    await goToUsersPage(page);
    
    // Click the first Reset Password button
    const resetPasswordButtons = page.locator('.users-grid').getByRole('button', { name: /^reset password$/i });
    await resetPasswordButtons.first().click();
    
    // Verify modal is displayed
    await expect(page.locator('.group-modal-backdrop')).toBeVisible();
    await expect(page.locator('.group-modal h2')).toContainText(/Reset Password for/i);

    const modal = page.locator('.group-modal');
    await expect(modal).toBeVisible();
    
    // Verify password input field is present
    await expect(page.getByLabel(/^new password$/i)).toBeVisible();
    
    // Verify buttons are present
    await expect(modal.getByRole('button', { name: /^reset password$/i })).toBeVisible();
    await expect(modal.getByRole('button', { name: /^cancel$/i })).toBeVisible();
  });

  test('password reset modal should validate minimum password length', async ({ page }) => {
    // Login as admin
    await loginAsAdmin(page);
    
    // Navigate to users page
    await goToUsersPage(page);
    
    // Click the first Reset Password button
    const resetPasswordButtons = page.locator('.users-grid').getByRole('button', { name: /^reset password$/i });
    await resetPasswordButtons.first().click();
    
    // Try to enter a short password
    const passwordInput = page.getByLabel(/^new password$/i);
    await passwordInput.fill('short');
    
    // Try to submit
    await page.locator('.group-modal button[type="submit"]:has-text("Reset Password")').click();
    
    // The form should have HTML5 validation preventing submission
    // Check that we're still on the modal (not closed)
    await expect(page.locator('.group-modal-backdrop')).toBeVisible();
  });

  test('admin should be able to reset user password and user can login with new password', async ({ page }) => {
    // Login as admin
    await loginAsAdmin(page);
    
    // Navigate to users page
    await goToUsersPage(page);
    
    // Find the test user row and click Reset Password
    // We need to find the row containing testuser
    const userRow = page.locator(`.user-card-new:has-text("${TEST_USER.username}")`);
    await expect(userRow).toBeVisible();
    
    // Click Reset Password button for this user
    await userRow.getByRole('button', { name: /^reset password$/i }).click();
    
    // Wait for modal to appear
    await expect(page.locator('.group-modal-backdrop')).toBeVisible();
    await expect(page.locator('.group-modal h2')).toContainText(TEST_USER.username);
    
    // Enter new password
    const passwordInput = page.getByLabel(/^new password$/i);
    await passwordInput.fill(NEW_PASSWORD);
    
    // Submit the form
    await page.locator('.group-modal button[type="submit"]:has-text("Reset Password")').click();
    
    // Wait for success message
    await expect(page.locator('.users-success')).toContainText(/reset.*successfully/i);
    
    // Wait for modal to close
    await page.waitForTimeout(2000); // Wait for auto-close after success
    
    // Logout as admin
    await logout(page);
    
    // Try to login as the test user with the NEW password
    await page.goto('/login');
    await page.getByRole('textbox', { name: /^username/i }).fill(TEST_USER.username);
    await page.getByRole('textbox', { name: /^password/i }).fill(NEW_PASSWORD);
    await page.getByRole('button', { name: /^login$/i }).click();
    
    // Verify successful login - should redirect to dashboard
    await page.waitForURL(/\/(dashboard|groups)/, { timeout: 15000 });
    await expect(page.locator('body')).toContainText(/(dashboard|groups|welcome)/i);
    
    // Logout
    await logout(page);
    
    // Now reset the password back to original for other tests
    await loginAsAdmin(page);
    await goToUsersPage(page);
    
    const userRow2 = page.locator(`.user-card-new:has-text("${TEST_USER.username}")`);
    await userRow2.getByRole('button', { name: /^reset password$/i }).click();
    await page.getByLabel(/^new password$/i).fill(TEST_USER.password);
    await page.locator('.group-modal button[type="submit"]:has-text("Reset Password")').click();
    await expect(page.locator('.users-success')).toContainText(/reset.*successfully/i);
  });

  test('should close modal when clicking Cancel button', async ({ page }) => {
    // Login as admin
    await loginAsAdmin(page);
    
    // Navigate to users page
    await goToUsersPage(page);
    
    // Click Reset Password button
    const resetPasswordButtons = page.locator('.users-grid').getByRole('button', { name: /^reset password$/i });
    await resetPasswordButtons.first().click();
    
    // Verify modal is visible
    await expect(page.locator('.group-modal-backdrop')).toBeVisible();
    
    // Click Cancel button
    await page.locator('.group-modal').getByRole('button', { name: /^cancel$/i }).click();
    
    // Verify modal is closed
    await expect(page.locator('.group-modal-backdrop')).not.toBeVisible();
  });

  test('should close modal when clicking backdrop', async ({ page }) => {
    // Login as admin
    await loginAsAdmin(page);
    
    // Navigate to users page
    await goToUsersPage(page);
    
    // Click Reset Password button
    const resetPasswordButtons = page.locator('.users-grid').getByRole('button', { name: /^reset password$/i });
    await resetPasswordButtons.first().click();
    
    // Verify modal is visible
    await expect(page.locator('.group-modal-backdrop')).toBeVisible();
    
    // Click backdrop (outside modal)
    await page.locator('.group-modal-backdrop').click({ position: { x: 10, y: 10 } });
    
    // Verify modal is closed
    await expect(page.locator('.group-modal-backdrop')).not.toBeVisible();
  });

  test('Reset Password button should be disabled for deleted users', async ({ page }) => {
    // Login as admin
    await loginAsAdmin(page);
    
    // Navigate to users page
    await goToUsersPage(page);
    
    // Click "Show Deleted Users" button
    await page.getByRole('button', { name: /^show deleted users$/i }).click();
    
    // Wait for deleted users to load
    await page.waitForTimeout(1000);
    
    // Check if there are any deleted users
    const userRows = page.locator('.user-card-new');
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
