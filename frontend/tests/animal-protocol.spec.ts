import { test, expect } from '@playwright/test';

/**
 * Animal Protocol E2E Tests
 * 
 * These tests validate the animal protocol functionality including:
 * - Viewing protocol entries
 * - Adding new protocol entries
 * - Character limit validation (1000 chars)
 * - Image upload
 * - Delete protocol (admin only)
 * - Modal interactions
 */

// Test configuration
const BASE_URL = 'http://localhost:5173';

// Test credentials
const ADMIN_USER = {
  username: 'testadmin',
  password: 'password123'
};

const REGULAR_USER = {
  username: 'testuser',
  password: 'password123'
};

test.describe('Animal Protocol Functionality', () => {
  test.beforeEach(async ({ page }) => {
    // Navigate to login page
    await page.goto(BASE_URL + '/login');
    
    // Login as admin user
    await page.fill('input[name="username"]', ADMIN_USER.username);
    await page.fill('input[name="password"]', ADMIN_USER.password);
    await page.click('button[type="submit"]');
    
    // Wait for successful login
    await page.waitForURL(/\/(dashboard|groups)/i, { timeout: 10000 });
  });

  test('should display protocol section on animal detail page', async ({ page }) => {
    // Navigate to a specific animal's detail page
    // This assumes there's at least one animal in the database
    await page.goto(BASE_URL + '/dashboard');
    
    // Click on first animal card (if available)
    const animalCard = page.locator('.animal-card').first();
    if (await animalCard.isVisible()) {
      await animalCard.click();
      
      // Wait for animal detail page to load
      await page.waitForURL(/\/animals\/\d+\/view/, { timeout: 5000 });
      
      // Verify protocol section exists
      await expect(page.locator('text=Protocol Entries')).toBeVisible();
      
      // Verify "Add Protocol" button exists
      await expect(page.locator('button:has-text("Add Protocol")')).toBeVisible();
    }
  });

  test('should open modal when "Add Protocol" button is clicked', async ({ page }) => {
    // Navigate to a specific animal's detail page
    await page.goto(BASE_URL + '/dashboard');
    
    const animalCard = page.locator('.animal-card').first();
    if (await animalCard.isVisible()) {
      await animalCard.click();
      await page.waitForURL(/\/animals\/\d+\/view/, { timeout: 5000 });
      
      // Click "Add Protocol" button
      await page.click('button:has-text("Add Protocol")');
      
      // Verify modal opens
      await expect(page.locator('text=Add Protocol Entry')).toBeVisible();
      
      // Verify form fields exist
      await expect(page.locator('textarea#protocol-content')).toBeVisible();
      await expect(page.locator('button:has-text("Save Protocol")')).toBeVisible();
      await expect(page.locator('button:has-text("Cancel")')).toBeVisible();
    }
  });

  test('should validate protocol content is required', async ({ page }) => {
    // Navigate to animal detail page
    await page.goto(BASE_URL + '/dashboard');
    
    const animalCard = page.locator('.animal-card').first();
    if (await animalCard.isVisible()) {
      await animalCard.click();
      await page.waitForURL(/\/animals\/\d+\/view/, { timeout: 5000 });
      
      // Open protocol form
      await page.click('button:has-text("Add Protocol")');
      
      // Try to submit without content
      const submitButton = page.locator('button:has-text("Save Protocol")');
      
      // Submit button should be disabled when content is empty
      await expect(submitButton).toBeDisabled();
    }
  });

  test('should show character count and limit to 1000 characters', async ({ page }) => {
    // Navigate to animal detail page
    await page.goto(BASE_URL + '/dashboard');
    
    const animalCard = page.locator('.animal-card').first();
    if (await animalCard.isVisible()) {
      await animalCard.click();
      await page.waitForURL(/\/animals\/\d+\/view/, { timeout: 5000 });
      
      // Open protocol form
      await page.click('button:has-text("Add Protocol")');
      
      // Type content
      const textarea = page.locator('textarea#protocol-content');
      await textarea.fill('Test protocol content');
      
      // Verify character count is displayed
      await expect(page.locator('text=/\\d+\\/1000 characters/')).toBeVisible();
      
      // Try to type more than 1000 characters (textarea has maxlength attribute)
      const longText = 'a'.repeat(1001);
      await textarea.fill(longText);
      
      // Verify content is limited to 1000 chars
      const content = await textarea.inputValue();
      expect(content.length).toBeLessThanOrEqual(1000);
    }
  });

  test('should successfully create a protocol entry', async ({ page }) => {
    // Navigate to animal detail page
    await page.goto(BASE_URL + '/dashboard');
    
    const animalCard = page.locator('.animal-card').first();
    if (await animalCard.isVisible()) {
      await animalCard.click();
      await page.waitForURL(/\/animals\/\d+\/view/, { timeout: 5000 });
      
      // Count existing protocols
      const existingProtocols = await page.locator('.protocol-card').count();
      
      // Open protocol form
      await page.click('button:has-text("Add Protocol")');
      
      // Fill in content
      const protocolContent = 'Test protocol: Ensure proper feeding schedule. Feed twice daily at 8am and 6pm.';
      await page.fill('textarea#protocol-content', protocolContent);
      
      // Submit form
      await page.click('button:has-text("Save Protocol")');
      
      // Wait for success toast
      await expect(page.locator('text=/Protocol saved successfully/i')).toBeVisible({ timeout: 5000 });
      
      // Verify new protocol appears in the list
      await expect(page.locator('.protocol-card')).toHaveCount(existingProtocols + 1);
      
      // Verify protocol content is displayed
      await expect(page.locator(`text=${protocolContent}`)).toBeVisible();
    }
  });

  test('should close modal when cancel button is clicked', async ({ page }) => {
    // Navigate to animal detail page
    await page.goto(BASE_URL + '/dashboard');
    
    const animalCard = page.locator('.animal-card').first();
    if (await animalCard.isVisible()) {
      await animalCard.click();
      await page.waitForURL(/\/animals\/\d+\/view/, { timeout: 5000 });
      
      // Open protocol form
      await page.click('button:has-text("Add Protocol")');
      
      // Verify modal is open
      await expect(page.locator('text=Add Protocol Entry')).toBeVisible();
      
      // Click cancel
      await page.click('button:has-text("Cancel")');
      
      // Verify modal is closed
      await expect(page.locator('text=Add Protocol Entry')).not.toBeVisible();
    }
  });

  test('should allow image upload for protocol', async ({ page }) => {
    // Navigate to animal detail page
    await page.goto(BASE_URL + '/dashboard');
    
    const animalCard = page.locator('.animal-card').first();
    if (await animalCard.isVisible()) {
      await animalCard.click();
      await page.waitForURL(/\/animals\/\d+\/view/, { timeout: 5000 });
      
      // Open protocol form
      await page.click('button:has-text("Add Protocol")');
      
      // Verify image upload button exists
      await expect(page.locator('button:has-text("Add Image")')).toBeVisible();
      
      // Note: Actual file upload would require a test image file
      // and would be tested in integration tests with the backend running
    }
  });

  test('admin should be able to delete protocol entry', async ({ page }) => {
    // Navigate to animal detail page
    await page.goto(BASE_URL + '/dashboard');
    
    const animalCard = page.locator('.animal-card').first();
    if (await animalCard.isVisible()) {
      await animalCard.click();
      await page.waitForURL(/\/animals\/\d+\/view/, { timeout: 5000 });
      
      // First, add a protocol to delete
      await page.click('button:has-text("Add Protocol")');
      await page.fill('textarea#protocol-content', 'Protocol to be deleted');
      await page.click('button:has-text("Save Protocol")');
      await expect(page.locator('text=/Protocol saved successfully/i')).toBeVisible({ timeout: 5000 });
      
      // Wait for modal to close
      await page.waitForTimeout(1000);
      
      // Find the delete button (admin only)
      const deleteButton = page.locator('button:has-text("Delete")').first();
      
      if (await deleteButton.isVisible()) {
        // Count protocols before deletion
        const protocolCount = await page.locator('.protocol-card').count();
        
        // Click delete button
        await deleteButton.click();
        
        // Confirm deletion in dialog
        await expect(page.locator('text=Delete Protocol Entry?')).toBeVisible();
        await page.click('button:has-text("Delete")');
        
        // Wait for success message
        await expect(page.locator('text=/Protocol deleted successfully/i')).toBeVisible({ timeout: 5000 });
        
        // Verify protocol count decreased
        await expect(page.locator('.protocol-card')).toHaveCount(protocolCount - 1);
      }
    }
  });

  test('should display empty state when no protocols exist', async ({ page }) => {
    // Navigate to animal detail page
    await page.goto(BASE_URL + '/dashboard');
    
    const animalCard = page.locator('.animal-card').first();
    if (await animalCard.isVisible()) {
      await animalCard.click();
      await page.waitForURL(/\/animals\/\d+\/view/, { timeout: 5000 });
      
      // Check if empty state is visible (when no protocols exist)
      const protocolCards = await page.locator('.protocol-card').count();
      
      if (protocolCards === 0) {
        await expect(page.locator('text=No Protocol Entries')).toBeVisible();
        await expect(page.locator('text=Add First Protocol')).toBeVisible();
      }
    }
  });
});

test.describe('Animal Protocol - Accessibility', () => {
  test.beforeEach(async ({ page }) => {
    // Login
    await page.goto(BASE_URL + '/login');
    await page.fill('input[name="username"]', ADMIN_USER.username);
    await page.fill('input[name="password"]', ADMIN_USER.password);
    await page.click('button[type="submit"]');
    await page.waitForURL(/\/(dashboard|groups)/i, { timeout: 10000 });
  });

  test('should have proper ARIA labels', async ({ page }) => {
    await page.goto(BASE_URL + '/dashboard');
    
    const animalCard = page.locator('.animal-card').first();
    if (await animalCard.isVisible()) {
      await animalCard.click();
      await page.waitForURL(/\/animals\/\d+\/view/, { timeout: 5000 });
      
      // Open protocol form
      await page.click('button:has-text("Add Protocol")');
      
      // Check for proper ARIA attributes
      const textarea = page.locator('textarea#protocol-content');
      await expect(textarea).toHaveAttribute('aria-label');
      await expect(textarea).toHaveAttribute('aria-required', 'true');
    }
  });

  test('should be keyboard accessible', async ({ page }) => {
    await page.goto(BASE_URL + '/dashboard');
    
    const animalCard = page.locator('.animal-card').first();
    if (await animalCard.isVisible()) {
      await animalCard.click();
      await page.waitForURL(/\/animals\/\d+\/view/, { timeout: 5000 });
      
      // Use Tab to navigate to "Add Protocol" button
      await page.keyboard.press('Tab');
      
      // Verify focus is on an interactive element
      const focusedElement = await page.evaluate(() => document.activeElement?.tagName);
      expect(focusedElement).toBeTruthy();
    }
  });
});
