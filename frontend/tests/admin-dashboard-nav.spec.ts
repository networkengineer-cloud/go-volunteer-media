import { test, expect } from '@playwright/test';
import { loginAsAdmin } from './helpers/auth';

test.describe('Admin nav → dashboard', () => {
  test('clicking Admin in the nav goes to the admin dashboard', async ({ page }) => {
    await loginAsAdmin(page, { waitForUrl: /\/(dashboard|groups)/i });

    await page.getByRole('link', { name: 'Admin', exact: true }).click();

    await expect(page).toHaveURL(/\/admin\/dashboard$/);
    await expect(page.locator('h1')).toContainText('Admin Dashboard');
  });

  test('the dashboard Quick Links reach Site Settings and API Tokens', async ({ page }) => {
    await loginAsAdmin(page, { waitForUrl: /\/(dashboard|groups)/i });
    await page.goto('/admin/dashboard');

    await page.getByRole('link', { name: 'Site Settings' }).click();
    await expect(page).toHaveURL(/\/admin\/site-settings$/);

    await page.goto('/admin/dashboard');
    await page.getByRole('link', { name: 'API Tokens' }).click();
    await expect(page).toHaveURL(/\/admin\/api-tokens$/);
  });
});
