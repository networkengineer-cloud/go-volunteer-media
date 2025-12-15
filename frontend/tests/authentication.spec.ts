import { test, expect } from '@playwright/test';
import { loginAsAdmin, loginAsVolunteer, testUsers } from './helpers/auth';

/**
 * Authentication E2E Tests
 *
 * Uses seeded demo accounts (see cmd/seed output):
 * - admin / demo1234
 * - terry / demo1234
 */

test.describe('Authentication', () => {
  test('admin can login and receives a JWT token', async ({ page }) => {
    await loginAsAdmin(page, { waitForUrl: /\/(dashboard|groups)/i });

    const token = await page.evaluate(() => localStorage.getItem('token'));
    expect(token).toBeTruthy();
    expect(token).toContain('eyJ');
    expect(page.url()).not.toContain('/login');
  });

  test('regular user can login', async ({ page }) => {
    await loginAsVolunteer(page, { waitForUrl: /\/(dashboard|groups)/i });

    const token = await page.evaluate(() => localStorage.getItem('token'));
    expect(token).toBeTruthy();
    expect(page.url()).not.toContain('/login');
  });

  test('invalid credentials do not create a token', async ({ page }) => {
    await page.goto('/login');
    await page.evaluate(() => localStorage.clear());

    await page.getByLabel('Username').fill('nonexistent');
    await page.getByRole('textbox', { name: /^password/i }).fill('wrongpassword');
    await page.getByRole('button', { name: /^login$/i }).click();

    await expect(page).toHaveURL(/\/login/i);
    const token = await page.evaluate(() => localStorage.getItem('token'));
    expect(token).toBeNull();

    await expect(page.locator('#main-content .error[role="alert"]')).toBeVisible();
  });

  test('session token persists after reload', async ({ page }) => {
    await loginAsAdmin(page, { waitForUrl: /\/(dashboard|groups)/i });

    const tokenBefore = await page.evaluate(() => localStorage.getItem('token'));
    expect(tokenBefore).toBeTruthy();

    await page.reload();
    await page.waitForTimeout(500);

    const tokenAfter = await page.evaluate(() => localStorage.getItem('token'));
    expect(tokenAfter).toBe(tokenBefore);
    expect(page.url()).not.toContain('/login');
  });

  test('logout clears token (if logout control exists)', async ({ page }) => {
    await loginAsVolunteer(page, { waitForUrl: /\/(dashboard|groups)/i });

    const logoutSelectors = [
      'button:has-text("Logout")',
      'a:has-text("Logout")',
      '[data-testid="logout"]',
      '[aria-label*="logout" i]',
    ];

    let clicked = false;
    for (const selector of logoutSelectors) {
      const el = page.locator(selector).first();
      if (await el.isVisible().catch(() => false)) {
        await el.click();
        clicked = true;
        break;
      }
    }

    if (!clicked) test.skip(true, 'Logout control not found');

    await expect(page).toHaveURL(/\/(login|$)/);
    const token = await page.evaluate(() => localStorage.getItem('token'));
    expect(token).toBeNull();
  });
});

test.describe('Authorization routing', () => {
  test('non-admin does not see Users navigation', async ({ page }) => {
    await loginAsVolunteer(page, { waitForUrl: /\/(dashboard|groups)/i });
    await expect(page.getByRole('link', { name: /^users$/i })).toHaveCount(0);
  });

  test('admin sees Users navigation', async ({ page }) => {
    await loginAsAdmin(page, { waitForUrl: /\/(dashboard|groups)/i });
    await expect(page.getByRole('link', { name: /^users$/i })).toBeVisible();
    expect(testUsers.admin.username).toBe('admin');
  });
});
