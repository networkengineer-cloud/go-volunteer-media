import { expect, type Page } from '@playwright/test';

type Credentials = {
  username: string;
  password: string;
};

const ADMIN: Credentials = {
  username: process.env.E2E_ADMIN_USERNAME ?? 'admin',
  password: process.env.E2E_ADMIN_PASSWORD ?? 'demo1234',
};

const GROUP_ADMIN: Credentials = {
  username: process.env.E2E_GROUP_ADMIN_USERNAME ?? 'merry',
  password: process.env.E2E_GROUP_ADMIN_PASSWORD ?? 'demo1234',
};

const VOLUNTEER: Credentials = {
  username: process.env.E2E_VOLUNTEER_USERNAME ?? 'terry',
  password: process.env.E2E_VOLUNTEER_PASSWORD ?? 'demo1234',
};

export const testUsers = {
  admin: ADMIN,
  groupAdmin: GROUP_ADMIN,
  volunteer: VOLUNTEER,
};

export async function login(page: Page, credentials: Credentials, options?: { waitForUrl?: RegExp }) {
  // Ensure we start from a clean auth state per test.
  // Some engines can retain storage more aggressively than expected.
  await page.goto('/');
  await page
    .evaluate(() => {
      try {
        localStorage.removeItem('token');
      } catch {
        // ignore
      }
      try {
        sessionStorage.clear();
      } catch {
        // ignore
      }
    })
    .catch(() => {});

  await page.goto('/login');

  const usernameInput = page.getByRole('textbox', { name: /^username/i });
  await expect(usernameInput).toBeVisible();
  await usernameInput.fill(credentials.username);

  // Use role-based selector to avoid strict-mode collisions with the
  // "Show password" toggle button.
  const passwordInput = page.getByRole('textbox', { name: /^password/i });
  await expect(passwordInput).toBeVisible();
  await passwordInput.fill(credentials.password);

  await page.getByRole('button', { name: /^login$/i }).click();

  // Prefer a token-based wait (SPA login may not trigger a navigation event).
  await expect
    .poll(async () => page.evaluate(() => localStorage.getItem('token')), {
      timeout: 15000,
    })
    .not.toBeNull();

  // Best-effort URL wait for callers that rely on post-login routing.
  await page
    .waitForURL(options?.waitForUrl ?? /\/(dashboard|groups)/i, { timeout: 15000 })
    .catch(() => {});

  // Confirm we're not stuck on the login page.
  await expect(page.getByRole('button', { name: /^logout$/i })).toBeVisible({ timeout: 15000 });
}

export async function loginAsAdmin(page: Page, options?: { waitForUrl?: RegExp }) {
  return login(page, ADMIN, options);
}

export async function loginAsGroupAdmin(page: Page, options?: { waitForUrl?: RegExp }) {
  return login(page, GROUP_ADMIN, options);
}

export async function loginAsVolunteer(page: Page, options?: { waitForUrl?: RegExp }) {
  return login(page, VOLUNTEER, options);
}
