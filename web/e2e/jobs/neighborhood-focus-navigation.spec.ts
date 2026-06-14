import { test, expect, type Page } from '@playwright/test';
import { resetSpace } from '../helpers/reset';

let spaceSlug = '';

/** Create a job and open its full-width Graph tab. */
async function createJobAndOpenGraph(page: Page, title: string): Promise<void> {
  await page.goto(`/spaces/${spaceSlug}/jobs`);
  await page.waitForLoadState('networkidle');
  await page.getByRole('button', { name: 'New Job' }).click();
  await page.locator('#job-title').fill(title);
  await page.getByRole('button', { name: 'Create Job' }).click();
  await page.waitForURL(new RegExp(`/spaces/${spaceSlug}/.+`));
  await page.getByRole('tab', { name: 'Graph' }).click();
  await expect(page.getByRole('button', { name: 'Add Node', exact: true })).toBeVisible({
    timeout: 10_000,
  });
}

/** Add an unconnected node via the toolbar and commit it with the given title. */
async function addNamedNode(page: Page, title: string): Promise<void> {
  await page.getByRole('button', { name: 'Add Node', exact: true }).click();
  const input = page.locator('.svelte-flow__node input');
  await expect(input).toBeVisible({ timeout: 10_000 });
  await input.fill(title);
  await page.keyboard.press('Enter');
}

test.describe('Leaf-task neighborhood: current task is focused for keyboard nav (nori-tu8)', () => {
  test.beforeEach(async ({ page }) => {
    const result = await resetSpace(page);
    spaceSlug = result.slug;
  });

  test("current task starts selected and 'l' moves downstream (not to the left node)", async ({
    page,
  }) => {
    await createJobAndOpenGraph(page, 'Neighborhood Focus Job');

    // Build Focus → Down so the focus task has a downstream (right) neighbor.
    await addNamedNode(page, 'Focus');
    await expect(page.locator('.svelte-flow__node')).toHaveCount(1, { timeout: 10_000 });

    await page.keyboard.press('j'); // select the only node (Focus)
    await expect(page.locator('.svelte-flow__node.selected')).toHaveCount(1, { timeout: 5_000 });
    await page.keyboard.press('Alt+s'); // create Down, edge Focus → Down
    const inlineInput = page.locator('.svelte-flow__node input');
    await expect(inlineInput).toBeVisible({ timeout: 10_000 });
    await inlineInput.fill('Down');
    await page.keyboard.press('Enter');

    await expect(page.locator('.svelte-flow__node')).toHaveCount(2, { timeout: 10_000 });
    await expect(page.locator('.svelte-flow__edge')).toHaveCount(1, { timeout: 10_000 });

    // Capture the task ids of both nodes.
    const focusId = await page
      .locator('.svelte-flow__node', { hasText: 'Focus' })
      .getAttribute('data-id');
    const downId = await page
      .locator('.svelte-flow__node', { hasText: 'Down' })
      .getAttribute('data-id');
    expect(focusId).toBeTruthy();
    expect(downId).toBeTruthy();
    expect(downId).not.toBe(focusId);

    // Navigate to the Focus task's own detail page — this renders the
    // neighborhood graph with Focus as the in-focus center node.
    await page.goto(`/spaces/${spaceSlug}/${focusId}`);
    await expect(page.getByRole('heading', { level: 2, name: 'Focus' })).toBeVisible({
      timeout: 10_000,
    });
    // Neighborhood = Focus + its downstream Down.
    await expect(page.locator('.svelte-flow__node')).toHaveCount(2, { timeout: 10_000 });

    // The current task (Focus) should be keyboard-selected on load.
    const selected = page.locator('.svelte-flow__node.selected');
    await expect(selected).toHaveCount(1, { timeout: 5_000 });
    await expect(selected).toHaveAttribute('data-id', focusId!);

    // 'l' moves the selection DOWNSTREAM (right) to Down — not to the leftmost
    // node, which was the previous (buggy) fallback behavior.
    await page.keyboard.press('l');
    await expect(selected).toHaveCount(1);
    await expect(selected).toHaveAttribute('data-id', downId!, { timeout: 5_000 });
  });
});
