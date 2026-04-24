import { test, expect } from '@playwright/test';
import { resetSpace } from '../helpers/reset';

let spaceSlug = '';

test.describe('Recipe Authoring (Graph Editor)', () => {
  test.beforeEach(async ({ page }) => {
    const result = await resetSpace(page);
    spaceSlug = result.slug;
  });

  test('create a new recipe and verify it opens in draft state with graph editor', async ({ page }) => {
    await page.goto(`/spaces/${spaceSlug}/recipes`);
    await page.waitForLoadState('networkidle');
    await expect(page.getByRole('heading', { name: 'Recipes' })).toBeVisible();

    // Open create dialog
    await page.getByRole('button', { name: 'New Recipe' }).click();
    await page.locator('#recipe-name').fill('Oak Dining Table');
    await page.locator('#recipe-description').fill('Standard 6-seat dining table build');
    await page.getByRole('button', { name: 'Create Recipe' }).click();

    // Should navigate to the recipe detail page
    await page.waitForURL(/\/spaces\/[^/]+\/recipes\/.+/);

    // Verify recipe title and draft status
    await expect(page.locator('h1', { hasText: 'Oak Dining Table' })).toBeVisible();
    await expect(page.getByText('Draft')).toBeVisible();

    // Recipe creation automatically creates a root task, so the graph editor loads.
    // Wait for graph toolbar to appear (loading skeleton clears).
    await expect(page.getByRole('button', { name: 'Add Node' })).toBeVisible({ timeout: 10000 });
  });

  test('graph editor toolbar shows controls after loading', async ({ page }) => {
    await page.goto(`/spaces/${spaceSlug}/recipes`);
    await page.waitForLoadState('networkidle');
    await page.getByRole('button', { name: 'New Recipe' }).click();
    await page.locator('#recipe-name').fill('Controls Test');
    await page.getByRole('button', { name: 'Create Recipe' }).click();
    await page.waitForURL(/\/spaces\/[^/]+\/recipes\/.+/);

    // Wait for graph to load
    await expect(page.getByRole('button', { name: 'Add Node' })).toBeVisible({ timeout: 10000 });
    await expect(page.getByRole('button', { name: 'Re-layout' })).toBeVisible();
    await expect(page.getByRole('button', { name: 'Refresh' })).toBeVisible();
  });

  test('back to recipes navigates to the list', async ({ page }) => {
    await page.goto(`/spaces/${spaceSlug}/recipes`);
    await page.waitForLoadState('networkidle');
    await page.getByRole('button', { name: 'New Recipe' }).click();
    await page.locator('#recipe-name').fill('Nav Test Recipe');
    await page.getByRole('button', { name: 'Create Recipe' }).click();
    await page.waitForURL(/\/spaces\/[^/]+\/recipes\/.+/);
    await page.waitForLoadState('networkidle');

    // Click "Back to Recipes"
    await page.getByRole('link', { name: 'Back to Recipes' }).click();
    await page.waitForURL(new RegExp(`/spaces/${spaceSlug}/recipes$`));

    // Verify we're on the recipes list
    await expect(page.getByRole('heading', { name: 'Recipes' })).toBeVisible();
    await expect(page.getByText('Nav Test Recipe')).toBeVisible();
  });

  test('add node creates a visible task in the graph', async ({ page }) => {
    await page.goto(`/spaces/${spaceSlug}/recipes`);
    await page.waitForLoadState('networkidle');
    await page.getByRole('button', { name: 'New Recipe' }).click();
    await page.locator('#recipe-name').fill('Add Node Test');
    await page.getByRole('button', { name: 'Create Recipe' }).click();
    await page.waitForURL(/\/spaces\/[^/]+\/recipes\/.+/);

    // Wait for graph to load
    const addNodeBtn = page.getByRole('button', { name: 'Add Node' });
    await expect(addNodeBtn).toBeVisible({ timeout: 10000 });

    // Count graph nodes before adding
    const nodesBefore = await page.locator('.svelte-flow__node').count();

    // Click Add Node
    await addNodeBtn.click();

    // New node should appear in the graph
    await expect(page.locator('.svelte-flow__node')).toHaveCount(nodesBefore + 1, { timeout: 10000 });

    // The new node should have the default title "New Task"
    await expect(page.locator('.svelte-flow__node', { hasText: 'New Task' })).toBeVisible();
  });

  test('recipe appears in the recipes list after creation', async ({ page }) => {
    await page.goto(`/spaces/${spaceSlug}/recipes`);
    await page.waitForLoadState('networkidle');
    await page.getByRole('button', { name: 'New Recipe' }).click();
    await page.locator('#recipe-name').fill('Walnut Bookshelf');
    await page.getByRole('button', { name: 'Create Recipe' }).click();
    await page.waitForURL(/\/spaces\/[^/]+\/recipes\/.+/);
    await page.waitForLoadState('networkidle');

    // Navigate back to recipes list
    await page.getByRole('link', { name: 'Back to Recipes' }).click();
    await page.waitForURL(new RegExp(`/spaces/${spaceSlug}/recipes$`));
    await page.waitForLoadState('networkidle');

    await expect(page.getByText('Walnut Bookshelf')).toBeVisible();
    await expect(page.getByText('Draft')).toBeVisible();
  });
});
