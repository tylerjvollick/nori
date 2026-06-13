import { test, expect, type Page } from '@playwright/test';
import { resetSpace } from './helpers/reset';

let spaceSlug = '';

/** Create a job with the given child task titles (each a sibling under the job). */
async function createJobWithChildren(
  page: Page,
  jobTitle: string,
  childTitles: string[],
): Promise<void> {
  await page.goto(`/spaces/${spaceSlug}/jobs`);
  await page.waitForLoadState('networkidle');
  await page.getByRole('button', { name: 'New Job' }).click();
  await page.locator('#job-title').fill(jobTitle);
  await page.getByRole('button', { name: 'Create Job' }).click();

  const addNodeBtn = page.getByRole('button', { name: 'Add Node', exact: true });
  await expect(addNodeBtn).toBeVisible({ timeout: 10_000 });

  for (const title of childTitles) {
    await addNodeBtn.click();
    const input = page.locator('.svelte-flow__node input');
    await expect(input).toBeVisible({ timeout: 10_000 });
    await input.fill(title);
    await page.keyboard.press('Enter');
    await expect(page.locator('.svelte-flow__node', { hasText: title })).toBeVisible({
      timeout: 10_000,
    });
  }
}

// nori-oyq: keyboard shortcut to delete the selected node from the jobs graph.
test.describe('Graph view: keyboard delete shortcut', () => {
  test.beforeEach(async ({ page }) => {
    const result = await resetSpace(page);
    spaceSlug = result.slug;
  });

  test('select a node with j, press Delete, confirm — node is removed', async ({ page }) => {
    await createJobWithChildren(page, 'KB Delete Job', ['Keep Me', 'Remove Me']);
    await expect(page.locator('.svelte-flow__node')).toHaveCount(2, { timeout: 10_000 });

    // Select a node via keyboard.
    await page.keyboard.press('j');
    const selected = page.locator('.svelte-flow__node.selected');
    await expect(selected).toHaveCount(1, { timeout: 5_000 });
    const removedId = await selected.getAttribute('data-id');
    expect(removedId).toBeTruthy();

    // Delete opens a confirmation dialog — nothing is deleted yet.
    await page.keyboard.press('Delete');
    const dialog = page.getByRole('dialog');
    await expect(dialog).toBeVisible({ timeout: 5_000 });
    await expect(dialog).toContainText('Delete task?');
    await expect(page.locator('.svelte-flow__node')).toHaveCount(2);

    // Confirm — the selected node is removed.
    await dialog.getByRole('button', { name: 'Delete', exact: true }).click();
    await expect(
      page.locator(`.svelte-flow__node[data-id="${removedId}"]`),
    ).toHaveCount(0, { timeout: 10_000 });
    await expect(page.locator('.svelte-flow__node')).toHaveCount(1, { timeout: 10_000 });
  });

  test('keyboard help overlay documents the delete shortcut', async ({ page }) => {
    await createJobWithChildren(page, 'KB Help Job', ['One']);

    await page.keyboard.press('?');
    const dialog = page.getByRole('dialog', { name: 'Keyboard Shortcuts' });
    await expect(dialog).toBeVisible({ timeout: 5_000 });

    const row = dialog.locator('div.flex').filter({ hasText: 'Delete selected node' });
    await expect(row.locator('kbd')).toHaveText('Del / ⌫');
  });

  test('canceling the confirmation keeps the node (no accidental delete)', async ({ page }) => {
    await createJobWithChildren(page, 'KB Cancel Job', ['Alpha', 'Beta']);
    await expect(page.locator('.svelte-flow__node')).toHaveCount(2, { timeout: 10_000 });

    await page.keyboard.press('j');
    await expect(page.locator('.svelte-flow__node.selected')).toHaveCount(1, { timeout: 5_000 });

    await page.keyboard.press('Delete');
    const dialog = page.getByRole('dialog');
    await expect(dialog).toBeVisible({ timeout: 5_000 });
    await dialog.getByRole('button', { name: 'Cancel' }).click();

    await expect(dialog).toBeHidden({ timeout: 5_000 });
    await expect(page.locator('.svelte-flow__node')).toHaveCount(2);
  });
});
