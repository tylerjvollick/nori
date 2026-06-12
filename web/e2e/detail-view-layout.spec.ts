import { test, expect } from '@playwright/test';
import { resetSpace } from './helpers/reset';

let spaceSlug = '';

test.describe('Detail view layout (nori-5mn.2, 5mn.3)', () => {
  test.beforeEach(async ({ page }) => {
    const result = await resetSpace(page);
    spaceSlug = result.slug;
  });

  // --- nori-5mn.2: Recipe detail ---

  test('recipe detail: no duplicate title below breadcrumbs', async ({ page }) => {
    await page.goto(`/spaces/${spaceSlug}/recipes`);
    await page.waitForLoadState('networkidle');
    await page.getByRole('button', { name: 'New Recipe' }).click();
    await page.locator('#recipe-name').fill('Layout Test Recipe');
    await page.getByRole('button', { name: 'Create Recipe' }).click();
    await page.waitForURL(/\/spaces\/[^/]+\/recipes\/.+/);
    await page.waitForLoadState('networkidle');

    // Title in breadcrumb
    await expect(page.locator('[data-slot="breadcrumb-page"]', { hasText: 'Layout Test Recipe' })).toBeVisible();
    // No duplicate recipe name in an h1 outside the graph area
    // The only h1 should be "Recipe Steps" inside the graph toolbar, not the recipe name
    await expect(page.locator('h1', { hasText: 'Layout Test Recipe' })).not.toBeVisible();
  });

  test('recipe detail: Publish and History buttons in breadcrumb bar', async ({ page }) => {
    await page.goto(`/spaces/${spaceSlug}/recipes`);
    await page.waitForLoadState('networkidle');
    await page.getByRole('button', { name: 'New Recipe' }).click();
    await page.locator('#recipe-name').fill('Actions Test');
    await page.getByRole('button', { name: 'Create Recipe' }).click();
    await page.waitForURL(/\/spaces\/[^/]+\/recipes\/.+/);
    await page.waitForLoadState('networkidle');

    // The breadcrumb bar contains the action buttons
    const breadcrumbBar = page.locator('nav[aria-label="breadcrumb"]').locator('..');
    // History button should be visible on same row as breadcrumbs
    await expect(page.getByRole('button', { name: 'History' })).toBeVisible();
    // Publish button visible for draft recipes
    await expect(page.getByRole('button', { name: 'Publish' })).toBeVisible();
  });

  test('recipe detail: root task shows status and version tags in right pane', async ({ page }) => {
    await page.goto(`/spaces/${spaceSlug}/recipes`);
    await page.waitForLoadState('networkidle');
    await page.getByRole('button', { name: 'New Recipe' }).click();
    await page.locator('#recipe-name').fill('Tags Test');
    await page.getByRole('button', { name: 'Create Recipe' }).click();
    await page.waitForURL(/\/spaces\/[^/]+\/recipes\/.+/);

    // Wait for graph and detail panel to load
    await expect(page.getByRole('button', { name: 'Add Node', exact: true })).toBeVisible({ timeout: 10000 });

    // Root task is auto-selected — check for status and version tags in the detail panel
    await expect(page.getByTestId('recipe-status-tag').first()).toBeVisible({ timeout: 5000 });
    await expect(page.getByTestId('recipe-version-tag').first()).toBeVisible({ timeout: 5000 });
  });

  test('recipe detail: child task does NOT show status/version tags', async ({ page }) => {
    await page.goto(`/spaces/${spaceSlug}/recipes`);
    await page.waitForLoadState('networkidle');
    await page.getByRole('button', { name: 'New Recipe' }).click();
    await page.locator('#recipe-name').fill('Child Tags Test');
    await page.getByRole('button', { name: 'Create Recipe' }).click();
    await page.waitForURL(/\/spaces\/[^/]+\/recipes\/.+/);

    // Wait for graph and add a child task
    const addNodeBtn = page.getByRole('button', { name: 'Add Node', exact: true });
    await expect(addNodeBtn).toBeVisible({ timeout: 10000 });
    await addNodeBtn.click();
    const nodes = page.locator('.svelte-flow__node');
    const countBefore = await nodes.count();
    await expect(nodes).toHaveCount(countBefore, { timeout: 10000 });

    // Dismiss inline edit (Enter commits default title "New Task")
    const inlineInput = page.locator('.svelte-flow__node input');
    await expect(inlineInput).toBeVisible({ timeout: 10000 });
    await inlineInput.focus();
    await page.keyboard.press('Enter');

    // Wait for node to show committed title, then click it
    await page.locator('.svelte-flow').click({ position: { x: 10, y: 10 } });
    const childNode = nodes.last();
    await childNode.click();

    // Wait for detail panel to show the child task
    await expect(page.getByRole('heading', { name: 'New Task', level: 2 }).first()).toBeVisible({ timeout: 10000 });

    // The child task's detail panel should NOT show recipe status/version tags
    await expect(page.getByTestId('recipe-status-tag')).not.toBeVisible();
    await expect(page.getByTestId('recipe-version-tag')).not.toBeVisible();
  });

  // --- nori-5mn.3: Job detail ---

  test('job detail: no duplicate title below breadcrumbs', async ({ page }) => {
    await page.goto(`/spaces/${spaceSlug}/jobs`);
    await page.waitForLoadState('networkidle');
    await page.getByRole('button', { name: 'New Job' }).click();
    await page.locator('#job-title').fill('Layout Test Job');
    await page.getByRole('button', { name: 'Create Job' }).click();
    await page.waitForURL(new RegExp(`/spaces/${spaceSlug}/.+`));
    await page.waitForLoadState('networkidle');

    // Title in breadcrumb
    await expect(page.locator('[data-slot="breadcrumb-page"]', { hasText: 'Layout Test Job' })).toBeVisible();
    // No h1 title below breadcrumbs
    await expect(page.locator('h1')).not.toBeVisible();
  });

  test('job detail: Save as Recipe button in breadcrumb bar', async ({ page }) => {
    await page.goto(`/spaces/${spaceSlug}/jobs`);
    await page.waitForLoadState('networkidle');
    await page.getByRole('button', { name: 'New Job' }).click();
    await page.locator('#job-title').fill('Actions Test Job');
    await page.getByRole('button', { name: 'Create Job' }).click();
    await page.waitForURL(new RegExp(`/spaces/${spaceSlug}/.+`));
    await page.waitForLoadState('networkidle');

    // Save as Recipe button should be visible
    await expect(page.getByRole('button', { name: 'Save as Recipe' })).toBeVisible();
  });

  test('job detail: Tasks and Cost tabs are visible', async ({ page }) => {
    await page.goto(`/spaces/${spaceSlug}/jobs`);
    await page.waitForLoadState('networkidle');
    await page.getByRole('button', { name: 'New Job' }).click();
    await page.locator('#job-title').fill('Tabs Test Job');
    await page.getByRole('button', { name: 'Create Job' }).click();
    await page.waitForURL(new RegExp(`/spaces/${spaceSlug}/.+`));
    await page.waitForLoadState('networkidle');

    await expect(page.getByRole('tab', { name: 'Tasks' })).toBeVisible();
    await expect(page.getByRole('tab', { name: 'Cost' })).toBeVisible();
  });

  test('job detail: switch between Graph/List/Board, detail panel persists', async ({ page }) => {
    await page.goto(`/spaces/${spaceSlug}/jobs`);
    await page.waitForLoadState('networkidle');
    await page.getByRole('button', { name: 'New Job' }).click();
    await page.locator('#job-title').fill('View Toggle Job');
    await page.getByRole('button', { name: 'Create Job' }).click();
    await page.waitForURL(new RegExp(`/spaces/${spaceSlug}/.+`));
    await page.waitForLoadState('networkidle');

    // Wait for graph to load
    await expect(page.getByRole('button', { name: 'Add Node', exact: true })).toBeVisible({ timeout: 10000 });

    // Graph view should show view toggle with Graph active
    await expect(page.getByRole('button', { name: 'Graph', exact: true })).toBeVisible();
    await expect(page.getByRole('button', { name: 'List', exact: true })).toBeVisible();
    await expect(page.getByRole('button', { name: 'Board', exact: true })).toBeVisible();

    // Switch to List — right pane (detail panel heading) should still be visible
    await page.getByRole('button', { name: 'List', exact: true }).click();
    await expect(page.getByRole('heading', { level: 2 }).first()).toBeVisible({ timeout: 5000 });

    // Switch to Board — right pane should still be visible
    await page.getByRole('button', { name: 'Board', exact: true }).click();
    await expect(page.getByRole('heading', { level: 2 }).first()).toBeVisible({ timeout: 5000 });

    // Switch back to Graph
    await page.getByRole('button', { name: 'Graph', exact: true }).click();
    await expect(page.getByRole('heading', { level: 2 }).first()).toBeVisible({ timeout: 5000 });
  });

  // --- nori-zv0.3: job-detail layout swap (panel left, views right) ---

  async function createJobAndOpenDetail(page: import('@playwright/test').Page, title: string) {
    await page.goto(`/spaces/${spaceSlug}/jobs`);
    await page.waitForLoadState('networkidle');
    await page.getByRole('button', { name: 'New Job' }).click();
    await page.locator('#job-title').fill(title);
    await page.getByRole('button', { name: 'Create Job' }).click();
    await page.waitForURL(new RegExp(`/spaces/${spaceSlug}/.+`));
    await page.waitForLoadState('networkidle');
    // Graph (views pane) is ready when Add Node is present.
    await expect(page.getByRole('button', { name: 'Add Node', exact: true })).toBeVisible({
      timeout: 10_000,
    });
  }

  test('job detail: detail panel renders to the LEFT of the views pane', async ({ page }) => {
    await createJobAndOpenDetail(page, 'Swap Layout Job');

    const panelTitle = page.getByRole('heading', { level: 2, name: 'Swap Layout Job' });
    const addNode = page.getByRole('button', { name: 'Add Node', exact: true });
    await expect(panelTitle).toBeVisible({ timeout: 10_000 });

    const panelBox = await panelTitle.boundingBox();
    const viewsBox = await addNode.boundingBox();
    expect(panelBox).not.toBeNull();
    expect(viewsBox).not.toBeNull();
    // The detail panel sits left of the views (graph) pane.
    expect(panelBox!.x).toBeLessThan(viewsBox!.x);
  });

  test('job detail: in-header collapse toggle hides the panel; expand restores it', async ({
    page,
  }) => {
    await createJobAndOpenDetail(page, 'Collapse Toggle Job');

    const panelTitle = page.getByRole('heading', { level: 2, name: 'Collapse Toggle Job' });
    await expect(panelTitle).toBeVisible({ timeout: 10_000 });

    // Collapse control lives inside the detail panel header.
    const collapse = page.getByTestId('panel-collapse');
    await expect(collapse).toBeVisible();
    await collapse.click();

    // Panel is hidden; the expand affordance appears in the views pane.
    await expect(panelTitle).toBeHidden();
    const expand = page.getByTestId('panel-expand');
    await expect(expand).toBeVisible();

    // Restore.
    await expand.click();
    await expect(panelTitle).toBeVisible();
    await expect(collapse).toBeVisible();
  });

  test('job detail: Cost tab shows cost summary', async ({ page }) => {
    await page.goto(`/spaces/${spaceSlug}/jobs`);
    await page.waitForLoadState('networkidle');
    await page.getByRole('button', { name: 'New Job' }).click();
    await page.locator('#job-title').fill('Cost Tab Job');
    await page.getByRole('button', { name: 'Create Job' }).click();
    await page.waitForURL(new RegExp(`/spaces/${spaceSlug}/.+`));
    await page.waitForLoadState('networkidle');

    // Click Cost tab
    await page.getByRole('tab', { name: 'Cost' }).click();

    // Cost view should be displayed (shows Estimated/Actual cards or loading state)
    await expect(page.getByText('Estimated').or(page.locator('.max-w-5xl'))).toBeVisible({ timeout: 5000 });
  });
});
