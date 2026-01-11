import { test, expect } from '@playwright/test';
import { loginAsGroupAdmin } from './helpers/auth';

test.describe('Dark Mode Activity Filter Bar', () => {
  test.beforeEach(async ({ page }) => {
    await loginAsGroupAdmin(page);
  });

  test('activity filter bar has correct dark mode colors', async ({ page }) => {
    // Navigate to a group page
    await page.goto('/groups/1');
    
    // Wait for activity filter bar to load
    await page.waitForSelector('.activity-filter-bar', { timeout: 10000 });
    
    // Enable dark mode
    const themeToggle = page.locator('button[aria-label*="theme"], button:has-text("Dark"), button:has-text("Light")');
    if (await themeToggle.count() > 0) {
      const currentTheme = await page.evaluate(() => document.documentElement.getAttribute('data-theme'));
      if (currentTheme !== 'dark') {
        await themeToggle.click();
        await page.waitForTimeout(500); // Wait for theme transition
      }
    } else {
      // Manually set dark mode if no toggle exists
      await page.evaluate(() => {
        document.documentElement.setAttribute('data-theme', 'dark');
      });
    }
    
    // Verify dark mode is active
    const theme = await page.evaluate(() => document.documentElement.getAttribute('data-theme'));
    expect(theme).toBe('dark');
    
    // Check activity filter bar background color
    const filterBar = page.locator('.activity-filter-bar');
    const filterBarBg = await filterBar.evaluate((el) => {
      return window.getComputedStyle(el).backgroundColor;
    });
    
    // Should be #1f2937 which is rgb(31, 41, 55)
    expect(filterBarBg).toBe('rgb(31, 41, 55)');
    
    // Check activity card background color for comparison
    const activityCard = page.locator('.activity-card').first();
    if (await activityCard.count() > 0) {
      const cardBg = await activityCard.evaluate((el) => {
        return window.getComputedStyle(el).backgroundColor;
      });
      
      // Activity card should have same background as filter bar
      expect(cardBg).toBe('rgb(31, 41, 55)');
      
      // Both should match
      expect(filterBarBg).toBe(cardBg);
    }
  });

  test('activity filter bar inputs have correct dark mode colors', async ({ page }) => {
    await page.goto('/groups/1');
    await page.waitForSelector('.activity-filter-bar', { timeout: 10000 });
    
    // Enable dark mode
    await page.evaluate(() => {
      document.documentElement.setAttribute('data-theme', 'dark');
    });
    
    // Check filter select background
    const filterSelect = page.locator('.activity-filter-bar .filter-select').first();
    const selectBg = await filterSelect.evaluate((el) => {
      return window.getComputedStyle(el).backgroundColor;
    });
    
    // Should be #111827 which is rgb(17, 24, 39)
    expect(selectBg).toBe('rgb(17, 24, 39)');
  });

  test('quick filter buttons have correct dark mode colors', async ({ page }) => {
    await page.goto('/groups/1');
    await page.waitForSelector('.activity-filter-bar', { timeout: 10000 });
    
    // Enable dark mode
    await page.evaluate(() => {
      document.documentElement.setAttribute('data-theme', 'dark');
    });
    
    // Check quick filter button background
    const quickFilterBtn = page.locator('.quick-filter-btn').first();
    if (await quickFilterBtn.count() > 0) {
      const btnBg = await quickFilterBtn.evaluate((el) => {
        return window.getComputedStyle(el).backgroundColor;
      });
      
      // Should be #1f2937 which is rgb(31, 41, 55)
      expect(btnBg).toBe('rgb(31, 41, 55)');
    }
  });

  test('dark mode colors match between filter bar and activity cards', async ({ page }) => {
    await page.goto('/groups/1');
    await page.waitForSelector('.activity-filter-bar', { timeout: 10000 });
    
    // Enable dark mode
    await page.evaluate(() => {
      document.documentElement.setAttribute('data-theme', 'dark');
    });
    
    // Get filter bar colors
    const filterBar = page.locator('.activity-filter-bar');
    const filterBarStyles = await filterBar.evaluate((el) => {
      const styles = window.getComputedStyle(el);
      return {
        background: styles.backgroundColor,
        border: styles.borderColor,
      };
    });
    
    // Get activity card colors
    const activityCard = page.locator('.activity-card').first();
    if (await activityCard.count() > 0) {
      const cardStyles = await activityCard.evaluate((el) => {
        const styles = window.getComputedStyle(el);
        return {
          background: styles.backgroundColor,
          border: styles.borderColor,
        };
      });
      
      // Both should use the same dark background
      expect(filterBarStyles.background).toBe(cardStyles.background);
      expect(filterBarStyles.background).toBe('rgb(31, 41, 55)');
      
      // Both should use the same dark border
      expect(filterBarStyles.border).toBe(cardStyles.border);
      expect(filterBarStyles.border).toBe('rgb(55, 65, 81)');
    }
  });
});
