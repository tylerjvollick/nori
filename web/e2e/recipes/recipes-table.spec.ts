import { test, expect } from '@playwright/test';
import { resetSpace } from '../helpers/reset';

let spaceSlug = '';

test.describe('Recipes Table View', () => {
  test.beforeEach(async ({ page }) => {
    const result = await resetSpace(page);
    spaceSlug = result.slug;
  });

  test('recipes page renders table with correct columns', async ({ page }) => {
    await page.goto(`/spaces/${spaceSlug}/recipes`);
    await expect(page.getByRole('heading', { name: 'Recipes' })).toBeVisible();

    // Create a recipe so the table has at least one row
    await page.getByRole('button', { name: 'New Recipe' }).click();
    await page.locator('#recipe-name').fill('Table View Test Recipe');
    await page.locator('#recipe-description').fill('A recipe to verify table columns');
    await page.getByRole('button', { name: 'Create Recipe' }).click();
    await page.waitForURL(/\/recipes\/.+/);

    // Navigate back to the recipes list (space-scoped, preserves currentSpace)
    await page.goto(`/spaces/${spaceSlug}/recipes`);

    // Verify table column headers are present
    await expect(page.getByRole('columnheader', { name: /Name/i })).toBeVisible();
    await expect(page.getByRole('columnheader', { name: /Description/i })).toBeVisible();
    await expect(page.getByRole('columnheader', { name: /Status/i })).toBeVisible();
    await expect(page.getByRole('columnheader', { name: /Version/i })).toBeVisible();
    await expect(page.getByRole('columnheader', { name: /Updated/i })).toBeVisible();

    // Verify the recipe appears in a table row
    await expect(page.getByRole('cell', { name: 'Table View Test Recipe' })).toBeVisible();
    await expect(page.getByText('Draft')).toBeVisible();
  });

  test('clicking a table row navigates to recipe detail', async ({ page }) => {
    await page.goto(`/spaces/${spaceSlug}/recipes`);

    // Create a recipe
    await page.getByRole('button', { name: 'New Recipe' }).click();
    await page.locator('#recipe-name').fill('Clickable Row Recipe');
    await page.getByRole('button', { name: 'Create Recipe' }).click();
    await page.waitForURL(/\/recipes\/.+/);

    // Navigate back to the list
    await page.goto(`/spaces/${spaceSlug}/recipes`);

    // Click the row for our recipe
    await page.getByRole('cell', { name: 'Clickable Row Recipe' }).click();

    // Should navigate to the recipe detail page
    await page.waitForURL(/\/recipes\/.+/);
    await expect(page.getByRole('heading', { name: 'Clickable Row Recipe' })).toBeVisible();
  });

  test('New Recipe button is visible in the toolbar', async ({ page }) => {
    await page.goto(`/spaces/${spaceSlug}/recipes`);
    await expect(page.getByRole('button', { name: 'New Recipe' })).toBeVisible();
  });

  test('search filter narrows table rows', async ({ page }) => {
    await page.goto(`/spaces/${spaceSlug}/recipes`);

    // Create two recipes
    await page.getByRole('button', { name: 'New Recipe' }).click();
    await page.locator('#recipe-name').fill('Oak Dining Table');
    await page.getByRole('button', { name: 'Create Recipe' }).click();
    await page.waitForURL(/\/recipes\/.+/);

    await page.goto(`/spaces/${spaceSlug}/recipes`);
    await page.getByRole('button', { name: 'New Recipe' }).click();
    await page.locator('#recipe-name').fill('Walnut Bookshelf');
    await page.getByRole('button', { name: 'Create Recipe' }).click();
    await page.waitForURL(/\/recipes\/.+/);

    await page.goto(`/spaces/${spaceSlug}/recipes`);

    // Both recipes visible initially
    await expect(page.getByRole('cell', { name: 'Oak Dining Table' })).toBeVisible();
    await expect(page.getByRole('cell', { name: 'Walnut Bookshelf' })).toBeVisible();

    // Filter to only "Oak"
    await page.getByPlaceholder('Search recipes by name or description...').fill('Oak');
    await expect(page.getByRole('cell', { name: 'Oak Dining Table' })).toBeVisible();
    await expect(page.getByRole('cell', { name: 'Walnut Bookshelf' })).not.toBeVisible();
  });
});
