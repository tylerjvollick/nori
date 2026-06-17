import { test, expect, type Page } from '@playwright/test';
import { resetSpace } from '../helpers/reset';

let spaceSlug = '';

/** Create a job and open its Graph tab; return the job page path (no query). */
async function createJobAndOpenGraph(page: Page, title: string): Promise<string> {
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
  return new URL(page.url()).pathname;
}

/** Add an unconnected node via the toolbar and commit it with the given title. */
async function addNamedNode(page: Page, title: string): Promise<void> {
  await page.getByRole('button', { name: 'Add Node', exact: true }).click();
  const input = page.locator('.svelte-flow__node input');
  await expect(input).toBeVisible({ timeout: 10_000 });
  await input.fill(title);
  await page.keyboard.press('Enter');
}

/** Read the task UUID (xyflow node data-id) of the node containing the given title. */
async function nodeIdByTitle(page: Page, title: string): Promise<string> {
  const node = page.locator('.svelte-flow__node', { hasText: title }).first();
  await expect(node).toBeVisible({ timeout: 10_000 });
  const id = await node.getAttribute('data-id');
  if (!id) throw new Error(`node "${title}" has no data-id`);
  return id;
}

/** Build Alpha → Bravo on a fresh job graph and return ids + the job page path. */
async function buildAlphaBravo(
  page: Page,
  jobTitle: string,
): Promise<{ jobPath: string; alphaId: string; bravoId: string }> {
  const jobPath = await createJobAndOpenGraph(page, jobTitle);

  await addNamedNode(page, 'Alpha');
  await expect(page.locator('.svelte-flow__node')).toHaveCount(1, { timeout: 10_000 });
  await page.keyboard.press('j'); // select Alpha (the only node)
  await expect(page.locator('.svelte-flow__node.selected')).toHaveCount(1, { timeout: 5_000 });
  await page.keyboard.press('Alt+s'); // create Bravo, edge Alpha → Bravo
  const input = page.locator('.svelte-flow__node input');
  await expect(input).toBeVisible({ timeout: 10_000 });
  await input.fill('Bravo');
  await page.keyboard.press('Enter');
  await expect(page.locator('.svelte-flow__node')).toHaveCount(2, { timeout: 10_000 });
  await expect(page.locator('.svelte-flow__edge')).toHaveCount(1, { timeout: 10_000 });

  // Reload so the parent task tree (which the side panel reads from) is fresh
  // from the server — avoids any lag between an inline rename and the tree
  // reload. The panel behavior under test is independent of how the job is built.
  await page.reload();
  await page.getByRole('tab', { name: 'Graph' }).click();
  await expect(page.getByRole('button', { name: 'Add Node', exact: true })).toBeVisible({
    timeout: 10_000,
  });
  await expect(page.locator('.svelte-flow__node')).toHaveCount(2, { timeout: 10_000 });
  await page.waitForTimeout(400); // let the fitView animation settle

  const alphaId = await nodeIdByTitle(page, 'Alpha');
  const bravoId = await nodeIdByTitle(page, 'Bravo');
  return { jobPath, alphaId, bravoId };
}

/** The visible panel title heading (the side panel renders the task title as an h2). */
function panelTitle(page: Page, name: string) {
  return page.locator(`h2:visible`, { hasText: name });
}

// nori-1rz.3: on the Graph tab, clicking a node activates it and loads it into a
// read-only side panel instead of navigating. hjkl moves the active node and the
// panel follows. Enter and the panel's open-in-detail button navigate to the full
// task page.
test.describe('Graph view: node side panel (nori-1rz.3)', () => {
  test.beforeEach(async ({ page }) => {
    const result = await resetSpace(page);
    spaceSlug = result.slug;
  });

  test('clicking a node opens the side panel without navigating; the expand button opens the detail page', async ({
    page,
  }) => {
    const { jobPath, bravoId } = await buildAlphaBravo(page, 'Panel Job');

    // Dismiss any lingering inline edit by clicking empty canvas.
    await page.locator('.svelte-flow').click({ position: { x: 12, y: 12 } });

    // Click Alpha — it activates and loads into the side panel; the URL stays put.
    await page.locator('.svelte-flow__node', { hasText: 'Alpha' }).click();
    await expect(page.locator('.svelte-flow__node.selected')).toHaveCount(1, { timeout: 5_000 });
    await expect(panelTitle(page, 'Alpha')).toBeVisible({ timeout: 5_000 });
    expect(new URL(page.url()).pathname).toBe(jobPath);

    // hjkl navigation follows the panel: 'l' moves downstream (LR layout) to Bravo.
    await page.keyboard.press('l');
    await expect(panelTitle(page, 'Bravo')).toBeVisible({ timeout: 5_000 });
    expect(new URL(page.url()).pathname).toBe(jobPath);

    // The panel's open-in-detail (expand) button opens the active task's full page.
    await page.locator('[data-testid="panel-open-detail"]:visible').click();
    await page.waitForURL((url) => url.pathname === `/spaces/${spaceSlug}/${bravoId}`, {
      timeout: 10_000,
    });
    await expect(page.getByRole('heading', { level: 2, name: 'Bravo' })).toBeVisible({
      timeout: 10_000,
    });
  });

  test('collapsing the panel hides it and the expand control restores it', async ({ page }) => {
    const { jobPath } = await buildAlphaBravo(page, 'Collapse Panel Job');

    await page.locator('.svelte-flow').click({ position: { x: 12, y: 12 } });
    await page.locator('.svelte-flow__node', { hasText: 'Alpha' }).click();
    await expect(panelTitle(page, 'Alpha')).toBeVisible({ timeout: 5_000 });

    // Collapse the panel.
    await page.locator('[data-testid="panel-collapse"]:visible').click();
    await expect(panelTitle(page, 'Alpha')).toBeHidden({ timeout: 5_000 });

    // The graph-side expand control brings it back — still no navigation.
    await page.locator('[data-testid="graph-panel-expand"]:visible').click();
    await expect(panelTitle(page, 'Alpha')).toBeVisible({ timeout: 5_000 });
    expect(new URL(page.url()).pathname).toBe(jobPath);
  });
});
