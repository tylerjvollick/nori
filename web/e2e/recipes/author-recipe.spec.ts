import { test, expect } from '@playwright/test';
import { resetSpace } from '../helpers/reset';

let spaceSlug = '';

test.describe('Recipe Authoring (Graph Editor)', () => {
  test.beforeEach(async ({ page }) => {
    const result = await resetSpace(page);
    spaceSlug = result.slug;
  });

  test('create a new recipe and verify it opens in draft state with graph editor', async ({ page }) => {
    await page.goto(`/spaces/${spaceSlug}/recipes`);
    await page.waitForLoadState('networkidle');
    await expect(page.getByRole('heading', { name: 'Recipes' })).toBeVisible();

    // Open create dialog
    await page.getByRole('button', { name: 'New Recipe' }).click();
    await page.locator('#recipe-name').fill('Oak Dining Table');
    await page.locator('#recipe-description').fill('Standard 6-seat dining table build');
    await page.getByRole('button', { name: 'Create Recipe' }).click();

    // Should navigate to the recipe detail page
    await page.waitForURL(/\/spaces\/[^/]+\/recipes\/.+/);

    // Verify recipe title and draft status
    await expect(page.locator('h1', { hasText: 'Oak Dining Table' })).toBeVisible();
    await expect(page.getByText('Draft')).toBeVisible();

    // Recipe creation automatically creates a root task, so the graph editor loads.
    // Wait for graph toolbar to appear (loading skeleton clears).
    await expect(page.getByRole('button', { name: 'Add Node' })).toBeVisible({ timeout: 10000 });
  });

  test('graph editor toolbar shows controls after loading', async ({ page }) => {
    await page.goto(`/spaces/${spaceSlug}/recipes`);
    await page.waitForLoadState('networkidle');
    await page.getByRole('button', { name: 'New Recipe' }).click();
    await page.locator('#recipe-name').fill('Controls Test');
    await page.getByRole('button', { name: 'Create Recipe' }).click();
    await page.waitForURL(/\/spaces\/[^/]+\/recipes\/.+/);

    // Wait for graph to load
    await expect(page.getByRole('button', { name: 'Add Node' })).toBeVisible({ timeout: 10000 });
    await expect(page.getByRole('button', { name: 'Re-layout' })).toBeVisible();
    await expect(page.getByRole('button', { name: 'Refresh' })).toBeVisible();
  });

  test('back to recipes navigates to the list', async ({ page }) => {
    await page.goto(`/spaces/${spaceSlug}/recipes`);
    await page.waitForLoadState('networkidle');
    await page.getByRole('button', { name: 'New Recipe' }).click();
    await page.locator('#recipe-name').fill('Nav Test Recipe');
    await page.getByRole('button', { name: 'Create Recipe' }).click();
    await page.waitForURL(/\/spaces\/[^/]+\/recipes\/.+/);
    await page.waitForLoadState('networkidle');

    // Click "Back to Recipes"
    await page.getByRole('link', { name: 'Back to Recipes' }).click();
    await page.waitForURL(new RegExp(`/spaces/${spaceSlug}/recipes$`));

    // Verify we're on the recipes list
    await expect(page.getByRole('heading', { name: 'Recipes' })).toBeVisible();
    await expect(page.getByText('Nav Test Recipe')).toBeVisible();
  });

  test('add node creates a visible task in the graph', async ({ page }) => {
    await page.goto(`/spaces/${spaceSlug}/recipes`);
    await page.waitForLoadState('networkidle');
    await page.getByRole('button', { name: 'New Recipe' }).click();
    await page.locator('#recipe-name').fill('Add Node Test');
    await page.getByRole('button', { name: 'Create Recipe' }).click();
    await page.waitForURL(/\/spaces\/[^/]+\/recipes\/.+/);

    // Wait for graph to load
    const addNodeBtn = page.getByRole('button', { name: 'Add Node' });
    await expect(addNodeBtn).toBeVisible({ timeout: 10000 });

    // Count graph nodes before adding
    const nodesBefore = await page.locator('.svelte-flow__node').count();

    // Click Add Node
    await addNodeBtn.click();

    // New node should appear in the graph
    await expect(page.locator('.svelte-flow__node')).toHaveCount(nodesBefore + 1, { timeout: 10000 });

    // The new node should show inline editing input with default title
    const newNode = page.locator('.svelte-flow__node').last();
    const input = newNode.locator('input');
    await expect(input).toBeVisible({ timeout: 5000 });
    await expect(input).toHaveValue('New Task');
  });

  test('recipe graph nodes do not show status icons', async ({ page }) => {
    await page.goto(`/spaces/${spaceSlug}/recipes`);
    await page.waitForLoadState('networkidle');
    await page.getByRole('button', { name: 'New Recipe' }).click();
    await page.locator('#recipe-name').fill('Status Icon Test');
    await page.getByRole('button', { name: 'Create Recipe' }).click();
    await page.waitForURL(/\/spaces\/[^/]+\/recipes\/.+/);

    // Wait for graph to load, add a node so we have a visible node to check
    const addNodeBtn = page.getByRole('button', { name: 'Add Node' });
    await expect(addNodeBtn).toBeVisible({ timeout: 10000 });

    const nodesBefore = await page.locator('.svelte-flow__node').count();
    await addNodeBtn.click();
    await expect(page.locator('.svelte-flow__node')).toHaveCount(nodesBefore + 1, { timeout: 10000 });

    // Press Escape to close inline edit, then click away to deselect
    await page.keyboard.press('Escape');
    await page.locator('.svelte-flow').click({ position: { x: 10, y: 10 } });

    // In recipe mode, status icons should not render.
    // In task/job mode, every node has a [data-testid="status-icon"] SVG.
    const nodes = page.locator('.svelte-flow__node');
    const count = await nodes.count();
    expect(count).toBeGreaterThan(0);
    await expect(page.locator('[data-testid="status-icon"]')).toHaveCount(0);
  });

  test('dependency section shows task title instead of ID', async ({ page }) => {
    await page.goto(`/spaces/${spaceSlug}/recipes`);
    await page.waitForLoadState('networkidle');
    await page.getByRole('button', { name: 'New Recipe' }).click();
    await page.locator('#recipe-name').fill('Dep Title Test');
    await page.getByRole('button', { name: 'Create Recipe' }).click();
    await page.waitForURL(/\/spaces\/[^/]+\/recipes\/.+/);

    // Wait for graph
    const addNodeBtn = page.getByRole('button', { name: 'Add Node' });
    await expect(addNodeBtn).toBeVisible({ timeout: 10000 });

    // Add first node
    const nodes = page.locator('.svelte-flow__node');
    const nodesBefore = await nodes.count();
    await addNodeBtn.click();
    await expect(nodes).toHaveCount(nodesBefore + 1, { timeout: 10000 });

    // Wait for inline editing input, focus it, then press Tab to commit and create serial downstream
    const editInput = page.locator('.svelte-flow__node input');
    await expect(editInput).toBeVisible({ timeout: 10000 });
    await editInput.focus();
    await page.keyboard.press('Tab');
    await expect(nodes).toHaveCount(nodesBefore + 2, { timeout: 10000 });

    // Dismiss editing on new serial node
    await expect(page.locator('.svelte-flow__node input')).toBeVisible({ timeout: 10000 });
    await page.keyboard.press('Escape');
    await page.locator('.svelte-flow').click({ position: { x: 10, y: 10 } });
    const lastNode = page.locator('.svelte-flow__node').last();
    await lastNode.click();

    // The detail panel should show "Blocked by" with a task title, not a UUID
    const blockedByLabel = page.getByText('Blocked by').first();
    await expect(blockedByLabel).toBeVisible({ timeout: 10000 });
    // The blocker badge should show "New Task" (the title), not a UUID
    const depSection = blockedByLabel.locator('..').locator('..');
    await expect(depSection.getByText('New Task')).toBeVisible({ timeout: 5000 });
  });

  test('Tab commits title and creates serial downstream node', async ({ page }) => {
    await page.goto(`/spaces/${spaceSlug}/recipes`);
    await page.waitForLoadState('networkidle');
    await page.getByRole('button', { name: 'New Recipe' }).click();
    await page.locator('#recipe-name').fill('Tab Test');
    await page.getByRole('button', { name: 'Create Recipe' }).click();
    await page.waitForURL(/\/spaces\/[^/]+\/recipes\/.+/);

    const addNodeBtn = page.getByRole('button', { name: 'Add Node' });
    await expect(addNodeBtn).toBeVisible({ timeout: 10000 });

    const nodes = page.locator('.svelte-flow__node');
    const nodesBefore = await nodes.count();

    // Add node — inline input should appear
    await addNodeBtn.click();
    await expect(nodes).toHaveCount(nodesBefore + 1, { timeout: 10000 });

    // Type a name and press Tab — should commit and create serial node
    const inlineInput = page.locator('.svelte-flow__node input');
    await expect(inlineInput).toBeVisible({ timeout: 10000 });
    await inlineInput.fill('Cut Wood');
    await page.keyboard.press('Tab');

    // Should now have 2 new nodes (the named one + serial downstream)
    await expect(nodes).toHaveCount(nodesBefore + 2, { timeout: 10000 });

    // The first node should have the committed title
    await expect(page.locator('.svelte-flow__node', { hasText: 'Cut Wood' })).toBeVisible({ timeout: 5000 });
  });

  test('Ctrl+R renames an existing node', async ({ page }) => {
    await page.goto(`/spaces/${spaceSlug}/recipes`);
    await page.waitForLoadState('networkidle');
    await page.getByRole('button', { name: 'New Recipe' }).click();
    await page.locator('#recipe-name').fill('Rename Test');
    await page.getByRole('button', { name: 'Create Recipe' }).click();
    await page.waitForURL(/\/spaces\/[^/]+\/recipes\/.+/);

    const addNodeBtn = page.getByRole('button', { name: 'Add Node' });
    await expect(addNodeBtn).toBeVisible({ timeout: 10000 });

    const nodes = page.locator('.svelte-flow__node');
    const nodesBefore = await nodes.count();

    // Add a node and dismiss editing
    await addNodeBtn.click();
    await expect(nodes).toHaveCount(nodesBefore + 1, { timeout: 10000 });
    await page.keyboard.press('Escape');
    await page.locator('.svelte-flow').click({ position: { x: 10, y: 10 } });

    // Click the node to select it
    const node = page.locator('.svelte-flow__node').first();
    await node.click();

    // Ctrl+R should open inline rename input
    await page.keyboard.press('Control+r');
    const renameInput = page.locator('.svelte-flow__node input');
    await expect(renameInput).toBeVisible({ timeout: 5000 });

    // Type new name and commit
    await renameInput.fill('Sand Edges');
    await page.keyboard.press('Enter');

    // Node should show the new title
    await expect(page.locator('.svelte-flow__node', { hasText: 'Sand Edges' })).toBeVisible({ timeout: 10000 });
  });

  test('recipe appears in the recipes list after creation', async ({ page }) => {
    await page.goto(`/spaces/${spaceSlug}/recipes`);
    await page.waitForLoadState('networkidle');
    await page.getByRole('button', { name: 'New Recipe' }).click();
    await page.locator('#recipe-name').fill('Walnut Bookshelf');
    await page.getByRole('button', { name: 'Create Recipe' }).click();
    await page.waitForURL(/\/spaces\/[^/]+\/recipes\/.+/);
    await page.waitForLoadState('networkidle');

    // Navigate back to recipes list
    await page.getByRole('link', { name: 'Back to Recipes' }).click();
    await page.waitForURL(new RegExp(`/spaces/${spaceSlug}/recipes$`));
    await page.waitForLoadState('networkidle');

    await expect(page.getByText('Walnut Bookshelf')).toBeVisible();
    await expect(page.getByText('Draft')).toBeVisible();
  });
});
