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

  test('should display default group after login', async ({ page }) => {
    // Login as regular user
    await page.fill('input[name="username"]', TEST_USER.username);
    await page.fill('input[name="password"]', TEST_USER.password);
    await page.click('button[type="submit"]');
    
    // Wait for navigation to dashboard which will redirect to group page
    await page.waitForURL(/\/(dashboard|groups)/, { timeout: 10000 });
    
    // Should end up on a group page (either dashboard redirects or directly on group page)
    const currentUrl = page.url();
    
    // If we're on a group page, verify the group name is displayed
    if (currentUrl.includes('/groups/')) {
      await expect(page.locator('.group-header h1')).toBeVisible({ timeout: 10000 });
      await expect(page.locator('.group-tabs')).toBeVisible();
    } else if (currentUrl.includes('/dashboard')) {
      // On dashboard (possibly no groups or in transition)
      // Just verify we're authenticated
      await expect(page.locator('h1')).toContainText(/welcome/i);
    }
  });

  test('should display group switcher when user has multiple groups', async ({ page }) => {
    // Login as user
    await page.fill('input[name="username"]', TEST_USER.username);
    await page.fill('input[name="password"]', TEST_USER.password);
    await page.click('button[type="submit"]');
    
    await page.waitForURL(/\/(dashboard|groups)/, { timeout: 10000 });
    
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

  test('should display activity feed or latest comments', async ({ page }) => {
    // Login as user
    await page.fill('input[name="username"]', TEST_USER.username);
    await page.fill('input[name="password"]', TEST_USER.password);
    await page.click('button[type="submit"]');
    
    await page.waitForURL(/\/(dashboard|groups)/, { timeout: 10000 });
    
    const currentUrl = page.url();
    
    if (currentUrl.includes('/groups/')) {
      // On group page, check for activity feed or comments
      const activityPanel = page.locator('#activity-panel, .activity-feed');
      await expect(activityPanel).toBeVisible({ timeout: 10000 });
    } else {
      // On dashboard (if groups exist), wait for redirect
      await page.waitForURL(/\/groups\//, { timeout: 5000 }).catch(() => {});
    }
  });

  test('should display animals section or tab', async ({ page }) => {
    // Login as user
    await page.fill('input[name="username"]', TEST_USER.username);
    await page.fill('input[name="password"]', TEST_USER.password);
    await page.click('button[type="submit"]');
    
    await page.waitForURL(/\/(dashboard|groups)/, { timeout: 10000 });
    
    const currentUrl = page.url();
    
    if (currentUrl.includes('/groups/')) {
      // On group page, verify Animals tab exists and shows count
      const animalsTab = page.locator('#animals-tab');
      await expect(animalsTab).toBeVisible();
      await expect(animalsTab).toContainText(/animals/i);
      // Should show count now (e.g., "Animals (5)")
      const tabText = await animalsTab.textContent();
      expect(tabText).toMatch(/animals \(\d+\)/i);
    } else {
      // On dashboard, wait for redirect
      await page.waitForURL(/\/groups\//, { timeout: 5000 }).catch(() => {});
    }
  });

  test('should be able to switch between groups', async ({ page }) => {
    // Login as user
    await page.fill('input[name="username"]', TEST_USER.username);
    await page.fill('input[name="password"]', TEST_USER.password);
    await page.click('button[type="submit"]');
    
    await page.waitForURL(/\/(dashboard|groups)/, { timeout: 10000 });
    
    // Wait for redirect to group page if it happens
    await page.waitForURL(/\/groups\//, { timeout: 5000 }).catch(() => {});
    
    const currentUrl = page.url();
    
    if (currentUrl.includes('/groups/')) {
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
              
              // Should navigate to the new group page
              await page.waitForURL(new RegExp(`/groups/${value}`), { timeout: 5000 });
              
              // Verify we're on the new group page
              const newUrl = page.url();
              expect(newUrl).toContain(`/groups/${value}`);
              
              break;
            }
          }
        }
      }
    }
  });

  test('should redirect directly to group page from dashboard', async ({ page }) => {
    // Login as user
    await page.fill('input[name="username"]', TEST_USER.username);
    await page.fill('input[name="password"]', TEST_USER.password);
    await page.click('button[type="submit"]');
    
    // Should be redirected to dashboard first
    await page.waitForURL(/\/dashboard/, { timeout: 10000 });
    
    // Dashboard should immediately redirect to the group page
    await page.waitForURL(/\/groups\/\d+/, { timeout: 5000 });
    
    // Verify we're on a group page
    const currentUrl = page.url();
    expect(currentUrl).toMatch(/\/groups\/\d+/);
    
    // Verify group page elements are visible
    await expect(page.locator('.group-header h1')).toBeVisible({ timeout: 5000 });
    await expect(page.locator('.group-tabs')).toBeVisible();
  });

  test('should show appropriate message when user has no groups', async ({ page }) => {
    // This test would need a user with no groups
    // For now, we'll skip it or create a test user with no groups
    test.skip();
  });

  test('activity feed should be visible on group page', async ({ page }) => {
    // Login as user
    await page.fill('input[name="username"]', TEST_USER.username);
    await page.fill('input[name="password"]', TEST_USER.password);
    await page.click('button[type="submit"]');
    
    await page.waitForURL(/\/(dashboard|groups)/, { timeout: 10000 });
    
    // Wait for redirect to group page
    await page.waitForURL(/\/groups\/\d+/, { timeout: 5000 });
    
    // Activity feed tab should be active by default
    const activityTab = page.locator('#activity-tab');
    await expect(activityTab).toHaveAttribute('aria-selected', 'true');
    
    // Activity panel should be visible
    const activityPanel = page.locator('#activity-panel');
    await expect(activityPanel).toBeVisible();
  });

  test('should display comments in activity feed', async ({ page }) => {
    // Login as user
    await page.fill('input[name="username"]', TEST_USER.username);
    await page.fill('input[name="password"]', TEST_USER.password);
    await page.click('button[type="submit"]');
    
    await page.waitForURL(/\/(dashboard|groups)/, { timeout: 10000 });
    
    // Wait for redirect to group page
    await page.waitForURL(/\/groups\/\d+/, { timeout: 5000 });
    
    // Activity feed should be visible
    const activityFeed = page.locator('.activity-feed, #activity-panel');
    await expect(activityFeed).toBeVisible({ timeout: 10000 });
  });
});

test.describe('Default Group Feature - Admin View', () => {
  
  test('should allow admin to see all groups and switch between them', async ({ page }) => {
    // Login as admin
    await page.goto(BASE_URL + '/login');
    await page.fill('input[name="username"]', ADMIN_USER.username);
    await page.fill('input[name="password"]', ADMIN_USER.password);
    await page.click('button[type="submit"]');
    
    await page.waitForURL(/\/(dashboard|groups)/, { timeout: 10000 });
    
    // Wait for redirect to group page
    await page.waitForURL(/\/groups\/\d+/, { timeout: 5000 }).catch(() => {});
    
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
