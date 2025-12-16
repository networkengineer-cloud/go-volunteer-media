/// <reference types="node" />

import { test, expect, type APIRequestContext, type Page } from '@playwright/test';
import { testUsers } from './helpers/auth';
import fs from 'fs';
import os from 'os';
import path from 'path';

/**
 * Visual Comparison Tests for Protocol Document Viewer
 * 
 * Creates screenshots demonstrating the mobile-responsive improvements
 * for the protocol document viewer on different device sizes.
 */

test.describe('Protocol Document Visual Comparison @visual', () => {
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
    expect(groupsResp.ok()).toBeTruthy();
    const groups = (await groupsResp.json()) as ApiGroup[];
    expect(groups.length).toBeGreaterThan(0);
    return groups[0].id;
  };

  const createTestAnimalWithProtocol = async (request: APIRequestContext, token: string) => {
    const groupId = await getFirstGroupId(request, token);
    const animalName = `Visual Test Animal ${Date.now()}`;
    
    const createAnimalResp = await request.post(`/api/groups/${groupId}/animals`, {
      headers: { Authorization: `Bearer ${token}` },
      data: {
        name: animalName,
        species: 'Dog',
        breed: 'Test Breed',
        age: 2,
        description: 'Visual comparison test animal',
        status: 'available',
      },
    });
    expect(createAnimalResp.ok()).toBeTruthy();
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
    expect(uploadResp.ok()).toBeTruthy();

    return { groupId, animalId };
  };

  test.beforeAll(async ({ request }) => {
    const readCachedToken = async (): Promise<string | null> => {
      try {
        const raw = fs.readFileSync(tokenCachePath, 'utf8');
        const parsed = JSON.parse(raw) as { token?: string };
        return parsed.token || null;
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
    expect(resp.ok()).toBeTruthy();
    const json = (await resp.json()) as { token?: string };
    expect(json.token).toBeTruthy();
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
      // ignore
    }
  });

  test('desktop: protocol modal with download button', async ({ page, request }) => {
    const entity = await createTestAnimalWithProtocol(request, adminToken);
    createdEntity = entity;

    // Desktop viewport
    await page.setViewportSize({ width: 1280, height: 800 });
    await page.goto(`/groups/${entity.groupId}/animals/${entity.animalId}/view`);
    await page.waitForSelector('.animal-detail-page', { timeout: 10000 });

    // Open protocol modal
    await page.click('.btn-view-document');
    await expect(page.locator('.protocol-modal-overlay')).toBeVisible({ timeout: 10000 });

    // Wait for content to load
    await page.waitForTimeout(1000);

    // Take screenshot
    await page.screenshot({
      path: 'test-results/protocol-desktop-1280px.png',
      fullPage: false,
    });
  });

  test('tablet: full-screen modal with download button', async ({ page, request }) => {
    const entity = await createTestAnimalWithProtocol(request, adminToken);
    createdEntity = entity;

    // Tablet viewport (iPad)
    await page.setViewportSize({ width: 768, height: 1024 });
    await page.goto(`/groups/${entity.groupId}/animals/${entity.animalId}/view`);
    await page.waitForSelector('.animal-detail-page', { timeout: 10000 });

    // Open protocol modal
    await page.click('.btn-view-document');
    await expect(page.locator('.protocol-modal-overlay')).toBeVisible({ timeout: 10000 });

    // Wait for content to load
    await page.waitForTimeout(1000);

    // Take screenshot
    await page.screenshot({
      path: 'test-results/protocol-tablet-768px.png',
      fullPage: false,
    });
  });

  test('phone: full-screen modal with prominent download button', async ({ page, request }) => {
    const entity = await createTestAnimalWithProtocol(request, adminToken);
    createdEntity = entity;

    // Phone viewport (iPhone 12)
    await page.setViewportSize({ width: 390, height: 844 });
    await page.goto(`/groups/${entity.groupId}/animals/${entity.animalId}/view`);
    await page.waitForSelector('.animal-detail-page', { timeout: 10000 });

    // Open protocol modal
    await page.click('.btn-view-document');
    await expect(page.locator('.protocol-modal-overlay')).toBeVisible({ timeout: 10000 });

    // Wait for content to load
    await page.waitForTimeout(1000);

    // Take screenshot
    await page.screenshot({
      path: 'test-results/protocol-phone-390px.png',
      fullPage: false,
    });
  });

  test('small phone: mobile hint and download button visible', async ({ page, request }) => {
    const entity = await createTestAnimalWithProtocol(request, adminToken);
    createdEntity = entity;

    // Small phone viewport (iPhone SE)
    await page.setViewportSize({ width: 375, height: 667 });
    await page.goto(`/groups/${entity.groupId}/animals/${entity.animalId}/view`);
    await page.waitForSelector('.animal-detail-page', { timeout: 10000 });

    // Open protocol modal
    await page.click('.btn-view-document');
    await expect(page.locator('.protocol-modal-overlay')).toBeVisible({ timeout: 10000 });

    // Wait for content to load
    await page.waitForTimeout(1000);

    // Highlight the mobile hint and download button
    await page.evaluate(() => {
      const hint = document.querySelector('.protocol-mobile-hint');
      const button = document.querySelector('.btn-download-protocol');
      if (hint) {
        (hint as HTMLElement).style.outline = '2px solid red';
        (hint as HTMLElement).style.outlineOffset = '2px';
      }
      if (button) {
        (button as HTMLElement).style.outline = '2px solid blue';
        (button as HTMLElement).style.outlineOffset = '2px';
      }
    });

    await page.waitForTimeout(500);

    // Take screenshot
    await page.screenshot({
      path: 'test-results/protocol-phone-375px-annotated.png',
      fullPage: false,
    });
  });

  test('animal detail page on mobile', async ({ page, request }) => {
    const entity = await createTestAnimalWithProtocol(request, adminToken);
    createdEntity = entity;

    // Phone viewport
    await page.setViewportSize({ width: 390, height: 844 });
    await page.goto(`/groups/${entity.groupId}/animals/${entity.animalId}/view`);
    await page.waitForSelector('.animal-detail-page', { timeout: 10000 });

    // Scroll to protocol section
    await page.evaluate(() => {
      const protocolSection = document.querySelector('.protocol-document-section');
      if (protocolSection) {
        protocolSection.scrollIntoView({ behavior: 'smooth', block: 'center' });
      }
    });

    await page.waitForTimeout(1000);

    // Take screenshot
    await page.screenshot({
      path: 'test-results/animal-detail-protocol-section-mobile.png',
      fullPage: false,
    });
  });
});
