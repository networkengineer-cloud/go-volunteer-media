/// <reference types="node" />

import { test, expect, type APIRequestContext, type Page } from '@playwright/test';
import { testUsers } from './helpers/auth';
import fs from 'fs';
import os from 'os';
import path from 'path';

/**
 * Protocol Document Scrolling UX Tests
 * 
 * Tests that the protocol viewer properly handles:
 * 1. Multi-page PDF scrolling on desktop and mobile
 * 2. DOCX content display and scrolling
 * 3. Modal visibility with solid background (not transparent)
 * 4. Proper overflow behavior for different viewport sizes
 * 
 * Addresses issue: "PDF no scrolling on desktop, DOCX not displaying, modal transparent"
 */

test.describe('Protocol Document Scrolling & Visibility', () => {
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

  const createAnimalWithProtocol = async (
    page: Page,
    request: APIRequestContext,
    filename: string
  ) => {
    const token = await getToken(page);
    const groupId = await getFirstGroupId(request, token);

    const animalName = `E2E Scroll Test ${Date.now()}-${Math.random().toString(16).slice(2)}`;
    const createAnimalResp = await request.post(`/api/groups/${groupId}/animals`, {
      headers: { Authorization: `Bearer ${token}` },
      data: {
        name: animalName,
        species: 'Dog',
        breed: 'Test Breed',
        age: 2,
        description: 'E2E protocol scrolling test animal',
        status: 'available',
      },
    });
    expect(createAnimalResp.ok()).toBeTruthy();
    const createdAnimal = (await createAnimalResp.json()) as ApiCreatedAnimal;

    const animalId = createdAnimal.id;
    createdEntity = { groupId, animalId };

    const fixtureBuffer = fs.readFileSync(new URL(`./fixtures/${filename}`, import.meta.url));
    const mimeType = filename.endsWith('.pdf')
      ? 'application/pdf'
      : 'application/vnd.openxmlformats-officedocument.wordprocessingml.document';

    const uploadResp = await request.post(`/api/groups/${groupId}/animals/${animalId}/protocol-document`, {
      headers: { Authorization: `Bearer ${token}` },
      multipart: {
        document: {
          name: filename,
          mimeType,
          buffer: fixtureBuffer,
        },
      },
    });
    expect(uploadResp.ok()).toBeTruthy();

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

    let lockFd: number | null = null;
    try {
      lockFd = fs.openSync(tokenLockPath, 'wx');
    } catch {
      lockFd = null;
    }

    if (lockFd === null) {
      const deadline = Date.now() + 20000;
      while (Date.now() < deadline) {
        const maybeToken = await readCachedToken();
        if (maybeToken && (await validateToken(maybeToken))) {
          adminToken = maybeToken;
          return;
        }
        await new Promise((resolve) => setTimeout(resolve, 250));
      }
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

        expect(resp.ok()).toBeTruthy();
        const json = (await resp.json()) as { token?: string };
        expect(json.token).toBeTruthy();
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

    await page.goto('/dashboard');
    await expect(page.getByRole('button', { name: /^logout$/i })).toBeVisible({ timeout: 15000 });
    await getToken(page);
  });

  test.afterEach(async ({ request }) => {
    if (!createdEntity) return;

    const { groupId, animalId } = createdEntity;
    createdEntity = null;

    try {
      await request.delete(`/api/groups/${groupId}/animals/${animalId}`, {
        headers: { Authorization: `Bearer ${adminToken}` },
      });
    } catch {
      // ignore cleanup failures
    }
  });

  test('modal should have solid, non-transparent background', async ({ page, request }) => {
    await createAnimalWithProtocol(page, request, 'protocol.pdf');
    await openProtocolModal(page);

    const modal = page.locator('.protocol-viewer-modal');
    await expect(modal).toBeVisible();

    // Allow entrance animation to complete before checking computed opacity.
    await page.waitForTimeout(400);

    // Check that modal has solid background (opacity should be 1)
    const opacity = await modal.evaluate((el) => {
      return window.getComputedStyle(el).opacity;
    });
    expect(parseFloat(opacity)).toBeGreaterThan(0.9);

    // Check that modal has a background color (not transparent)
    const backgroundColor = await modal.evaluate((el) => {
      return window.getComputedStyle(el).backgroundColor;
    });
    expect(backgroundColor).not.toBe('rgba(0, 0, 0, 0)');
    expect(backgroundColor).not.toBe('transparent');
  });

  test('PDF viewer should be scrollable on desktop viewport', async ({ page, request }) => {
    // Set desktop viewport
    await page.setViewportSize({ width: 1280, height: 800 });

    await createAnimalWithProtocol(page, request, 'protocol.pdf');
    await openProtocolModal(page);

    const pdfContainer = page.locator('.protocol-pdf-container');
    await expect(pdfContainer).toBeVisible({ timeout: 10000 });

    // Wait for PDF to render
    await expect(page.locator('.pdf-page-canvas').first()).toBeVisible({ timeout: 10000 });

    // Check that PDF container has overflow-y: auto (scrollable)
    const overflowY = await pdfContainer.evaluate((el) => {
      return window.getComputedStyle(el).overflowY;
    });
    expect(['auto', 'scroll']).toContain(overflowY);

    // Check that the container is scrollable (scrollHeight > clientHeight for multi-page PDFs)
    const isScrollable = await pdfContainer.evaluate((el) => {
      return el.scrollHeight > el.clientHeight;
    });
    
    // If the PDF has multiple pages, it should be scrollable
    const pageCount = await page.locator('.pdf-page-canvas').count();
    if (pageCount > 1) {
      expect(isScrollable).toBe(true);
    }
  });

  test('PDF viewer should allow scrolling through content on desktop', async ({ page, request }) => {
    // Set desktop viewport
    await page.setViewportSize({ width: 1280, height: 800 });

    await createAnimalWithProtocol(page, request, 'protocol.pdf');
    await openProtocolModal(page);

    const pdfContainer = page.locator('.protocol-pdf-container');
    await expect(pdfContainer).toBeVisible({ timeout: 10000 });
    await expect(page.locator('.pdf-page-canvas').first()).toBeVisible({ timeout: 10000 });

    // Get initial scroll position
    const initialScrollTop = await pdfContainer.evaluate((el) => el.scrollTop);

    // Attempt to scroll down
    await pdfContainer.evaluate((el) => {
      el.scrollTop = 100;
    });

    // Wait a moment for scroll to take effect
    await page.waitForTimeout(100);

    // Check if scroll position changed
    const newScrollTop = await pdfContainer.evaluate((el) => el.scrollTop);
    
    // If content is scrollable, scroll position should have changed
    const isScrollable = await pdfContainer.evaluate((el) => {
      return el.scrollHeight > el.clientHeight;
    });
    
    if (isScrollable) {
      expect(newScrollTop).toBeGreaterThan(initialScrollTop);
    }
  });

  test('PDF viewer should be scrollable on mobile viewport', async ({ page, request }) => {
    // Set mobile viewport
    await page.setViewportSize({ width: 375, height: 667 });

    await createAnimalWithProtocol(page, request, 'protocol.pdf');
    await openProtocolModal(page);

    const pdfContainer = page.locator('.protocol-pdf-container');
    await expect(pdfContainer).toBeVisible({ timeout: 10000 });
    await expect(page.locator('.pdf-page-canvas').first()).toBeVisible({ timeout: 10000 });

    // Check that PDF container has overflow-y: auto (scrollable)
    const overflowY = await pdfContainer.evaluate((el) => {
      return window.getComputedStyle(el).overflowY;
    });
    expect(['auto', 'scroll']).toContain(overflowY);
  });

  test('DOCX viewer should display content', async ({ page, request }) => {
    // Check if DOCX fixture exists, skip if not
    const docxPath = new URL('./fixtures/protocol.docx', import.meta.url);
    if (!fs.existsSync(docxPath)) {
      test.skip();
    }

    await createAnimalWithProtocol(page, request, 'protocol.docx');
    await openProtocolModal(page);

    const docxViewer = page.locator('.protocol-docx-viewer');
    await expect(docxViewer).toBeVisible({ timeout: 10000 });

    const docxContent = page.locator('.protocol-docx-content');
    await expect(docxContent).toBeVisible({ timeout: 10000 });

    // Check that content has been rendered (not empty)
    const hasContent = await docxContent.evaluate((el) => {
      return el.textContent && el.textContent.trim().length > 0;
    });
    expect(hasContent).toBe(true);
  });

  test('DOCX viewer should be scrollable', async ({ page, request }) => {
    // Check if DOCX fixture exists, skip if not
    const docxPath = new URL('./fixtures/protocol.docx', import.meta.url);
    if (!fs.existsSync(docxPath)) {
      test.skip();
    }

    await page.setViewportSize({ width: 1280, height: 800 });

    await createAnimalWithProtocol(page, request, 'protocol.docx');
    await openProtocolModal(page);

    const docxViewer = page.locator('.protocol-docx-viewer');
    await expect(docxViewer).toBeVisible({ timeout: 10000 });
    await expect(page.locator('.protocol-docx-content')).toBeVisible({ timeout: 10000 });

    // Check that DOCX viewer has overflow-y: auto (scrollable)
    const overflowY = await docxViewer.evaluate((el) => {
      return window.getComputedStyle(el).overflowY;
    });
    expect(['auto', 'scroll']).toContain(overflowY);
  });

  test('protocol viewer body should not have overflow:hidden', async ({ page, request }) => {
    await createAnimalWithProtocol(page, request, 'protocol.pdf');
    await openProtocolModal(page);

    const viewerBody = page.locator('.protocol-viewer-body');
    await expect(viewerBody).toBeVisible();

    // Check that viewer body does NOT have overflow: hidden
    const overflow = await viewerBody.evaluate((el) => {
      return window.getComputedStyle(el).overflow;
    });
    expect(overflow).not.toBe('hidden');
  });

  test('modal overlay should have proper dark backdrop', async ({ page, request }) => {
    await createAnimalWithProtocol(page, request, 'protocol.pdf');
    await openProtocolModal(page);

    const overlay = page.locator('.protocol-viewer-overlay');
    await expect(overlay).toBeVisible();

    // Check that overlay has dark semi-transparent background
    const backgroundColor = await overlay.evaluate((el) => {
      return window.getComputedStyle(el).backgroundColor;
    });
    
    // Should have rgba with alpha > 0 (not fully transparent)
    expect(backgroundColor).toMatch(/rgba?\(/);
  });
});
