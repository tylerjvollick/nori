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

// nori-c98.2: adding a serial/parallel node via the keyboard used to leave the
// new node detached (no arrows) until a full page refresh, because the parent's
// deps reload early-returned on the cached `depsLoaded` flag. These tests assert
// the connecting edges appear immediately, with no reload.
test.describe('Graph view: keyboard-added nodes connect immediately (nori-c98.2)', () => {
  test.beforeEach(async ({ page }) => {
    const result = await resetSpace(page);
    spaceSlug = result.slug;
  });

  test('Alt+S adds a serial node connected by an edge without a reload', async ({ page }) => {
    await createJobAndOpenGraph(page, 'Serial Edge Job');

    await addNamedNode(page, 'Step 1');
    await expect(page.locator('.svelte-flow__node')).toHaveCount(1, { timeout: 10_000 });
    await expect(page.locator('.svelte-flow__edge')).toHaveCount(0);

    // Select the node (j selects the only node) and add a serial node downstream.
    await page.keyboard.press('j');
    await expect(page.locator('.svelte-flow__node.selected')).toHaveCount(1, { timeout: 5_000 });
    await page.keyboard.press('Alt+s');

    // The new node and its connecting edge appear WITHOUT a page reload.
    await expect(page.locator('.svelte-flow__node')).toHaveCount(2, { timeout: 10_000 });
    await expect(page.locator('.svelte-flow__edge')).toHaveCount(1, { timeout: 10_000 });

    // Commit the new node's default title to settle inline editing.
    await page.keyboard.press('Enter');
  });

  test('Alt+P adds a parallel node sharing the upstream edge without a reload', async ({ page }) => {
    await createJobAndOpenGraph(page, 'Parallel Edge Job');

    // Build Up → Down so Down has an upstream blocker.
    await addNamedNode(page, 'Up');
    await expect(page.locator('.svelte-flow__node')).toHaveCount(1, { timeout: 10_000 });

    await page.keyboard.press('j'); // select the only node (Up)
    await expect(page.locator('.svelte-flow__node.selected')).toHaveCount(1, { timeout: 5_000 });
    await page.keyboard.press('Alt+s'); // create Down, edge Up → Down
    const input = page.locator('.svelte-flow__node input');
    await expect(input).toBeVisible({ timeout: 10_000 });
    await input.fill('Down');
    await page.keyboard.press('Enter');
    await expect(page.locator('.svelte-flow__node')).toHaveCount(2, { timeout: 10_000 });
    await expect(page.locator('.svelte-flow__edge')).toHaveCount(1, { timeout: 10_000 });

    // Select Down (downstream of Up): selection reset after the rebuild, so
    // j selects Up (leftmost), then j again moves downstream to Down.
    await page.keyboard.press('j');
    await page.keyboard.press('j');
    await expect(page.locator('.svelte-flow__node.selected')).toContainText('Down', {
      timeout: 5_000,
    });

    // Parallel node of Down inherits Up as its upstream → a second Up→X edge.
    await page.keyboard.press('Alt+p');
    await expect(page.locator('.svelte-flow__node')).toHaveCount(3, { timeout: 10_000 });
    await expect(page.locator('.svelte-flow__edge')).toHaveCount(2, { timeout: 10_000 });
  });
});
