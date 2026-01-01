import { test, expect } from '@playwright/test';
import { loginAsAdmin } from './helpers/auth';

/**
 * Dark Mode Field Fixes Test
 * 
 * Tests for the three specific dark mode issues:
 * 1. Login page - username field shows with white background
 * 2. Activity feed page - filters section shows as light mode
 * 3. Password setup page - password field shows with white background
 */

test.describe('Dark Mode Field Fixes', () => {
  test('login page - username and password fields should have dark backgrounds in dark mode', async ({ page }) => {
    // Enable dark mode before navigating
    await page.goto('/login');
    await page.evaluate(() => {
      document.documentElement.setAttribute('data-theme', 'dark');
      localStorage.setItem('theme', 'dark');
    });
    
    // Wait for page to render
    await page.waitForSelector('.login-container');
    
    // Take screenshot before checking
    await page.screenshot({ path: 'playwright-report/login-dark-mode-before.png', fullPage: true });
    
    // Check username field (FormField component)
    const usernameInput = page.locator('#username');
    await expect(usernameInput).toBeVisible();
    
    const usernameStyles = await usernameInput.evaluate((el) => {
      const styles = window.getComputedStyle(el);
      return {
        background: styles.backgroundColor,
        color: styles.color,
        border: styles.borderColor
      };
    });
    
    console.log('Login - Username field styles in dark mode:', usernameStyles);
    
    // Background should NOT be white (rgb(255, 255, 255))
    expect(usernameStyles.background).not.toBe('rgb(255, 255, 255)');
    expect(usernameStyles.background).not.toBe('rgba(0, 0, 0, 0)'); // Not transparent
    
    // Text color should be light (for readability on dark background)
    const colorMatch = usernameStyles.color.match(/rgb[a]?\((\d+),\s*(\d+),\s*(\d+)/);
    if (colorMatch) {
      const [, r, g, b] = colorMatch.map(Number);
      const avgColor = (r + g + b) / 3;
      expect(avgColor).toBeGreaterThan(150); // Light text
      console.log('Username text color average:', avgColor);
    }
    
    // Check password field (PasswordField component, which uses FormField internally)
    const passwordInput = page.locator('#password');
    await expect(passwordInput).toBeVisible();
    
    const passwordStyles = await passwordInput.evaluate((el) => {
      const styles = window.getComputedStyle(el);
      return {
        background: styles.backgroundColor,
        color: styles.color,
        border: styles.borderColor
      };
    });
    
    console.log('Login - Password field styles in dark mode:', passwordStyles);
    
    // Background should NOT be white
    expect(passwordStyles.background).not.toBe('rgb(255, 255, 255)');
    expect(passwordStyles.background).not.toBe('rgba(0, 0, 0, 0)');
    
    // Text color should be light
    const passwordColorMatch = passwordStyles.color.match(/rgb[a]?\((\d+),\s*(\d+),\s*(\d+)/);
    if (passwordColorMatch) {
      const [, r, g, b] = passwordColorMatch.map(Number);
      const avgColor = (r + g + b) / 3;
      expect(avgColor).toBeGreaterThan(150);
      console.log('Password text color average:', avgColor);
    }
    
    // Take screenshot after verifying
    await page.screenshot({ path: 'playwright-report/login-dark-mode-after.png', fullPage: true });
  });

  test('activity feed page - filter inputs should have dark backgrounds in dark mode', async ({ page }) => {
    await loginAsAdmin(page);
    
    // Enable dark mode
    await page.evaluate(() => {
      document.documentElement.setAttribute('data-theme', 'dark');
      localStorage.setItem('theme', 'dark');
    });
    
    // Navigate to activity feed (assuming group 1 exists)
    await page.goto('/dashboard');
    
    // Try to navigate to a group first
    const groupCard = page.locator('.group-card').first();
    if (await groupCard.isVisible()) {
      await groupCard.click();
      
      // Look for activity feed link
      const activityFeedLink = page.locator('a[href*="/activity-feed"]').first();
      if (await activityFeedLink.isVisible()) {
        await activityFeedLink.click();
      } else {
        // Direct navigation if link not found
        await page.goto('/groups/1/activity-feed');
      }
    } else {
      // Direct navigation if no groups
      await page.goto('/groups/1/activity-feed');
    }
    
    // Wait for page to load
    await page.waitForSelector('.activity-feed-page', { timeout: 10000 });
    
    // Expand filters if collapsed
    const filterToggleBtn = page.locator('.filter-toggle-btn');
    if (await filterToggleBtn.isVisible()) {
      const isExpanded = await filterToggleBtn.getAttribute('aria-expanded');
      if (isExpanded === 'false') {
        await filterToggleBtn.click();
        await page.waitForTimeout(500); // Wait for animation
      }
    }
    
    // Take screenshot
    await page.screenshot({ path: 'playwright-report/activity-feed-dark-mode.png', fullPage: true });
    
    // Check filter select elements
    const filterSelect = page.locator('.filter-select').first();
    if (await filterSelect.isVisible()) {
      const selectStyles = await filterSelect.evaluate((el) => {
        const styles = window.getComputedStyle(el);
        return {
          background: styles.backgroundColor,
          color: styles.color,
          border: styles.borderColor
        };
      });
      
      console.log('Activity Feed - Filter select styles in dark mode:', selectStyles);
      
      // Background should NOT be white
      expect(selectStyles.background).not.toBe('rgb(255, 255, 255)');
      expect(selectStyles.background).not.toBe('rgba(0, 0, 0, 0)');
      
      // Text color should be light
      const colorMatch = selectStyles.color.match(/rgb[a]?\((\d+),\s*(\d+),\s*(\d+)/);
      if (colorMatch) {
        const [, r, g, b] = colorMatch.map(Number);
        const avgColor = (r + g + b) / 3;
        expect(avgColor).toBeGreaterThan(150);
        console.log('Filter select text color average:', avgColor);
      }
    }
    
    // Check filter input (date inputs)
    const filterInput = page.locator('.filter-input').first();
    if (await filterInput.isVisible()) {
      const inputStyles = await filterInput.evaluate((el) => {
        const styles = window.getComputedStyle(el);
        return {
          background: styles.backgroundColor,
          color: styles.color,
          border: styles.borderColor
        };
      });
      
      console.log('Activity Feed - Filter input styles in dark mode:', inputStyles);
      
      // Background should NOT be white
      expect(inputStyles.background).not.toBe('rgb(255, 255, 255)');
      expect(inputStyles.background).not.toBe('rgba(0, 0, 0, 0)');
      
      // Text color should be light
      const colorMatch = inputStyles.color.match(/rgb[a]?\((\d+),\s*(\d+),\s*(\d+)/);
      if (colorMatch) {
        const [, r, g, b] = colorMatch.map(Number);
        const avgColor = (r + g + b) / 3;
        expect(avgColor).toBeGreaterThan(150);
        console.log('Filter input text color average:', avgColor);
      }
    }
  });

  test('password setup page - password fields should have dark backgrounds in dark mode', async ({ page }) => {
    // Navigate to setup password page (with a fake token, we'll just check the styling)
    await page.goto('/setup-password?token=fake-token-for-testing');
    
    // Enable dark mode
    await page.evaluate(() => {
      document.documentElement.setAttribute('data-theme', 'dark');
      localStorage.setItem('theme', 'dark');
    });
    
    // Wait for page to render
    await page.waitForSelector('.login-container');
    
    // Take screenshot
    await page.screenshot({ path: 'playwright-report/setup-password-dark-mode.png', fullPage: true });
    
    // Check new password field
    const newPasswordInput = page.locator('#new-password');
    await expect(newPasswordInput).toBeVisible();
    
    const newPasswordStyles = await newPasswordInput.evaluate((el) => {
      const styles = window.getComputedStyle(el);
      return {
        background: styles.backgroundColor,
        color: styles.color,
        border: styles.borderColor
      };
    });
    
    console.log('Setup Password - New password field styles in dark mode:', newPasswordStyles);
    
    // Background should NOT be white
    expect(newPasswordStyles.background).not.toBe('rgb(255, 255, 255)');
    expect(newPasswordStyles.background).not.toBe('rgba(0, 0, 0, 0)');
    
    // Text color should be light
    const colorMatch = newPasswordStyles.color.match(/rgb[a]?\((\d+),\s*(\d+),\s*(\d+)/);
    if (colorMatch) {
      const [, r, g, b] = colorMatch.map(Number);
      const avgColor = (r + g + b) / 3;
      expect(avgColor).toBeGreaterThan(150);
      console.log('New password text color average:', avgColor);
    }
    
    // Check confirm password field
    const confirmPasswordInput = page.locator('#confirm-password');
    await expect(confirmPasswordInput).toBeVisible();
    
    const confirmPasswordStyles = await confirmPasswordInput.evaluate((el) => {
      const styles = window.getComputedStyle(el);
      return {
        background: styles.backgroundColor,
        color: styles.color,
        border: styles.borderColor
      };
    });
    
    console.log('Setup Password - Confirm password field styles in dark mode:', confirmPasswordStyles);
    
    // Background should NOT be white
    expect(confirmPasswordStyles.background).not.toBe('rgb(255, 255, 255)');
    expect(confirmPasswordStyles.background).not.toBe('rgba(0, 0, 0, 0)');
    
    // Text color should be light
    const confirmColorMatch = confirmPasswordStyles.color.match(/rgb[a]?\((\d+),\s*(\d+),\s*(\d+)/);
    if (confirmColorMatch) {
      const [, r, g, b] = confirmColorMatch.map(Number);
      const avgColor = (r + g + b) / 3;
      expect(avgColor).toBeGreaterThan(150);
      console.log('Confirm password text color average:', avgColor);
    }
  });

  test('visual regression - all three pages with dark mode enabled', async ({ page }) => {
    // This test creates side-by-side screenshots for visual comparison
    
    // 1. Login page
    await page.goto('/login');
    await page.evaluate(() => {
      document.documentElement.setAttribute('data-theme', 'dark');
      localStorage.setItem('theme', 'dark');
    });
    await page.waitForSelector('.login-container');
    await page.screenshot({ 
      path: 'playwright-report/dark-mode-login-full.png', 
      fullPage: true 
    });
    
    // 2. Setup Password page
    await page.goto('/setup-password?token=test-token');
    await page.waitForSelector('.login-container');
    await page.screenshot({ 
      path: 'playwright-report/dark-mode-setup-password-full.png', 
      fullPage: true 
    });
    
    // 3. Activity Feed page (requires auth)
    await loginAsAdmin(page);
    await page.evaluate(() => {
      document.documentElement.setAttribute('data-theme', 'dark');
      localStorage.setItem('theme', 'dark');
    });
    
    // Navigate to activity feed
    await page.goto('/dashboard');
    const groupCard = page.locator('.group-card').first();
    if (await groupCard.isVisible()) {
      await groupCard.click();
      const activityLink = page.locator('a[href*="activity-feed"]').first();
      if (await activityLink.isVisible()) {
        await activityLink.click();
      } else {
        await page.goto('/groups/1/activity-feed');
      }
    } else {
      await page.goto('/groups/1/activity-feed');
    }
    
    await page.waitForSelector('.activity-feed-page', { timeout: 10000 });
    
    // Expand filters
    const filterToggle = page.locator('.filter-toggle-btn');
    if (await filterToggle.isVisible()) {
      const isExpanded = await filterToggle.getAttribute('aria-expanded');
      if (isExpanded === 'false') {
        await filterToggle.click();
        await page.waitForTimeout(500);
      }
    }
    
    await page.screenshot({ 
      path: 'playwright-report/dark-mode-activity-feed-full.png', 
      fullPage: true 
    });
    
    console.log('✅ Visual regression screenshots saved to playwright-report/');
  });
});
