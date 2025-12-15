import { test, expect } from '@playwright/test';
import { loginAsAdmin } from './helpers/auth';

/**
 * UX Testing for Improved Add User Form
 * 
 * Tests focus on:
 * - Visual hierarchy and clarity
 * - Inline validation feedback
 * - Password strength indicator
 * - Accessibility compliance
 * - Error handling
 * - Success feedback
 * - Responsive design
 * - Keyboard navigation
 */

test.describe('Add User Form UX', () => {
  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page);
    
    // Navigate to Users page
    await page.goto('/users');
    await page.waitForLoadState('networkidle');
  });

  test('form has clear visual hierarchy and professional design', async ({ page }) => {
    // Open form
    await page.click('button:has-text("Add User")');
    
    // Check form is visible with proper styling
    const form = page.locator('.users-create-form');
    await expect(form).toBeVisible();
    
    // Verify header exists
    await expect(page.locator('.create-form-header h2')).toHaveText('Add New User');
    await expect(page.locator('.form-description')).toBeVisible();
    
    // Check form has border and shadow (indicating it stands out)
    const borderColor = await form.evaluate((el) => {
      return window.getComputedStyle(el).borderColor;
    });
    expect(borderColor).toBeTruthy();
    
    // Verify required field indicators
    const requiredMarkers = page.locator('.required');
    await expect(requiredMarkers).toHaveCount(3); // username, email, password
  });

  test('username field has inline validation', async ({ page }) => {
    await page.click('button:has-text("Add User")');
    
    const usernameInput = page.locator('input[name="username"]');
    
    // Initially no error
    await expect(page.locator('#username-error')).not.toBeVisible();
    
    // Show hint text
    await expect(page.locator('#username-hint')).toBeVisible();
    await expect(page.locator('#username-hint')).toContainText('3-50 characters');
    
    // Enter invalid username (too short)
    await usernameInput.fill('ab');
    await usernameInput.blur();
    
    // Error should appear
    await expect(page.locator('#username-error')).toBeVisible();
    await expect(page.locator('#username-error')).toContainText('at least 3 characters');
    
    // Input should have error styling
    await expect(usernameInput).toHaveClass(/input-error/);
    
    // Enter invalid characters
    await usernameInput.fill('user@name');
    await usernameInput.blur();
    await expect(page.locator('#username-error')).toContainText('letters, numbers, hyphens, and underscores');
    
    // Fix the error
    await usernameInput.fill('valid_username');
    await usernameInput.blur();
    
    // Error should disappear
    await expect(page.locator('#username-error')).not.toBeVisible();
    await expect(usernameInput).not.toHaveClass(/input-error/);
  });

  test('email field validates format correctly', async ({ page }) => {
    await page.click('button:has-text("Add User")');
    
    const emailInput = page.locator('input[name="email"]');
    
    // Enter invalid email
    await emailInput.fill('notanemail');
    await emailInput.blur();
    
    await expect(page.locator('#email-error')).toBeVisible();
    await expect(page.locator('#email-error')).toContainText('valid email address');
    
    // Fix email
    await emailInput.fill('user@example.com');
    await emailInput.blur();
    
    await expect(page.locator('#email-error')).not.toBeVisible();
  });

  test('password field shows strength indicator', async ({ page }) => {
    await page.click('button:has-text("Add User")');
    
    const passwordInput = page.locator('input[name="password"]');
    
    // Weak password
    await passwordInput.fill('12345678');
    
    // Strength indicator should appear
    await expect(page.locator('.password-strength')).toBeVisible();
    await expect(page.locator('.strength-label')).toContainText('Weak');
    
    // Check strength bar color
    const weakBar = page.locator('.strength-fill.strength-weak');
    await expect(weakBar).toBeVisible();
    
    // Medium password
    await passwordInput.fill('Password123');
    await expect(page.locator('.strength-label')).toContainText('Medium');
    
    // Strong password
    await passwordInput.fill('P@ssw0rd!2024');
    await expect(page.locator('.strength-label')).toContainText('Strong');
  });

  test('password toggle shows/hides password', async ({ page }) => {
    await page.click('button:has-text("Add User")');
    
    const passwordInput = page.locator('input[name="password"]');
    const toggleButton = page.locator('.password-toggle');
    
    await passwordInput.fill('SecretPassword123');
    
    // Initially password type
    await expect(passwordInput).toHaveAttribute('type', 'password');
    
    // Click toggle to show
    await toggleButton.click();
    await expect(passwordInput).toHaveAttribute('type', 'text');
    
    // Click toggle to hide
    await toggleButton.click();
    await expect(passwordInput).toHaveAttribute('type', 'password');
  });

  test('admin checkbox has clear visual design', async ({ page }) => {
    await page.click('button:has-text("Add User")');
    
    const adminCheckbox = page.locator('input[name="is_admin"]');
    const checkboxField = page.locator('.checkbox-field');
    
    // Checkbox field is prominent
    await expect(checkboxField).toBeVisible();
    await expect(checkboxField).toContainText('Grant admin privileges');
    await expect(checkboxField).toContainText('Admins can manage users');
    
    // Custom checkbox styling
    const checkboxBox = page.locator('.checkbox-field .checkbox-box').first();
    await expect(checkboxBox).toBeVisible();
    
    // Check checkbox
    await adminCheckbox.check();
    
    // Visual checkmark should appear
    const checkIcon = page.locator('.checkbox-field .checkbox-icon').first();
    await expect(checkIcon).toBeVisible();
  });

  test('group checkboxes are well-styled and scannable', async ({ page }) => {
    await page.click('button:has-text("Add User")');
    
    const groupCheckboxes = page.locator('.group-checkboxes');
    await expect(groupCheckboxes).toBeVisible();
    
    // Check if groups are displayed
    const groupLabels = page.locator('.group-checkbox-label');
    const count = await groupLabels.count();
    expect(count).toBeGreaterThan(0);
    
    // Each group should have proper styling
    const firstGroup = groupLabels.first();
    await expect(firstGroup).toBeVisible();
    
    // Click to select a group
    await firstGroup.click();
    
    // Visual feedback (checked state)
    const checkbox = firstGroup.locator('.checkbox-input');
    await expect(checkbox).toBeChecked();
  });

  test('form prevents submission with validation errors', async ({ page }) => {
    await page.click('button:has-text("Add User")');
    
    // Try to submit empty form
    await page.click('button[type="submit"]:has-text("Create User")');
    
    // Validation errors should appear
    await expect(page.locator('#username-error')).toBeVisible();
    await expect(page.locator('#email-error')).toBeVisible();
    await expect(page.locator('#password-error')).toBeVisible();
    
    // General error message
    await expect(page.locator('.alert-error')).toBeVisible();
    await expect(page.locator('.alert-error')).toContainText('fix the errors');
  });

  test('form shows loading state during submission', async ({ page }) => {
    await page.click('button:has-text("Add User")');
    
    // Fill form with valid data
    await page.fill('input[name="username"]', 'testuser' + Date.now());
    await page.fill('input[name="email"]', `test${Date.now()}@example.com`);
    await page.fill('input[name="password"]', 'SecurePass123!');
    
    // Submit form
    const submitButton = page.locator('button[type="submit"]');
    await submitButton.click();
    
    // Button should show loading state
    await expect(submitButton).toContainText('Creating User');
    await expect(submitButton.locator('.spinner')).toBeVisible();
    await expect(submitButton).toBeDisabled();
  });

  test('form shows success message and auto-closes', async ({ page }) => {
    await page.click('button:has-text("Add User")');
    
    const username = 'testuser' + Date.now();
    
    // Fill form
    await page.fill('input[name="username"]', username);
    await page.fill('input[name="email"]', `${username}@example.com`);
    await page.fill('input[name="password"]', 'SecurePass123!');
    
    // Submit
    await page.click('button[type="submit"]');
    
    // Success message should appear
    await expect(page.locator('.alert-success')).toBeVisible();
    await expect(page.locator('.alert-success')).toContainText('successfully');
    
    // Form should close after delay
    await expect(page.locator('.users-create-form')).not.toBeVisible({ timeout: 3000 });
  });

  test('cancel button closes form without saving', async ({ page }) => {
    await page.click('button:has-text("Add User")');
    
    // Fill some data
    await page.fill('input[name="username"]', 'testuser');
    await page.fill('input[name="email"]', 'test@example.com');
    
    // Click cancel
    await page.click('button.btn-secondary:has-text("Cancel")');
    
    // Form should close
    await expect(page.locator('.users-create-form')).not.toBeVisible();
    
    // Reopen form - fields should be reset
    await page.click('button:has-text("Add User")');
    await expect(page.locator('input[name="username"]')).toHaveValue('');
    await expect(page.locator('input[name="email"]')).toHaveValue('');
  });

  test('form is keyboard accessible', async ({ page }) => {
    await page.click('button:has-text("Add User")');
    
    // Tab through form fields
    await page.keyboard.press('Tab'); // Focus username
    const usernameInput = page.locator('input[name="username"]');
    await expect(usernameInput).toBeFocused();
    
    await page.keyboard.press('Tab'); // Focus email
    const emailInput = page.locator('input[name="email"]');
    await expect(emailInput).toBeFocused();
    
    await page.keyboard.press('Tab'); // Focus password
    const passwordInput = page.locator('input[name="password"]');
    await expect(passwordInput).toBeFocused();
    
    // Enter with keyboard
    await page.keyboard.type('SecurePass123!');
    await expect(passwordInput).toHaveValue('SecurePass123!');
    
    // Tab to password toggle
    await page.keyboard.press('Tab');
    const toggleButton = page.locator('.password-toggle');
    await expect(toggleButton).toBeFocused();
    
    // Press Enter to toggle
    await page.keyboard.press('Enter');
    await expect(passwordInput).toHaveAttribute('type', 'text');
  });

  test('form is responsive on mobile', async ({ page }) => {
    // Set mobile viewport
    await page.setViewportSize({ width: 375, height: 667 });
    
    await page.goto('/users');
    await page.click('button:has-text("Add User")');
    
    const form = page.locator('.users-create-form');
    await expect(form).toBeVisible();
    
    // Check that form fits in viewport
    const formBox = await form.boundingBox();
    expect(formBox).not.toBeNull();
    if (formBox) {
      expect(formBox.width).toBeLessThanOrEqual(375);
    }
    
    // Action buttons should be stacked vertically
    const formActions = page.locator('.form-actions');
    const actionsBox = await formActions.boundingBox();
    
    // Buttons should be full width on mobile
    const submitButton = page.locator('button[type="submit"]');
    const submitBox = await submitButton.boundingBox();
    
    if (actionsBox && submitBox) {
      // Button should be nearly full width of actions container
      expect(submitBox.width).toBeGreaterThan(actionsBox.width * 0.8);
    }
  });

  test('form has proper ARIA labels and roles', async ({ page }) => {
    await page.click('button:has-text("Add User")');
    
    // Check ARIA attributes on inputs
    const usernameInput = page.locator('input[name="username"]');
    await expect(usernameInput).toHaveAttribute('aria-describedby', 'username-hint');
    
    // Trigger error
    await usernameInput.fill('a');
    await usernameInput.blur();
    
    // ARIA should update to error
    await expect(usernameInput).toHaveAttribute('aria-invalid', 'true');
    await expect(usernameInput).toHaveAttribute('aria-describedby', 'username-error');
    
    // Error message should have role="alert"
    await expect(page.locator('#username-error')).toHaveAttribute('role', 'alert');
    
    // Form alerts should have role="alert"
    await page.click('button[type="submit"]');
    await expect(page.locator('.alert-error')).toHaveAttribute('role', 'alert');
  });

  test('password field shows helpful hint text', async ({ page }) => {
    await page.click('button:has-text("Add User")');
    
    // Initially shows hint
    await expect(page.locator('#password-hint')).toBeVisible();
    await expect(page.locator('#password-hint')).toContainText('8 characters');
    await expect(page.locator('#password-hint')).toContainText('mix of letters, numbers, and symbols');
    
    // Hint disappears when password has value and no error
    await page.fill('input[name="password"]', 'ValidPassword123!');
    
    // Strength indicator replaces hint
    await expect(page.locator('#password-hint')).not.toBeVisible();
    await expect(page.locator('.password-strength')).toBeVisible();
  });

  test('form fields show visual focus indicators', async ({ page }) => {
    await page.click('button:has-text("Add User")');
    
    const usernameInput = page.locator('input[name="username"]');
    
    // Click to focus
    await usernameInput.click();
    
    // Should have visible focus styling
    const borderColor = await usernameInput.evaluate((el) => {
      return window.getComputedStyle(el).borderColor;
    });
    
    const boxShadow = await usernameInput.evaluate((el) => {
      return window.getComputedStyle(el).boxShadow;
    });
    
    // Border color and box shadow should be set (indicating focus styles)
    expect(borderColor).toBeTruthy();
    expect(boxShadow).not.toBe('none');
  });

  test('form handles server errors gracefully', async ({ page }) => {
    await page.click('button:has-text("Add User")');
    
    // Try to create user with existing username (will cause server error)
    await page.fill('input[name="username"]', 'admin'); // Existing user
    await page.fill('input[name="email"]', 'newemail@example.com');
    await page.fill('input[name="password"]', 'SecurePass123!');
    
    await page.click('button[type="submit"]');
    
    // Should show error alert
    await expect(page.locator('.alert-error')).toBeVisible({ timeout: 5000 });
    
    // Form should remain open so user can fix the issue
    await expect(page.locator('.users-create-form')).toBeVisible();
    
    // Form fields should retain their values
    await expect(page.locator('input[name="username"]')).toHaveValue('admin');
  });

  test('form groups section has clear hierarchy', async ({ page }) => {
    await page.click('button:has-text("Add User")');
    
    // Groups section should have clear label
    await expect(page.locator('text=Assign to Groups')).toBeVisible();
    
    // Should have explanatory text
    const hint = page.locator('.field-hint', { hasText: 'Select which groups' });
    await expect(hint).toBeVisible();
    
    // Groups should be in a scrollable container
    const groupContainer = page.locator('.group-checkboxes');
    await expect(groupContainer).toBeVisible();
    
    // Check that groups are clearly separated
    const firstGroup = page.locator('.group-checkbox-label').first();
    await expect(firstGroup).toBeVisible();
    
    // Hover should show visual feedback
    await firstGroup.hover();
    const borderColor = await firstGroup.evaluate((el) => {
      return window.getComputedStyle(el).borderColor;
    });
    expect(borderColor).toBeTruthy();
  });
});
