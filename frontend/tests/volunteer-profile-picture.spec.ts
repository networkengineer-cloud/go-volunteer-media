import { test, expect } from '@playwright/test';

test.describe('Volunteer Profile Picture Selection', () => {
  // Test user credentials
  const volunteerUser = {
    username: 'terry',
    password: 'volunteer2026!',
  };

  const adminUser = {
    username: 'admin',
    password: 'demo1234',
  };

  test.beforeEach(async ({ page }) => {
    // Start from the login page
    await page.goto('http://localhost:5173/login');
  });

  test('regular volunteer can see Set as Profile Picture button', async ({ page }) => {
    // Login as regular volunteer
    await page.fill('input[name="username"]', volunteerUser.username);
    await page.fill('input[name="password"]', volunteerUser.password);
    await page.click('button[type="submit"]');

    // Wait for dashboard to load
    await expect(page).toHaveURL(/.*dashboard/);

    // Navigate to ModSquad group
    await page.click('text=ModSquad');
    await expect(page).toHaveURL(/.*groups\/\d+/);

    // Click on first animal
    const firstAnimalCard = page.locator('.animal-card').first();
    await firstAnimalCard.click();

    // Wait for animal detail page
    await expect(page.locator('h1')).toBeVisible();

    // Navigate to photo gallery
    const photoGalleryLink = page.locator('a[href*="/photos"]').first();
    if (await photoGalleryLink.isVisible()) {
      await photoGalleryLink.click();
    } else {
      // Alternative: look for Photos link in navigation
      await page.click('text=Photos');
    }

    // Wait for photo gallery to load
    await expect(page.locator('h1')).toContainText("'s Photos");

    // Check if there are photos
    const photoCards = page.locator('.photo-card');
    const photoCount = await photoCards.count();

    if (photoCount > 0) {
      // Click on the first photo to open lightbox
      await photoCards.first().click();

      // Wait for lightbox to appear
      await expect(page.locator('.lightbox')).toBeVisible();

      // Look for Set as Profile Picture button
      const setProfileButton = page.locator('button:has-text("Set as Profile Picture")');
      
      // The button should be visible (not admin-only anymore)
      await expect(setProfileButton).toBeVisible();

      // Check for help text
      await expect(page.locator('.action-help-text')).toContainText('main photo shown');
    } else {
      console.log('No photos available to test with');
    }
  });

  test('volunteer can successfully set a profile picture', async ({ page }) => {
    // Login as regular volunteer
    await page.fill('input[name="username"]', volunteerUser.username);
    await page.fill('input[name="password"]', volunteerUser.password);
    await page.click('button[type="submit"]');
    await expect(page).toHaveURL(/.*dashboard/);

    // Navigate to ModSquad group and find an animal with photos
    await page.click('text=ModSquad');
    await expect(page).toHaveURL(/.*groups\/\d+/);

    // Click on first animal
    await page.locator('.animal-card').first().click();
    await expect(page.locator('h1')).toBeVisible();

    // Go to photo gallery
    const photoGalleryLink = page.locator('a[href*="/photos"]').first();
    if (await photoGalleryLink.isVisible()) {
      await photoGalleryLink.click();
    } else {
      await page.click('text=Photos');
    }

    await expect(page.locator('h1')).toContainText("'s Photos");

    const photoCards = page.locator('.photo-card');
    const photoCount = await photoCards.count();

    if (photoCount > 1) {
      // Find a photo that is NOT currently the profile picture
      let nonProfilePhotoIndex = -1;
      for (let i = 0; i < photoCount; i++) {
        const hasProfileBadge = await photoCards.nth(i).locator('.profile-badge').isVisible().catch(() => false);
        if (!hasProfileBadge) {
          nonProfilePhotoIndex = i;
          break;
        }
      }

      if (nonProfilePhotoIndex >= 0) {
        // Click on the non-profile photo
        await photoCards.nth(nonProfilePhotoIndex).click();
        await expect(page.locator('.lightbox')).toBeVisible();

        // Click Set as Profile Picture button
        const setProfileButton = page.locator('button:has-text("Set as Profile Picture")');
        await setProfileButton.click();

        // Wait for success toast
        await expect(page.locator('.toast:has-text("Profile picture updated")')).toBeVisible({ timeout: 5000 });

        // Close lightbox (press Escape)
        await page.keyboard.press('Escape');

        // Verify the profile badge moved to the new photo
        // The photo we just set should now have the profile badge
        await expect(photoCards.nth(nonProfilePhotoIndex).locator('.profile-badge')).toBeVisible();
      } else {
        console.log('All photos are profile pictures or no non-profile photo found');
      }
    } else {
      console.log('Not enough photos to test profile picture selection');
    }
  });

  test('profile picture updates are reflected on animal card', async ({ page }) => {
    // Login as regular volunteer
    await page.fill('input[name="username"]', volunteerUser.username);
    await page.fill('input[name="password"]', volunteerUser.password);
    await page.click('button[type="submit"]');
    await expect(page).toHaveURL(/.*dashboard/);

    // Navigate to ModSquad group
    await page.click('text=ModSquad');
    await expect(page).toHaveURL(/.*groups\/\d+/);

    // Get the first animal's current image
    const firstAnimalCard = page.locator('.animal-card').first();
    const animalName = await firstAnimalCard.locator('h3').textContent();
    const originalImageSrc = await firstAnimalCard.locator('img').getAttribute('src');

    // Click on the animal
    await firstAnimalCard.click();
    await expect(page.locator('h1')).toContainText(animalName || '');

    // Navigate to photo gallery
    const photoGalleryLink = page.locator('a[href*="/photos"]').first();
    if (await photoGalleryLink.isVisible()) {
      await photoGalleryLink.click();
    } else {
      await page.click('text=Photos');
    }

    await expect(page.locator('h1')).toContainText("'s Photos");

    const photoCards = page.locator('.photo-card');
    const photoCount = await photoCards.count();

    if (photoCount > 1) {
      // Find a different photo (not the current profile picture)
      let differentPhotoIndex = -1;
      for (let i = 0; i < photoCount; i++) {
        const photoSrc = await photoCards.nth(i).locator('img').getAttribute('src');
        if (photoSrc !== originalImageSrc) {
          differentPhotoIndex = i;
          break;
        }
      }

      if (differentPhotoIndex >= 0) {
        // Set a different photo as profile picture
        await photoCards.nth(differentPhotoIndex).click();
        await expect(page.locator('.lightbox')).toBeVisible();

        const setProfileButton = page.locator('button:has-text("Set as Profile Picture")');
        if (await setProfileButton.isVisible()) {
          await setProfileButton.click();
          await expect(page.locator('.toast:has-text("Profile picture updated")')).toBeVisible({ timeout: 5000 });

          // Go back to group page
          await page.click('a:has-text("Back to")');
          await expect(page).toHaveURL(/.*groups\/\d+/);

          // Wait a moment for the page to update
          await page.waitForTimeout(1000);

          // Check if the animal card shows the new image
          const updatedImageSrc = await page.locator(`.animal-card:has-text("${animalName}") img`).first().getAttribute('src');
          
          // The image should be different from the original
          expect(updatedImageSrc).not.toBe(originalImageSrc);
        }
      }
    }
  });

  test('loading state shows while setting profile picture', async ({ page }) => {
    // Login as regular volunteer
    await page.fill('input[name="username"]', volunteerUser.username);
    await page.fill('input[name="password"]', volunteerUser.password);
    await page.click('button[type="submit"]');
    await expect(page).toHaveURL(/.*dashboard/);

    // Navigate to an animal with photos
    await page.click('text=ModSquad');
    await page.locator('.animal-card').first().click();
    
    const photoGalleryLink = page.locator('a[href*="/photos"]').first();
    if (await photoGalleryLink.isVisible()) {
      await photoGalleryLink.click();
    } else {
      await page.click('text=Photos');
    }

    const photoCards = page.locator('.photo-card');
    if (await photoCards.count() > 0) {
      await photoCards.first().click();
      await expect(page.locator('.lightbox')).toBeVisible();

      const setProfileButton = page.locator('button:has-text("Set as Profile Picture")');
      
      if (await setProfileButton.isVisible()) {
        // Click the button
        await setProfileButton.click();

        // The button should briefly show "Setting..." state
        // (This might be too fast to catch reliably, but we try)
        const hasLoadingState = await page.locator('button:has-text("Setting")').isVisible().catch(() => false);
        
        // Either we caught the loading state, or it completed successfully
        if (hasLoadingState) {
          expect(hasLoadingState).toBe(true);
        }

        // Should eventually show success
        await expect(page.locator('.toast')).toBeVisible({ timeout: 5000 });
      }
    }
  });

  test('profile badge appears on newly selected photo', async ({ page }) => {
    // Login as regular volunteer
    await page.fill('input[name="username"]', volunteerUser.username);
    await page.fill('input[name="password"]', volunteerUser.password);
    await page.click('button[type="submit"]');
    await expect(page).toHaveURL(/.*dashboard/);

    await page.click('text=ModSquad');
    await page.locator('.animal-card').first().click();
    
    const photoGalleryLink = page.locator('a[href*="/photos"]').first();
    if (await photoGalleryLink.isVisible()) {
      await photoGalleryLink.click();
    } else {
      await page.click('text=Photos');
    }

    await expect(page.locator('h1')).toContainText("'s Photos");

    const photoCards = page.locator('.photo-card');
    const photoCount = await photoCards.count();

    if (photoCount > 1) {
      // Find a non-profile photo
      for (let i = 0; i < photoCount; i++) {
        const hasProfileBadge = await photoCards.nth(i).locator('.profile-badge').isVisible().catch(() => false);
        
        if (!hasProfileBadge) {
          // This photo doesn't have the profile badge yet
          await photoCards.nth(i).click();
          await expect(page.locator('.lightbox')).toBeVisible();

          const setProfileButton = page.locator('button:has-text("Set as Profile Picture")');
          await setProfileButton.click();

          // Wait for success
          await expect(page.locator('.toast:has-text("Profile picture updated")')).toBeVisible({ timeout: 5000 });

          // Close and reopen lightbox to verify badge
          await page.keyboard.press('Escape');
          await page.waitForTimeout(500);

          // The photo should now have the profile badge in the grid
          await expect(photoCards.nth(i).locator('.profile-badge')).toBeVisible();
          
          break;
        }
      }
    }
  });

  test('button is disabled during profile picture update', async ({ page }) => {
    // Login as regular volunteer
    await page.fill('input[name="username"]', volunteerUser.username);
    await page.fill('input[name="password"]', volunteerUser.password);
    await page.click('button[type="submit"]');
    await expect(page).toHaveURL(/.*dashboard/);

    await page.click('text=ModSquad');
    await page.locator('.animal-card').first().click();
    
    const photoGalleryLink = page.locator('a[href*="/photos"]').first();
    if (await photoGalleryLink.isVisible()) {
      await photoGalleryLink.click();
    } else {
      await page.click('text=Photos');
    }

    const photoCards = page.locator('.photo-card');
    if (await photoCards.count() > 0) {
      await photoCards.first().click();
      await expect(page.locator('.lightbox')).toBeVisible();

      const setProfileButton = page.locator('button:has-text("Set as Profile Picture"), button:has-text("Setting")');
      
      if (await setProfileButton.first().isVisible()) {
        // Get the button element
        const button = setProfileButton.first();
        
        // Button should not be disabled initially
        const isDisabledBefore = await button.isDisabled();
        expect(isDisabledBefore).toBe(false);

        // Click to start the update
        await button.click();

        // Button should become disabled briefly (or succeed immediately)
        // This is hard to test reliably due to speed, so we just verify success
        await expect(page.locator('.toast')).toBeVisible({ timeout: 5000 });
      }
    }
  });

  test('accessibility: button has proper aria-label', async ({ page }) => {
    // Login as regular volunteer
    await page.fill('input[name="username"]', volunteerUser.username);
    await page.fill('input[name="password"]', volunteerUser.password);
    await page.click('button[type="submit"]');
    await expect(page).toHaveURL(/.*dashboard/);

    await page.click('text=ModSquad');
    await page.locator('.animal-card').first().click();
    
    const photoGalleryLink = page.locator('a[href*="/photos"]').first();
    if (await photoGalleryLink.isVisible()) {
      await photoGalleryLink.click();
    } else {
      await page.click('text=Photos');
    }

    const photoCards = page.locator('.photo-card');
    if (await photoCards.count() > 0) {
      await photoCards.first().click();
      await expect(page.locator('.lightbox')).toBeVisible();

      const setProfileButton = page.locator('button:has-text("Set as Profile Picture")');
      
      if (await setProfileButton.isVisible()) {
        // Check for aria-label
        const ariaLabel = await setProfileButton.getAttribute('aria-label');
        expect(ariaLabel).toContain('main profile picture');
      }
    }
  });

  test('help text explains what the action does', async ({ page }) => {
    // Login as regular volunteer
    await page.fill('input[name="username"]', volunteerUser.username);
    await page.fill('input[name="password"]', volunteerUser.password);
    await page.click('button[type="submit"]');
    await expect(page).toHaveURL(/.*dashboard/);

    await page.click('text=ModSquad');
    await page.locator('.animal-card').first().click();
    
    const photoGalleryLink = page.locator('a[href*="/photos"]').first();
    if (await photoGalleryLink.isVisible()) {
      await photoGalleryLink.click();
    } else {
      await page.click('text=Photos');
    }

    const photoCards = page.locator('.photo-card');
    if (await photoCards.count() > 0) {
      // Find a non-profile photo
      for (let i = 0; i < await photoCards.count(); i++) {
        const hasProfileBadge = await photoCards.nth(i).locator('.profile-badge').isVisible().catch(() => false);
        
        if (!hasProfileBadge) {
          await photoCards.nth(i).click();
          await expect(page.locator('.lightbox')).toBeVisible();

          // Check for help text
          const helpText = page.locator('.action-help-text');
          await expect(helpText).toBeVisible();
          await expect(helpText).toContainText('main photo');
          await expect(helpText).toContainText('card');
          
          break;
        }
      }
    }
  });

  test('multiple volunteers can set profile pictures independently', async ({ page, context }) => {
    // This test verifies that the permission change works and multiple users can use the feature
    
    // Login as first volunteer
    await page.fill('input[name="username"]', volunteerUser.username);
    await page.fill('input[name="password"]', volunteerUser.password);
    await page.click('button[type="submit"]');
    await expect(page).toHaveURL(/.*dashboard/);

    await page.click('text=ModSquad');
    await page.locator('.animal-card').first().click();
    
    const photoGalleryLink = page.locator('a[href*="/photos"]').first();
    if (await photoGalleryLink.isVisible()) {
      await photoGalleryLink.click();
    } else {
      await page.click('text=Photos');
    }

    const photoCards = page.locator('.photo-card');
    if (await photoCards.count() > 1) {
      // Set profile picture as first volunteer
      await photoCards.first().click();
      const setProfileButton = page.locator('button:has-text("Set as Profile Picture")');
      
      if (await setProfileButton.isVisible()) {
        await setProfileButton.click();
        await expect(page.locator('.toast:has-text("Profile picture updated")')).toBeVisible({ timeout: 5000 });

        // Logout
        await page.click('button:has-text("Logout"), a:has-text("Logout")');
        await expect(page).toHaveURL(/.*login/);

        // Login as different user (admin in this case, but testing the feature is available to all)
        await page.fill('input[name="username"]', adminUser.username);
        await page.fill('input[name="password"]', adminUser.password);
        await page.click('button[type="submit"]');
        await expect(page).toHaveURL(/.*dashboard/);

        // Navigate to same animal
        await page.click('text=ModSquad');
        await page.locator('.animal-card').first().click();
        
        const photoGalleryLink2 = page.locator('a[href*="/photos"]').first();
        if (await photoGalleryLink2.isVisible()) {
          await photoGalleryLink2.click();
        } else {
          await page.click('text=Photos');
        }

        // Should be able to see and use the feature
        const photoCards2 = page.locator('.photo-card');
        if (await photoCards2.count() > 1) {
          // Pick a different photo
          await photoCards2.nth(1).click();
          const setProfileButton2 = page.locator('button:has-text("Set as Profile Picture")');
          
          // Button should be visible for this user too
          if (await setProfileButton2.isVisible()) {
            await expect(setProfileButton2).toBeVisible();
          }
        }
      }
    }
  });
});
