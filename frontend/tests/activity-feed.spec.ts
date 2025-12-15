import { test, expect, type Page } from '@playwright/test';
import { loginAsAdmin, loginAsGroupAdmin } from './helpers/auth';

test.describe('Activity Feed UX', () => {
  test.describe.configure({ mode: 'serial' });

  const MODSQUAD_GROUP_NAME = 'modsquad';
  const SANDBOX_GROUP_NAME = 'activity-sandbox';

  const getGroupIdByName = async (page: Page, groupName: string) => {
    const groupId = await page.evaluate(async (targetName) => {
      const token = localStorage.getItem('token');
      const res = await fetch('/api/groups', {
        headers: token ? { Authorization: `Bearer ${token}` } : {},
      });

      if (!res.ok) {
        throw new Error(`Failed to fetch groups: ${res.status}`);
      }

      const groups = await res.json();
      const match = groups.find((g: { name: string }) => g.name.toLowerCase() === targetName.toLowerCase());
      return match?.id ?? null;
    }, groupName);

    if (!groupId) {
      throw new Error(`Group not found: ${groupName}`);
    }

    return groupId as number;
  };

  // Setup: Login before each test
  test.beforeEach(async ({ page }) => {
    await loginAsGroupAdmin(page);
  });

  test('displays activity feed tabs on group page', async ({ page }) => {
    const modsquadId = await getGroupIdByName(page, MODSQUAD_GROUP_NAME);
    await page.goto(`/groups/${modsquadId}`);
    
    // Check that tabs are visible
    const activityTab = page.getByRole('tab', { name: /^Activity Feed$/ });
    const animalsTab = page.getByRole('tab', { name: /^Animals\s*\(\d+\)$/ });
    await expect(activityTab).toBeVisible();
    await expect(animalsTab).toBeVisible();
    
    // Activity Feed tab should be active by default
    await expect(activityTab).toHaveClass(/group-tab--active/);
  });

  test('switches between activity feed and animals view', async ({ page }) => {
    const modsquadId = await getGroupIdByName(page, MODSQUAD_GROUP_NAME);
    await page.goto(`/groups/${modsquadId}`);
    
    // Click on Animals tab
    const animalsTab = page.getByRole('tab', { name: /^Animals\s*\(\d+\)$/ });
    await animalsTab.click();
    
    // Animals tab should now be active
    await expect(animalsTab).toHaveClass(/group-tab--active/);
    
    // Animals grid should be visible
    await expect(page.locator('.animals-grid')).toBeVisible();
    
    // Switch back to Activity Feed
    await page.getByRole('tab', { name: /^Activity Feed$/ }).click();
    
    // Activity feed should be visible
    await expect(page.locator('.activity-feed')).toBeVisible();
  });

  test('shows quick actions bar on activity feed', async ({ page }) => {
    const modsquadId = await getGroupIdByName(page, MODSQUAD_GROUP_NAME);
    await page.goto(`/groups/${modsquadId}`);
    
    // Quick actions bar should be visible
    await expect(page.locator('.quick-actions-bar')).toBeVisible();
    
    // Filter dropdown should be visible
    await expect(page.locator('#activity-filter')).toBeVisible();

    // Group admins should see an Announcement quick action
    const adminNav = page.getByRole('navigation', { name: 'Group administration links' });
    await expect(adminNav).toBeVisible();
    await expect(adminNav.getByRole('button', { name: /^Announcement$/ })).toBeVisible();
  });

  test('filters activity by type', async ({ page }) => {
    const modsquadId = await getGroupIdByName(page, MODSQUAD_GROUP_NAME);
    await page.goto(`/groups/${modsquadId}`);
    
    // Change filter to comments only
    await page.selectOption('#activity-filter', 'comments');
    
    // Wait for feed to update
    await page.waitForTimeout(1000);
    
    // Change filter to announcements only
    await page.selectOption('#activity-filter', 'announcements');
    
    // Wait for feed to update
    await page.waitForTimeout(1000);
    
    // Change back to all
    await page.selectOption('#activity-filter', 'all');
  });

  test('opens announcement form modal', async ({ page }) => {
    const modsquadId = await getGroupIdByName(page, MODSQUAD_GROUP_NAME);
    await page.goto(`/groups/${modsquadId}`);
    
    // Click Announcement quick action
    const adminNav = page.getByRole('navigation', { name: 'Group administration links' });
    await expect(adminNav).toBeVisible();
    await adminNav.getByRole('button', { name: /^Announcement$/ }).click();
    
    // Modal should open
    await expect(page.getByRole('dialog', { name: 'Post Announcement' })).toBeVisible();
    
    // Form fields should be visible
    await expect(page.locator('#announcement-title')).toBeVisible();
    await expect(page.locator('#announcement-content')).toBeVisible();
    
    // Close modal
    await page.getByRole('button', { name: /^Cancel$/ }).click();
    await expect(page.locator('.modal')).not.toBeVisible();
  });

  test('displays activity items correctly', async ({ page }) => {
    const modsquadId = await getGroupIdByName(page, MODSQUAD_GROUP_NAME);
    await page.goto(`/groups/${modsquadId}`);
    
    // Wait for activity items to load
    await page.waitForSelector('.activity-item', { timeout: 10000 });
    
    // At least one activity item should be visible
    const activityItems = page.locator('.activity-item');
    await expect(activityItems.first()).toBeVisible();
    
    // Activity item should have required elements
    await expect(activityItems.first().locator('.activity-item__username')).toBeVisible();
    await expect(activityItems.first().locator('.activity-item__timestamp')).toBeVisible();
    await expect(activityItems.first().locator('.activity-item__text')).toBeVisible();
  });

  test('supports keyboard navigation', async ({ page }) => {
    const modsquadId = await getGroupIdByName(page, MODSQUAD_GROUP_NAME);
    await page.goto(`/groups/${modsquadId}`);

    const activityTab = page.getByRole('tab', { name: /^Activity Feed$/ });
    const animalsTab = page.getByRole('tab', { name: /^Animals\s*\(\d+\)$/ });

    // On mobile-safari, Tab navigation can be inconsistent; verify keyboard activation works
    // once a tab is focused.
    await animalsTab.focus();
    await expect(animalsTab).toBeFocused();

    await page.keyboard.press('Enter');
    await expect(animalsTab).toHaveClass(/group-tab--active/);
  });

  test('is responsive on mobile', async ({ page }) => {
    // Set mobile viewport
    await page.setViewportSize({ width: 375, height: 667 });
    const modsquadId = await getGroupIdByName(page, MODSQUAD_GROUP_NAME);
    await page.goto(`/groups/${modsquadId}`);
    
    // Tabs should still be visible
    await expect(page.getByRole('tab', { name: /^Activity Feed$/ })).toBeVisible();
    await expect(page.getByRole('tab', { name: /^Animals\s*\(\d+\)$/ })).toBeVisible();
    
    // Quick actions should stack vertically
    const quickActions = page.locator('.quick-actions-bar');
    await expect(quickActions).toBeVisible();
  });

  test('shows empty state when no activity', async ({ page }) => {
    await loginAsAdmin(page);
    const sandboxId = await getGroupIdByName(page, SANDBOX_GROUP_NAME);
    await page.goto(`/groups/${sandboxId}`);
    await expect(page.locator('.empty-state')).toBeVisible();
  });

  test('supports infinite scroll loading', async ({ page }) => {
    const modsquadId = await getGroupIdByName(page, MODSQUAD_GROUP_NAME);
    await page.goto(`/groups/${modsquadId}`);

    // Wait for initial items to load
    await page.waitForSelector('.activity-feed__list', { timeout: 10000 });
    const items = page.locator('.activity-item');
    await expect(items.first()).toBeVisible();
    const initialCount = await items.count();

    const loadTrigger = page.locator('.activity-feed__load-trigger');
    const endOfFeed = page.locator('.activity-feed__end-message');

    if (await loadTrigger.count()) {
      await loadTrigger.scrollIntoViewIfNeeded();
      await expect
        .poll(async () => {
          const currentCount = await items.count();
          const endVisible = await endOfFeed.isVisible().catch(() => false);
          return currentCount > initialCount || endVisible;
        }, { timeout: 15000 })
        .toBe(true);
    } else {
      // No load trigger means the feed is already finite; end-of-feed should be present.
      await endOfFeed.scrollIntoViewIfNeeded();
      await expect(endOfFeed).toBeVisible({ timeout: 15000 });
    }
  });
});
