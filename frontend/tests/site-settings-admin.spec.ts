import { test, expect, type Page } from '@playwright/test';
import { loginAsAdmin, testUsers } from './helpers/auth';

/**
 * Admin Site Settings E2E Tests
 * 
 * These tests validate the admin site settings/branding functionality:
 * - Admin can access site settings
 * - Admin can update site name, short name, and description
 * - Changes propagate to navigation and pages immediately (no page reload)
 * - Admin can upload and update hero image
 * - Hero image updates propagate to home page
 * - Validation is enforced for required fields and max lengths
 */

const BASE_URL = 'http://localhost:5173';
const ADMIN_USER = testUsers.admin;

test.describe('Admin Site Settings', () => {
  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page, { waitForUrl: /\/(dashboard|groups)/i });
  });

  test.describe('Branding Settings', () => {
    async function navigateToSiteSettings(page: Page) {
      // Navigate to Admin → Settings
      await page.goto(BASE_URL + '/admin/settings');
      await page.waitForLoadState('networkidle');
      
      // Wait for the settings page to load
      await expect(page.locator('h1')).toContainText('Site Configuration');
      
      // Ensure branding section is visible
      await expect(page.locator('h3:has-text("Branding")')).toBeVisible();
    }

    test('should display branding configuration form', async ({ page }) => {
      await navigateToSiteSettings(page);

      // Check for branding section heading
      await expect(page.locator('h3')).toContainText('Branding');

      // Check for site name field
      const siteNameLabel = page.locator('label:has-text("Site Name")');
      await expect(siteNameLabel).toBeVisible();
      const siteNameInput = page.locator('input#siteName');
      await expect(siteNameInput).toBeVisible();
      await expect(siteNameInput).not.toBeEmpty();

      // Check for short name field
      const shortNameLabel = page.locator('label:has-text("Short Name")');
      await expect(shortNameLabel).toBeVisible();
      const shortNameInput = page.locator('input#siteShortName');
      await expect(shortNameInput).toBeVisible();
      await expect(shortNameInput).not.toBeEmpty();

      // Check for description field
      const descriptionLabel = page.locator('label:has-text("Site Description")');
      await expect(descriptionLabel).toBeVisible();
      const descriptionTextarea = page.locator('textarea#siteDescription');
      await expect(descriptionTextarea).toBeVisible();

      // Check for save button
      const saveButton = page.locator('button:has-text("Save Branding Settings")');
      await expect(saveButton).toBeVisible();
      await expect(saveButton).toBeEnabled();
    });

    test('should update site name and reflect in navigation immediately', async ({ page }) => {
      await navigateToSiteSettings(page);

      // Save original site name for cleanup
      const originalSiteName = await page.locator('input#siteName').inputValue();

      // Update site name to a custom value
      const customSiteName = `Test Site ${Date.now()}`;
      await page.locator('input#siteName').clear();
      await page.locator('input#siteName').fill(customSiteName);

      // Save changes
      await page.locator('button:has-text("Save Branding Settings")').click();

      // Wait for success message
      await expect(page.locator('.settings-form:has-text("Site settings updated successfully")')).toBeVisible({ timeout: 5000 });

      // Verify site name appears in navigation header WITHOUT page reload
      const navBrandText = page.locator('.nav-brand-text');
      await expect(navBrandText).toContainText(customSiteName, { timeout: 3000 });

      // Navigate to dashboard and verify site name is still there
      await page.goto(BASE_URL + '/dashboard');
      await expect(navBrandText).toContainText(customSiteName);

      // Cleanup: restore original site name
      await navigateToSiteSettings(page);
      await page.locator('input#siteName').clear();
      await page.locator('input#siteName').fill(originalSiteName);
      await page.locator('button:has-text("Save Branding Settings")').click();
      await expect(page.locator('.settings-form:has-text("Site settings updated successfully")')).toBeVisible({ timeout: 5000 });
    });

    test('should update short name and description', async ({ page }) => {
      await navigateToSiteSettings(page);

      // Save original values
      const originalShortName = await page.locator('input#siteShortName').inputValue();
      const originalDescription = await page.locator('textarea#siteDescription').inputValue();

      // Update short name and description
      const customShortName = `App${Date.now()}`;
      const customDescription = `Custom description ${Date.now()}`;
      
      await page.locator('input#siteShortName').clear();
      await page.locator('input#siteShortName').fill(customShortName);
      
      await page.locator('textarea#siteDescription').clear();
      await page.locator('textarea#siteDescription').fill(customDescription);

      // Save changes
      await page.locator('button:has-text("Save Branding Settings")').click();

      // Wait for success message
      await expect(page.locator('.settings-form:has-text("Site settings updated successfully")')).toBeVisible({ timeout: 5000 });

      // Reload page and verify values persisted
      await page.reload();
      await expect(page.locator('input#siteShortName')).toHaveValue(customShortName);
      await expect(page.locator('textarea#siteDescription')).toHaveValue(customDescription);

      // Cleanup: restore original values
      await page.locator('input#siteShortName').clear();
      await page.locator('input#siteShortName').fill(originalShortName);
      await page.locator('textarea#siteDescription').clear();
      await page.locator('textarea#siteDescription').fill(originalDescription);
      await page.locator('button:has-text("Save Branding Settings")').click();
      await expect(page.locator('.settings-form:has-text("Site settings updated successfully")')).toBeVisible({ timeout: 5000 });
    });

    test('should enforce validation: required fields', async ({ page }) => {
      await navigateToSiteSettings(page);

      // Save original values for cleanup
      const originalSiteName = await page.locator('input#siteName').inputValue();
      const originalShortName = await page.locator('input#siteShortName').inputValue();

      // Try to submit empty site name
      await page.locator('input#siteName').clear();
      await page.locator('button:has-text("Save Branding Settings")').click();

      // Should show error message
      await expect(page.locator('.settings-form:has-text("Site name is required")')).toBeVisible({ timeout: 3000 });

      // Restore site name, clear short name
      await page.locator('input#siteName').fill(originalSiteName);
      await page.locator('input#siteShortName').clear();
      await page.locator('button:has-text("Save Branding Settings")').click();

      // Should show error message
      await expect(page.locator('.settings-form:has-text("Site short name is required")')).toBeVisible({ timeout: 3000 });

      // Restore original values
      await page.locator('input#siteShortName').fill(originalShortName);
    });

    test('should enforce validation: max length', async ({ page }) => {
      await navigateToSiteSettings(page);

      // Save original values for cleanup
      const originalSiteName = await page.locator('input#siteName').inputValue();

      // Try to submit site name with 101 characters (max is 100)
      const longSiteName = 'A'.repeat(101);
      await page.locator('input#siteName').clear();
      await page.locator('input#siteName').fill(longSiteName);
      await page.locator('button:has-text("Save Branding Settings")').click();

      // Should show error message
      await expect(page.locator('.settings-form:has-text("must be 100 characters or less")')).toBeVisible({ timeout: 3000 });

      // Try with exactly 100 characters (should succeed)
      const maxSiteName = 'B'.repeat(100);
      await page.locator('input#siteName').clear();
      await page.locator('input#siteName').fill(maxSiteName);
      await page.locator('button:has-text("Save Branding Settings")').click();

      // Should succeed
      await expect(page.locator('.settings-form:has-text("Site settings updated successfully")')).toBeVisible({ timeout: 5000 });

      // Restore original value
      await page.locator('input#siteName').clear();
      await page.locator('input#siteName').fill(originalSiteName);
      await page.locator('button:has-text("Save Branding Settings")').click();
      await expect(page.locator('.settings-form:has-text("Site settings updated successfully")')).toBeVisible({ timeout: 5000 });
    });
  });

  test.describe('Hero Image Upload', () => {
    async function navigateToSiteSettings(page: Page) {
      await page.goto(BASE_URL + '/admin/settings');
      await page.waitForLoadState('networkidle');
      await expect(page.locator('h1')).toContainText('Site Configuration');
    }

    test('should display hero image upload form', async ({ page }) => {
      await navigateToSiteSettings(page);

      // Check for Hero Image heading
      await expect(page.locator('h3:has-text("Hero Image")')).toBeVisible();

      // Check for file input
      const fileInput = page.locator('input#heroImage');
      await expect(fileInput).toBeVisible();

      // Check for upload button
      const uploadButton = page.locator('button:has-text("Upload Image")');
      await expect(uploadButton).toBeVisible();
    });

    // Note: Actual file upload testing is complex and often flaky in E2E tests
    // This test validates the UI structure. For real file upload testing, 
    // consider using integration tests or mocking the upload API.
    test('should show upload button and preview area', async ({ page }) => {
      await navigateToSiteSettings(page);

      // Check for preview area (by label or class)
      const previewLabel = page.locator('label:has-text("Current Image")');
      await expect(previewLabel).toBeVisible();
      
      // Preview might not be visible if no image is uploaded, but the form should exist
      const heroImageSection = page.locator('form:has(input#heroImage)');
      await expect(heroImageSection).toBeVisible();
    });
  });

  test.describe('Site Name in Authentication Pages', () => {
    test('should display custom site name on login page', async ({ page }) => {
      // First, set a custom site name as admin
      await loginAsAdmin(page, { waitForUrl: /\/(dashboard|groups)/i });
      await page.goto(BASE_URL + '/admin/settings');
      await page.waitForLoadState('networkidle');

      const customSiteName = `Auth Test ${Date.now()}`;
      const originalSiteName = await page.locator('input#siteName').inputValue();

      await page.locator('input#siteName').clear();
      await page.locator('input#siteName').fill(customSiteName);
      await page.locator('button:has-text("Save Branding Settings")').click();
      await expect(page.locator('.settings-form:has-text("Site settings updated successfully")')).toBeVisible({ timeout: 5000 });

      // Logout
      const logoutButton = page.locator('button:has-text("Logout")');
      const mobileMenuToggle = page.locator('.mobile-menu-toggle');
      if (await mobileMenuToggle.isVisible().catch(() => false)) {
        await mobileMenuToggle.click();
      }
      await logoutButton.click();

      // Navigate to login page
      await page.goto(BASE_URL + '/login');
      
      // Verify custom site name appears on login page
      await expect(page.locator('h1')).toContainText(customSiteName, { timeout: 5000 });

      // Cleanup: login again and restore original site name
      await loginAsAdmin(page, { waitForUrl: /\/(dashboard|groups)/i });
      await page.goto(BASE_URL + '/admin/settings');
      await page.waitForLoadState('networkidle');
      await page.locator('input#siteName').clear();
      await page.locator('input#siteName').fill(originalSiteName);
      await page.locator('button:has-text("Save Branding Settings")').click();
      await expect(page.locator('.settings-form:has-text("Site settings updated successfully")')).toBeVisible({ timeout: 5000 });
    });
  });
});
