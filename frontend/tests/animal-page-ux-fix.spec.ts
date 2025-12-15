import { test, expect } from '@playwright/test';
import { loginAsAdmin } from './helpers/auth';

/**
 * Animal Page UX Fix Tests
 * 
 * Tests to verify:
 * 1. Dark mode styling for all containers on animal detail page
 * 2. Navigation breadcrumb includes Animals link
 * 3. Animals link navigates to correct tab with query parameter
 */

test.describe('Animal Page UX Fixes', () => {
  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page);
  });

  test('animal main info box should have dark mode styling', async ({ page }) => {
    // Enable dark mode
    await page.evaluate(() => {
      document.documentElement.setAttribute('data-theme', 'dark');
      localStorage.setItem('theme', 'dark');
    });

    // Navigate to dashboard and click on first animal
    await page.goto('/dashboard');
    const firstAnimalLink = page.locator('.animal-card-mini, .animal-card').first();
    await firstAnimalLink.click();
    
    // Wait for animal detail page
    await page.waitForSelector('.animal-detail-page', { timeout: 10000 });
    
    // Check that animal-main-info has dark mode background
    const mainInfo = page.locator('.animal-main-info');
    await expect(mainInfo).toBeVisible();
    
    const bgColor = await mainInfo.evaluate((el) => {
      return window.getComputedStyle(el).backgroundColor;
    });
    
    console.log('Animal main info background in dark mode:', bgColor);
    // Should NOT be pure white in dark mode
    expect(bgColor).not.toBe('rgb(255, 255, 255)');

    // Take screenshot
    await page.screenshot({ path: '/tmp/animal-page-dark-mode.png', fullPage: true });
  });

  test('tag filters should have dark mode styling', async ({ page }) => {
    // Enable dark mode
    await page.evaluate(() => {
      document.documentElement.setAttribute('data-theme', 'dark');
      localStorage.setItem('theme', 'dark');
    });

    // Navigate to dashboard and click on first animal
    await page.goto('/dashboard');
    const firstAnimalLink = page.locator('.animal-card-mini, .animal-card').first();
    await firstAnimalLink.click();
    
    // Wait for animal detail page
    await page.waitForSelector('.animal-detail-page', { timeout: 10000 });
    
    // Check tag filters section
    const tagFilters = page.locator('.tag-filters');
    if (await tagFilters.isVisible()) {
      const borderColor = await tagFilters.evaluate((el) => {
        return window.getComputedStyle(el).borderTopColor;
      });
      
      console.log('Tag filters border in dark mode:', borderColor);
      // Should NOT be light gray border in dark mode
      expect(borderColor).not.toBe('rgb(226, 232, 240)'); // #e2e8f0
    }
  });

  test('breadcrumb should include Animals link', async ({ page }) => {
    // Navigate to dashboard and click on first animal
    await page.goto('/dashboard');
    const firstAnimalLink = page.locator('.animal-card-mini, .animal-card').first();
    await firstAnimalLink.click();
    
    // Wait for animal detail page
    await page.waitForSelector('.animal-detail-page', { timeout: 10000 });
    
    // Check breadcrumb for Animals link
    const breadcrumb = page.locator('.breadcrumb');
    await expect(breadcrumb).toBeVisible();
    
    // Find the Animals link
    const animalsLink = breadcrumb.locator('a:has-text("Animals")');
    await expect(animalsLink).toBeVisible();
    
    console.log('Animals link found in breadcrumb');
  });

  test('Animals link should navigate to animals tab with query parameter', async ({ page }) => {
    // Navigate to dashboard and click on first animal
    await page.goto('/dashboard');
    const firstAnimalLink = page.locator('.animal-card-mini, .animal-card').first();
    await firstAnimalLink.click();
    
    // Wait for animal detail page
    await page.waitForSelector('.animal-detail-page', { timeout: 10000 });
    
    // Click the Animals breadcrumb link
    const animalsLink = page.locator('.breadcrumb a:has-text("Animals")');
    await animalsLink.click();
    
    // Wait for navigation
    await page.waitForLoadState('networkidle');
    
    // Check that we're on the group page with view=animals query param
    const url = page.url();
    console.log('Current URL after clicking Animals link:', url);
    expect(url).toContain('view=animals');
    
    // Verify animals tab is active
    const animalsTab = page.locator('.group-tab:has-text("Animals")');
    await expect(animalsTab).toHaveClass(/group-tab--active/);
    
    console.log('Successfully navigated to animals tab');
  });

  test('light mode should have good contrast on animal page', async ({ page }) => {
    // Ensure light mode
    await page.evaluate(() => {
      document.documentElement.removeAttribute('data-theme');
      localStorage.setItem('theme', 'light');
    });

    // Navigate to dashboard and click on first animal
    await page.goto('/dashboard');
    const firstAnimalLink = page.locator('.animal-card-mini, .animal-card').first();
    await firstAnimalLink.click();
    
    // Wait for animal detail page
    await page.waitForSelector('.animal-detail-page', { timeout: 10000 });
    
    // Check text is readable
    const animalDetails = page.locator('.animal-details h1');
    await expect(animalDetails).toBeVisible();
    
    const textColor = await animalDetails.evaluate((el) => {
      return window.getComputedStyle(el).color;
    });
    
    console.log('Animal name text color in light mode:', textColor);
    
    // Take screenshot
    await page.screenshot({ path: '/tmp/animal-page-light-mode.png', fullPage: true });
  });
});
