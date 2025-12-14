import { test, expect, type Page } from '@playwright/test';

/**
 * Animal Form UX E2E Tests
 * 
 * These tests validate the UX improvements for the animal form:
 * - Cancel button functionality
 * - Single date entry for Bite Quarantine (in modal only)
 * - Modal focus preservation during typing
 */

const BASE_URL = 'http://localhost:5173';

// Test credentials (admin required for animal editing)
const ADMIN_USER = {
  username: 'testadmin',
  password: 'password123'
};

// Helper function to login
async function loginAsAdmin(page: Page) {
  await page.goto(BASE_URL + '/login');
  await page.fill('input[name="username"]', ADMIN_USER.username);
  await page.fill('input[name="password"]', ADMIN_USER.password);
  await page.click('button[type="submit"]');
  await page.waitForURL(/\/(dashboard|groups)/i, { timeout: 10000 });
}

test.describe('Animal Form UX Improvements', () => {
  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page);
  });

  test('should display Cancel button on animal edit form', async ({ page }) => {
    // Navigate to a group (assuming group ID 1 exists)
    await page.goto(BASE_URL + '/groups/1');
    await page.waitForLoadState('networkidle');

    // Click on an animal to view details
    const animalCard = page.locator('.animal-card').first();
    if (await animalCard.count() > 0) {
      await animalCard.click();
      await page.waitForLoadState('networkidle');

      // Click Edit button
      const editButton = page.locator('button:has-text("Edit Animal Details")');
      if (await editButton.count() > 0) {
        await editButton.click();
        await page.waitForLoadState('networkidle');

        // Verify Cancel button exists
        const cancelButton = page.locator('button:has-text("Cancel")');
        await expect(cancelButton).toBeVisible();

        console.log('✅ Cancel button is present on animal edit form');
      }
    }
  });

  test('should navigate back to group when Cancel is clicked', async ({ page }) => {
    // Navigate to a group
    await page.goto(BASE_URL + '/groups/1');
    await page.waitForLoadState('networkidle');

    const animalCard = page.locator('.animal-card').first();
    if (await animalCard.count() > 0) {
      await animalCard.click();
      await page.waitForLoadState('networkidle');

      const editButton = page.locator('button:has-text("Edit Animal Details")');
      if (await editButton.count() > 0) {
        await editButton.click();
        await page.waitForLoadState('networkidle');

        // Make some changes to the form
        await page.fill('input[name="name"]', 'Modified Name');

        // Click Cancel button
        const cancelButton = page.locator('button:has-text("Cancel")');
        await cancelButton.click();

        // Should navigate back to group page
        await expect(page).toHaveURL(/\/groups\/1$/);

        console.log('✅ Cancel button navigates back to group without saving');
      }
    }
  });

  test('should NOT show duplicate date field when Bite Quarantine is selected', async ({ page }) => {
    // Navigate to add new animal page
    await page.goto(BASE_URL + '/groups/1/animals/new');
    await page.waitForLoadState('networkidle');

    // Fill required fields
    await page.fill('input[id="name"]', 'Test Animal');
    await page.fill('input[id="age"]', '3');

    // Select Bite Quarantine status
    await page.selectOption('select[id="status"]', 'bite_quarantine');

    // Wait a moment for any conditional rendering
    await page.waitForTimeout(500);

    // Check that there's NO date field in the main form
    // The date field should ONLY appear in the modal
    const dateFieldInForm = page.locator('input[id="quarantine_start_date"]');
    await expect(dateFieldInForm).not.toBeVisible();

    console.log('✅ No duplicate quarantine date field in main form');
  });

  test('should show date field in modal when Bite Quarantine is selected and Update is clicked', async ({ page }) => {
    // Navigate to edit existing animal
    await page.goto(BASE_URL + '/groups/1');
    await page.waitForLoadState('networkidle');

    const animalCard = page.locator('.animal-card').first();
    if (await animalCard.count() > 0) {
      await animalCard.click();
      await page.waitForLoadState('networkidle');

      const editButton = page.locator('button:has-text("Edit Animal Details")');
      if (await editButton.count() > 0) {
        await editButton.click();
        await page.waitForLoadState('networkidle');

        // Select Bite Quarantine status
        await page.selectOption('select[id="status"]', 'bite_quarantine');

        // Click Update button
        const updateButton = page.locator('button[type="submit"]:has-text("Update Animal")');
        await updateButton.click();

        // Modal should appear with date field
        const modal = page.locator('.modal');
        await expect(modal).toBeVisible({ timeout: 5000 });

        // Date field should be in the modal
        const dateFieldInModal = modal.locator('input[type="date"]');
        await expect(dateFieldInModal).toBeVisible();

        console.log('✅ Quarantine date field appears in modal');
      }
    }
  });

  test('should preserve focus when typing in modal textarea', async ({ page }) => {
    // Navigate to edit existing animal
    await page.goto(BASE_URL + '/groups/1');
    await page.waitForLoadState('networkidle');

    const animalCard = page.locator('.animal-card').first();
    if (await animalCard.count() > 0) {
      await animalCard.click();
      await page.waitForLoadState('networkidle');

      const editButton = page.locator('button:has-text("Edit Animal Details")');
      if (await editButton.count() > 0) {
        await editButton.click();
        await page.waitForLoadState('networkidle');

        // Select Bite Quarantine status
        await page.selectOption('select[id="status"]', 'bite_quarantine');

        // Click Update button to open modal
        const updateButton = page.locator('button[type="submit"]:has-text("Update Animal")');
        await updateButton.click();

        // Wait for modal
        const modal = page.locator('.modal');
        await expect(modal).toBeVisible({ timeout: 5000 });

        // Find the textarea for incident details
        const textarea = modal.locator('textarea[id="quarantine-context"]');
        await expect(textarea).toBeVisible();

        // Focus on the textarea
        await textarea.focus();

        // Type a multi-character string character by character to test focus retention
        // This simulates real user typing and catches the bug where focus was lost on each keystroke
        const testText = 'This is a test of focus preservation during typing.';
        await textarea.type(testText, { delay: 50 }); // Add delay to simulate real typing

        // Verify the full text was entered (proves focus was maintained throughout)
        const enteredText = await textarea.inputValue();
        expect(enteredText).toBe(testText);

        // Verify textarea still has focus after typing
        const focusedElement = await page.evaluate(() => document.activeElement?.id);
        expect(focusedElement).toBe('quarantine-context');

        // Additional check: Type more text to ensure continued focus
        const additionalText = ' More text.';
        await textarea.type(additionalText, { delay: 50 });
        const finalText = await textarea.inputValue();
        expect(finalText).toBe(testText + additionalText);

        console.log('✅ Focus is preserved while typing in modal textarea (no focus loss on keystroke)');
      }
    }
  });

  test('should preserve focus when typing in modal date input', async ({ page }) => {
    // Navigate to edit existing animal
    await page.goto(BASE_URL + '/groups/1');
    await page.waitForLoadState('networkidle');

    const animalCard = page.locator('.animal-card').first();
    if (await animalCard.count() > 0) {
      await animalCard.click();
      await page.waitForLoadState('networkidle');

      const editButton = page.locator('button:has-text("Edit Animal Details")');
      if (await editButton.count() > 0) {
        await editButton.click();
        await page.waitForLoadState('networkidle');

        // Select Bite Quarantine status
        await page.selectOption('select[id="status"]', 'bite_quarantine');

        // Click Update button to open modal
        const updateButton = page.locator('button[type="submit"]:has-text("Update Animal")');
        await updateButton.click();

        // Wait for modal
        const modal = page.locator('.modal');
        await expect(modal).toBeVisible({ timeout: 5000 });

        // Find the date input
        const dateInput = modal.locator('input[type="date"][id="quarantine-date"]');
        await expect(dateInput).toBeVisible();

        // Focus on the date input
        await dateInput.focus();

        // Type a date
        await dateInput.fill('2024-01-15');

        // Verify the date was entered
        const enteredDate = await dateInput.inputValue();
        expect(enteredDate).toBe('2024-01-15');

        console.log('✅ Focus is preserved when entering date in modal');
      }
    }
  });

  test('should allow closing modal with Cancel button', async ({ page }) => {
    // Navigate to edit existing animal
    await page.goto(BASE_URL + '/groups/1');
    await page.waitForLoadState('networkidle');

    const animalCard = page.locator('.animal-card').first();
    if (await animalCard.count() > 0) {
      await animalCard.click();
      await page.waitForLoadState('networkidle');

      const editButton = page.locator('button:has-text("Edit Animal Details")');
      if (await editButton.count() > 0) {
        await editButton.click();
        await page.waitForLoadState('networkidle');

        // Select Bite Quarantine status
        await page.selectOption('select[id="status"]', 'bite_quarantine');

        // Click Update button to open modal
        const updateButton = page.locator('button[type="submit"]:has-text("Update Animal")');
        await updateButton.click();

        // Wait for modal
        const modal = page.locator('.modal');
        await expect(modal).toBeVisible({ timeout: 5000 });

        // Click Cancel button in modal
        const cancelButton = modal.locator('button:has-text("Cancel")');
        await cancelButton.click();

        // Modal should close
        await expect(modal).not.toBeVisible();

        console.log('✅ Modal closes properly when Cancel is clicked');
      }
    }
  });

  test('should allow closing modal with Escape key', async ({ page }) => {
    // Navigate to edit existing animal
    await page.goto(BASE_URL + '/groups/1');
    await page.waitForLoadState('networkidle');

    const animalCard = page.locator('.animal-card').first();
    if (await animalCard.count() > 0) {
      await animalCard.click();
      await page.waitForLoadState('networkidle');

      const editButton = page.locator('button:has-text("Edit Animal Details")');
      if (await editButton.count() > 0) {
        await editButton.click();
        await page.waitForLoadState('networkidle');

        // Select Bite Quarantine status
        await page.selectOption('select[id="status"]', 'bite_quarantine');

        // Click Update button to open modal
        const updateButton = page.locator('button[type="submit"]:has-text("Update Animal")');
        await updateButton.click();

        // Wait for modal
        const modal = page.locator('.modal');
        await expect(modal).toBeVisible({ timeout: 5000 });

        // Press Escape key
        await page.keyboard.press('Escape');

        // Modal should close
        await expect(modal).not.toBeVisible();

        console.log('✅ Modal closes properly when Escape key is pressed');
      }
    }
  });

  test('should have proper button order: Update, Cancel, Delete', async ({ page }) => {
    // Navigate to edit existing animal
    await page.goto(BASE_URL + '/groups/1');
    await page.waitForLoadState('networkidle');

    const animalCard = page.locator('.animal-card').first();
    if (await animalCard.count() > 0) {
      await animalCard.click();
      await page.waitForLoadState('networkidle');

      const editButton = page.locator('button:has-text("Edit Animal Details")');
      if (await editButton.count() > 0) {
        await editButton.click();
        await page.waitForLoadState('networkidle');

        // Get form actions container
        const formActions = page.locator('.form-actions');
        await expect(formActions).toBeVisible();

        // Get all buttons in order
        const buttons = formActions.locator('button');
        const buttonCount = await buttons.count();

        // Should have at least 3 buttons (Update, Cancel, Delete)
        expect(buttonCount).toBeGreaterThanOrEqual(3);

        // Check button order
        const firstButton = await buttons.nth(0).textContent();
        const secondButton = await buttons.nth(1).textContent();
        const thirdButton = await buttons.nth(2).textContent();

        expect(firstButton).toContain('Update Animal');
        expect(secondButton).toContain('Cancel');
        expect(thirdButton).toContain('Delete');

        console.log('✅ Buttons are in correct order: Update, Cancel, Delete');
      }
    }
  });
});

test.describe('Animal Form UX - Accessibility', () => {
  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page);
  });

  test('Cancel button should be keyboard accessible', async ({ page }) => {
    await page.goto(BASE_URL + '/groups/1');
    await page.waitForLoadState('networkidle');

    const animalCard = page.locator('.animal-card').first();
    if (await animalCard.count() > 0) {
      await animalCard.click();
      await page.waitForLoadState('networkidle');

      const editButton = page.locator('button:has-text("Edit Animal Details")');
      if (await editButton.count() > 0) {
        await editButton.click();
        await page.waitForLoadState('networkidle');

        // Tab to Cancel button (after Update button)
        await page.keyboard.press('Tab');
        await page.keyboard.press('Tab');

        // Check if Cancel button has focus
        const focusedElement = await page.evaluate(() => {
          const el = document.activeElement;
          return el?.textContent || '';
        });

        expect(focusedElement).toContain('Cancel');

        console.log('✅ Cancel button is keyboard accessible');
      }
    }
  });

  test('Modal should trap focus within itself', async ({ page }) => {
    await page.goto(BASE_URL + '/groups/1');
    await page.waitForLoadState('networkidle');

    const animalCard = page.locator('.animal-card').first();
    if (await animalCard.count() > 0) {
      await animalCard.click();
      await page.waitForLoadState('networkidle');

      const editButton = page.locator('button:has-text("Edit Animal Details")');
      if (await editButton.count() > 0) {
        await editButton.click();
        await page.waitForLoadState('networkidle');

        // Select Bite Quarantine status
        await page.selectOption('select[id="status"]', 'bite_quarantine');

        // Click Update button to open modal
        const updateButton = page.locator('button[type="submit"]:has-text("Update Animal")');
        await updateButton.click();

        // Wait for modal
        const modal = page.locator('.modal');
        await expect(modal).toBeVisible({ timeout: 5000 });

        // Tab through all focusable elements
        // Should cycle back to first element
        for (let i = 0; i < 10; i++) {
          await page.keyboard.press('Tab');
          
          // Check if focus is still within modal
          const focusedElement = await page.evaluate(() => {
            const el = document.activeElement;
            const modal = document.querySelector('.modal');
            return modal?.contains(el) || false;
          });

          if (!focusedElement) {
            throw new Error('Focus escaped from modal');
          }
        }

        console.log('✅ Modal properly traps focus');
      }
    }
  });
});
