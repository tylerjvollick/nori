import { test, expect } from '@playwright/test';
import { resetSpace } from '../helpers/reset';

let spaceSlug = '';

test.describe('Jobs tab: board/list view toggle', () => {
  test.beforeEach(async ({ page }) => {
    const result = await resetSpace(page);
    spaceSlug = result.slug;
  });

  test('jobs tab shows board|list toggle', async ({ page }) => {
    await page.goto(`/spaces/${spaceSlug}/jobs`);
    await page.waitForLoadState('networkidle');

    await expect(page.getByRole('button', { name: 'Board' })).toBeVisible();
    await expect(page.getByRole('button', { name: 'List' })).toBeVisible();
  });

  test('default view is board (kanban)', async ({ page }) => {
    await page.goto(`/spaces/${spaceSlug}/jobs`);
    await page.waitForLoadState('networkidle');

    // Kanban columns should be visible
    await expect(page.getByText('Ready')).toBeVisible();
    await expect(page.getByText('In Progress')).toBeVisible();
    await expect(page.getByText('Done')).toBeVisible();

    // List table should not be present
    await expect(page.locator('[data-testid="jobs-list-table"]')).not.toBeVisible();
  });

  test('clicking List switches to list view and updates URL', async ({ page }) => {
    await page.goto(`/spaces/${spaceSlug}/jobs`);
    await page.waitForLoadState('networkidle');

    await page.getByRole('button', { name: 'List' }).click();

    // URL should have ?view=list
    await expect(page).toHaveURL(/\?view=list/);
  });

  test('list view renders table with correct columns', async ({ page }) => {
    // Create a job first so the table has data
    await page.goto(`/spaces/${spaceSlug}/jobs`);
    await page.waitForLoadState('networkidle');
    await page.getByRole('button', { name: 'New Job' }).click();
    await page.locator('#job-title').fill('Cherry Dresser');
    await page.getByRole('button', { name: 'Create Job' }).click();
    await page.waitForURL(new RegExp(`/spaces/${spaceSlug}/.+`));

    // Navigate back and switch to list view
    await page.goto(`/spaces/${spaceSlug}/jobs?view=list`);
    await page.waitForLoadState('networkidle');

    // Table should be visible
    const table = page.locator('[data-testid="jobs-list-table"]');
    await expect(table).toBeVisible();

    // All expected column headers should be present
    await expect(table.getByText('ID')).toBeVisible();
    await expect(table.getByText('Title')).toBeVisible();
    await expect(table.getByText('Status')).toBeVisible();
    await expect(table.getByText('Progress')).toBeVisible();
    await expect(table.getByText('Priority')).toBeVisible();
    await expect(table.getByText('Station')).toBeVisible();
    await expect(table.getByText('Time')).toBeVisible();
    await expect(table.getByText('Due')).toBeVisible();

    // The created job should appear in the table
    await expect(table.getByText('Cherry Dresser')).toBeVisible();
  });

  test('?view=list in URL directly loads list view', async ({ page }) => {
    await page.goto(`/spaces/${spaceSlug}/jobs?view=list`);
    await page.waitForLoadState('networkidle');

    // List table container should be visible (even if empty)
    // The loading skeleton or empty state or table should appear
    // Board columns (Ready/In Progress/Done headers) should not be visible
    const readyColumn = page.getByRole('heading', { name: 'Ready' });
    await expect(readyColumn).not.toBeVisible();
  });

  test('clicking Board from list view switches back to kanban', async ({ page }) => {
    await page.goto(`/spaces/${spaceSlug}/jobs?view=list`);
    await page.waitForLoadState('networkidle');

    await page.getByRole('button', { name: 'Board' }).click();

    // URL should not have ?view param
    await expect(page).not.toHaveURL(/\?view=/);

    // Kanban columns should be visible
    await expect(page.getByText('Ready')).toBeVisible();
  });

  test('keyboard shortcut l switches to list view', async ({ page }) => {
    await page.goto(`/spaces/${spaceSlug}/jobs`);
    await page.waitForLoadState('networkidle');

    // Press l to switch to list
    await page.keyboard.press('l');

    await expect(page).toHaveURL(/\?view=list/);
  });

  test('keyboard shortcut b switches back to board view', async ({ page }) => {
    await page.goto(`/spaces/${spaceSlug}/jobs?view=list`);
    await page.waitForLoadState('networkidle');

    // Press b to switch to board
    await page.keyboard.press('b');

    await expect(page).not.toHaveURL(/\?view=/);
    await expect(page.getByText('Ready')).toBeVisible();
  });
});
