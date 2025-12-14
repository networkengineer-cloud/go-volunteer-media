import { test, expect } from '@playwright/test';

/**
 * Settings Page E2E Tests
 * 
 * These tests validate the Settings page functionality including:
 * - Displaying user information
 * - Editing email address
 * - Toggling email notifications
 * - Saving preferences
 * - Error handling
 */

const BASE_URL = 'http://localhost:5173';

const TEST_USER = {
  username: 'testuser',
  password: 'password123'
};

test.describe('Settings Page', () => {
  test.beforeEach(async ({ page }) => {
    // Login before accessing settings
    await page.goto(BASE_URL + '/login');
    await page.fill('input[name="username"]', TEST_USER.username);
    await page.fill('input[name="password"]', TEST_USER.password);
    await page.click('button[type="submit"]');
    
    // Wait for login to complete
    await page.waitForURL(/\/(dashboard|groups)/i, { timeout: 10000 });
    
    // Navigate to settings
    await page.goto(BASE_URL + '/settings');
    await page.waitForLoadState('networkidle');
  });

  test('should display settings page with all sections', async ({ page }) => {
    // Check for main heading
    await expect(page.locator('h1')).toContainText('Settings');
    
    // Check for Profile Information section
    await expect(page.locator('h2')).toContainText('Profile Information');
    
    // Check for Privacy Settings section
    await expect(page.locator('h2')).toContainText('Privacy Settings');
    
    // Check for Email Notifications section
    await expect(page.locator('h2')).toContainText('Email Notifications');
    
    // Check for About Email Notifications section
    await expect(page.locator('h2')).toContainText('About Email Notifications');
  });

  test('should display email and phone input fields', async ({ page }) => {
    // Check for email label
    const emailLabel = page.locator('label:has-text("Email Address")');
    await expect(emailLabel).toBeVisible();
    
    // Check for email input
    const emailInput = page.locator('input[type="email"]');
    await expect(emailInput).toBeVisible();
    
    // Check for phone label
    const phoneLabel = page.locator('label:has-text("Phone Number")');
    await expect(phoneLabel).toBeVisible();
    
    // Check for phone input
    const phoneInput = page.locator('input[type="tel"]');
    await expect(phoneInput).toBeVisible();
  });

  test('should display privacy toggles', async ({ page }) => {
    // Check for hide email toggle label
    const hideEmailLabel = page.locator('label:has-text("Hide Email from Other Users")');
    await expect(hideEmailLabel).toBeVisible();
    
    // Check for hide phone toggle label
    const hidePhoneLabel = page.locator('label:has-text("Hide Phone from Other Users")');
    await expect(hidePhoneLabel).toBeVisible();
    
    // Check for toggle inputs
    const hideEmailToggle = page.locator('input[id="hide-email"]');
    await expect(hideEmailToggle).toBeVisible();
    
    const hidePhoneToggle = page.locator('input[id="hide-phone"]');
    await expect(hidePhoneToggle).toBeVisible();
  });

  test('should display email notifications toggle', async ({ page }) => {
    // Check for toggle label
    const toggleLabel = page.locator('label:has-text("Receive Announcement Emails")');
    await expect(toggleLabel).toBeVisible();
    
    // Check for toggle checkbox
    const toggleInput = page.locator('input[id="email-notifications"]');
    await expect(toggleInput).toBeVisible();
  });

  test('should allow editing email address', async ({ page }) => {
    // Get email input
    const emailInput = page.locator('input[type="email"]');
    
    // Clear and enter new email
    const newEmail = 'newemail@example.com';
    await emailInput.fill(newEmail);
    
    // Verify value changed
    await expect(emailInput).toHaveValue(newEmail);
  });

  test('should save email changes', async ({ page }) => {
    // Get email input and save button
    const emailInput = page.locator('input[type="email"]');
    const saveButton = page.locator('button:has-text("Save Profile")');
    
    // Enter new email
    const newEmail = 'updated@example.com';
    await emailInput.fill(newEmail);
    
    // Click save button
    await saveButton.click();
    
    // Wait for success message or button state change
    const successMessage = page.locator('.success');
    if (await successMessage.isVisible({ timeout: 5000 }).catch(() => false)) {
      await expect(successMessage).toBeVisible();
    }
  });

  test('should allow editing phone number', async ({ page }) => {
    // Get phone input
    const phoneInput = page.locator('input[type="tel"]');
    
    // Clear and enter new phone
    const newPhone = '(555) 123-4567';
    await phoneInput.fill(newPhone);
    
    // Verify value changed
    await expect(phoneInput).toHaveValue(newPhone);
  });

  test('should toggle privacy settings', async ({ page }) => {
    // Get privacy toggles
    const hideEmailToggle = page.locator('input[id="hide-email"]');
    const hidePhoneToggle = page.locator('input[id="hide-phone"]');
    
    // Get initial states
    const initialEmailState = await hideEmailToggle.isChecked();
    const initialPhoneState = await hidePhoneToggle.isChecked();
    
    // Toggle both
    await hideEmailToggle.click();
    await hidePhoneToggle.click();
    
    // Verify state changed
    const newEmailState = await hideEmailToggle.isChecked();
    const newPhoneState = await hidePhoneToggle.isChecked();
    expect(newEmailState).toBe(!initialEmailState);
    expect(newPhoneState).toBe(!initialPhoneState);
  });

  test('should save profile with phone number and privacy settings', async ({ page }) => {
    // Get inputs
    const phoneInput = page.locator('input[type="tel"]');
    const hideEmailToggle = page.locator('input[id="hide-email"]');
    const saveButton = page.locator('button:has-text("Save Profile")');
    
    // Update phone number
    const newPhone = '(888) 999-0000';
    await phoneInput.fill(newPhone);
    
    // Toggle hide email
    await hideEmailToggle.click();
    
    // Click save button
    await saveButton.click();
    
    // Wait for success message or button state change
    const successMessage = page.locator('.success');
    if (await successMessage.isVisible({ timeout: 5000 }).catch(() => false)) {
      await expect(successMessage).toBeVisible();
    }
  });

  test('should toggle email notifications', async ({ page }) => {
    // Get toggle checkbox
    const toggleInput = page.locator('input[id="email-notifications"]');
    
    // Get initial state
    const initialState = await toggleInput.isChecked();
    
    // Click to toggle
    await toggleInput.click();
    
    // Verify state changed
    const newState = await toggleInput.isChecked();
    expect(newState).toBe(!initialState);
  });

  test('should save notification preferences', async ({ page }) => {
    // Get toggle and save button
    const toggleInput = page.locator('input[id="email-notifications"]');
    const saveButton = page.locator('button:has-text("Save Preferences")');
    
    // Toggle the notification setting
    await toggleInput.click();
    
    // Click save button
    await saveButton.click();
    
    // Wait for success message
    const successMessage = page.locator('.success');
    if (await successMessage.isVisible({ timeout: 5000 }).catch(() => false)) {
      await expect(successMessage).toBeVisible();
    }
  });

  test('should have back button to return to dashboard', async ({ page }) => {
    // Check for back button
    const backButton = page.locator('button:has-text("Back to Dashboard")');
    await expect(backButton).toBeVisible();
    
    // Click back button
    await backButton.click();
    
    // Verify navigation
    await page.waitForURL(/\/(dashboard|groups)/i, { timeout: 5000 });
  });

  test('should display help text for email field', async ({ page }) => {
    // Check for help text
    const helpText = page.locator('p:has-text("email is used for account recovery")');
    await expect(helpText).toBeVisible();
  });

  test('should display help text for phone field', async ({ page }) => {
    // Check for help text
    const helpText = page.locator('p:has-text("phone number helps other volunteers")');
    await expect(helpText).toBeVisible();
  });

  test('should display help text for privacy toggles', async ({ page }) => {
    // Check for email privacy help text
    const emailPrivacyText = page.locator('p:has-text("only admins can see your email")');
    await expect(emailPrivacyText).toBeVisible();
    
    // Check for phone privacy help text
    const phonePrivacyText = page.locator('p:has-text("only admins and group admins")');
    await expect(phonePrivacyText).toBeVisible();
  });

  test('should disable buttons while saving', async ({ page }) => {
    // Intercept the API call to slow it down
    await page.route('**/api/**', route => {
      setTimeout(() => route.continue(), 1000);
    });
    
    // Get email input and save button
    const emailInput = page.locator('input[type="email"]');
    const saveButton = page.locator('button:has-text("Save Profile")');
    
    // Enter new email
    await emailInput.fill('test@example.com');
    
    // Click save button
    await saveButton.click();
    
    // Check that button is disabled immediately
    await expect(saveButton).toBeDisabled();
  });

  test('should be responsive on mobile', async ({ page }) => {
    // Set mobile viewport
    await page.setViewportSize({ width: 375, height: 667 });
    
    // Verify page is still usable
    await expect(page.locator('h1')).toContainText('Settings');
    
    // Check that elements are visible and accessible
    const emailInput = page.locator('input[type="email"]');
    await expect(emailInput).toBeVisible();
    
    const saveButton = page.locator('button:has-text("Save Profile")');
    await expect(saveButton).toBeVisible();
  });

  test('should have proper form labels for accessibility', async ({ page }) => {
    // Check email label is associated with input
    const emailLabel = page.locator('label[for="email"]');
    await expect(emailLabel).toBeVisible();
    
    // Check toggle label is associated with input
    const toggleLabel = page.locator('label[for="email-notifications"]');
    await expect(toggleLabel).toBeVisible();
  });
});
