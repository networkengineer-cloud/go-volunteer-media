/// <reference types="node" />

import { test, expect, type APIRequestContext, type Page } from '@playwright/test';
import { testUsers } from './helpers/auth';
import fs from 'fs';
import os from 'os';
import path from 'path';

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
  let createdEntity: { groupId: number; animalId: number } | null = null;
  const tokenCachePath = path.join(os.tmpdir(), 'go-volunteer-media-e2e-admin-token.json');
  const tokenLockPath = `${tokenCachePath}.lock`;

  interface ApiGroup {
    id: number;
  }

  interface ApiCreatedAnimal {
    id: number;
  }


  const getToken = async (page: Page): Promise<string> => {
    const token = await page.evaluate(() => localStorage.getItem('token'));
    expect(token, 'Missing auth token after login').toBeTruthy();
    return token as string;
  };

  const getFirstGroupId = async (
    request: APIRequestContext,
    token: string
  ): Promise<number> => {
    const groupsResp = await request.get('/api/groups', {
      headers: { Authorization: `Bearer ${token}` },
    });
    expect(groupsResp.ok(), `Failed to load groups: ${groupsResp.status()}`).toBeTruthy();

    const groups = (await groupsResp.json()) as ApiGroup[];
    expect(groups.length, 'No groups returned from /api/groups').toBeGreaterThan(0);
    return groups[0].id;
  };

  const openAnimalDetailWithProtocol = async (page: Page, request: APIRequestContext) => {
    const token = await getToken(page);
    const groupId = await getFirstGroupId(request, token);

    // Create a dedicated animal and upload a known-good fixture doc.
    // This avoids flaky seed-data mutations where protocol_document_url changes mid-test.
    const animalName = `E2E Protocol Animal ${Date.now()}-${Math.random().toString(16).slice(2)}`;
    const createAnimalResp = await request.post(`/api/groups/${groupId}/animals`, {
      headers: { Authorization: `Bearer ${token}` },
      data: {
        name: animalName,
        species: 'Dog',
        breed: 'Test Breed',
        age: 2,
        description: 'E2E protocol document modal test animal',
        status: 'available',
      },
    });
    expect(createAnimalResp.ok(), `Failed to create test animal: ${createAnimalResp.status()}`).toBeTruthy();
    const createdAnimal = (await createAnimalResp.json()) as ApiCreatedAnimal;
    expect(createdAnimal.id, 'Created animal missing id').toBeTruthy();

    const animalId = createdAnimal.id;
    createdEntity = { groupId, animalId };

    const fixtureBuffer = fs.readFileSync(new URL('./fixtures/protocol.pdf', import.meta.url));
    const uploadResp = await request.post(`/api/groups/${groupId}/animals/${animalId}/protocol-document`, {
      headers: { Authorization: `Bearer ${token}` },
      multipart: {
        document: {
          name: 'protocol.pdf',
          mimeType: 'application/pdf',
          buffer: fixtureBuffer,
        },
      },
    });
    expect(uploadResp.ok(), `Failed to upload protocol fixture: ${uploadResp.status()}`).toBeTruthy();

    await page.goto(`/groups/${groupId}/animals/${animalId}/view`);
    await page.waitForSelector('.animal-detail-page', { timeout: 10000 });
    await expect(page.locator('.btn-view-document')).toBeVisible({ timeout: 10000 });
    return { groupId, animalId };
  };

  const openProtocolModal = async (page: Page) => {
    await page.click('.btn-view-document');
    await expect(page.locator('.protocol-viewer-overlay')).toBeVisible({ timeout: 10000 });
  };

  test.beforeAll(async ({ request }) => {
    const validateToken = async (token: string): Promise<boolean> => {
      const resp = await request.get('/api/me', {
        headers: { Authorization: `Bearer ${token}` },
      });
      return resp.ok();
    };

    const readCachedToken = async (): Promise<string | null> => {
      try {
        const raw = fs.readFileSync(tokenCachePath, 'utf8');
        const parsed = JSON.parse(raw) as { token?: string };
        if (typeof parsed.token !== 'string' || parsed.token.length === 0) return null;
        return parsed.token;
      } catch {
        return null;
      }
    };

    const writeCachedToken = (token: string) => {
      fs.writeFileSync(tokenCachePath, JSON.stringify({ token }), 'utf8');
    };

    const cached = await readCachedToken();
    if (cached && (await validateToken(cached))) {
      adminToken = cached;
      return;
    }

    // Try to acquire a simple file lock so only one project hits /api/login.
    let lockFd: number | null = null;
    try {
      lockFd = fs.openSync(tokenLockPath, 'wx');
    } catch {
      lockFd = null;
    }

    if (lockFd === null) {
      // Another project is logging in. Wait for the token file to appear.
      const deadline = Date.now() + 20000;
      while (Date.now() < deadline) {
        const maybeToken = await readCachedToken();
        if (maybeToken && (await validateToken(maybeToken))) {
          adminToken = maybeToken;
          return;
        }
        await new Promise((resolve) => setTimeout(resolve, 250));
      }
      // Fall through to login attempt if the lock holder failed.
    }

    try {
      const maxAttempts = 10;
      for (let attempt = 1; attempt <= maxAttempts; attempt++) {
        const resp = await request.post('/api/login', {
          data: {
            username: testUsers.admin.username,
            password: testUsers.admin.password,
          },
        });

        if (resp.status() === 429 && attempt < maxAttempts) {
          const backoffMs = Math.min(1000 * 2 ** (attempt - 1), 10000);
          await new Promise((resolve) => setTimeout(resolve, backoffMs));
          continue;
        }

        expect(resp.ok(), `Failed to login via API: ${resp.status()}`).toBeTruthy();
        const json = (await resp.json()) as { token?: string };
        expect(json.token, 'Missing token in /api/login response').toBeTruthy();
        adminToken = json.token as string;
        writeCachedToken(adminToken);
        return;
      }

      throw new Error('Failed to login via API after retries');
    } finally {
      if (lockFd !== null) {
        try {
          fs.closeSync(lockFd);
        } catch {
          // ignore
        }
        try {
          fs.unlinkSync(tokenLockPath);
        } catch {
          // ignore
        }
      }
    }
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

  test.afterEach(async ({ request }) => {
    if (!createdEntity) return;

    const { groupId, animalId } = createdEntity;
    createdEntity = null;

    // Best-effort cleanup: if the server was restarted/reseeded, this may 404.
    try {
      await request.delete(`/api/groups/${groupId}/animals/${animalId}`, {
        headers: { Authorization: `Bearer ${adminToken}` },
      });
    } catch {
      // ignore cleanup failures
    }
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
        return res.url().includes('/api/documents/');
      }, { timeout: 20000 }),
      page.click('.btn-view-document'),
    ]);
    expect(response.status(), `Protocol document fetch failed: ${response.status()} ${response.url()}`).toBe(200);
    const contentType = response.headers()['content-type'] ?? '';
    expect(contentType, 'Unexpected content-type for protocol document')
      .toMatch(/^(application\/pdf|application\/vnd\.openxmlformats-officedocument\.wordprocessingml\.document)/);

    // Modal should appear
    await expect(page.locator('.protocol-viewer-overlay')).toBeVisible({ timeout: 5000 });
    
    // Modal content should be visible
    await expect(page.locator('.protocol-viewer-modal')).toBeVisible();
    
    // Modal should have proper ARIA attributes
    await expect(page.locator('.protocol-viewer-overlay')).toHaveAttribute('role', 'dialog');
    await expect(page.locator('.protocol-viewer-overlay')).toHaveAttribute('aria-modal', 'true');
    
    // Modal header should show the title
    await expect(page.locator('#protocol-viewer-title')).toHaveText(/Protocol Document/);
    
    if (contentType.startsWith('application/pdf')) {
      const body = await response.body();
      expect(body.length, 'Protocol PDF response body should not be empty').toBeGreaterThan(100);
      expect(body.subarray(0, 4).toString('utf8'), 'Protocol PDF should start with %PDF').toBe('%PDF');

      // PDF should render with PDF.js viewer (canvas-based, not iframe)
      const pdfViewer = page.locator('.protocol-pdf-viewer');
      await expect(pdfViewer).toBeVisible({ timeout: 8000 });
      
      // PDF controls should be visible
      const pdfControls = page.locator('.protocol-pdf-controls');
      await expect(pdfControls).toBeVisible();
      
      // PDF canvas should be rendered
      const pdfCanvas = page.locator('.pdf-page-canvas');
      await expect(pdfCanvas.first()).toBeVisible({ timeout: 8000 });
    } else {
      // DOCX should render into container with converted HTML content
      await expect(page.locator('.protocol-docx-viewer')).toBeVisible({ timeout: 8000 });
      await expect(page.locator('.protocol-docx-content')).toBeVisible({ timeout: 8000 });
    }
  });

  test('should close modal on Escape key press', async ({ page, request }) => {
    await openAnimalDetailWithProtocol(page, request);

    await openProtocolModal(page);

    // Press Escape key
    await page.keyboard.press('Escape');

    // Modal should be closed
    await expect(page.locator('.protocol-viewer-overlay')).not.toBeVisible();
  });

  test('should close modal when clicking outside', async ({ page, request }) => {
    await openAnimalDetailWithProtocol(page, request);

    await openProtocolModal(page);

    const overlay = page.locator('.protocol-viewer-overlay');

    const clickOutside = async () => {
      const overlayBox = await overlay.boundingBox();
      expect(overlayBox, 'Missing overlay bounding box').toBeTruthy();

      const o = overlayBox as NonNullable<typeof overlayBox>;
      const candidates = [
        { x: Math.floor(o.x + 2), y: Math.floor(o.y + 2) },
        { x: Math.floor(o.x + o.width - 2), y: Math.floor(o.y + 2) },
        { x: Math.floor(o.x + 2), y: Math.floor(o.y + o.height - 2) },
        { x: Math.floor(o.x + o.width - 2), y: Math.floor(o.y + o.height - 2) },
      ];

      let target: { x: number; y: number } | null = null;
      for (const p of candidates) {
        const isOverlayOutsideContent = await page.evaluate(({ x, y }) => {
          const el = document.elementFromPoint(x, y);
          if (!el) return false;
          if (el.closest('.protocol-viewer-modal')) return false;
          return !!el.closest('.protocol-viewer-overlay');
        }, p);

        if (isOverlayOutsideContent) {
          target = p;
          break;
        }
      }

      test.skip(!target, 'Modal content covers overlay; no outside click target on this viewport');
      await page.mouse.click((target as { x: number; y: number }).x, (target as { x: number; y: number }).y);
    };

    await clickOutside();
    // Retry once for occasional mobile/WebKit gesture quirks.
    if (await overlay.isVisible()) {
      await clickOutside();
    }

    // Modal should be closed
    await expect(page.locator('.protocol-viewer-overlay')).not.toBeVisible({ timeout: 10000 });
  });

  test('should close modal when clicking close button', async ({ page, request }) => {
    await openAnimalDetailWithProtocol(page, request);

    await openProtocolModal(page);

    // Click close button
    await page.click('.protocol-viewer-close');

    // Modal should be closed
    await expect(page.locator('.protocol-viewer-overlay')).not.toBeVisible();
  });

  test('should have accessible close button with proper attributes', async ({ page, request }) => {
    await openAnimalDetailWithProtocol(page, request);

    await openProtocolModal(page);

    // Close button should have aria-label
    const closeButton = page.locator('.protocol-viewer-close');
    await expect(closeButton).toHaveAttribute('aria-label', 'Close protocol document');
    
    // Close button should have title
    await expect(closeButton).toHaveAttribute('title', 'Close (Esc)');
  });

  test('should not leave page context when viewing protocol', async ({ page, context, request }) => {
    await openAnimalDetailWithProtocol(page, request);

    // Track if new tab/page is created
    const pagesBefore = context.pages().length;
    const urlBefore = page.url();

    await openProtocolModal(page);

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
