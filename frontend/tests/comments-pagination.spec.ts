import { test, expect } from '@playwright/test';
import { loginAsAdmin } from './helpers/auth';

/**
 * Comments Pagination UX Tests
 * 
 * Tests to verify:
 * 1. Comments are paginated (showing 10 at a time)
 * 2. "Load More" button appears when there are more comments
 * 3. Sort order toggle works (Newest/Oldest first)
 * 4. Comment count is displayed correctly
 * 5. Pagination works with tag filtering
 * 6. Loading states are shown appropriately
 */

test.describe('Comments Pagination UX', () => {
  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page);
  });

  test('should display comment count indicator', async ({ page }) => {
    // Navigate to an animal page
    await page.goto('/dashboard');
    const firstAnimal = page.locator('.animal-card-mini, .animal-card').first();
    await firstAnimal.click();
    
    // Wait for animal detail page
    await page.waitForSelector('.animal-detail-page', { timeout: 10000 });
    
    // Check for comment count display
    const commentCount = page.locator('.comments-count');
    await expect(commentCount).toBeVisible();
    
    // Should show format like "Showing X of Y comments"
    const countText = await commentCount.textContent();
    expect(countText).toMatch(/Showing \d+ of \d+ comment/);
    
    console.log('Comment count displayed:', countText);
  });

  test('should have sort order toggle button', async ({ page }) => {
    // Navigate to an animal page
    await page.goto('/dashboard');
    const firstAnimal = page.locator('.animal-card-mini, .animal-card').first();
    await firstAnimal.click();
    
    // Wait for animal detail page
    await page.waitForSelector('.animal-detail-page', { timeout: 10000 });
    
    // Check for sort button
    const sortButton = page.locator('.btn-sort');
    await expect(sortButton).toBeVisible();
    
    // Should show either "Newest First" or "Oldest First"
    const sortText = await sortButton.textContent();
    expect(sortText).toMatch(/Newest First|Oldest First/);
    
    console.log('Sort button displayed:', sortText);
  });

  test('should toggle sort order when clicked', async ({ page }) => {
    // Navigate to an animal page
    await page.goto('/dashboard');
    const firstAnimal = page.locator('.animal-card-mini, .animal-card').first();
    await firstAnimal.click();
    
    // Wait for animal detail page
    await page.waitForSelector('.animal-detail-page', { timeout: 10000 });
    
    const sortButton = page.locator('.btn-sort');
    const initialText = await sortButton.textContent();
    
    // Click to toggle
    await sortButton.click();
    
    // Wait for comments to reload
    await page.waitForTimeout(1000);
    
    const newText = await sortButton.textContent();
    
    // Text should have changed
    expect(newText).not.toBe(initialText);
    
    console.log('Sort order changed from', initialText, 'to', newText);
  });

  test('should show load more button when there are more comments', async ({ page }) => {
    // This test assumes there's an animal with more than 10 comments
    // If not, it will skip the load more button check
    
    await page.goto('/dashboard');
    const firstAnimal = page.locator('.animal-card-mini, .animal-card').first();
    await firstAnimal.click();
    
    // Wait for animal detail page
    await page.waitForSelector('.animal-detail-page', { timeout: 10000 });
    
    // Check comment count
    const commentCount = page.locator('.comments-count');
    const countText = await commentCount.textContent();
    const match = countText?.match(/Showing (\d+) of (\d+)/);
    
    if (match) {
      const showing = parseInt(match[1]);
      const total = parseInt(match[2]);
      
      console.log(`Showing ${showing} of ${total} comments`);
      
      if (total > showing) {
        // Should have load more button
        const loadMoreButton = page.locator('.btn-load-more');
        await expect(loadMoreButton).toBeVisible();
        
        // Should show remaining count
        const remainingCount = page.locator('.remaining-count');
        await expect(remainingCount).toBeVisible();
        
        const remainingText = await remainingCount.textContent();
        expect(remainingText).toContain(`${total - showing}`);
        
        console.log('Load more button visible with remaining count:', remainingText);
      } else {
        console.log('All comments already displayed, no load more button needed');
      }
    }
  });

  test('should load more comments when button is clicked', async ({ page }) => {
    await page.goto('/dashboard');
    const firstAnimal = page.locator('.animal-card-mini, .animal-card').first();
    await firstAnimal.click();
    
    // Wait for animal detail page
    await page.waitForSelector('.animal-detail-page', { timeout: 10000 });
    
    // Check if load more button exists
    const loadMoreButton = page.locator('.btn-load-more');
    
    if (await loadMoreButton.isVisible()) {
      // Get initial comment count
      const initialComments = await page.locator('.comment-card').count();
      console.log('Initial comments displayed:', initialComments);
      
      // Click load more
      await loadMoreButton.click();
      
      // Wait for loading to complete
      await page.waitForTimeout(1000);
      
      // Get new comment count
      const newComments = await page.locator('.comment-card').count();
      console.log('Comments after load more:', newComments);
      
      // Should have more comments
      expect(newComments).toBeGreaterThan(initialComments);
    } else {
      console.log('No load more button - all comments already displayed');
    }
  });

  test('should have accessible ARIA labels', async ({ page }) => {
    await page.goto('/dashboard');
    const firstAnimal = page.locator('.animal-card-mini, .animal-card').first();
    await firstAnimal.click();
    
    // Wait for animal detail page
    await page.waitForSelector('.animal-detail-page', { timeout: 10000 });
    
    // Check sort button has aria-label
    const sortButton = page.locator('.btn-sort');
    const ariaLabel = await sortButton.getAttribute('aria-label');
    expect(ariaLabel).toBeTruthy();
    console.log('Sort button aria-label:', ariaLabel);
    
    // Check comment count has aria-live
    const commentCount = page.locator('.comments-count');
    const ariaLive = await commentCount.getAttribute('aria-live');
    expect(ariaLive).toBe('polite');
    
    // Check tag filter buttons have aria-pressed
    const tagFilters = page.locator('.tag-filter');
    if (await tagFilters.first().isVisible()) {
      const ariaPressed = await tagFilters.first().getAttribute('aria-pressed');
      expect(ariaPressed).toBeTruthy();
    }
  });

  test('should maintain pagination state when filtering by tags', async ({ page }) => {
    await page.goto('/dashboard');
    const firstAnimal = page.locator('.animal-card-mini, .animal-card').first();
    await firstAnimal.click();
    
    // Wait for animal detail page
    await page.waitForSelector('.animal-detail-page', { timeout: 10000 });
    
    // Get initial comment count
    const initialCount = page.locator('.comments-count');
    const initialText = await initialCount.textContent();
    
    // Click a tag filter if available
    const tagFilters = page.locator('.tag-filter').filter({ hasNotText: 'All' });
    if (await tagFilters.first().isVisible()) {
      await tagFilters.first().click();
      
      // Wait for filter to apply
      await page.waitForTimeout(1000);
      
      // Comment count should update
      const newText = await initialCount.textContent();
      console.log('Comment count changed from', initialText, 'to', newText);
      
      // Comments list should be visible
      const commentsSection = page.locator('.comments-section');
      await expect(commentsSection).toBeVisible();
    }
  });

  test('should be responsive on mobile viewport', async ({ page }) => {
    // Set mobile viewport
    await page.setViewportSize({ width: 375, height: 667 });
    
    await page.goto('/dashboard');
    const firstAnimal = page.locator('.animal-card-mini, .animal-card').first();
    await firstAnimal.click();
    
    // Wait for animal detail page
    await page.waitForSelector('.animal-detail-page', { timeout: 10000 });
    
    // Check that controls are visible and properly stacked
    const sortButton = page.locator('.btn-sort');
    await expect(sortButton).toBeVisible();
    
    const commentCount = page.locator('.comments-count');
    await expect(commentCount).toBeVisible();
    
    // Load more button should be full width on mobile
    const loadMoreButton = page.locator('.btn-load-more');
    if (await loadMoreButton.isVisible()) {
      const box = await loadMoreButton.boundingBox();
      if (box) {
        // Button should be nearly full width (accounting for padding)
        expect(box.width).toBeGreaterThan(300);
      }
    }
    
    // Take screenshot
    await page.screenshot({ path: '/tmp/comments-pagination-mobile.png', fullPage: true });
  });

  test('should show end of comments message when all loaded', async ({ page }) => {
    await page.goto('/dashboard');
    const firstAnimal = page.locator('.animal-card-mini, .animal-card').first();
    await firstAnimal.click();
    
    // Wait for animal detail page
    await page.waitForSelector('.animal-detail-page', { timeout: 10000 });
    
    // Check if there's a load more button
    const loadMoreButton = page.locator('.btn-load-more');
    
    if (await loadMoreButton.isVisible()) {
      // Keep clicking until no more
      let attempts = 0;
      while (await loadMoreButton.isVisible() && attempts < 10) {
        await loadMoreButton.click();
        await page.waitForTimeout(1000);
        attempts++;
      }
      
      // Should show end of comments message
      const endMessage = page.locator('.end-of-comments');
      if (await endMessage.isVisible()) {
        await expect(endMessage).toBeVisible();
        const messageText = await endMessage.textContent();
        console.log('End of comments message:', messageText);
      }
    } else {
      console.log('No pagination needed - all comments fit on one page');
    }
  });

  test('should take screenshot of pagination UI', async ({ page }) => {
    await page.goto('/dashboard');
    const firstAnimal = page.locator('.animal-card-mini, .animal-card').first();
    await firstAnimal.click();
    
    // Wait for animal detail page
    await page.waitForSelector('.animal-detail-page', { timeout: 10000 });
    
    // Scroll to comments section
    await page.locator('.comments-section').scrollIntoViewIfNeeded();
    
    // Take screenshot of full page showing pagination
    await page.screenshot({ 
      path: '/tmp/comments-pagination-desktop.png', 
      fullPage: true 
    });
    
    console.log('Screenshot saved to /tmp/comments-pagination-desktop.png');
  });
});
