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
    // Attempt login; if environment is not running backend, skip gracefully
    try {
      await loginAsAdmin(page);
    } catch {
      // If login fails or no token, skip subsequent steps
    }
    const token = await page.evaluate(() => localStorage.getItem('token'));
    if (!token) {
      test.skip();
      return;
    }
  });

  test('should open protocol document in modal instead of new tab', async ({ page, request }) => {
    // Assert auth header and allow request to continue
    await page.route('**/api/documents/*', async (route) => {
      const req = route.request();
      const auth = req.headers()['authorization'];
      // Expect Authorization header to be present and Bearer format
      expect(auth, 'Authorization header missing on protocol fetch').toMatch(/^Bearer\s.+/);
      await route.continue();
    });

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
    const firstLinkHref = await animalLinks.getAttribute('href');
    await animalLinks.click();
    await page.waitForSelector('.animal-detail-page', { timeout: 10000 });

    // Check if protocol document section exists
    const hasProtocol = await page.locator('.protocol-document-section').count();
    
    // If no protocol document, upload a small fixture PDF via authenticated API and reload
    if (hasProtocol === 0 && firstLinkHref) {
      const token = await page.evaluate(() => localStorage.getItem('token'));
      const match = firstLinkHref.match(/\/groups\/(\d+)\/animals\/(\d+)/);
      if (token && match) {
        const [_, groupId, animalId] = match;
        const resp = await request.post(`/api/groups/${groupId}/animals/${animalId}/protocol-document`, {
          headers: { Authorization: `Bearer ${token}` },
          multipart: {
            document: {
              name: 'protocol.pdf',
              mimeType: 'application/pdf',
              file: 'tests/fixtures/protocol.pdf',
            },
          },
        });
        expect(resp.ok(), `Failed to upload protocol fixture: ${resp.status()}`).toBeTruthy();
        await page.reload();
        await page.waitForSelector('.animal-detail-page', { timeout: 10000 });
      } else {
        test.skip();
        return;
      }
    }
    
    // Click the view protocol button
    await page.click('.btn-view-document');

    // Wait for the protocol document network response and validate headers
    const response = await page.waitForResponse((res) => {
      return res.url().includes('/api/documents/') && res.status() === 200;
    }, { timeout: 10000 });
    const contentType = response.headers()['content-type'] ?? '';
    expect(contentType, 'Unexpected content-type for protocol document')
      .toMatch(/^(application\/pdf|application\/vnd\.openxmlformats-officedocument\.wordprocessingml\.document)/);

    // Modal should appear
    await expect(page.locator('.protocol-modal-overlay')).toBeVisible({ timeout: 5000 });
    
    // Modal content should be visible
    await expect(page.locator('.protocol-modal-content')).toBeVisible();
    
    // Modal should have proper ARIA attributes
    await expect(page.locator('.protocol-modal-overlay')).toHaveAttribute('role', 'dialog');
    await expect(page.locator('.protocol-modal-overlay')).toHaveAttribute('aria-modal', 'true');
    
    // Modal header should show the title
    await expect(page.locator('#protocol-modal-title')).toHaveText(/Protocol Document/);
    
    // Iframe should render (blob URL)
    const iframe = page.locator('.protocol-iframe');
    await expect(iframe).toBeVisible({ timeout: 8000 });
    const iframeSrc = await iframe.evaluate((el) => (el as HTMLIFrameElement).src);
    expect(iframeSrc.startsWith('blob:'), 'Protocol iframe should use blob URL').toBeTruthy();
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
    await expect(page.locator('.protocol-modal-overlay')).toBeVisible({ timeout: 5000 });

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
    await expect(page.locator('.protocol-modal-overlay')).toBeVisible({ timeout: 5000 });

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
    await expect(page.locator('.protocol-modal-overlay')).toBeVisible({ timeout: 5000 });

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
    await expect(page.locator('.protocol-modal-overlay')).toBeVisible({ timeout: 5000 });

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
