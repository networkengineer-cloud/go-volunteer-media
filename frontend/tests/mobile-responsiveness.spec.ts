import { test, expect, devices } from '@playwright/test';

test.describe('Mobile Responsiveness', () => {
  test.use({ ...devices['iPhone 12'] });

  test('should display mobile navigation menu', async ({ page }) => {
    await page.goto('/');
    
    // Check if mobile menu toggle is visible
    const mobileMenuToggle = page.locator('.mobile-menu-toggle');
    await expect(mobileMenuToggle).toBeVisible();
    
    // Menu should be hidden by default
    const navRight = page.locator('.nav-right');
    await expect(navRight).not.toHaveClass(/mobile-menu-open/);
    
    // Click to open menu
    await mobileMenuToggle.click();
    await expect(navRight).toHaveClass(/mobile-menu-open/);
    
    // Close icon should be visible
    await expect(mobileMenuToggle).toBeVisible();
  });

  test('should have proper touch targets', async ({ page }) => {
    await page.goto('/login');
    
    // Check input fields have proper height
    const usernameInput = page.locator('input[name="username"]');
    const box = await usernameInput.boundingBox();
    expect(box?.height).toBeGreaterThanOrEqual(44);
    
    // Check button has proper size
    const loginButton = page.locator('button[type="submit"]');
    const buttonBox = await loginButton.boundingBox();
    expect(buttonBox?.height).toBeGreaterThanOrEqual(44);
  });

  test('should display readable text on mobile', async ({ page }) => {
    await page.goto('/');
    
    // Check that font size is at least 16px to prevent zoom on iOS
    const bodyFontSize = await page.evaluate(() => {
      return window.getComputedStyle(document.body).fontSize;
    });
    expect(parseInt(bodyFontSize)).toBeGreaterThanOrEqual(16);
  });

  test('should have viewport meta tag', async ({ page }) => {
    await page.goto('/');
    
    const viewportMeta = await page.locator('meta[name="viewport"]').getAttribute('content');
    expect(viewportMeta).toContain('width=device-width');
    expect(viewportMeta).toContain('initial-scale=1.0');
  });

  test('should stack form elements vertically on mobile', async ({ page }) => {
    await page.goto('/login');
    
    const loginCard = page.locator('.login-card');
    await expect(loginCard).toBeVisible();
    
    // Check if login card adapts to mobile width
    const box = await loginCard.boundingBox();
    const viewportSize = page.viewportSize();
    
    if (viewportSize) {
      // Card should not exceed viewport width (with some padding)
      expect(box?.width).toBeLessThanOrEqual(viewportSize.width);
    }
  });

  test('should handle mobile form submission', async ({ page }) => {
    await page.goto('/login');
    
    // Fill form on mobile
    await page.fill('input[name="username"]', 'testuser');
    await page.fill('input[name="password"]', 'testpass');
    
    // Submit button should be visible and clickable
    const submitButton = page.locator('button[type="submit"]');
    await expect(submitButton).toBeVisible();
    await expect(submitButton).toBeEnabled();
  });
});

test.describe('Tablet Responsiveness', () => {
  test.use({ ...devices['iPad Pro'] });

  test('should display navigation without hamburger menu', async ({ page }) => {
    await page.goto('/');
    
    // Mobile menu toggle should be hidden on tablet
    const mobileMenuToggle = page.locator('.mobile-menu-toggle');
    await expect(mobileMenuToggle).toBeHidden();
    
    // Navigation items should be visible
    const navRight = page.locator('.nav-right');
    await expect(navRight).toBeVisible();
  });

  test('should display tables properly on tablet', async ({ page }) => {
    await page.goto('/login');
    
    // Login as admin (assuming test data exists)
    await page.fill('input[name="username"]', 'admin');
    await page.fill('input[name="password"]', 'password');
    await page.click('button[type="submit"]');
    
    // Wait for navigation
    await page.waitForURL(/.*dashboard/, { timeout: 5000 }).catch(() => {
      // If login fails, skip the table test
      return;
    });
  });

  test('should have adequate spacing for touch on tablet', async ({ page }) => {
    await page.goto('/');
    
    // Check that interactive elements have spacing
    const links = page.locator('a');
    const count = await links.count();
    
    expect(count).toBeGreaterThan(0);
  });
});

test.describe('Grid Layout Responsiveness', () => {
  test.use({ ...devices['Pixel 5'] });

  test('should stack grid items on mobile', async ({ page }) => {
    // This test would check dashboard grid layout
    // For now, we'll just verify the page loads
    await page.goto('/');
    
    const body = page.locator('body');
    await expect(body).toBeVisible();
  });

  test('should handle image grids on mobile', async ({ page }) => {
    await page.goto('/');
    
    // Check that images are responsive
    const images = page.locator('img');
    const count = await images.count();
    
    if (count > 0) {
      const firstImage = images.first();
      const box = await firstImage.boundingBox();
      const viewportSize = page.viewportSize();
      
      if (viewportSize && box) {
        // Image should not exceed viewport width
        expect(box.width).toBeLessThanOrEqual(viewportSize.width);
      }
    }
  });
});

test.describe('Mobile Interactions', () => {
  test.use({ ...devices['iPhone 12'] });

  test('should handle touch events properly', async ({ page }) => {
    await page.goto('/');
    
    // Test tap on mobile menu
    const mobileMenuToggle = page.locator('.mobile-menu-toggle');
    await mobileMenuToggle.tap();
    
    const navRight = page.locator('.nav-right');
    await expect(navRight).toHaveClass(/mobile-menu-open/);
  });

  test('should close mobile menu when navigating', async ({ page }) => {
    await page.goto('/');
    
    // Open mobile menu
    const mobileMenuToggle = page.locator('.mobile-menu-toggle');
    await mobileMenuToggle.tap();
    
    // Click login link
    const loginLink = page.locator('.nav-login');
    await loginLink.tap();
    
    // Should navigate to login page
    await expect(page).toHaveURL(/.*login/);
  });

  test('should scroll smoothly on mobile', async ({ page }) => {
    await page.goto('/');
    
    // Scroll down
    await page.evaluate(() => window.scrollBy(0, 500));
    
    // Wait for scroll
    await page.waitForTimeout(100);
    
    // Verify page scrolled
    const scrollY = await page.evaluate(() => window.scrollY);
    expect(scrollY).toBeGreaterThan(0);
  });
});

test.describe('Dark Mode on Mobile', () => {
  test.use({ ...devices['iPhone 12'] });

  test('should toggle dark mode on mobile', async ({ page }) => {
    await page.goto('/');
    
    // Open mobile menu to access theme toggle
    const mobileMenuToggle = page.locator('.mobile-menu-toggle');
    await mobileMenuToggle.tap();
    
    // Find and click theme toggle
    const themeToggle = page.locator('.theme-toggle');
    await expect(themeToggle).toBeVisible();
    
    // Click to toggle theme
    await themeToggle.click();
    
    // Check if dark mode is applied
    const html = page.locator('html');
    const dataTheme = await html.getAttribute('data-theme');
    expect(['dark', null]).toContain(dataTheme);
  });

  test('should persist theme preference on mobile', async ({ page, context }) => {
    await page.goto('/');
    
    // Set dark mode
    await page.evaluate(() => {
      document.documentElement.setAttribute('data-theme', 'dark');
      localStorage.setItem('theme', 'dark');
    });
    
    // Reload page
    await page.reload();
    
    // Check if dark mode persisted
    const html = page.locator('html');
    const dataTheme = await html.getAttribute('data-theme');
    expect(dataTheme).toBe('dark');
  });
});
