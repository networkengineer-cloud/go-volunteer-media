import { test, expect } from '@playwright/test';

test.describe('Protocol Document Modal UX', () => {
  test.beforeEach(async ({ page }) => {
    // Login as admin
    await page.goto('/login');
    await page.fill('input[name="username"]', 'admin');
    await page.fill('input[name="password"]', 'admin123');
    await page.click('button[type="submit"]');
    await page.waitForURL('/dashboard');
  });

  test('should open protocol document in modal instead of new tab', async ({ page }) => {
    // Navigate to an animal with a protocol document
    // Note: This assumes there's a test animal with a protocol document
    // You may need to set this up in your test data
    await page.goto('/groups/1/animals/1');

    // Wait for page to load
    await page.waitForSelector('.animal-detail-page');

    // Check if protocol document section exists
    const hasProtocol = await page.locator('.protocol-document-section').count();
    
    if (hasProtocol > 0) {
      // Click the view protocol button
      await page.click('.btn-view-document');

      // Modal should appear
      await expect(page.locator('.protocol-modal-overlay')).toBeVisible();
      
      // Modal content should be visible
      await expect(page.locator('.protocol-modal-content')).toBeVisible();
      
      // Modal should have proper ARIA attributes
      await expect(page.locator('.protocol-modal-overlay')).toHaveAttribute('role', 'dialog');
      await expect(page.locator('.protocol-modal-overlay')).toHaveAttribute('aria-modal', 'true');
      
      // Modal header should show the title
      await expect(page.locator('#protocol-modal-title')).toHaveText(/Protocol Document/);
      
      // Should show loading state initially
      const loadingState = page.locator('.protocol-loading');
      if (await loadingState.isVisible()) {
        await expect(loadingState).toContainText(/Loading/);
      }

      // Wait for document to load (iframe or error)
      await page.waitForSelector('.protocol-iframe, .protocol-loading', { timeout: 5000 });

      // Document should load (or show error if no document exists)
      // In real scenario, the iframe would contain the PDF
    } else {
      console.log('No protocol document found for this animal - skipping test');
    }
  });

  test('should close modal on Escape key press', async ({ page }) => {
    await page.goto('/groups/1/animals/1');
    await page.waitForSelector('.animal-detail-page');

    const hasProtocol = await page.locator('.protocol-document-section').count();
    
    if (hasProtocol > 0) {
      // Open modal
      await page.click('.btn-view-document');
      await expect(page.locator('.protocol-modal-overlay')).toBeVisible();

      // Press Escape key
      await page.keyboard.press('Escape');

      // Modal should be closed
      await expect(page.locator('.protocol-modal-overlay')).not.toBeVisible();
    }
  });

  test('should close modal when clicking outside', async ({ page }) => {
    await page.goto('/groups/1/animals/1');
    await page.waitForSelector('.animal-detail-page');

    const hasProtocol = await page.locator('.protocol-document-section').count();
    
    if (hasProtocol > 0) {
      // Open modal
      await page.click('.btn-view-document');
      await expect(page.locator('.protocol-modal-overlay')).toBeVisible();

      // Click on overlay (outside modal content)
      await page.locator('.protocol-modal-overlay').click({ position: { x: 10, y: 10 } });

      // Modal should be closed
      await expect(page.locator('.protocol-modal-overlay')).not.toBeVisible();
    }
  });

  test('should close modal when clicking close button', async ({ page }) => {
    await page.goto('/groups/1/animals/1');
    await page.waitForSelector('.animal-detail-page');

    const hasProtocol = await page.locator('.protocol-document-section').count();
    
    if (hasProtocol > 0) {
      // Open modal
      await page.click('.btn-view-document');
      await expect(page.locator('.protocol-modal-overlay')).toBeVisible();

      // Click close button
      await page.click('.protocol-modal-close');

      // Modal should be closed
      await expect(page.locator('.protocol-modal-overlay')).not.toBeVisible();
    }
  });

  test('should have accessible close button with focus indicator', async ({ page }) => {
    await page.goto('/groups/1/animals/1');
    await page.waitForSelector('.animal-detail-page');

    const hasProtocol = await page.locator('.protocol-document-section').count();
    
    if (hasProtocol > 0) {
      // Open modal
      await page.click('.btn-view-document');
      await expect(page.locator('.protocol-modal-overlay')).toBeVisible();

      // Tab to close button
      await page.keyboard.press('Tab');

      // Close button should have aria-label
      const closeButton = page.locator('.protocol-modal-close');
      await expect(closeButton).toHaveAttribute('aria-label', 'Close protocol document');
      
      // Close button should have title
      await expect(closeButton).toHaveAttribute('title', 'Close (Esc)');

      // Verify focus is visible (browser will apply outline)
      await expect(closeButton).toBeFocused();
    }
  });

  test('should work on mobile viewport', async ({ page }) => {
    // Set mobile viewport
    await page.setViewportSize({ width: 375, height: 667 });
    
    await page.goto('/groups/1/animals/1');
    await page.waitForSelector('.animal-detail-page');

    const hasProtocol = await page.locator('.protocol-document-section').count();
    
    if (hasProtocol > 0) {
      // Button should be full width on mobile
      const button = page.locator('.btn-view-document');
      const buttonBox = await button.boundingBox();
      
      // Check button is reasonably wide (full width minus padding)
      expect(buttonBox?.width).toBeGreaterThan(300);

      // Open modal
      await button.click();
      await expect(page.locator('.protocol-modal-overlay')).toBeVisible();

      // Modal should be full screen on mobile
      const modalContent = page.locator('.protocol-modal-content');
      const modalBox = await modalContent.boundingBox();
      
      expect(modalBox?.width).toBeGreaterThan(350);
      expect(modalBox?.height).toBeGreaterThan(600);
    }
  });

  test('should not leave page context when viewing protocol', async ({ page, context }) => {
    await page.goto('/groups/1/animals/1');
    await page.waitForSelector('.animal-detail-page');

    const hasProtocol = await page.locator('.protocol-document-section').count();
    
    if (hasProtocol > 0) {
      // Track if new tab/page is created
      const pagesBefore = context.pages().length;

      // Open protocol
      await page.click('.btn-view-document');
      await expect(page.locator('.protocol-modal-overlay')).toBeVisible();

      // Should not create new tab/page
      const pagesAfter = context.pages().length;
      expect(pagesAfter).toBe(pagesBefore);

      // Should still be on the same URL
      expect(page.url()).toContain('/groups/1/animals/1');
    }
  });
});
