import { test, expect } from '@playwright/test';

/**
 * Dark Mode Contrast & Readability Tests
 * 
 * These tests ensure that all UI elements remain readable and accessible
 * in dark mode, with proper contrast ratios and visibility.
 * 
 * Focus areas:
 * - Comment forms and cards
 * - Group selector dropdowns
 * - Form inputs and textareas
 * - Text content readability
 */

test.describe('Dark Mode Contrast & Readability', () => {
  test.beforeEach(async ({ page }) => {
    // Login as admin to access all features
    await page.goto('/login');
    await page.fill('input[name="email"]', 'admin@example.com');
    await page.fill('input[name="password"]', 'AdminPass123!');
    await page.click('button[type="submit"]');
    await page.waitForURL('/dashboard');
    
    // Enable dark mode
    await page.evaluate(() => {
      document.documentElement.setAttribute('data-theme', 'dark');
      localStorage.setItem('theme', 'dark');
    });
  });

  test.describe('Comment Form Dark Mode', () => {
    test('comment form should be visible and readable in dark mode', async ({ page }) => {
      // Navigate to an animal detail page
      await page.goto('/dashboard');
      
      // Find and click on the first animal
      const firstAnimalLink = page.locator('.animal-card-mini, .animal-card').first();
      await firstAnimalLink.click();
      
      // Wait for animal detail page to load
      await page.waitForSelector('.animal-detail-page', { timeout: 10000 });
      
      // Verify comment form is visible
      const commentForm = page.locator('.comment-form');
      await expect(commentForm).toBeVisible();
      
      // Check that the form has appropriate dark mode styling
      const formBg = await commentForm.evaluate((el) => {
        return window.getComputedStyle(el).backgroundColor;
      });
      
      // Background should NOT be pure white (rgb(255, 255, 255)) in dark mode
      expect(formBg).not.toBe('rgb(255, 255, 255)');
      console.log('Comment form background in dark mode:', formBg);
      
      // Check textarea visibility
      const textarea = commentForm.locator('textarea');
      await expect(textarea).toBeVisible();
      
      const textareaStyles = await textarea.evaluate((el) => {
        const styles = window.getComputedStyle(el);
        return {
          background: styles.backgroundColor,
          color: styles.color,
          border: styles.borderColor
        };
      });
      
      console.log('Textarea styles in dark mode:', textareaStyles);
      
      // Textarea should not have white background in dark mode
      expect(textareaStyles.background).not.toBe('rgb(255, 255, 255)');
      
      // Text color should be light (not dark gray/black) in dark mode
      // Light colors have higher RGB values (typically > 200)
      const colorMatch = textareaStyles.color.match(/rgb\((\d+),\s*(\d+),\s*(\d+)\)/);
      if (colorMatch) {
        const [_, r, g, b] = colorMatch.map(Number);
        const avgColor = (r + g + b) / 3;
        expect(avgColor).toBeGreaterThan(150); // Should be light colored text
        console.log('Textarea text color average:', avgColor);
      }
    });

    test('comment form buttons should be readable in dark mode', async ({ page }) => {
      // Navigate to an animal detail page
      await page.goto('/dashboard');
      const firstAnimalLink = page.locator('.animal-card-mini, .animal-card').first();
      await firstAnimalLink.click();
      await page.waitForSelector('.animal-detail-page', { timeout: 10000 });
      
      // Check "Add Comment" toggle button
      const toggleButton = page.locator('.btn-toggle-form');
      if (await toggleButton.isVisible()) {
        const btnStyles = await toggleButton.evaluate((el) => {
          const styles = window.getComputedStyle(el);
          return {
            background: styles.backgroundColor,
            color: styles.color,
            border: styles.borderColor
          };
        });
        
        console.log('Toggle button styles in dark mode:', btnStyles);
        
        // Button should have appropriate dark mode styling
        expect(btnStyles.background).not.toBe('rgb(255, 255, 255)');
      }
      
      // Check post comment button
      const postButton = page.locator('.btn-post');
      await expect(postButton).toBeVisible();
      
      // Post button should maintain brand color (should be visible)
      const postBtnColor = await postButton.evaluate((el) => {
        return window.getComputedStyle(el).backgroundColor;
      });
      
      console.log('Post button background in dark mode:', postBtnColor);
      // Brand color should be present (not white, not pure black)
      expect(postBtnColor).not.toBe('rgb(255, 255, 255)');
      expect(postBtnColor).not.toBe('rgb(0, 0, 0)');
    });
  });

  test.describe('Comment Cards Dark Mode', () => {
    test('existing comment cards should be readable in dark mode', async ({ page }) => {
      // Navigate to an animal with comments
      await page.goto('/dashboard');
      const firstAnimalLink = page.locator('.animal-card-mini, .animal-card').first();
      await firstAnimalLink.click();
      await page.waitForSelector('.animal-detail-page', { timeout: 10000 });
      
      // Check if there are any comment cards
      const commentCards = page.locator('.comment-card');
      const count = await commentCards.count();
      
      if (count > 0) {
        const firstCard = commentCards.first();
        await expect(firstCard).toBeVisible();
        
        // Check card background
        const cardBg = await firstCard.evaluate((el) => {
          return window.getComputedStyle(el).backgroundColor;
        });
        
        console.log('Comment card background in dark mode:', cardBg);
        
        // Card background should NOT be pure white in dark mode
        expect(cardBg).not.toBe('rgb(255, 255, 255)');
        
        // Check text content is visible
        const commentContent = firstCard.locator('.comment-content');
        if (await commentContent.isVisible()) {
          const textColor = await commentContent.evaluate((el) => {
            return window.getComputedStyle(el).color;
          });
          
          console.log('Comment text color in dark mode:', textColor);
          
          // Text should be light colored in dark mode
          const colorMatch = textColor.match(/rgb\((\d+),\s*(\d+),\s*(\d+)\)/);
          if (colorMatch) {
            const [_, r, g, b] = colorMatch.map(Number);
            const avgColor = (r + g + b) / 3;
            expect(avgColor).toBeGreaterThan(150); // Light text
            console.log('Comment text color average:', avgColor);
          }
        }
      } else {
        console.log('No comments found to test - will post a test comment');
        
        // Post a test comment to verify the styling
        await page.fill('textarea', 'Test comment for dark mode validation');
        await page.click('.btn-post');
        
        // Wait for comment to appear
        await page.waitForSelector('.comment-card', { timeout: 5000 });
        
        // Now verify the newly created comment
        const newCard = page.locator('.comment-card').first();
        const cardBg = await newCard.evaluate((el) => {
          return window.getComputedStyle(el).backgroundColor;
        });
        
        expect(cardBg).not.toBe('rgb(255, 255, 255)');
        console.log('New comment card background in dark mode:', cardBg);
      }
    });
  });

  test.describe('Group Selector Dark Mode', () => {
    test('group selector on dashboard should be readable in dark mode', async ({ page }) => {
      await page.goto('/dashboard');
      
      // Find the group selector
      const groupSelect = page.locator('.group-select');
      await expect(groupSelect).toBeVisible();
      
      // Check select element styling
      const selectStyles = await groupSelect.evaluate((el) => {
        const styles = window.getComputedStyle(el);
        return {
          background: styles.backgroundColor,
          color: styles.color,
          border: styles.borderColor
        };
      });
      
      console.log('Group selector styles on dashboard (dark mode):', selectStyles);
      
      // Background should be dark, not white
      expect(selectStyles.background).not.toBe('rgb(255, 255, 255)');
      
      // Text color should be light in dark mode
      const colorMatch = selectStyles.color.match(/rgb\((\d+),\s*(\d+),\s*(\d+)\)/);
      if (colorMatch) {
        const [_, r, g, b] = colorMatch.map(Number);
        const avgColor = (r + g + b) / 3;
        expect(avgColor).toBeGreaterThan(150); // Light colored text
        console.log('Group selector text color average:', avgColor);
      }
      
      // Verify selector options are also readable
      await groupSelect.click();
      
      // Check if dropdown options are visible (implementation may vary)
      const options = await groupSelect.locator('option').all();
      expect(options.length).toBeGreaterThan(0);
      
      console.log('Group selector has', options.length, 'options');
    });

    test('group selector on group page should be readable in dark mode', async ({ page }) => {
      // Navigate to a group page
      await page.goto('/dashboard');
      
      // Click on a group card or navigate to first group
      const groupCard = page.locator('.group-card').first();
      if (await groupCard.isVisible()) {
        await groupCard.click();
      } else {
        // If no group cards, try direct navigation
        await page.goto('/groups/1');
      }
      
      // Wait for group page to load
      await page.waitForSelector('.group-page, .group-header', { timeout: 10000 });
      
      // Find the group selector
      const groupSelect = page.locator('.group-select');
      if (await groupSelect.isVisible()) {
        // Check select element styling
        const selectStyles = await groupSelect.evaluate((el) => {
          const styles = window.getComputedStyle(el);
          return {
            background: styles.backgroundColor,
            color: styles.color,
            border: styles.borderColor
          };
        });
        
        console.log('Group selector styles on group page (dark mode):', selectStyles);
        
        // Background should be dark, not white
        expect(selectStyles.background).not.toBe('rgb(255, 255, 255)');
        
        // Text color should be light in dark mode
        const colorMatch = selectStyles.color.match(/rgb\((\d+),\s*(\d+),\s*(\d+)\)/);
        if (colorMatch) {
          const [_, r, g, b] = colorMatch.map(Number);
          const avgColor = (r + g + b) / 3;
          expect(avgColor).toBeGreaterThan(150);
          console.log('Group selector text color average:', avgColor);
        }
      }
    });
  });

  test.describe('Form Inputs Dark Mode', () => {
    test('all text inputs should be readable in dark mode', async ({ page }) => {
      // Test settings page inputs
      await page.goto('/settings');
      
      const textInputs = page.locator('input[type="text"], input[type="email"], input[type="password"]');
      const count = await textInputs.count();
      
      if (count > 0) {
        const firstInput = textInputs.first();
        await expect(firstInput).toBeVisible();
        
        const inputStyles = await firstInput.evaluate((el) => {
          const styles = window.getComputedStyle(el);
          return {
            background: styles.backgroundColor,
            color: styles.color,
            border: styles.borderColor
          };
        });
        
        console.log('Text input styles in dark mode:', inputStyles);
        
        // Input should not have white background in dark mode
        expect(inputStyles.background).not.toBe('rgb(255, 255, 255)');
        
        // Text color should be light
        const colorMatch = inputStyles.color.match(/rgb\((\d+),\s*(\d+),\s*(\d+)\)/);
        if (colorMatch) {
          const [_, r, g, b] = colorMatch.map(Number);
          const avgColor = (r + g + b) / 3;
          expect(avgColor).toBeGreaterThan(150);
        }
      }
    });
  });

  test.describe('Mobile Dark Mode', () => {
    test('comment form should be readable in dark mode on mobile', async ({ page }) => {
      // Set mobile viewport
      await page.setViewportSize({ width: 375, height: 667 });
      
      // Navigate to animal detail page
      await page.goto('/dashboard');
      const firstAnimalLink = page.locator('.animal-card-mini, .animal-card').first();
      await firstAnimalLink.click();
      await page.waitForSelector('.animal-detail-page', { timeout: 10000 });
      
      // Check comment form
      const commentForm = page.locator('.comment-form');
      await expect(commentForm).toBeVisible();
      
      const formBg = await commentForm.evaluate((el) => {
        return window.getComputedStyle(el).backgroundColor;
      });
      
      // Should not be white in dark mode
      expect(formBg).not.toBe('rgb(255, 255, 255)');
      console.log('Mobile comment form background in dark mode:', formBg);
    });

    test('group selector should be readable in dark mode on mobile', async ({ page }) => {
      // Set mobile viewport
      await page.setViewportSize({ width: 375, height: 667 });
      
      await page.goto('/dashboard');
      
      const groupSelect = page.locator('.group-select');
      await expect(groupSelect).toBeVisible();
      
      const selectStyles = await groupSelect.evaluate((el) => {
        const styles = window.getComputedStyle(el);
        return {
          background: styles.backgroundColor,
          color: styles.color
        };
      });
      
      console.log('Mobile group selector styles in dark mode:', selectStyles);
      
      // Should not be white background
      expect(selectStyles.background).not.toBe('rgb(255, 255, 255)');
    });
  });

  test.describe('Contrast Validation', () => {
    test('should log color combinations for manual contrast review', async ({ page }) => {
      console.log('\n=== DARK MODE COLOR AUDIT ===');
      
      // Navigate to various pages and log color combinations
      await page.goto('/dashboard');
      
      // Check group selector
      const groupSelect = page.locator('.group-select').first();
      if (await groupSelect.isVisible()) {
        const colors = await groupSelect.evaluate((el) => {
          const styles = window.getComputedStyle(el);
          return {
            component: 'Group Selector',
            background: styles.backgroundColor,
            text: styles.color,
            border: styles.borderColor
          };
        });
        console.log(colors);
      }
      
      // Navigate to animal detail and check comment form
      const firstAnimalLink = page.locator('.animal-card-mini, .animal-card').first();
      if (await firstAnimalLink.isVisible()) {
        await firstAnimalLink.click();
        await page.waitForSelector('.animal-detail-page', { timeout: 10000 });
        
        const commentForm = page.locator('.comment-form').first();
        if (await commentForm.isVisible()) {
          const formColors = await commentForm.evaluate((el) => {
            const styles = window.getComputedStyle(el);
            return {
              component: 'Comment Form',
              background: styles.backgroundColor,
              text: styles.color,
              border: styles.borderColor
            };
          });
          console.log(formColors);
          
          const textarea = commentForm.locator('textarea');
          if (await textarea.isVisible()) {
            const textareaColors = await textarea.evaluate((el) => {
              const styles = window.getComputedStyle(el);
              return {
                component: 'Comment Textarea',
                background: styles.backgroundColor,
                text: styles.color,
                border: styles.borderColor
              };
            });
            console.log(textareaColors);
          }
        }
        
        // Check comment cards if present
        const commentCard = page.locator('.comment-card').first();
        if (await commentCard.isVisible()) {
          const cardColors = await commentCard.evaluate((el) => {
            const styles = window.getComputedStyle(el);
            return {
              component: 'Comment Card',
              background: styles.backgroundColor,
              text: styles.color,
              border: styles.borderColor
            };
          });
          console.log(cardColors);
        }
      }
      
      console.log('=== END COLOR AUDIT ===\n');
    });
  });
});
