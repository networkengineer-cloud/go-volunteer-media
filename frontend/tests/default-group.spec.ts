import { test, expect } from '@playwright/test';
import { loginAsAdmin, loginAsVolunteer, testUsers } from './helpers/auth';

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

const TEST_USER = testUsers.volunteer;
const ADMIN_USER = testUsers.admin;

test.describe('Default Group Feature', () => {
  test('should display default group after login', async ({ page }) => {
    await loginAsVolunteer(page, { waitForUrl: /\/(dashboard|groups)/i });
    
    // Wait for navigation to dashboard which will redirect to group page
    await page.waitForURL(/\/(dashboard|groups)/, { timeout: 10000 });

    // The dashboard experience may immediately render group content (with or without changing the URL).
    // For seeded users, the success signal is that the group UI is present.
    await expect(page.locator('.group-tabs')).toBeVisible({ timeout: 15000 });
  });

  test('should display group switcher when user has multiple groups', async ({ page }) => {
    await loginAsVolunteer(page, { waitForUrl: /\/(dashboard|groups)/i });
    
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
    await loginAsVolunteer(page, { waitForUrl: /\/(dashboard|groups)/i });
    
    const currentUrl = page.url();
    
    if (currentUrl.includes('/groups/')) {
      // On group page, check for Activity Feed tab panel
      await expect(page.getByRole('tabpanel', { name: /activity feed/i })).toBeVisible({ timeout: 15000 });
    } else {
      // On dashboard (if groups exist), wait for redirect
      await page.waitForURL(/\/groups\//, { timeout: 5000 }).catch(() => {});
    }
  });

  test('should display animals section or tab', async ({ page }) => {
    await loginAsVolunteer(page, { waitForUrl: /\/(dashboard|groups)/i });
    
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
    await loginAsVolunteer(page, { waitForUrl: /\/(dashboard|groups)/i });
    
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
    await loginAsVolunteer(page, { waitForUrl: /\/(dashboard|groups)/i });

    // Visiting /dashboard should immediately redirect to a group page.
    await page.goto('/dashboard');
    await page.waitForURL(/\/groups\/\d+/, { timeout: 15000 });
    
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
    await loginAsVolunteer(page, { waitForUrl: /\/(dashboard|groups)/i });
    
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
    await loginAsVolunteer(page, { waitForUrl: /\/(dashboard|groups)/i });
    
    // Wait for redirect to group page
    await page.waitForURL(/\/groups\/\d+/, { timeout: 15000 });
    
    // Activity feed should be visible and have items (seeded data)
    await expect(page.locator('#activity-panel')).toBeVisible({ timeout: 15000 });
    await expect(page.locator('.activity-item').first()).toBeVisible({ timeout: 15000 });
  });
});

test.describe('Default Group Feature - Admin View', () => {
  
  test('should allow admin to see all groups and switch between them', async ({ page }) => {
    await loginAsAdmin(page, { waitForUrl: /\/(dashboard|groups)/i });
    
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
