import { Page } from '@playwright/test';

/**
 * API Mock Helpers for Playwright Tests
 */

export interface MockUser {
  id: number;
  username: string;
  email: string;
  is_admin: boolean;
  created_at: string;
  updated_at: string;
}

export async function mockLoginSuccess(page: Page, user: MockUser, token: string = 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.test') {
  await page.route('**/api/login', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ token, user })
    });
  });
}

export async function mockLoginFailure(page: Page, message: string = 'Invalid credentials') {
  await page.route('**/api/login', async (route) => {
    await route.fulfill({
      status: 401,
      contentType: 'application/json',
      body: JSON.stringify({ error: message })
    });
  });
}
