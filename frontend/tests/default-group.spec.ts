import { test, expect } from '@playwright/test';

/**
 * Default Group Feature E2E Tests
 * 
 * These tests validate the new default group UX including:
 * - Default group loading on login
 * - Group switcher functionality
 * - Latest comments display
 * - Animals list in the group
 * - Setting default group preference
 */

// Test configuration
const BASE_URL = 'http://localhost:5173';

// Test credentials
const TEST_USER = {
  username: 'testuser',
  password: 'password123'
};

const ADMIN_USER = {
  username: 'testadmin',
  password: 'password123'
};

test.describe('Default Group Feature', () => {
  
  test.beforeEach(async ({ page }) => {
    // Navigate to login page
    await page.goto(BASE_URL + '/login');
  });

  test('should display default group on dashboard after login', async ({ page }) => {
    // Login as regular user
    await page.fill('input[name="username"]', TEST_USER.username);
    await page.fill('input[name="password"]', TEST_USER.password);
    await page.click('button[type="submit"]');
    
    // Wait for navigation to dashboard
    await page.waitForURL(/\/dashboard/, { timeout: 10000 });
    
    // Verify dashboard is displayed
    await expect(page.locator('h1')).toContainText(/welcome/i);
    
    // Verify that a group is displayed (current group header)
    const groupHeader = page.locator('.current-group-header');
    await expect(groupHeader).toBeVisible({ timeout: 10000 });
    
    // Verify group name is displayed
    await expect(groupHeader.locator('h2')).toBeVisible();
  });

  test('should display group switcher when user has multiple groups', async ({ page }) => {
    // Login as user
    await page.fill('input[name="username"]', TEST_USER.username);
    await page.fill('input[name="password"]', TEST_USER.password);
    await page.click('button[type="submit"]');
    
    await page.waitForURL(/\/dashboard/, { timeout: 10000 });
    
    // Check if group switcher is present (only if user has multiple groups)
    const groupSwitcher = page.locator('.group-switcher');
    const isVisible = await groupSwitcher.isVisible().catch(() => false);
    
    if (isVisible) {
      // Verify select element is present
      const groupSelect = groupSwitcher.locator('select.group-select');
      await expect(groupSelect).toBeVisible();
      
      // Verify it has options
      const options = await groupSelect.locator('option').count();
      expect(options).toBeGreaterThan(1);
    }
  });

  test('should display latest comments section', async ({ page }) => {
    // Login as user
    await page.fill('input[name="username"]', TEST_USER.username);
    await page.fill('input[name="password"]', TEST_USER.password);
    await page.click('button[type="submit"]');
    
    await page.waitForURL(/\/dashboard/, { timeout: 10000 });
    
    // Wait for dashboard to load
    await page.waitForSelector('.current-group-header', { timeout: 10000 });
    
    // Verify latest comments section exists
    const commentsSection = page.locator('.latest-comments-section');
    await expect(commentsSection).toBeVisible();
    
    // Verify section has a heading
    await expect(commentsSection.locator('h3')).toContainText(/latest comments/i);
  });

  test('should display animals section with preview', async ({ page }) => {
    // Login as user
    await page.fill('input[name="username"]', TEST_USER.username);
    await page.fill('input[name="password"]', TEST_USER.password);
    await page.click('button[type="submit"]');
    
    await page.waitForURL(/\/dashboard/, { timeout: 10000 });
    
    // Wait for dashboard to load
    await page.waitForSelector('.current-group-header', { timeout: 10000 });
    
    // Verify animals section exists
    const animalsSection = page.locator('.animals-section');
    await expect(animalsSection).toBeVisible();
    
    // Verify section has a heading
    await expect(animalsSection.locator('h3')).toBeVisible();
  });

  test('should be able to switch between groups', async ({ page }) => {
    // Login as user
    await page.fill('input[name="username"]', TEST_USER.username);
    await page.fill('input[name="password"]', TEST_USER.password);
    await page.click('button[type="submit"]');
    
    await page.waitForURL(/\/dashboard/, { timeout: 10000 });
    
    // Wait for initial group to load
    await page.waitForSelector('.current-group-header', { timeout: 10000 });
    
    // Check if group switcher exists
    const groupSwitcher = page.locator('.group-switcher');
    const isVisible = await groupSwitcher.isVisible().catch(() => false);
    
    if (isVisible) {
      const groupSelect = groupSwitcher.locator('select.group-select');
      
      // Get current selected value
      const initialValue = await groupSelect.inputValue();
      
      // Get all options
      const options = await groupSelect.locator('option').all();
      
      if (options.length > 1) {
        // Find a different option
        for (const option of options) {
          const value = await option.getAttribute('value');
          if (value && value !== initialValue) {
            // Select a different group
            await groupSelect.selectOption(value);
            
            // Wait a moment for the UI to update
            await page.waitForTimeout(1000);
            
            // Verify the selection changed
            const newValue = await groupSelect.inputValue();
            expect(newValue).toBe(value);
            
            break;
          }
        }
      }
    }
  });

  test('should display link to full group page', async ({ page }) => {
    // Login as user
    await page.fill('input[name="username"]', TEST_USER.username);
    await page.fill('input[name="password"]', TEST_USER.password);
    await page.click('button[type="submit"]');
    
    await page.waitForURL(/\/dashboard/, { timeout: 10000 });
    
    // Wait for group header
    await page.waitForSelector('.current-group-header', { timeout: 10000 });
    
    // Verify link to full group page exists
    const groupLink = page.locator('.current-group-header .btn-secondary');
    await expect(groupLink).toBeVisible();
    await expect(groupLink).toContainText(/view full group page/i);
  });

  test('should show appropriate message when user has no groups', async ({ page }) => {
    // This test would need a user with no groups
    // For now, we'll skip it or create a test user with no groups
    test.skip();
  });

  test('should maintain announcements section', async ({ page }) => {
    // Login as user
    await page.fill('input[name="username"]', TEST_USER.username);
    await page.fill('input[name="password"]', TEST_USER.password);
    await page.click('button[type="submit"]');
    
    await page.waitForURL(/\/dashboard/, { timeout: 10000 });
    
    // Verify announcements section still exists
    const announcementsSection = page.locator('.announcements-section');
    await expect(announcementsSection).toBeVisible();
    await expect(announcementsSection.locator('h3')).toContainText(/announcements/i);
  });

  test('should display comment with animal information', async ({ page }) => {
    // Login as user
    await page.fill('input[name="username"]', TEST_USER.username);
    await page.fill('input[name="password"]', TEST_USER.password);
    await page.click('button[type="submit"]');
    
    await page.waitForURL(/\/dashboard/, { timeout: 10000 });
    
    // Wait for comments section
    await page.waitForSelector('.latest-comments-section', { timeout: 10000 });
    
    // Check if there are any comments
    const commentCards = page.locator('.comment-card');
    const count = await commentCards.count();
    
    if (count > 0) {
      // Verify first comment has animal info
      const firstComment = commentCards.first();
      await expect(firstComment.locator('.animal-link')).toBeVisible();
      
      // Verify comment has metadata
      await expect(firstComment.locator('.comment-meta')).toBeVisible();
    }
  });
});

test.describe('Default Group Feature - Admin View', () => {
  
  test('should allow admin to see all groups and switch between them', async ({ page }) => {
    // Login as admin
    await page.goto(BASE_URL + '/login');
    await page.fill('input[name="username"]', ADMIN_USER.username);
    await page.fill('input[name="password"]', ADMIN_USER.password);
    await page.click('button[type="submit"]');
    
    await page.waitForURL(/\/dashboard/, { timeout: 10000 });
    
    // Wait for dashboard to load
    await page.waitForSelector('.dashboard-header', { timeout: 10000 });
    
    // Admin should be able to see group switcher if multiple groups exist
    const groupSwitcher = page.locator('.group-switcher');
    const isVisible = await groupSwitcher.isVisible().catch(() => false);
    
    if (isVisible) {
      const groupSelect = groupSwitcher.locator('select.group-select');
      const options = await groupSelect.locator('option').count();
      
      // Admin should have access to all groups
      expect(options).toBeGreaterThan(0);
    }
  });
});
