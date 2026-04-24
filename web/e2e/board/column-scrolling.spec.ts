import { test, expect } from '@playwright/test';
import { resetSpace } from '../helpers/reset';

let spaceSlug = '';

test.describe('Board column scrolling', () => {
  test.beforeEach(async ({ page }) => {
    const result = await resetSpace(page);
    spaceSlug = result.slug;
  });

  test('tasks board: column headers are visible and columns exist', async ({ page }) => {
    await page.goto(`/spaces/${spaceSlug}/tasks`);

    // Board should render column headers
    await expect(page.getByText('Ready')).toBeVisible();
    await expect(page.getByText('In Progress')).toBeVisible();
    await expect(page.getByText('Done')).toBeVisible();

    // Columns container should not cause page-level scroll — check that the
    // document body height does not exceed the viewport height significantly.
    const bodyHeight = await page.evaluate(() => document.body.scrollHeight);
    const viewportHeight = await page.evaluate(() => window.innerHeight);
    expect(bodyHeight).toBeLessThanOrEqual(viewportHeight + 10);
  });

  test('jobs board: column headers are visible and columns exist', async ({ page }) => {
    await page.goto(`/spaces/${spaceSlug}/jobs`);

    // Board should render column headers
    await expect(page.getByText('Ready')).toBeVisible();
    await expect(page.getByText('In Progress')).toBeVisible();
    await expect(page.getByText('Done')).toBeVisible();

    // Columns container should not cause page-level scroll
    const bodyHeight = await page.evaluate(() => document.body.scrollHeight);
    const viewportHeight = await page.evaluate(() => window.innerHeight);
    expect(bodyHeight).toBeLessThanOrEqual(viewportHeight + 10);
  });

  test('tasks board: column headers remain visible after scrolling', async ({ page }) => {
    await page.goto(`/spaces/${spaceSlug}/tasks`);

    // Wait for board to load
    await expect(page.getByText('Ready')).toBeVisible();

    // Scroll the page
    await page.evaluate(() => window.scrollBy(0, 500));
    await page.waitForTimeout(100);

    // Column headers should still be visible (not scrolled off)
    await expect(page.getByText('Ready')).toBeVisible();
    await expect(page.getByText('In Progress')).toBeVisible();
    await expect(page.getByText('Done')).toBeVisible();
  });
});
