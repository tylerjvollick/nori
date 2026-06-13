import { test, expect } from '@playwright/test';
import { resetSpace } from '../helpers/reset';

let spaceSlug = '';

test.describe('Job Graph Node Creation', () => {
  test.beforeEach(async ({ page }) => {
    const result = await resetSpace(page);
    spaceSlug = result.slug;
  });

  test('add node creates a visible task in the job graph', async ({ page }) => {
    // Create a job
    await page.goto(`/spaces/${spaceSlug}/jobs`);
    await page.waitForLoadState('networkidle');
    await page.getByRole('button', { name: 'New Job' }).click();
    await page.locator('#job-title').fill('Graph Node Test');
    await page.getByRole('button', { name: 'Create Job' }).click();
    await page.waitForURL(new RegExp(`/spaces/${spaceSlug}/.+`));

    // Open the Graph tab — wait for graph toolbar
    await page.getByRole('tab', { name: 'Graph' }).click();
    const addNodeBtn = page.getByRole('button', { name: 'Add Node' });
    await expect(addNodeBtn).toBeVisible({ timeout: 10000 });

    // Count nodes before
    const nodesBefore = await page.locator('.svelte-flow__node').count();

    // Click Add Node
    await addNodeBtn.click();

    // New node should appear (inline editing activates, so look for the input)
    await expect(page.locator('.svelte-flow__node')).toHaveCount(nodesBefore + 1, { timeout: 10000 });
  });

  test('added node persists after page reload', async ({ page }) => {
    // Create a job
    await page.goto(`/spaces/${spaceSlug}/jobs`);
    await page.waitForLoadState('networkidle');
    await page.getByRole('button', { name: 'New Job' }).click();
    await page.locator('#job-title').fill('Persist Node Test');
    await page.getByRole('button', { name: 'Create Job' }).click();
    await page.waitForURL(new RegExp(`/spaces/${spaceSlug}/.+`));

    await page.getByRole('tab', { name: 'Graph' }).click();
    const addNodeBtn = page.getByRole('button', { name: 'Add Node' });
    await expect(addNodeBtn).toBeVisible({ timeout: 10000 });

    const nodesBefore = await page.locator('.svelte-flow__node').count();
    await addNodeBtn.click();
    await expect(page.locator('.svelte-flow__node')).toHaveCount(nodesBefore + 1, { timeout: 10000 });

    // Reload and verify the node persists (proves addChildTask was used)
    await page.reload();
    await expect(page.getByRole('button', { name: 'Add Node' })).toBeVisible({ timeout: 10000 });
    await expect(page.locator('.svelte-flow__node')).toHaveCount(nodesBefore + 1, { timeout: 10000 });
  });
});
