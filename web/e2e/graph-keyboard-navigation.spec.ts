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

test.describe('Graph keyboard navigation', () => {
  test.beforeEach(async ({ page }) => {
    const result = await resetSpace(page);
    spaceSlug = result.slug;
  });

  // nori-zv0.5: hjkl selects a node, Enter navigates to it.
  test('hjkl selects a node and Enter navigates to that task detail page', async ({ page }) => {
    await createJobWithChildren(page, 'Enter Nav Job', ['Alpha', 'Beta']);
    await expect(page.locator('.svelte-flow__node')).toHaveCount(2, { timeout: 10_000 });

    // j selects a node.
    await page.keyboard.press('j');
    const selected = page.locator('.svelte-flow__node.selected');
    await expect(selected).toHaveCount(1, { timeout: 5_000 });
    const firstId = await selected.getAttribute('data-id');

    // j again moves the selection to the sibling node.
    await page.keyboard.press('j');
    await expect(selected).toHaveCount(1);
    const secondId = await selected.getAttribute('data-id');
    expect(secondId).toBeTruthy();
    expect(secondId).not.toBe(firstId);

    // Enter navigates to the currently selected node's detail page.
    await page.keyboard.press('Enter');
    await page.waitForURL((url) => url.pathname === `/spaces/${spaceSlug}/${secondId}`, {
      timeout: 10_000,
    });
    // That task is now the focused center node (its own detail page renders).
    await expect(page.getByRole('heading', { level: 2 }).first()).toBeVisible({ timeout: 10_000 });
  });

  // nori-zv0.6: 's' switches to List; 'l' is freed for graph move-right.
  test("'l' does not switch views; 's' switches to the List view", async ({ page }) => {
    await createJobWithChildren(page, 'Rebind Job', ['One', 'Two']);
    await expect(page.locator('.svelte-flow__node')).toHaveCount(2, { timeout: 10_000 });

    // Graph view has no ?view param.
    expect(new URL(page.url()).searchParams.get('view')).toBeNull();

    // 'l' must NOT switch to the list view (reserved for graph move-right).
    await page.keyboard.press('l');
    await page.waitForTimeout(500); // allow any (incorrect) navigation to happen
    expect(new URL(page.url()).searchParams.get('view')).toBeNull();
    await expect(page.locator('.svelte-flow')).toBeVisible();
    await expect(page.locator('tr[role="link"]')).toHaveCount(0);

    // 's' switches to the List view.
    await page.keyboard.press('s');
    await page.waitForURL((url) => url.searchParams.get('view') === 'list', { timeout: 10_000 });
    await expect(page.locator('tr[role="link"]')).toHaveCount(2, { timeout: 10_000 });
  });

  // nori-zv0.6: keyboard help overlay reflects the rebind.
  test('keyboard help overlay shows s = Switch to list view', async ({ page }) => {
    await createJobWithChildren(page, 'Help Job', ['One']);

    await page.keyboard.press('?');
    const dialog = page.getByRole('dialog', { name: 'Keyboard Shortcuts' });
    await expect(dialog).toBeVisible({ timeout: 5_000 });

    const row = dialog.locator('div.flex').filter({ hasText: 'Switch to list view' });
    await expect(row.locator('kbd')).toHaveText('s');
  });
});
