import { test, expect, type APIRequestContext, type Page } from '@playwright/test';
import { testUsers } from './helpers/auth';
import fs from 'node:fs';

/**
 * Protocol Document Modal UX Tests
 * 
 * Tests the modal dialog implementation for viewing protocol documents
 * with proper authentication and user experience.
 * 
 * Note: These tests skip if no animals with protocol documents exist.
 */

test.describe('Protocol Document Modal UX', () => {
  let adminToken: string;

  interface ApiGroup {
    id: number;
  }

  interface ApiAnimal {
    id: number;
  }

  const getToken = async (page: Page): Promise<string> => {
    const token = await page.evaluate(() => localStorage.getItem('token'));
    expect(token, 'Missing auth token after login').toBeTruthy();
    return token as string;
  };

  const getFirstAnimalIds = async (
    request: APIRequestContext,
    token: string
  ): Promise<{ groupId: number; animalId: number }> => {
    const groupsResp = await request.get('/api/groups', {
      headers: { Authorization: `Bearer ${token}` },
    });
    expect(groupsResp.ok(), `Failed to load groups: ${groupsResp.status()}`).toBeTruthy();

    const groups = (await groupsResp.json()) as ApiGroup[];
    expect(groups.length, 'No groups returned from /api/groups').toBeGreaterThan(0);
    const groupId = groups[0].id;

    const animalsResp = await request.get(`/api/groups/${groupId}/animals?status=all`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    expect(animalsResp.ok(), `Failed to load animals: ${animalsResp.status()}`).toBeTruthy();

    const animals = (await animalsResp.json()) as ApiAnimal[];
    expect(animals.length, `No animals returned for group ${groupId}`).toBeGreaterThan(0);

    return { groupId, animalId: animals[0].id };
  };

  const openAnimalDetailWithProtocol = async (page: Page, request: APIRequestContext) => {
    const token = await getToken(page);
    const { groupId, animalId } = await getFirstAnimalIds(request, token);

    await page.goto(`/groups/${groupId}/animals/${animalId}/view`);
    await page.waitForSelector('.animal-detail-page', { timeout: 10000 });

    // If no protocol document section/button exists, upload a small fixture PDF via authenticated API and reload.
    const hasProtocolSection = (await page.locator('.protocol-document-section').count()) > 0;
    const hasViewButton = (await page.locator('.btn-view-document').count()) > 0;
    if (!hasProtocolSection || !hasViewButton) {
      const fixtureBuffer = fs.readFileSync(new URL('./fixtures/protocol.pdf', import.meta.url));
      const resp = await request.post(`/api/groups/${groupId}/animals/${animalId}/protocol-document`, {
        headers: { Authorization: `Bearer ${token}` },
        multipart: {
          document: {
            name: 'protocol.pdf',
            mimeType: 'application/pdf',
            buffer: fixtureBuffer,
          },
        },
      });
      expect(resp.ok(), `Failed to upload protocol fixture: ${resp.status()}`).toBeTruthy();
      await page.reload();
      await page.waitForSelector('.animal-detail-page', { timeout: 10000 });
    }

    await expect(page.locator('.btn-view-document')).toBeVisible({ timeout: 10000 });
    return { groupId, animalId };
  };

  test.beforeAll(async ({ request }) => {
    const resp = await request.post('/api/login', {
      data: {
        username: testUsers.admin.username,
        password: testUsers.admin.password,
      },
    });
    expect(resp.ok(), `Failed to login via API: ${resp.status()}`).toBeTruthy();
    const json = (await resp.json()) as { token?: string };
    expect(json.token, 'Missing token in /api/login response').toBeTruthy();
    adminToken = json.token as string;
  });

  test.beforeEach(async ({ page }) => {
    await page.addInitScript((token) => {
      localStorage.setItem('token', token);
    }, adminToken);

    // Confirm we're authenticated before running the test.
    await page.goto('/dashboard');
    await expect(page.getByRole('button', { name: /^logout$/i })).toBeVisible({ timeout: 15000 });
    await getToken(page);
  });

  test('should open protocol document in modal instead of new tab', async ({ page, request }) => {
    await openAnimalDetailWithProtocol(page, request);

    // Assert auth header and allow request to continue
    await page.route('**/api/documents/*', async (route) => {
      const req = route.request();
      const auth = req.headers()['authorization'];
      // Expect Authorization header to be present and Bearer format
      expect(auth, 'Authorization header missing on protocol fetch').toMatch(/^Bearer\s.+/);
      await route.continue();
    });
    
    // Click the view protocol button and wait for the protocol document response (avoid click/response race)
    const [response] = await Promise.all([
      page.waitForResponse((res) => {
        return res.url().includes('/api/documents/') && res.status() === 200;
      }, { timeout: 10000 }),
      page.click('.btn-view-document'),
    ]);
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
    
    if (contentType.startsWith('application/pdf')) {
      const body = await response.body();
      expect(body.length, 'Protocol PDF response body should not be empty').toBeGreaterThan(100);
      expect(body.subarray(0, 4).toString('utf8'), 'Protocol PDF should start with %PDF').toBe('%PDF');

      // Iframe should render (blob URL)
      const iframe = page.locator('.protocol-iframe');
      await expect(iframe).toBeVisible({ timeout: 8000 });
      const iframeSrc = await iframe.evaluate((el) => (el as HTMLIFrameElement).src);
      expect(iframeSrc.startsWith('blob:'), 'Protocol iframe should use blob URL').toBeTruthy();

      const box = await iframe.boundingBox();
      expect(box?.width ?? 0, 'Protocol iframe should have a width').toBeGreaterThan(50);
      expect(box?.height ?? 0, 'Protocol iframe should have a height').toBeGreaterThan(50);
    } else {
      // DOCX should render into container (no iframe)
      await expect(page.locator('.protocol-docx-container')).toBeVisible({ timeout: 8000 });
    }
  });

  test('should close modal on Escape key press', async ({ page, request }) => {
    await openAnimalDetailWithProtocol(page, request);

    // Open modal
    await page.click('.btn-view-document');
    await expect(page.locator('.protocol-modal-overlay')).toBeVisible({ timeout: 5000 });

    // Press Escape key
    await page.keyboard.press('Escape');

    // Modal should be closed
    await expect(page.locator('.protocol-modal-overlay')).not.toBeVisible();
  });

  test('should close modal when clicking outside', async ({ page, request }) => {
    await openAnimalDetailWithProtocol(page, request);

    // Open modal
    await page.click('.btn-view-document');
    await expect(page.locator('.protocol-modal-overlay')).toBeVisible({ timeout: 5000 });

    // Click on overlay (outside modal content)
    await page.locator('.protocol-modal-overlay').click({ position: { x: 10, y: 10 } });

    // Modal should be closed
    await expect(page.locator('.protocol-modal-overlay')).not.toBeVisible();
  });

  test('should close modal when clicking close button', async ({ page, request }) => {
    await openAnimalDetailWithProtocol(page, request);

    // Open modal
    await page.click('.btn-view-document');
    await expect(page.locator('.protocol-modal-overlay')).toBeVisible({ timeout: 5000 });

    // Click close button
    await page.click('.protocol-modal-close');

    // Modal should be closed
    await expect(page.locator('.protocol-modal-overlay')).not.toBeVisible();
  });

  test('should have accessible close button with proper attributes', async ({ page, request }) => {
    await openAnimalDetailWithProtocol(page, request);

    // Open modal
    await page.click('.btn-view-document');
    await expect(page.locator('.protocol-modal-overlay')).toBeVisible();

    // Close button should have aria-label
    const closeButton = page.locator('.protocol-modal-close');
    await expect(closeButton).toHaveAttribute('aria-label', 'Close protocol document');
    
    // Close button should have title
    await expect(closeButton).toHaveAttribute('title', 'Close (Esc)');
  });

  test('should not leave page context when viewing protocol', async ({ page, context, request }) => {
    await openAnimalDetailWithProtocol(page, request);

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

  test('button should have proper styling and be accessible', async ({ page, request }) => {
    await openAnimalDetailWithProtocol(page, request);

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
