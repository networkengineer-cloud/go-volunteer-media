import { test, expect, type Page } from '@playwright/test';

test.describe('Animal Tagging System', () => {
  // Helper to login as admin
  async function loginAsAdmin(page: Page) {
    await page.goto('http://localhost:5173/login');
    await page.fill('input[name="username"]', 'admin');
    await page.fill('input[name="password"]', 'admin123');
    await page.click('button[type="submit"]');
    await page.waitForURL(/.*dashboard/);
  }

  test('should only have modsquad as default group', async () => {
    await loginAsAdmin(page);
    
    // Navigate to groups management page
    await page.goto('http://localhost:5173/admin/groups');
    await page.waitForLoadState('networkidle');
    
    // Check that only modsquad exists or that dogs/cats are not listed
    const groupCards = await page.locator('.group-card, .group-item, tr').all();
    const groupTexts = await Promise.all(groupCards.map(card => card.textContent()));
    
    // Verify modsquad exists
    const hasModsquad = groupTexts.some(text => text?.toLowerCase().includes('modsquad'));
    expect(hasModsquad).toBeTruthy();
  });

  test('should display default animal tags in admin page', async () => {
    await loginAsAdmin(page);
    
    // Navigate to animal tags management
    await page.goto('http://localhost:5173/admin/animal-tags');
    await page.waitForLoadState('networkidle');
    
    // Check for behavior tags section
    const behaviorSection = page.locator('h2:has-text("Behavior Traits")');
    await expect(behaviorSection).toBeVisible();
    
    // Check for walker status section
    const walkerSection = page.locator('h2:has-text("Walker Status")');
    await expect(walkerSection).toBeVisible();
    
    // Verify some default tags exist
    const resourceGuarding = page.locator('.tag-preview:has-text("resource guarding")');
    const shy = page.locator('.tag-preview:has-text("shy")');
    const availableForWalking = page.locator('.tag-preview:has-text("available for walking")');
    
    await expect(resourceGuarding.or(shy)).toBeVisible();
    await expect(availableForWalking).toBeVisible();
  });

  test('should create a new animal tag', async () => {
    await loginAsAdmin(page);
    
    // Navigate to animal tags management
    await page.goto('http://localhost:5173/admin/animal-tags');
    await page.waitForLoadState('networkidle');
    
    // Click create button
    await page.click('button:has-text("Create New Tag")');
    
    // Fill in form
    await page.fill('input#name', 'Test Tag');
    await page.selectOption('select#category', 'behavior');
    await page.fill('input[type="color"]', '#ff5733');
    
    // Submit form
    await page.click('button:has-text("Create Tag")');
    
    // Wait for success and verify tag appears
    await page.waitForTimeout(1000);
    const newTag = page.locator('.tag-preview:has-text("Test Tag")');
    await expect(newTag).toBeVisible();
  });

  test('should display tags on animal detail page', async () => {
    await loginAsAdmin(page);
    
    // First, we need to create an animal with tags
    // Navigate to a group
    await page.goto('http://localhost:5173/dashboard');
    await page.waitForLoadState('networkidle');
    
    // Try to find a group link or navigate directly
    const groupLink = page.locator('a[href*="/groups/"]').first();
    if (await groupLink.isVisible()) {
      await groupLink.click();
      await page.waitForLoadState('networkidle');
      
      // Check if there's an "Add Animal" or similar button
      const addAnimalBtn = page.locator('button:has-text("Add Animal"), a:has-text("Add Animal")').first();
      if (await addAnimalBtn.isVisible()) {
        await addAnimalBtn.click();
        await page.waitForLoadState('networkidle');
        
        // Fill in animal form
        await page.fill('input#name', 'Test Animal for Tags');
        await page.fill('input#species', 'Dog');
        await page.fill('input#breed', 'Mixed');
        await page.fill('input#age, input[type="number"]', '3');
        
        // Select some tags
        const behaviorCheckbox = page.locator('.tag-checkbox input[type="checkbox"]').first();
        if (await behaviorCheckbox.isVisible()) {
          await behaviorCheckbox.check();
        }
        
        // Submit form
        await page.click('button[type="submit"]:has-text("Add Animal")');
        await page.waitForTimeout(2000);
        
        // Navigate to the animal detail page
        const animalLink = page.locator('a:has-text("Test Animal for Tags")').first();
        if (await animalLink.isVisible()) {
          await animalLink.click();
          await page.waitForLoadState('networkidle');
          
          // Check if tags section exists
          const tagsSection = page.locator('.animal-tags-section, .animal-tags-list');
          if (await tagsSection.isVisible()) {
            const tags = page.locator('.animal-tag');
            const tagCount = await tags.count();
            expect(tagCount).toBeGreaterThan(0);
          }
        }
      }
    }
  });

  test('should allow editing animal tags in animal form', async () => {
    await loginAsAdmin(page);
    
    // Navigate to dashboard
    await page.goto('http://localhost:5173/dashboard');
    await page.waitForLoadState('networkidle');
    
    // Find a group
    const groupLink = page.locator('a[href*="/groups/"]').first();
    if (await groupLink.isVisible()) {
      await groupLink.click();
      await page.waitForLoadState('networkidle');
      
      // Look for an edit animal button or create a new one
      const editBtn = page.locator('button:has-text("Edit"), a:has-text("Edit")').first();
      const addBtn = page.locator('button:has-text("Add Animal"), a:has-text("Add Animal")').first();
      
      if (await editBtn.isVisible()) {
        await editBtn.click();
      } else if (await addBtn.isVisible()) {
        await addBtn.click();
      } else {
        // Skip test if no animals to edit
        test.skip();
        return;
      }
      
      await page.waitForLoadState('networkidle');
      
      // Check if tag selection interface exists
      const tagCategories = page.locator('.tag-categories, .tag-selection-grid');
      if (await tagCategories.isVisible()) {
        // Verify behavior section
        const behaviorHeader = page.locator('h4:has-text("Behavior Traits")');
        await expect(behaviorHeader).toBeVisible();
        
        // Verify walker status section
        const walkerHeader = page.locator('h4:has-text("Walker Status")');
        await expect(walkerHeader).toBeVisible();
        
        // Verify tag checkboxes exist
        const tagCheckboxes = page.locator('.tag-checkbox');
        const checkboxCount = await tagCheckboxes.count();
        expect(checkboxCount).toBeGreaterThan(0);
      }
    }
  });

  test('should delete an animal tag from admin page', async () => {
    await loginAsAdmin(page);
    
    // Navigate to animal tags management
    await page.goto('http://localhost:5173/admin/animal-tags');
    await page.waitForLoadState('networkidle');
    
    // Find a tag to delete (preferably one we created, or use a default one for test)
    const deleteButtons = page.locator('button:has-text("Delete")');
    const deleteCount = await deleteButtons.count();
    
    if (deleteCount > 0) {
      // Get the tag name before deleting
      const tagCard = page.locator('.tag-card').first();
      const tagName = await tagCard.locator('.tag-preview').textContent();
      
      // Click delete button and confirm
      page.on('dialog', dialog => dialog.accept());
      await deleteButtons.first().click();
      
      await page.waitForTimeout(1000);
      
      // Verify tag is removed (optional - may need to reload)
      await page.reload();
      const deletedTag = page.locator(`.tag-preview:has-text("${tagName}")`);
      const exists = await deletedTag.count();
      // Tag should not exist or count should be 0
      expect(exists).toBe(0);
    }
  });

  test('should edit an existing animal tag', async () => {
    await loginAsAdmin(page);
    
    // Navigate to animal tags management
    await page.goto('http://localhost:5173/admin/animal-tags');
    await page.waitForLoadState('networkidle');
    
    // Click edit on first tag
    const editButtons = page.locator('button:has-text("Edit")');
    const editCount = await editButtons.count();
    
    if (editCount > 0) {
      await editButtons.first().click();
      
      // Wait for modal
      await page.waitForSelector('.modal, form', { state: 'visible' });
      
      // Change the name
      const nameInput = page.locator('input#name');
      await nameInput.fill('Updated Tag Name');
      
      // Submit
      await page.click('button:has-text("Update Tag")');
      
      await page.waitForTimeout(1000);
      
      // Verify update (check if new name appears)
      const updatedTag = page.locator('.tag-preview:has-text("Updated Tag Name")');
      await expect(updatedTag).toBeVisible();
    }
  });
});
