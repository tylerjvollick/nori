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

function center(box: { x: number; y: number; width: number; height: number }) {
  return { x: box.x + box.width / 2, y: box.y + box.height / 2 };
}

/** Let the post-mutation fitView animation settle so measured boxes are stable. */
async function settleGraph(page: Page): Promise<void> {
  await page.waitForTimeout(400);
}

/**
 * Drag a reconnect anchor onto a target handle until `assertion` holds. Canvas
 * drags are inherently imprecise (a few pixels off the small handle and xyflow
 * registers no connection), so we re-measure and re-drag a few times. A missed
 * drag leaves the graph unchanged, so retrying is safe and idempotent.
 */
async function reconnectUntil(
  page: Page,
  getFrom: () => Promise<{ x: number; y: number }>,
  getTo: () => Promise<{ x: number; y: number }>,
  check: () => Promise<boolean>,
): Promise<void> {
  await expect
    .poll(
      async () => {
        await dragPoint(page, await getFrom(), await getTo());
        // Give the reconnect handler time to persist + refresh before checking.
        await page.waitForTimeout(600);
        return check();
      },
      { timeout: 20_000, intervals: [400, 800, 1200] },
    )
    .toBe(true);
}

// The node's connect handle sits dead-center on top of the larger reconnect
// anchor, so a pointer-down at the anchor's center hits the handle (starting a
// fresh connection) instead of the anchor. Grab the anchor off-center, in its
// outer ring, on the side away from the node body: target anchors live on a
// node's left edge (grab further left), source anchors on the right edge (grab
// further right). The offset is a fraction of the anchor box so it clears the
// inner handle yet stays inside the anchor at any zoom (the anchor renders
// smaller when the graph is zoomed out).
function anchorGrabPoint(
  box: { x: number; y: number; width: number; height: number },
  type: 'source' | 'target',
): { x: number; y: number } {
  const c = center(box);
  const offset = box.width * 0.32;
  return { x: c.x + (type === 'source' ? offset : -offset), y: c.y };
}

/**
 * Reconnect anchors are portaled into a shared edge-labels layer, so they can't
 * be scoped by edge id in the DOM. With multiple edges present, pick the anchor
 * of the given type whose screen position is nearest the given handle — that is
 * the endpoint sitting on that handle — and return its off-center grab point.
 */
async function anchorNearHandle(
  page: Page,
  type: 'source' | 'target',
  handle: ReturnType<Page['locator']>,
): Promise<{ x: number; y: number }> {
  const handleBox = await handle.boundingBox();
  if (!handleBox) throw new Error('missing handle bounding box');
  const hc = center(handleBox);
  const anchors = page.locator(`.svelte-flow__edgeupdater-${type}`);
  const count = await anchors.count();
  let best: { x: number; y: number } | null = null;
  let bestDist = Infinity;
  for (let i = 0; i < count; i++) {
    const box = await anchors.nth(i).boundingBox();
    if (!box) continue;
    const c = center(box);
    const dist = (c.x - hc.x) ** 2 + (c.y - hc.y) ** 2;
    if (dist < bestDist) {
      bestDist = dist;
      best = anchorGrabPoint(box, type);
    }
  }
  if (!best) throw new Error(`no ${type} reconnect anchor found`);
  return best;
}

// nori-c98.4: dependency edges can be re-pointed by dragging an endpoint anchor
// to a different node. Only the TaskDep moves — the task nodes and their data are
// preserved. Cycles are rejected.
test.describe('Graph view: drag-to-reconnect dependency edges (nori-c98.4)', () => {
  test.beforeEach(async ({ page }) => {
    const result = await resetSpace(page);
    spaceSlug = result.slug;
  });

  test('dragging an edge endpoint re-points the dependency and preserves task data', async ({
    page,
  }) => {
    await createJobAndOpenGraph(page, 'Reconnect Job');

    // Build Alpha → Bravo, plus an unconnected Charlie.
    await addNamedNode(page, 'Alpha');
    await expect(page.locator('.svelte-flow__node')).toHaveCount(1, { timeout: 10_000 });

    await page.keyboard.press('j'); // select Alpha (the only node)
    await expect(page.locator('.svelte-flow__node.selected')).toHaveCount(1, { timeout: 5_000 });
    await page.keyboard.press('Alt+s'); // create Bravo, edge Alpha → Bravo
    const editInput = page.locator('.svelte-flow__node input');
    await expect(editInput).toBeVisible({ timeout: 10_000 });
    await editInput.fill('Bravo');
    await page.keyboard.press('Enter');
    await expect(page.locator('.svelte-flow__node')).toHaveCount(2, { timeout: 10_000 });
    await expect(page.locator('.svelte-flow__edge')).toHaveCount(1, { timeout: 10_000 });

    await addNamedNode(page, 'Charlie');
    await expect(page.locator('.svelte-flow__node')).toHaveCount(3, { timeout: 10_000 });
    await expect(page.locator('.svelte-flow__edge')).toHaveCount(1, { timeout: 10_000 });
    await settleGraph(page);

    const alphaId = await nodeIdByTitle(page, 'Alpha');
    const bravoId = await nodeIdByTitle(page, 'Bravo');
    const charlieId = await nodeIdByTitle(page, 'Charlie');

    // The edge encodes blocker(source=Alpha) → dependent(target=Bravo).
    await expect(
      page.locator(`.svelte-flow__edge[data-id="${alphaId}->${bravoId}"]`),
    ).toHaveCount(1);

    // Drag the edge's target endpoint anchor from Bravo onto Charlie's target
    // handle. Only one edge exists, so the lone target anchor is this edge's.
    const charlieTargetHandle = page.locator(
      `.svelte-flow__handle.target[data-nodeid="${charlieId}"]`,
    );
    const newEdge = page.locator(`.svelte-flow__edge[data-id="${alphaId}->${charlieId}"]`);
    await reconnectUntil(
      page,
      async () => {
        const box = await page.locator('.svelte-flow__edgeupdater-target').first().boundingBox();
        if (!box) throw new Error('missing target anchor');
        return anchorGrabPoint(box, 'target');
      },
      async () => {
        const box = await charlieTargetHandle.boundingBox();
        if (!box) throw new Error('missing Charlie target handle');
        return center(box);
      },
      async () => (await newEdge.count()) === 1,
    );

    // The dependency now points Alpha → Charlie; the old Alpha → Bravo edge is gone.
    await expect(newEdge).toHaveCount(1, { timeout: 10_000 });
    await expect(page.locator(`.svelte-flow__edge[data-id="${alphaId}->${bravoId}"]`)).toHaveCount(
      0,
      { timeout: 10_000 },
    );
    await expect(page.locator('.svelte-flow__edge')).toHaveCount(1);

    // Task nodes and their data are preserved — only the dependency moved.
    await expect(page.locator('.svelte-flow__node')).toHaveCount(3);
    for (const title of ['Alpha', 'Bravo', 'Charlie']) {
      await expect(page.locator('.svelte-flow__node', { hasText: title })).toHaveCount(1);
    }

    // Persisted: a reload shows the re-pointed dependency from the server.
    await page.reload();
    await page.getByRole('tab', { name: 'Graph' }).click();
    await expect(
      page.locator(`.svelte-flow__edge[data-id="${alphaId}->${charlieId}"]`),
    ).toHaveCount(1, { timeout: 10_000 });
    await expect(page.locator('.svelte-flow__edge')).toHaveCount(1, { timeout: 10_000 });
  });

  test('reconnecting an edge into a cycle is rejected and leaves the graph unchanged', async ({
    page,
  }) => {
    await createJobAndOpenGraph(page, 'Cycle Reject Job');

    // Build Alpha → Bravo → Charlie.
    await addNamedNode(page, 'Alpha');
    await expect(page.locator('.svelte-flow__node')).toHaveCount(1, { timeout: 10_000 });
    await page.keyboard.press('j'); // select Alpha
    await expect(page.locator('.svelte-flow__node.selected')).toHaveCount(1, { timeout: 5_000 });
    await page.keyboard.press('Alt+s'); // Alpha → Bravo
    let editInput = page.locator('.svelte-flow__node input');
    await expect(editInput).toBeVisible({ timeout: 10_000 });
    await editInput.fill('Bravo');
    await page.keyboard.press('Enter');
    await expect(page.locator('.svelte-flow__edge')).toHaveCount(1, { timeout: 10_000 });

    // Select Bravo (downstream) and add Charlie serially → Bravo → Charlie.
    await page.keyboard.press('j');
    await page.keyboard.press('j');
    await expect(page.locator('.svelte-flow__node.selected')).toContainText('Bravo', {
      timeout: 5_000,
    });
    await page.keyboard.press('Alt+s'); // Bravo → Charlie
    editInput = page.locator('.svelte-flow__node input');
    await expect(editInput).toBeVisible({ timeout: 10_000 });
    await editInput.fill('Charlie');
    await page.keyboard.press('Enter');
    await expect(page.locator('.svelte-flow__node')).toHaveCount(3, { timeout: 10_000 });
    await expect(page.locator('.svelte-flow__edge')).toHaveCount(2, { timeout: 10_000 });
    await settleGraph(page);

    const alphaId = await nodeIdByTitle(page, 'Alpha');
    const bravoId = await nodeIdByTitle(page, 'Bravo');
    const charlieId = await nodeIdByTitle(page, 'Charlie');

    // Re-point edge Alpha → Bravo's SOURCE endpoint from Alpha onto Charlie:
    // the new edge would be Charlie → Bravo, and since Bravo → Charlie already
    // exists, that closes a cycle. Drag the source anchor sitting on Alpha's
    // source handle to Charlie's source handle.
    const alphaSourceHandle = page.locator(
      `.svelte-flow__handle.source[data-nodeid="${alphaId}"]`,
    );
    const charlieSourceHandle = page.locator(
      `.svelte-flow__handle.source[data-nodeid="${charlieId}"]`,
    );
    const cycleToast = page.getByText(/would create a cycle/i);
    await reconnectUntil(
      page,
      async () => anchorNearHandle(page, 'source', alphaSourceHandle),
      async () => {
        const box = await charlieSourceHandle.boundingBox();
        if (!box) throw new Error('missing Charlie source handle');
        return center(box);
      },
      async () => (await cycleToast.count()) > 0,
    );

    // Rejected with feedback; the original two edges are intact and unchanged.
    await expect(cycleToast.first()).toBeVisible({ timeout: 5_000 });
    await expect(
      page.locator(`.svelte-flow__edge[data-id="${alphaId}->${bravoId}"]`),
    ).toHaveCount(1);
    await expect(
      page.locator(`.svelte-flow__edge[data-id="${bravoId}->${charlieId}"]`),
    ).toHaveCount(1);
    await expect(page.locator('.svelte-flow__edge')).toHaveCount(2);
  });
});
