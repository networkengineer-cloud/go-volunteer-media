import { test, expect } from '@playwright/test';
import { loginAsAdmin } from './helpers/auth';

/**
 * Protocol Document Modal UX Tests
 * 
 * Tests the modal dialog implementation for viewing protocol documents
 * with proper authentication and user experience.
 * 
 * Note: These tests skip if no animals with protocol documents exist.
 */

test.describe('Protocol Document Modal UX', () => {
  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page);
  });

  test('should open protocol document in modal instead of new tab', async ({ page }) => {
    // Try to find an animal detail page - navigate to dashboard first
    await page.goto('/dashboard');
    
    // Try to find any animal link
    const animalLinks = page.locator('a[href*="/groups/"][href*="/animals/"]').first();
    const linkCount = await animalLinks.count();
    
    // Skip if no animals exist
    if (linkCount === 0) {
      test.skip();
      return;
    }
    
    // Click the first animal link
    await animalLinks.click();
    await page.waitForSelector('.animal-detail-page', { timeout: 10000 });

    // Check if protocol document section exists
    const hasProtocol = await page.locator('.protocol-document-section').count();
    
    // Skip test if no protocol document
    if (hasProtocol === 0) {
      test.skip();
      return;
    }
    
    // Click the view protocol button
    await page.click('.btn-view-document');

    // Modal should appear
    await expect(page.locator('.protocol-modal-overlay')).toBeVisible({ timeout: 5000 });
    
    // Modal content should be visible
    await expect(page.locator('.protocol-modal-content')).toBeVisible();
    
    // Modal should have proper ARIA attributes
    await expect(page.locator('.protocol-modal-overlay')).toHaveAttribute('role', 'dialog');
    await expect(page.locator('.protocol-modal-overlay')).toHaveAttribute('aria-modal', 'true');
    
    // Modal header should show the title
    await expect(page.locator('#protocol-modal-title')).toHaveText(/Protocol Document/);
    
    // Should show loading state initially or iframe
    await page.waitForSelector('.protocol-iframe, .protocol-loading', { timeout: 5000 });
  });

  test('should close modal on Escape key press', async ({ page }) => {
    await page.goto('/dashboard');
    const animalLinks = page.locator('a[href*="/groups/"][href*="/animals/"]').first();
    if (await animalLinks.count() === 0) {
      test.skip();
      return;
    }
    
    await animalLinks.click();
    await page.waitForSelector('.animal-detail-page', { timeout: 10000 });
    
    const hasProtocol = await page.locator('.protocol-document-section').count();
    if (hasProtocol === 0) {
      test.skip();
      return;
    }
    
    // Open modal
    await page.click('.btn-view-document');
    await expect(page.locator('.protocol-modal-overlay')).toBeVisible();

    // Press Escape key
    await page.keyboard.press('Escape');

    // Modal should be closed
    await expect(page.locator('.protocol-modal-overlay')).not.toBeVisible();
  });

  test('should close modal when clicking outside', async ({ page }) => {
    await page.goto('/dashboard');
    const animalLinks = page.locator('a[href*="/groups/"][href*="/animals/"]').first();
    if (await animalLinks.count() === 0) {
      test.skip();
      return;
    }
    
    await animalLinks.click();
    await page.waitForSelector('.animal-detail-page', { timeout: 10000 });
    
    const hasProtocol = await page.locator('.protocol-document-section').count();
    if (hasProtocol === 0) {
      test.skip();
      return;
    }
    
    // Open modal
    await page.click('.btn-view-document');
    await expect(page.locator('.protocol-modal-overlay')).toBeVisible();

    // Click on overlay (outside modal content)
    await page.locator('.protocol-modal-overlay').click({ position: { x: 10, y: 10 } });

    // Modal should be closed
    await expect(page.locator('.protocol-modal-overlay')).not.toBeVisible();
  });

  test('should close modal when clicking close button', async ({ page }) => {
    await page.goto('/dashboard');
    const animalLinks = page.locator('a[href*="/groups/"][href*="/animals/"]').first();
    if (await animalLinks.count() === 0) {
      test.skip();
      return;
    }
    
    await animalLinks.click();
    await page.waitForSelector('.animal-detail-page', { timeout: 10000 });
    
    const hasProtocol = await page.locator('.protocol-document-section').count();
    if (hasProtocol === 0) {
      test.skip();
      return;
    }
    
    // Open modal
    await page.click('.btn-view-document');
    await expect(page.locator('.protocol-modal-overlay')).toBeVisible();

    // Click close button
    await page.click('.protocol-modal-close');

    // Modal should be closed
    await expect(page.locator('.protocol-modal-overlay')).not.toBeVisible();
  });

  test('should have accessible close button with proper attributes', async ({ page }) => {
    await page.goto('/dashboard');
    const animalLinks = page.locator('a[href*="/groups/"][href*="/animals/"]').first();
    if (await animalLinks.count() === 0) {
      test.skip();
      return;
    }
    
    await animalLinks.click();
    await page.waitForSelector('.animal-detail-page', { timeout: 10000 });
    
    const hasProtocol = await page.locator('.protocol-document-section').count();
    if (hasProtocol === 0) {
      test.skip();
      return;
    }
    
    // Open modal
    await page.click('.btn-view-document');
    await expect(page.locator('.protocol-modal-overlay')).toBeVisible();

    // Close button should have aria-label
    const closeButton = page.locator('.protocol-modal-close');
    await expect(closeButton).toHaveAttribute('aria-label', 'Close protocol document');
    
    // Close button should have title
    await expect(closeButton).toHaveAttribute('title', 'Close (Esc)');
  });

  test('should not leave page context when viewing protocol', async ({ page, context }) => {
    await page.goto('/dashboard');
    const animalLinks = page.locator('a[href*="/groups/"][href*="/animals/"]').first();
    if (await animalLinks.count() === 0) {
      test.skip();
      return;
    }
    
    await animalLinks.click();
    await page.waitForSelector('.animal-detail-page', { timeout: 10000 });
    
    const hasProtocol = await page.locator('.protocol-document-section').count();
    if (hasProtocol === 0) {
      test.skip();
      return;
    }
    
    // Track if new tab/page is created
    const pagesBefore = context.pages().length;
    const urlBefore = page.url();

    // Open protocol
    await page.click('.btn-view-document');
    await expect(page.locator('.protocol-modal-overlay')).toBeVisible();

    // Should not create new tab/page
    const pagesAfter = context.pages().length;
    expect(pagesAfter).toBe(pagesBefore);

    // Should still be on the same URL
    expect(page.url()).toBe(urlBefore);
  });

  test('button should have proper styling and be accessible', async ({ page }) => {
    await page.goto('/dashboard');
    const animalLinks = page.locator('a[href*="/groups/"][href*="/animals/"]').first();
    if (await animalLinks.count() === 0) {
      test.skip();
      return;
    }
    
    await animalLinks.click();
    await page.waitForSelector('.animal-detail-page', { timeout: 10000 });
    
    const hasProtocol = await page.locator('.protocol-document-section').count();
    if (hasProtocol === 0) {
      test.skip();
      return;
    }
    
    const button = page.locator('.btn-view-document');
    
    // Button should be visible
    await expect(button).toBeVisible();
    
    // Button should have aria-label
    await expect(button).toHaveAttribute('aria-label', 'View protocol document');
    
    // Button should have title
    await expect(button).toHaveAttribute('title', 'View protocol document');
    
    // Button should be a button element (not a link)
    const tagName = await button.evaluate((el) => el.tagName.toLowerCase());
    expect(tagName).toBe('button');
  });
});
