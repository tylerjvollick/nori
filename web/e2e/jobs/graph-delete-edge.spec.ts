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

/** Let the post-mutation fitView animation settle so measured boxes are stable. */
async function settleGraph(page: Page): Promise<void> {
  await page.waitForTimeout(400);
}

/**
 * Build a two-node Alpha → Bravo dependency on a fresh job graph and return the
 * node ids. Alpha is added first; selecting it and pressing Alt+S creates Bravo
 * downstream, wired by a single edge.
 */
async function buildAlphaBravoEdge(
  page: Page,
  jobTitle: string,
): Promise<{ alphaId: string; bravoId: string; edgeId: string }> {
  await createJobAndOpenGraph(page, jobTitle);

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
  await settleGraph(page);

  const alphaId = await nodeIdByTitle(page, 'Alpha');
  const bravoId = await nodeIdByTitle(page, 'Bravo');
  const edgeId = `${alphaId}->${bravoId}`;
  await expect(page.locator(`.svelte-flow__edge[data-id="${edgeId}"]`)).toHaveCount(1);
  return { alphaId, bravoId, edgeId };
}

/**
 * Click a dependency edge until it is selected. Clicking a thin edge path on the
 * canvas is imprecise, so re-measure and re-click a few times; a missed click is
 * idempotent (it just doesn't select anything).
 */
async function selectEdge(page: Page, edgeId: string): Promise<void> {
  const edge = page.locator(`.svelte-flow__edge[data-id="${edgeId}"]`);
  await expect
    .poll(
      async () => {
        const box = await edge.boundingBox();
        if (!box) return false;
        await page.mouse.click(box.x + box.width / 2, box.y + box.height / 2);
        await page.waitForTimeout(150);
        return (
          (await page.locator(`.svelte-flow__edge[data-id="${edgeId}"].selected`).count()) === 1
        );
      },
      { timeout: 15_000, intervals: [300, 600, 900] },
    )
    .toBe(true);
}

// nori-1rz.2: a dependency arrow can be removed by clicking it to select/highlight
// it, pressing Delete, and confirming in a dialog. The tasks are kept — only the
// TaskDep edge is removed.
test.describe('Graph view: delete a dependency edge (nori-1rz.2)', () => {
  test.beforeEach(async ({ page }) => {
    const result = await resetSpace(page);
    spaceSlug = result.slug;
  });

  test('click edge, Delete, confirm — the dependency is removed and stays removed', async ({
    page,
  }) => {
    const { edgeId } = await buildAlphaBravoEdge(page, 'Delete Edge Job');

    // Select the edge — it highlights (.selected class).
    await selectEdge(page, edgeId);
    await expect(page.locator(`.svelte-flow__edge[data-id="${edgeId}"].selected`)).toHaveCount(1);

    // Delete opens a confirm dialog — nothing removed yet.
    await page.keyboard.press('Delete');
    const dialog = page.getByRole('dialog');
    await expect(dialog).toBeVisible({ timeout: 5_000 });
    await expect(dialog).toContainText('Remove dependency?');
    await expect(page.locator('.svelte-flow__edge')).toHaveCount(1);

    // Confirm — the edge is removed without a reload; both task nodes remain.
    await dialog.getByRole('button', { name: 'Remove', exact: true }).click();
    await expect(page.locator(`.svelte-flow__edge[data-id="${edgeId}"]`)).toHaveCount(0, {
      timeout: 10_000,
    });
    await expect(page.locator('.svelte-flow__edge')).toHaveCount(0, { timeout: 10_000 });
    await expect(page.locator('.svelte-flow__node')).toHaveCount(2);

    // Persisted: a reload still shows no dependency edge.
    await page.reload();
    await page.getByRole('tab', { name: 'Graph' }).click();
    await expect(page.locator('.svelte-flow__node')).toHaveCount(2, { timeout: 10_000 });
    await expect(page.locator('.svelte-flow__edge')).toHaveCount(0, { timeout: 10_000 });
  });

  test('click edge, Delete, Cancel — the dependency remains', async ({ page }) => {
    const { edgeId } = await buildAlphaBravoEdge(page, 'Cancel Edge Job');

    await selectEdge(page, edgeId);

    await page.keyboard.press('Delete');
    const dialog = page.getByRole('dialog');
    await expect(dialog).toBeVisible({ timeout: 5_000 });
    await expect(dialog).toContainText('Remove dependency?');

    await dialog.getByRole('button', { name: 'Cancel' }).click();
    await expect(dialog).toBeHidden({ timeout: 5_000 });

    // The edge is untouched.
    await expect(page.locator(`.svelte-flow__edge[data-id="${edgeId}"]`)).toHaveCount(1);
    await expect(page.locator('.svelte-flow__edge')).toHaveCount(1);
  });
});
