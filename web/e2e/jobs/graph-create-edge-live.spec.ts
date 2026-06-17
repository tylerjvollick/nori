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

/** Read the task UUID (xyflow node data-id) of the node containing the given title. */
async function nodeIdByTitle(page: Page, title: string): Promise<string> {
  const node = page.locator('.svelte-flow__node', { hasText: title }).first();
  await expect(node).toBeVisible({ timeout: 10_000 });
  const id = await node.getAttribute('data-id');
  if (!id) throw new Error(`node "${title}" has no data-id`);
  return id;
}

function center(box: { x: number; y: number; width: number; height: number }) {
  return { x: box.x + box.width / 2, y: box.y + box.height / 2 };
}

/** Drag from a source point to a target point using manual pointer moves. */
async function dragPoint(
  page: Page,
  from: { x: number; y: number },
  to: { x: number; y: number },
): Promise<void> {
  await page.mouse.move(from.x, from.y);
  await page.mouse.down();
  // A short intermediate move clears xyflow's drag threshold before the drop.
  await page.mouse.move((from.x + to.x) / 2, (from.y + to.y) / 2, { steps: 6 });
  await page.mouse.move(to.x, to.y, { steps: 10 });
  // Settle over the target handle so xyflow registers it as the drop handle.
  await page.mouse.move(to.x, to.y);
  await page.waitForTimeout(120);
  await page.mouse.up();
}

/** Let the post-mutation fitView animation settle so measured boxes are stable. */
async function settleGraph(page: Page): Promise<void> {
  await page.waitForTimeout(400);
}

/**
 * Drag from a source handle to a target handle until `check` holds. Canvas drags
 * are imprecise (a few pixels off the small handle and xyflow registers no
 * connection), so we re-measure and re-drag a few times. A missed drag leaves the
 * graph unchanged, so retrying is safe and idempotent.
 */
async function connectUntil(
  page: Page,
  getFrom: () => Promise<{ x: number; y: number }>,
  getTo: () => Promise<{ x: number; y: number }>,
  check: () => Promise<boolean>,
): Promise<void> {
  await expect
    .poll(
      async () => {
        await dragPoint(page, await getFrom(), await getTo());
        // Give the connect handler time to persist + refresh before checking.
        await page.waitForTimeout(600);
        return check();
      },
      { timeout: 20_000, intervals: [400, 800, 1200] },
    )
    .toBe(true);
}

// nori-1rz.1: drag-creating a dependency edge between two nodes used to leave the
// new arrow invisible until a manual reload, because handleConnect rebuilt the
// scoped graph from the stale externalDeps prop instead of re-fetching via the
// parent's onmutate(). This test asserts the new edge appears WITHOUT a reload.
test.describe('Graph view: drag-created dependency edge appears live (nori-1rz.1)', () => {
  test.beforeEach(async ({ page }) => {
    const result = await resetSpace(page);
    spaceSlug = result.slug;
  });

  test('dragging from a source handle to a target handle adds the edge without a reload', async ({
    page,
  }) => {
    await createJobAndOpenGraph(page, 'Live Edge Job');

    // Two unconnected nodes.
    await addNamedNode(page, 'Alpha');
    await expect(page.locator('.svelte-flow__node')).toHaveCount(1, { timeout: 10_000 });
    await addNamedNode(page, 'Bravo');
    await expect(page.locator('.svelte-flow__node')).toHaveCount(2, { timeout: 10_000 });
    await expect(page.locator('.svelte-flow__edge')).toHaveCount(0);
    await settleGraph(page);

    const alphaId = await nodeIdByTitle(page, 'Alpha');
    const bravoId = await nodeIdByTitle(page, 'Bravo');

    // Drag Alpha's source handle onto Bravo's target handle → Alpha blocks Bravo.
    const alphaSource = page.locator(`.svelte-flow__handle.source[data-nodeid="${alphaId}"]`);
    const bravoTarget = page.locator(`.svelte-flow__handle.target[data-nodeid="${bravoId}"]`);
    const newEdge = page.locator(`.svelte-flow__edge[data-id="${alphaId}->${bravoId}"]`);

    await connectUntil(
      page,
      async () => {
        const box = await alphaSource.boundingBox();
        if (!box) throw new Error('missing Alpha source handle');
        return center(box);
      },
      async () => {
        const box = await bravoTarget.boundingBox();
        if (!box) throw new Error('missing Bravo target handle');
        return center(box);
      },
      async () => (await newEdge.count()) === 1,
    );

    // The new dependency edge is present immediately — no reload needed.
    await expect(newEdge).toHaveCount(1, { timeout: 10_000 });
    await expect(page.locator('.svelte-flow__edge')).toHaveCount(1);

    // Persisted: a reload shows the same edge from the server.
    await page.reload();
    await page.getByRole('tab', { name: 'Graph' }).click();
    await expect(
      page.locator(`.svelte-flow__edge[data-id="${alphaId}->${bravoId}"]`),
    ).toHaveCount(1, { timeout: 10_000 });
    await expect(page.locator('.svelte-flow__edge')).toHaveCount(1, { timeout: 10_000 });
  });
});
