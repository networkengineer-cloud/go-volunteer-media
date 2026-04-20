// frontend/tests/group-documents-ux.spec.ts
import { test, expect, type APIRequestContext, type Page } from '@playwright/test';
import { testUsers } from './helpers/auth';
import fs from 'fs';
import path from 'path';
import os from 'os';

/**
 * Group Documents UX Tests
 *
 * Verifies the card-grid layout and modal upload introduced in the
 * 2026-04-20 UX redesign. Tests run as a site admin (who is also a
 * group admin) so upload/delete controls are visible.
 */

test.describe('Group Documents UX', () => {
  let adminToken: string;
  let groupId: number;
  let uploadedDocId: number | null = null;
  const tokenCachePath = path.join(os.tmpdir(), 'go-volunteer-media-e2e-admin-token.json');

  const getToken = async (page: Page): Promise<string> => {
    const token = await page.evaluate(() => localStorage.getItem('token'));
    expect(token, 'Missing auth token').toBeTruthy();
    return token as string;
  };

  const readCachedToken = (): string | null => {
    try {
      const raw = fs.readFileSync(tokenCachePath, 'utf8');
      const parsed = JSON.parse(raw) as { token?: string };
      return typeof parsed.token === 'string' ? parsed.token : null;
    } catch {
      return null;
    }
  };

  const writeCachedToken = (token: string) => {
    fs.writeFileSync(tokenCachePath, JSON.stringify({ token }), 'utf8');
  };

  test.beforeAll(async ({ request }) => {
    // Reuse cached token if still valid
    const cached = readCachedToken();
    if (cached) {
      const resp = await request.get('/api/me', {
        headers: { Authorization: `Bearer ${cached}` },
      });
      if (resp.ok()) {
        adminToken = cached;
      }
    }

    if (!adminToken) {
      const resp = await request.post('/api/login', {
        data: { username: testUsers.admin.username, password: testUsers.admin.password },
      });
      expect(resp.ok(), `Login failed: ${resp.status()}`).toBeTruthy();
      const json = (await resp.json()) as { token: string };
      adminToken = json.token;
      writeCachedToken(adminToken);
    }

    // Resolve first group id
    const groupsResp = await request.get('/api/groups', {
      headers: { Authorization: `Bearer ${adminToken}` },
    });
    expect(groupsResp.ok()).toBeTruthy();
    const groups = (await groupsResp.json()) as { id: number }[];
    expect(groups.length).toBeGreaterThan(0);
    groupId = groups[0].id;
  });

  test.beforeEach(async ({ page }) => {
    await page.addInitScript((token) => {
      localStorage.setItem('token', token);
    }, adminToken);
    await page.goto('/dashboard');
    await expect(page.getByRole('button', { name: /^logout$/i })).toBeVisible({ timeout: 15000 });
  });

  test.afterEach(async ({ request }) => {
    if (uploadedDocId === null) return;
    const docId = uploadedDocId;
    uploadedDocId = null;
    await request
      .delete(`/api/groups/${groupId}/documents/${docId}`, {
        headers: { Authorization: `Bearer ${adminToken}` },
      })
      .catch(() => {});
  });

  const navigateToDocuments = async (page: Page) => {
    await page.goto(`/groups/${groupId}?view=documents`);
    await expect(page.locator('#documents-tab')).toBeVisible({ timeout: 10000 });
    await page.click('#documents-tab');
    await expect(page.locator('#documents-panel')).toBeVisible({ timeout: 10000 });
  };

  test('document list renders as card grid, not full-width list', async ({ page }) => {
    await navigateToDocuments(page);
    // Card grid should exist
    await expect(page.locator('.document-grid')).toBeVisible({ timeout: 8000 });
    // Old list should NOT exist
    await expect(page.locator('.document-list')).not.toBeAttached();
  });

  test('upload form is not visible inline — only a button', async ({ page }) => {
    await navigateToDocuments(page);
    // Inline form must not be present
    await expect(page.locator('.document-upload-form')).not.toBeAttached();
    // Upload button must be visible to admins
    await expect(page.getByRole('button', { name: /upload document/i })).toBeVisible({ timeout: 8000 });
  });

  test('clicking Upload Document opens the modal', async ({ page }) => {
    await navigateToDocuments(page);
    await page.getByRole('button', { name: /upload document/i }).click();
    await expect(page.locator('.modal-overlay')).toBeVisible({ timeout: 5000 });
    await expect(page.getByRole('heading', { name: /upload document/i })).toBeVisible();
  });

  test('modal closes on Cancel', async ({ page }) => {
    await navigateToDocuments(page);
    await page.getByRole('button', { name: /upload document/i }).click();
    await expect(page.locator('.modal-overlay')).toBeVisible({ timeout: 5000 });
    await page.getByRole('button', { name: /cancel/i }).click();
    await expect(page.locator('.modal-overlay')).not.toBeVisible({ timeout: 5000 });
  });

  test('each document card has Open button and icon buttons', async ({ page, request }) => {
    // Upload a document so there is at least one card to assert on
    const pdfBuffer = Buffer.from('%PDF-1.4 test');
    const uploadResp = await request.post(`/api/groups/${groupId}/documents`, {
      headers: { Authorization: `Bearer ${adminToken}` },
      multipart: {
        title: 'E2E Card Test Doc',
        description: 'test',
        file: { name: 'e2e-test.pdf', mimeType: 'application/pdf', buffer: pdfBuffer },
      },
    });
    if (uploadResp.ok()) {
      const json = (await uploadResp.json()) as { id: number };
      uploadedDocId = json.id;
    }

    await navigateToDocuments(page);
    const firstCard = page.locator('.document-card').first();
    await expect(firstCard).toBeVisible({ timeout: 8000 });

    // Primary open button
    await expect(firstCard.getByRole('button', { name: /open/i })).toBeVisible();
    // Download icon button
    await expect(firstCard.getByRole('button', { name: /download/i })).toBeVisible();
    // Delete icon button (site admin sees this)
    await expect(firstCard.getByRole('button', { name: /delete/i })).toBeVisible();
  });

  test('empty state shown when no documents exist and no list is rendered', async ({ page, request }) => {
    // This test only makes sense if the group has no docs — skip if it does
    const docsResp = await request.get(`/api/groups/${groupId}/documents`, {
      headers: { Authorization: `Bearer ${adminToken}` },
    });
    const docs = (await docsResp.json()) as unknown[];
    test.skip(docs.length > 0, 'Group has documents; skipping empty-state test');

    await navigateToDocuments(page);
    await expect(page.locator('.document-grid')).not.toBeAttached();
    await expect(page.locator('.empty-state')).toBeVisible({ timeout: 8000 });
  });
});
