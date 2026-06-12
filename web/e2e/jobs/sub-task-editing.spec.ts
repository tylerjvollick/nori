import { test, expect, type Page, type Locator } from '@playwright/test';
import { resetSpace } from '../helpers/reset';

let spaceSlug = '';

/**
 * The task detail panel. On a leaf task's own detail page it is the left
 * pane (border-r); we scope by the Sub-tasks heading it contains.
 */
function detailPanel(page: Page): Locator {
  return page
    .locator('div.border-r')
    .filter({ has: page.getByRole('heading', { name: 'Sub-tasks' }) })
    .first();
}

/**
 * Helper: create a job, add a child task, open that child's own detail page
 * (selecting a node now navigates there), and return its detail panel.
 */
async function createJobAndSelectChildTask(page: Page, jobTitle: string): Promise<Locator> {
  await page.goto(`/spaces/${spaceSlug}/jobs`);
  await page.waitForLoadState('networkidle');
  await page.getByRole('button', { name: 'New Job' }).click();
  await page.locator('#job-title').fill(jobTitle);
  await page.getByRole('button', { name: 'Create Job' }).click();
  await page.waitForURL(new RegExp(`/spaces/${spaceSlug}/.+`));

  // Wait for graph, add a child node (graph excludes root, so starts at 0)
  const addNodeBtn = page.getByRole('button', { name: 'Add Node' });
  await expect(addNodeBtn).toBeVisible({ timeout: 10000 });
  await addNodeBtn.click();
  const input = page.locator('.svelte-flow__node input');
  await expect(input).toBeVisible({ timeout: 10000 });
  await page.keyboard.press('Enter'); // commit the default "New Task" title
  await expect(page.locator('.svelte-flow__node')).toHaveCount(1, { timeout: 10000 });

  // Open the child task's own detail page, where its detail panel (with the
  // Sub-tasks section) renders as the left pane.
  const childId = await page.locator('.svelte-flow__node').first().getAttribute('data-id');
  await page.goto(`/spaces/${spaceSlug}/${childId}`);
  await page.waitForLoadState('networkidle');

  const panel = detailPanel(page);
  await expect(panel.getByRole('heading', { name: 'Sub-tasks' })).toBeVisible({ timeout: 10000 });

  return panel;
}

/** Add a sub-task within the panel and wait for it to appear. */
async function addSubTask(panel: Locator, title: string): Promise<void> {
  await panel.getByText('Add sub-task').click();
  await panel.getByPlaceholder('Sub-task title').fill(title);
  await panel.getByRole('button', { name: 'Add', exact: true }).click();
  await expect(panel.getByText(title)).toBeVisible();
}

test.describe('Sub-task editing (in-flight)', () => {
  test.beforeEach(async ({ page }) => {
    const result = await resetSpace(page);
    spaceSlug = result.slug;
  });

  test('add a sub-task via inline form', async ({ page }) => {
    const panel = await createJobAndSelectChildTask(page, 'SubTask Test A');

    await expect(panel.getByText('No sub-tasks yet.')).toBeVisible();
    await addSubTask(panel, 'Sand the edges');

    await expect(panel.getByText('Sand the edges')).toBeVisible();
    await expect(panel.getByText('No sub-tasks yet.')).not.toBeVisible();
  });

  test('add multiple sub-tasks in sequence', async ({ page }) => {
    const panel = await createJobAndSelectChildTask(page, 'SubTask Test B');

    await addSubTask(panel, 'Step 1: Cut wood');
    await addSubTask(panel, 'Step 2: Sand surfaces');

    await expect(panel.getByText('Step 1: Cut wood')).toBeVisible();
    await expect(panel.getByText('Step 2: Sand surfaces')).toBeVisible();
  });

  test('cancel adding a sub-task', async ({ page }) => {
    const panel = await createJobAndSelectChildTask(page, 'SubTask Test C');

    await panel.getByText('Add sub-task').click();
    const input = panel.getByPlaceholder('Sub-task title');
    await input.fill('Should not appear');
    await panel.getByRole('button', { name: 'Cancel', exact: true }).click();

    await expect(input).not.toBeVisible();
    await expect(panel.getByText('Should not appear')).not.toBeVisible();
    await expect(panel.getByText('No sub-tasks yet.')).toBeVisible();
  });

  test('delete a sub-task', async ({ page }) => {
    const panel = await createJobAndSelectChildTask(page, 'SubTask Test D');

    await addSubTask(panel, 'To be deleted');

    // Hover to reveal delete button, then click it
    const row = panel.locator('.group\\/item', { hasText: 'To be deleted' });
    await row.hover();
    await row.getByTitle('Delete').click();

    await expect(panel.getByText('To be deleted')).not.toBeVisible();
    await expect(panel.getByText('No sub-tasks yet.')).toBeVisible();
  });

  test('edit a sub-task title inline', async ({ page }) => {
    const panel = await createJobAndSelectChildTask(page, 'SubTask Test E');

    await addSubTask(panel, 'Original title');

    // Get all sub-task rows — there's only one, so use first()
    const row = panel.locator('.group\\/item').first();
    await row.hover();
    await row.getByTitle('Edit').click();

    // The row now has an input with the current title
    const editInput = row.locator('input');
    await expect(editInput).toBeVisible();
    await editInput.clear();
    await editInput.fill('Updated title');

    // Confirm (green check button)
    await row.locator('button.text-green-500').click();

    await expect(panel.getByText('Updated title')).toBeVisible();
    await expect(panel.getByText('Original title')).not.toBeVisible();
  });

  test('cancel editing a sub-task restores original title', async ({ page }) => {
    const panel = await createJobAndSelectChildTask(page, 'SubTask Test F');

    await addSubTask(panel, 'Keep this title');

    const row = panel.locator('.group\\/item').first();
    await row.hover();
    await row.getByTitle('Edit').click();

    const editInput = row.locator('input');
    await expect(editInput).toBeVisible();
    await editInput.clear();
    await editInput.fill('Changed but will cancel');

    // Cancel (X button — the second button in the edit controls)
    await row.locator('button.text-muted-foreground').last().click();

    await expect(panel.getByText('Keep this title')).toBeVisible();
    await expect(panel.getByText('Changed but will cancel')).not.toBeVisible();
  });

  test('sub-tasks persist after page reload', async ({ page }) => {
    const panel = await createJobAndSelectChildTask(page, 'SubTask Test G');

    await addSubTask(panel, 'Persistent step');

    // Reload the child detail page; its panel renders directly (no re-select).
    await page.reload();
    await page.waitForLoadState('networkidle');

    // Verify sub-task persisted
    const reloadedPanel = detailPanel(page);
    await expect(reloadedPanel.getByRole('heading', { name: 'Sub-tasks' })).toBeVisible({ timeout: 10000 });
    await expect(reloadedPanel.getByText('Persistent step')).toBeVisible({ timeout: 10000 });
  });
});
