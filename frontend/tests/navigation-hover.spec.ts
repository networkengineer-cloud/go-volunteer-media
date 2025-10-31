import { test, expect } from '@playwright/test';

/**
 * Navigation Hover Visibility Tests
 * 
 * These tests validate that navigation items remain visible when hovering
 * in both light and dark modes. This prevents regression of the issue where
 * hover text would blend into the background in light mode.
 */

const BASE_URL = 'http://localhost:5173';

test.describe('Navigation Hover Visibility - Light Mode', () => {
  test.beforeEach(async ({ page }) => {
    // Navigate to home page
    await page.goto(BASE_URL);
    
    // Ensure we're in light mode by checking the theme attribute
    await page.evaluate(() => {
      document.documentElement.removeAttribute('data-theme');
      localStorage.setItem('theme', 'light');
    });
    
    // Wait for page to be ready
    await page.waitForLoadState('domcontentloaded');
  });

  test('brand logo should be visible on hover in light mode', async ({ page }) => {
    const brandLink = page.locator('.nav-brand');
    
    // Verify the brand link exists
    await expect(brandLink).toBeVisible();
    
    // Hover over the brand link
    await brandLink.hover();
    
    // Check the computed color is not the background brand color
    const color = await brandLink.evaluate((el) => {
      return window.getComputedStyle(el).color;
    });
    
    // The hover color should be white (rgb(255, 255, 255))
    // Not the brand green which would make it invisible
    expect(color).toContain('255, 255, 255');
  });

  test('login link should be visible on hover in light mode', async ({ page }) => {
    const loginLink = page.locator('.nav-right a').filter({ hasText: 'Login' });
    
    // Verify the login link exists
    await expect(loginLink).toBeVisible();
    
    // Hover over the login link
    await loginLink.hover();
    
    // Check the computed color is white on hover
    const color = await loginLink.evaluate((el) => {
      return window.getComputedStyle(el).color;
    });
    
    // The hover color should be white (rgb(255, 255, 255))
    expect(color).toContain('255, 255, 255');
  });

  test('theme toggle should be visible on hover in light mode', async ({ page }) => {
    const themeToggle = page.locator('.theme-toggle');
    
    // Verify the theme toggle exists
    await expect(themeToggle).toBeVisible();
    
    // Hover over the theme toggle
    await themeToggle.hover();
    
    // Check the computed color is white on hover
    const color = await themeToggle.evaluate((el) => {
      return window.getComputedStyle(el).color;
    });
    
    // The hover color should be white (rgb(255, 255, 255))
    expect(color).toContain('255, 255, 255');
  });

  test('navigation background should remain dark green', async ({ page }) => {
    const navigation = page.locator('.navigation');
    
    // Verify the navigation exists
    await expect(navigation).toBeVisible();
    
    // Check the background color is the brand green
    const backgroundColor = await navigation.evaluate((el) => {
      return window.getComputedStyle(el).backgroundColor;
    });
    
    // The background should be the brand green (rgb(0, 107, 84))
    expect(backgroundColor).toContain('0, 107, 84');
  });
});

test.describe('Navigation Hover Visibility - Dark Mode', () => {
  test.beforeEach(async ({ page }) => {
    // Navigate to home page
    await page.goto(BASE_URL);
    
    // Switch to dark mode
    await page.evaluate(() => {
      document.documentElement.setAttribute('data-theme', 'dark');
      localStorage.setItem('theme', 'dark');
    });
    
    // Wait for page to be ready
    await page.waitForLoadState('domcontentloaded');
  });

  test('brand logo should be visible on hover in dark mode', async ({ page }) => {
    const brandLink = page.locator('.nav-brand');
    
    // Verify the brand link exists
    await expect(brandLink).toBeVisible();
    
    // Hover over the brand link
    await brandLink.hover();
    
    // In dark mode, the text should still be visible (white)
    const color = await brandLink.evaluate((el) => {
      return window.getComputedStyle(el).color;
    });
    
    // The hover color should be white (rgb(255, 255, 255))
    expect(color).toContain('255, 255, 255');
  });

  test('navigation links should be visible on hover in dark mode', async ({ page }) => {
    const loginLink = page.locator('.nav-right a').filter({ hasText: 'Login' });
    
    // Verify the login link exists
    await expect(loginLink).toBeVisible();
    
    // Hover over the login link
    await loginLink.hover();
    
    // Check the computed color is white on hover
    const color = await loginLink.evaluate((el) => {
      return window.getComputedStyle(el).color;
    });
    
    // The hover color should be white (rgb(255, 255, 255))
    expect(color).toContain('255, 255, 255');
  });

  test('theme toggle should be visible on hover in dark mode', async ({ page }) => {
    const themeToggle = page.locator('.theme-toggle');
    
    // Verify the theme toggle exists
    await expect(themeToggle).toBeVisible();
    
    // Hover over the theme toggle
    await themeToggle.hover();
    
    // Check the computed color is white on hover
    const color = await themeToggle.evaluate((el) => {
      return window.getComputedStyle(el).color;
    });
    
    // The hover color should be white (rgb(255, 255, 255))
    expect(color).toContain('255, 255, 255');
  });
});

test.describe('Navigation Hover Visual Feedback', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto(BASE_URL);
  });

  test('hovering should provide visual feedback with background change', async ({ page }) => {
    const loginLink = page.locator('.nav-right a').filter({ hasText: 'Login' });
    
    // Get the initial background color
    const initialBg = await loginLink.evaluate((el) => {
      return window.getComputedStyle(el).backgroundColor;
    });
    
    // Hover over the link
    await loginLink.hover();
    
    // Get the hover background color
    const hoverBg = await loginLink.evaluate((el) => {
      return window.getComputedStyle(el).backgroundColor;
    });
    
    // The background should change on hover (become more opaque/visible)
    expect(hoverBg).not.toBe(initialBg);
  });

  test('brand logo should show opacity change on hover', async ({ page }) => {
    const brandLink = page.locator('.nav-brand');
    
    // Hover over the brand link
    await brandLink.hover();
    
    // Check that opacity is set on hover
    const opacity = await brandLink.evaluate((el) => {
      return window.getComputedStyle(el).opacity;
    });
    
    // The opacity should be 0.9 on hover
    expect(parseFloat(opacity)).toBeCloseTo(0.9, 1);
  });
});
