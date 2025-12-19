import { test, expect, type Page } from '@playwright/test';
import { loginAsVolunteer, loginAsGroupAdmin } from './helpers/auth';

test.describe('Activity Feed Animal Links', () => {
  test.describe.configure({ mode: 'serial' });

  const MODSQUAD_GROUP_NAME = 'modsquad';

  const getGroupIdByName = async (page: Page, groupName: string) => {
    const groupId = await page.evaluate(async (targetName) => {
      const token = localStorage.getItem('token');
      const res = await fetch('/api/groups', {
        headers: token ? { Authorization: `Bearer ${token}` } : {},
      });

      if (!res.ok) {
        throw new Error(`Failed to fetch groups: ${res.status}`);
      }

      const groups = await res.json();
      const match = groups.find((g: { name: string }) => g.name.toLowerCase() === targetName.toLowerCase());
      return match?.id ?? null;
    }, groupName);

    if (!groupId) {
      throw new Error(`Group not found: ${groupName}`);
    }

    return groupId as number;
  };

  test('regular user can click animal name link in activity feed to view animal profile', async ({ page }) => {
    await loginAsVolunteer(page);
    const modsquadId = await getGroupIdByName(page, MODSQUAD_GROUP_NAME);
    await page.goto(`/groups/${modsquadId}`);
    
    // Wait for activity feed to load
    await page.waitForSelector('.activity-feed', { timeout: 10000 });
    
    // Find an activity item with an animal link
    const animalLink = page.locator('.activity-animal').first();
    
    // Check if there are any animal links in the activity feed
    const count = await animalLink.count();
    if (count === 0) {
      console.log('No animal links found in activity feed, skipping test');
      test.skip();
      return;
    }
    
    await expect(animalLink).toBeVisible();
    
    // Get the href to verify it includes /view
    const href = await animalLink.getAttribute('href');
    expect(href).toMatch(/\/groups\/\d+\/animals\/\d+\/view$/);
    
    // Click the link
    await animalLink.click();
    
    // Should navigate to the animal detail page, not the edit page
    await expect(page).toHaveURL(/\/groups\/\d+\/animals\/\d+\/view$/);
    
    // The animal detail page should be visible (not redirect to dashboard due to permissions)
    await expect(page.locator('.animal-detail-page, .animal-profile, h1')).toBeVisible({ timeout: 10000 });
  });

  test('regular user can click "View Profile" button in activity feed to view animal profile', async ({ page }) => {
    await loginAsVolunteer(page);
    const modsquadId = await getGroupIdByName(page, MODSQUAD_GROUP_NAME);
    await page.goto(`/groups/${modsquadId}`);
    
    // Wait for activity feed to load
    await page.waitForSelector('.activity-feed', { timeout: 10000 });
    
    // Find the "View Profile" button
    const viewProfileBtn = page.locator('.btn-view-profile').first();
    
    // Check if there are any view profile buttons
    const count = await viewProfileBtn.count();
    if (count === 0) {
      console.log('No view profile buttons found in activity feed, skipping test');
      test.skip();
      return;
    }
    
    await expect(viewProfileBtn).toBeVisible();
    
    // Get the href to verify it includes /view
    const href = await viewProfileBtn.getAttribute('href');
    expect(href).toMatch(/\/groups\/\d+\/animals\/\d+\/view$/);
    
    // Click the button
    await viewProfileBtn.click();
    
    // Should navigate to the animal detail page, not the edit page
    await expect(page).toHaveURL(/\/groups\/\d+\/animals\/\d+\/view$/);
    
    // The animal detail page should be visible
    await expect(page.locator('.animal-detail-page, .animal-profile, h1')).toBeVisible({ timeout: 10000 });
  });

  test('group admin can also access animal view page from activity feed', async ({ page }) => {
    await loginAsGroupAdmin(page);
    const modsquadId = await getGroupIdByName(page, MODSQUAD_GROUP_NAME);
    await page.goto(`/groups/${modsquadId}`);
    
    // Wait for activity feed to load
    await page.waitForSelector('.activity-feed', { timeout: 10000 });
    
    // Find an activity item with an animal link
    const animalLink = page.locator('.activity-animal').first();
    
    // Check if there are any animal links
    const count = await animalLink.count();
    if (count === 0) {
      console.log('No animal links found in activity feed, skipping test');
      test.skip();
      return;
    }
    
    await expect(animalLink).toBeVisible();
    
    // Get the href to verify it includes /view
    const href = await animalLink.getAttribute('href');
    expect(href).toMatch(/\/groups\/\d+\/animals\/\d+\/view$/);
    
    // Click the link
    await animalLink.click();
    
    // Should navigate to the animal view page
    await expect(page).toHaveURL(/\/groups\/\d+\/animals\/\d+\/view$/);
    
    // The animal detail page should be visible
    await expect(page.locator('.animal-detail-page, .animal-profile, h1')).toBeVisible({ timeout: 10000 });
  });
});
