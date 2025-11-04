import { test, expect } from '@playwright/test';

test.describe('Activity Feed UX', () => {
  // Setup: Login before each test
  test.beforeEach(async ({ page }) => {
    // Navigate to login page
    await page.goto('http://localhost:5173/login');
    
    // Login with test credentials (you may need to adjust these)
    await page.fill('input[name="username"]', 'admin');
    await page.fill('input[name="password"]', 'admin');
    await page.click('button[type="submit"]');
    
    // Wait for navigation to dashboard
    await page.waitForURL('**/');
  });

  test('displays activity feed tabs on group page', async () => {
    // Navigate to a group page (assuming group ID 1 exists)
    await page.goto('http://localhost:5173/groups/1');
    
    // Check that tabs are visible
    await expect(page.locator('text=Activity Feed')).toBeVisible();
    await expect(page.locator('text=Animals')).toBeVisible();
    
    // Activity Feed tab should be active by default
    const activityTab = page.locator('button:has-text("Activity Feed")');
    await expect(activityTab).toHaveClass(/group-tab--active/);
  });

  test('switches between activity feed and animals view', async () => {
    await page.goto('http://localhost:5173/groups/1');
    
    // Click on Animals tab
    await page.click('button:has-text("Animals")');
    
    // Animals tab should now be active
    const animalsTab = page.locator('button:has-text("Animals")');
    await expect(animalsTab).toHaveClass(/group-tab--active/);
    
    // Animals grid should be visible
    await expect(page.locator('.animals-grid')).toBeVisible();
    
    // Switch back to Activity Feed
    await page.click('button:has-text("Activity Feed")');
    
    // Activity feed should be visible
    await expect(page.locator('.activity-feed')).toBeVisible();
  });

  test('shows quick actions bar on activity feed', async () => {
    await page.goto('http://localhost:5173/groups/1');
    
    // Quick actions bar should be visible
    await expect(page.locator('.quick-actions-bar')).toBeVisible();
    
    // Filter dropdown should be visible
    await expect(page.locator('#activity-filter')).toBeVisible();
    
    // New Announcement button should be visible
    await expect(page.locator('button:has-text("New Announcement")')).toBeVisible();
  });

  test('filters activity by type', async () => {
    await page.goto('http://localhost:5173/groups/1');
    
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

  test('opens announcement form modal', async () => {
    await page.goto('http://localhost:5173/groups/1');
    
    // Click New Announcement button
    await page.click('button:has-text("New Announcement")');
    
    // Modal should open
    await expect(page.locator('.modal')).toBeVisible();
    await expect(page.locator('text=Post Announcement')).toBeVisible();
    
    // Form fields should be visible
    await expect(page.locator('#announcement-title')).toBeVisible();
    await expect(page.locator('#announcement-content')).toBeVisible();
    
    // Close modal
    await page.click('button:has-text("Cancel")');
    await expect(page.locator('.modal')).not.toBeVisible();
  });

  test('displays activity items correctly', async () => {
    await page.goto('http://localhost:5173/groups/1');
    
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

  test('supports keyboard navigation', async () => {
    await page.goto('http://localhost:5173/groups/1');
    
    // Tab to the first tab button
    await page.keyboard.press('Tab');
    await page.keyboard.press('Tab'); // Skip breadcrumb
    
    // Should focus on Activity Feed tab
    const activityTab = page.locator('button:has-text("Activity Feed")');
    await expect(activityTab).toBeFocused();
    
    // Tab to Animals tab
    await page.keyboard.press('Tab');
    const animalsTab = page.locator('button:has-text("Animals")');
    await expect(animalsTab).toBeFocused();
    
    // Press Enter to activate
    await page.keyboard.press('Enter');
    await expect(animalsTab).toHaveClass(/group-tab--active/);
  });

  test('is responsive on mobile', async () => {
    // Set mobile viewport
    await page.setViewportSize({ width: 375, height: 667 });
    await page.goto('http://localhost:5173/groups/1');
    
    // Tabs should still be visible
    await expect(page.locator('button:has-text("Activity Feed")')).toBeVisible();
    await expect(page.locator('button:has-text("Animals")')).toBeVisible();
    
    // Quick actions should stack vertically
    const quickActions = page.locator('.quick-actions-bar');
    await expect(quickActions).toBeVisible();
  });

  test('shows empty state when no activity', async () => {
    // Navigate to a new group with no activity
    await page.goto('http://localhost:5173/groups/999'); // Assuming this group doesn't exist or has no activity
    
    // Should show empty state or error
    const hasEmptyState = await page.locator('.empty-state').isVisible().catch(() => false);
    const hasError = await page.locator('.error-state').isVisible().catch(() => false);
    
    expect(hasEmptyState || hasError).toBeTruthy();
  });

  test('supports infinite scroll loading', async () => {
    await page.goto('http://localhost:5173/groups/1');
    
    // Wait for initial items to load
    await page.waitForSelector('.activity-item', { timeout: 10000 });
    
    // Scroll to bottom
    await page.evaluate(() => window.scrollTo(0, document.body.scrollHeight));
    
    // Wait a bit for potential loading
    await page.waitForTimeout(2000);
    
    // Check if "Loading more..." or "end of feed" message appears
    const loadingMore = page.locator('text=Loading more...');
    const endOfFeed = page.locator('text=reached the end');
    
    const hasLoadingIndicator = await loadingMore.isVisible().catch(() => false);
    const hasEndMessage = await endOfFeed.isVisible().catch(() => false);
    
    // Either should be visible (loading more or end of feed)
    expect(hasLoadingIndicator || hasEndMessage).toBeTruthy();
  });
});
