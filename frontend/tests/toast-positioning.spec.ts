import { test, expect } from '@playwright/test';
import { loginAsVolunteer, testUsers } from './helpers/auth';

/**
 * Toast Positioning E2E Tests
 * 
 * These tests validate that toast notifications don't overlap with
 * the navigation bar and logout button.
 */

const TEST_USER = testUsers.volunteer;

test.describe('Toast Positioning - Desktop', () => {
  test.beforeEach(async ({ page }) => {
    await page.setViewportSize({ width: 1280, height: 800 });
  });

  test('toast should not overlap with navigation on successful login', async ({ page }) => {
    await loginAsVolunteer(page, { waitForUrl: /\/(dashboard|groups)/i });
    
    // Wait for navigation and toast to appear
    await page.waitForTimeout(1000);
    
    // Get navigation element
    const navigation = page.locator('nav.navigation');
    const navBox = await navigation.boundingBox();
    
    // Wait for toast to appear (if successful login)
    const toast = page.locator('.toast').first();
    const toastVisible = await toast.isVisible().catch(() => false);
    
    if (toastVisible) {
      const toastBox = await toast.boundingBox();
      
      // Verify navigation exists
      expect(navBox).toBeTruthy();
      
      if (navBox && toastBox) {
        // Toast should be positioned below the navigation
        // Toast top should be greater than navigation bottom
        expect(toastBox.y).toBeGreaterThan(navBox.y + navBox.height);
        
        // There should be some space between them (at least 8px)
        const gap = toastBox.y - (navBox.y + navBox.height);
        expect(gap).toBeGreaterThanOrEqual(8);
        
        console.log(`✅ Toast positioned ${gap}px below navigation (nav bottom: ${navBox.y + navBox.height}px, toast top: ${toastBox.y}px)`);
      }
    }
  });

  test('logout button should not overlap with toast container', async ({ page }) => {
    await loginAsVolunteer(page, { waitForUrl: /\/(dashboard|groups)/i });
    
    // Wait for navigation
    await page.waitForTimeout(1500);
    
    // Find logout button
    const logoutButton = page.locator('button.nav-logout, button:has-text("Logout")').first();
    const logoutVisible = await logoutButton.isVisible().catch(() => false);
    
    if (logoutVisible) {
      const logoutBox = await logoutButton.boundingBox();
      
      // Get toast container position
      const toastContainer = page.locator('.toast-container');
      const containerBox = await toastContainer.boundingBox();
      
      if (logoutBox && containerBox) {
        // Verify logout button is above toast container
        expect(logoutBox.y + logoutBox.height).toBeLessThan(containerBox.y);
        
        console.log('✅ Logout button does not overlap with toast container');
        console.log(`   Logout button bottom: ${logoutBox.y + logoutBox.height}px`);
        console.log(`   Toast container top: ${containerBox.y}px`);
      }
    }
  });

  test('toast close button should be easily clickable without hitting logout', async ({ page }) => {
    await loginAsVolunteer(page, { waitForUrl: /\/(dashboard|groups)/i });
    
    await page.waitForTimeout(1000);
    
    // Check if toast appeared
    const toast = page.locator('.toast').first();
    const toastVisible = await toast.isVisible().catch(() => false);
    
    if (toastVisible) {
      // Get logout button position
      const logoutButton = page.locator('button.nav-logout, button:has-text("Logout")').first();
      const logoutBox = await logoutButton.boundingBox();
      
      // Get toast close button position
      const toastCloseButton = toast.locator('button.toast__close');
      const closeButtonBox = await toastCloseButton.boundingBox();
      
      if (logoutBox && closeButtonBox) {
        // Calculate vertical distance between logout button and toast close button
        const verticalDistance = Math.abs(closeButtonBox.y - logoutBox.y);
        
        // They should be at least 40px apart vertically to avoid accidental clicks
        expect(verticalDistance).toBeGreaterThan(40);
        
        // Click the toast close button - should NOT log out
        await toastCloseButton.click();
        await page.waitForTimeout(500);
        
        // Verify we're still logged in (logout button still visible)
        const stillLoggedIn = await logoutButton.isVisible();
        expect(stillLoggedIn).toBe(true);
        
        console.log('✅ Toast close button safely separated from logout button');
      }
    }
  });
});

test.describe('Toast Positioning - Mobile', () => {
  test.beforeEach(async ({ page }) => {
    await page.setViewportSize({ width: 375, height: 667 });
  });

  test('toast should not overlap with mobile navigation', async ({ page }) => {
    await loginAsVolunteer(page, { waitForUrl: /\/(dashboard|groups)/i });
    
    await page.waitForTimeout(1000);
    
    // Get navigation element
    const navigation = page.locator('nav.navigation');
    const navBox = await navigation.boundingBox();
    
    // Check if toast appeared
    const toast = page.locator('.toast').first();
    const toastVisible = await toast.isVisible().catch(() => false);
    
    if (toastVisible && navBox) {
      const toastBox = await toast.boundingBox();
      
      if (toastBox) {
        // On mobile, toast should still be below navigation
        expect(toastBox.y).toBeGreaterThan(navBox.y + navBox.height);
        
        console.log('✅ Toast positioned correctly on mobile');
      }
    }
  });

  test('toast should fit within mobile viewport', async ({ page }) => {
    await loginAsVolunteer(page, { waitForUrl: /\/(dashboard|groups)/i });
    
    await page.waitForTimeout(1000);
    
    // Check if toast appeared
    const toast = page.locator('.toast').first();
    const toastVisible = await toast.isVisible().catch(() => false);
    
    if (toastVisible) {
      const toastBox = await toast.boundingBox();
      
      if (toastBox) {
        // Toast should fit within viewport width
        expect(toastBox.x).toBeGreaterThanOrEqual(0);
        expect(toastBox.x + toastBox.width).toBeLessThanOrEqual(375);
        
        console.log('✅ Toast fits within mobile viewport');
      }
    }
  });
});

test.describe('Toast Positioning - Tablet', () => {
  test.beforeEach(async ({ page }) => {
    await page.setViewportSize({ width: 768, height: 1024 });
  });

  test('toast should not overlap with tablet navigation', async ({ page }) => {
    await loginAsVolunteer(page, { waitForUrl: /\/(dashboard|groups)/i });
    
    await page.waitForTimeout(1000);
    
    // Get navigation element
    const navigation = page.locator('nav.navigation');
    const navBox = await navigation.boundingBox();
    
    // Check if toast appeared
    const toast = page.locator('.toast').first();
    const toastVisible = await toast.isVisible().catch(() => false);
    
    if (toastVisible && navBox) {
      const toastBox = await toast.boundingBox();
      
      if (toastBox) {
        // Toast should be below navigation
        expect(toastBox.y).toBeGreaterThan(navBox.y + navBox.height);
        
        console.log('✅ Toast positioned correctly on tablet');
      }
    }
  });
});
