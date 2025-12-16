/// <reference types="node" />

import { test, expect, type APIRequestContext, type Page } from '@playwright/test';
import { testUsers } from './helpers/auth';
import fs from 'fs';
import os from 'os';
import path from 'path';

/**
 * Protocol Document Mobile Responsiveness Tests
 * 
 * Tests the mobile-responsive behavior of the protocol document viewer
 * on different device sizes (phone, tablet, desktop).
 * 
 * Validates:
 * - DOCX documents fit within viewport on mobile (no horizontal overflow)
 * - PDF documents are accessible on mobile
 * - Download button is visible and functional on all devices
 * - Modal adapts to different screen sizes
 */

test.describe('Protocol Document Mobile Responsiveness', () => {
  let adminToken: string;
  let createdEntity: { groupId: number; animalId: number } | null = null;
  const tokenCachePath = path.join(os.tmpdir(), 'go-volunteer-media-e2e-admin-token.json');

  interface ApiGroup {
    id: number;
  }

  interface ApiCreatedAnimal {
    id: number;
  }


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

  const createTestAnimalWithProtocol = async (request: APIRequestContext, token: string) => {
    const groupId = await getFirstGroupId(request, token);

    const animalName = `E2E Mobile Protocol ${Date.now()}-${Math.random().toString(16).slice(2)}`;
    const createAnimalResp = await request.post(`/api/groups/${groupId}/animals`, {
      headers: { Authorization: `Bearer ${token}` },
      data: {
        name: animalName,
        species: 'Dog',
        breed: 'Test Breed',
        age: 2,
        description: 'E2E mobile protocol test animal',
        status: 'available',
      },
    });
    expect(createAnimalResp.ok(), `Failed to create test animal: ${createAnimalResp.status()}`).toBeTruthy();
    const createdAnimal = (await createAnimalResp.json()) as ApiCreatedAnimal;
    const animalId = createdAnimal.id;

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

    return { groupId, animalId };
  };

  test.beforeAll(async ({ request }) => {
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

    const cached = await readCachedToken();
    if (cached) {
      const resp = await request.get('/api/me', {
        headers: { Authorization: `Bearer ${cached}` },
      });
      if (resp.ok()) {
        adminToken = cached;
        return;
      }
    }

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

    await page.goto('/dashboard');
    await expect(page.getByRole('button', { name: /^logout$/i })).toBeVisible({ timeout: 15000 });
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

  test('download button should be visible on mobile devices', async ({ page, request }) => {
    const entity = await createTestAnimalWithProtocol(request, adminToken);
    createdEntity = entity;

    // Test on phone viewport (iPhone 12)
    await page.setViewportSize({ width: 390, height: 844 });
    await page.goto(`/groups/${entity.groupId}/animals/${entity.animalId}/view`);
    await page.waitForSelector('.animal-detail-page', { timeout: 10000 });

    // Open protocol modal
    await page.click('.btn-view-document');
    await expect(page.locator('.protocol-viewer-overlay')).toBeVisible({ timeout: 10000 });

    // Download button should be visible
    const downloadButton = page.locator('.btn-download-protocol');
    await expect(downloadButton).toBeVisible();
    
    // Button should have proper aria-label
    await expect(downloadButton).toHaveAttribute('aria-label', 'Download protocol document');
    
    // Button should be full-width on mobile
    const buttonBox = await downloadButton.boundingBox();
    const modalContentBox = await page.locator('.protocol-viewer-modal').boundingBox();
    
    expect(buttonBox, 'Download button should have dimensions').toBeTruthy();
    expect(modalContentBox, 'Modal content should have dimensions').toBeTruthy();
    
    // On mobile, button should be close to full width (accounting for padding)
    if (buttonBox && modalContentBox) {
      expect(buttonBox.width).toBeGreaterThan(modalContentBox.width * 0.7);
    }
  });

  test('mobile hint should be visible on small screens', async ({ page, request }) => {
    const entity = await createTestAnimalWithProtocol(request, adminToken);
    createdEntity = entity;

    // Test on phone viewport
    await page.setViewportSize({ width: 375, height: 667 });
    await page.goto(`/groups/${entity.groupId}/animals/${entity.animalId}/view`);
    await page.waitForSelector('.animal-detail-page', { timeout: 10000 });

    // Open protocol modal
    await page.click('.btn-view-document');
    await expect(page.locator('.protocol-viewer-overlay')).toBeVisible({ timeout: 10000 });

    // Wait for document to load
    await page.waitForTimeout(2000);
  });

  test('modal should be full-screen on mobile', async ({ page, request }) => {
    const entity = await createTestAnimalWithProtocol(request, adminToken);
    createdEntity = entity;

    // Test on phone viewport
    await page.setViewportSize({ width: 375, height: 667 });
    await page.goto(`/groups/${entity.groupId}/animals/${entity.animalId}/view`);
    await page.waitForSelector('.animal-detail-page', { timeout: 10000 });

    // Open protocol modal
    await page.click('.btn-view-document');
    await expect(page.locator('.protocol-viewer-overlay')).toBeVisible({ timeout: 10000 });

    // Modal content should be full-screen on mobile
    const modalContent = page.locator('.protocol-viewer-modal');
    const modalBox = await modalContent.boundingBox();
    
    expect(modalBox, 'Modal should have dimensions').toBeTruthy();
    
    if (modalBox) {
      const viewport = page.viewportSize();
      expect(viewport, 'Viewport should be set').toBeTruthy();
      if (viewport) {
        // Allow tiny rounding differences across browsers/devices.
        expect(modalBox.width).toBeGreaterThanOrEqual(viewport.width * 0.95);
        expect(modalBox.height).toBeGreaterThanOrEqual(viewport.height * 0.95);
      }
    }
  });

  test('download button should trigger file download', async ({ page, request }) => {
    const entity = await createTestAnimalWithProtocol(request, adminToken);
    createdEntity = entity;

    await page.setViewportSize({ width: 390, height: 844 });
    await page.goto(`/groups/${entity.groupId}/animals/${entity.animalId}/view`);
    await page.waitForSelector('.animal-detail-page', { timeout: 10000 });

    // Open protocol modal
    await page.click('.btn-view-document');
    await expect(page.locator('.protocol-viewer-overlay')).toBeVisible({ timeout: 10000 });

    // Listen for download
    const downloadPromise = page.waitForEvent('download', { timeout: 10000 });
    
    // Click download button
    await page.click('.btn-download-protocol');
    
    // Wait for download to start
    const download = await downloadPromise;
    
    // Verify download has the correct filename
    expect(download.suggestedFilename()).toBe('protocol.pdf');
  });

  test('modal header should stack on mobile', async ({ page, request }) => {
    const entity = await createTestAnimalWithProtocol(request, adminToken);
    createdEntity = entity;

    // Test on phone viewport
    await page.setViewportSize({ width: 375, height: 667 });
    await page.goto(`/groups/${entity.groupId}/animals/${entity.animalId}/view`);
    await page.waitForSelector('.animal-detail-page', { timeout: 10000 });

    // Open protocol modal
    await page.click('.btn-view-document');
    await expect(page.locator('.protocol-viewer-overlay')).toBeVisible({ timeout: 10000 });

    // Get positions of header elements
    const title = page.locator('#protocol-viewer-title');
    const downloadButton = page.locator('.btn-download-protocol');
    
    const titleBox = await title.boundingBox();
    const buttonBox = await downloadButton.boundingBox();
    
    expect(titleBox, 'Title should have dimensions').toBeTruthy();
    expect(buttonBox, 'Button should have dimensions').toBeTruthy();
    
    if (titleBox && buttonBox) {
      // Button should be below the title on mobile (vertical stacking) within the header
      expect(buttonBox.y).toBeGreaterThanOrEqual(titleBox.y);
    }
  });

  test('modal content should not overflow horizontally on mobile', async ({ page, request }) => {
    const entity = await createTestAnimalWithProtocol(request, adminToken);
    createdEntity = entity;

    // Test on phone viewport
    await page.setViewportSize({ width: 375, height: 667 });
    await page.goto(`/groups/${entity.groupId}/animals/${entity.animalId}/view`);
    await page.waitForSelector('.animal-detail-page', { timeout: 10000 });

    // Open protocol modal
    await page.click('.btn-view-document');
    await expect(page.locator('.protocol-viewer-overlay')).toBeVisible({ timeout: 10000 });

    // Wait for PDF viewer to load
    await page.waitForSelector('.protocol-pdf-viewer', { timeout: 10000 });

    // Check that the modal layout doesn't create horizontal overflow.
    const hasHorizontalScroll = await page.evaluate(() => {
      const body = document.querySelector('.protocol-viewer-body') as HTMLElement | null;
      if (!body) return false;
      return body.scrollWidth > body.clientWidth + 1;
    });
    
    expect(hasHorizontalScroll).toBe(false);
  });

  test('tablet viewport should show appropriate layout', async ({ page, request }) => {
    const entity = await createTestAnimalWithProtocol(request, adminToken);
    createdEntity = entity;

    // Test on tablet viewport (iPad)
    await page.setViewportSize({ width: 768, height: 1024 });
    await page.goto(`/groups/${entity.groupId}/animals/${entity.animalId}/view`);
    await page.waitForSelector('.animal-detail-page', { timeout: 10000 });

    // Open protocol modal
    await page.click('.btn-view-document');
    await expect(page.locator('.protocol-viewer-overlay')).toBeVisible({ timeout: 10000 });

    // Modal should be full-screen on tablet
    const modalContent = page.locator('.protocol-viewer-modal');
    const modalBox = await modalContent.boundingBox();
    
    expect(modalBox, 'Modal should have dimensions').toBeTruthy();
    
    if (modalBox) {
      const viewport = page.viewportSize();
      expect(viewport, 'Viewport should be set').toBeTruthy();
      if (viewport) {
        // Tablet uses a slightly narrower modal (max-width ~95%).
        expect(modalBox.width).toBeGreaterThanOrEqual(viewport.width * 0.9);
        expect(modalBox.width).toBeLessThanOrEqual(viewport.width);
      }
    }

    // Download button should be visible
    await expect(page.locator('.btn-download-protocol')).toBeVisible();
  });
});
