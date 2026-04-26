import { test, expect } from '@playwright/test';
import { resetSpace } from '../helpers/reset';

let spaceSlug = '';

test.describe('Recipe Time Estimation', () => {
  test.beforeEach(async ({ page }) => {
    const result = await resetSpace(page);
    spaceSlug = result.slug;
  });

  /**
   * Helper: create a recipe with one task node and return the recipe URL.
   */
  async function createRecipeWithTask(page: import('@playwright/test').Page, recipeName: string, taskName: string) {
    await page.goto(`/spaces/${spaceSlug}/recipes`);
    await page.waitForLoadState('networkidle');
    await page.getByRole('button', { name: 'New Recipe' }).click();
    await page.locator('#recipe-name').fill(recipeName);
    await page.getByRole('button', { name: 'Create Recipe' }).click();
    await page.waitForURL(/\/spaces\/[^/]+\/recipes\/.+/);

    // Wait for graph to load
    const addNodeBtn = page.getByRole('button', { name: 'Add Node', exact: true });
    await expect(addNodeBtn).toBeVisible({ timeout: 10000 });

    // Add a node
    await addNodeBtn.click();
    const nodes = page.locator('.svelte-flow__node');
    await expect(nodes).toHaveCount(2, { timeout: 10000 }); // root + new node

    // Rename the node
    const inlineInput = page.locator('.svelte-flow__node input');
    await expect(inlineInput).toBeVisible({ timeout: 5000 });
    await inlineInput.fill(taskName);
    await page.keyboard.press('Enter');

    // Wait for title to commit
    await expect(page.locator('.svelte-flow__node', { hasText: taskName })).toBeVisible({ timeout: 10000 });
  }

  /**
   * Helper: click a task node to open the detail panel.
   */
  async function selectTask(page: import('@playwright/test').Page, taskName: string) {
    // Click away first to deselect
    await page.locator('.svelte-flow').click({ position: { x: 10, y: 10 } });
    // Click the node
    const node = page.locator('.svelte-flow__node', { hasText: taskName });
    await node.click();
    // Wait for detail panel to show the task title
    await expect(page.getByRole('heading', { name: taskName, level: 2 }).first()).toBeVisible({ timeout: 5000 });
  }

  test('enter formula on recipe task, save, reload, assert persists', async ({ page }) => {
    await createRecipeWithTask(page, 'Formula Persist Test', 'Cut Parts');
    await selectTask(page, 'Cut Parts');

    // Find the formula input
    const formulaInput = page.getByTestId('estimated-time-formula');
    await expect(formulaInput).toBeVisible({ timeout: 5000 });

    // Enter a formula
    await formulaInput.fill('5m * {{batch_size}}');
    await formulaInput.blur();

    // Wait for save to complete
    await page.waitForTimeout(500);

    // Reload the page
    await page.reload();
    await page.waitForLoadState('networkidle');

    // Wait for graph to load
    await expect(page.getByRole('button', { name: 'Add Node', exact: true })).toBeVisible({ timeout: 10000 });

    // Re-select the task
    await selectTask(page, 'Cut Parts');

    // Assert the formula persisted
    const reloadedInput = page.getByTestId('estimated-time-formula');
    await expect(reloadedInput).toHaveValue('5m * {{batch_size}}', { timeout: 5000 });
  });

  test('recipe task detail shows formula input with helper text', async ({ page }) => {
    await createRecipeWithTask(page, 'Helper Text Test', 'Sand Edges');
    await selectTask(page, 'Sand Edges');

    // Formula input should be visible in recipe mode
    const formulaInput = page.getByTestId('estimated-time-formula');
    await expect(formulaInput).toBeVisible({ timeout: 5000 });

    // Helper text should explain syntax
    await expect(page.getByText('Flat:')).toBeVisible();
    await expect(page.getByText('Formula:')).toBeVisible();
  });

  test('recipe task shows Inherit for nil batchSize', async ({ page }) => {
    await createRecipeWithTask(page, 'Batch Inherit Test', 'Assembly Step');
    await selectTask(page, 'Assembly Step');

    // Batch size should default to "Inherit" for new steps
    const batchSizeValue = page.getByTestId('batch-size-value');
    await expect(batchSizeValue).toBeVisible({ timeout: 5000 });
    await expect(batchSizeValue).toHaveText('Inherit');
  });

  test('set explicit batch size, reload, assert persists', async ({ page }) => {
    await createRecipeWithTask(page, 'Batch Set Test', 'Cut Parts');
    await selectTask(page, 'Cut Parts');

    // Click to edit batch size
    const batchSizeValue = page.getByTestId('batch-size-value');
    await expect(batchSizeValue).toBeVisible({ timeout: 5000 });
    await batchSizeValue.click();

    // Enter batch size 3
    const input = page.getByTestId('batch-size-input');
    await expect(input).toBeVisible({ timeout: 5000 });
    await input.fill('3');
    await input.blur();
    await page.waitForTimeout(500);

    // Reload
    await page.reload();
    await page.waitForLoadState('networkidle');
    await expect(page.getByRole('button', { name: 'Add Node', exact: true })).toBeVisible({ timeout: 10000 });

    // Re-select and verify
    await selectTask(page, 'Cut Parts');
    const reloadedValue = page.getByTestId('batch-size-value');
    await expect(reloadedValue).toHaveText('3', { timeout: 5000 });
  });

  test('change batch size back to Inherit, reload, assert Inherit', async ({ page }) => {
    await createRecipeWithTask(page, 'Batch Reset Test', 'Sand Edges');
    await selectTask(page, 'Sand Edges');

    // Set to 5 first
    const batchSizeValue = page.getByTestId('batch-size-value');
    await batchSizeValue.click();
    const input = page.getByTestId('batch-size-input');
    await input.fill('5');
    await input.blur();
    await page.waitForTimeout(500);

    // Verify it's 5
    await expect(page.getByTestId('batch-size-value')).toHaveText('5', { timeout: 5000 });

    // Now reset to inherit
    await page.getByTestId('batch-size-value').click();
    const input2 = page.getByTestId('batch-size-input');
    await input2.fill('');
    await input2.blur();
    await page.waitForTimeout(500);

    // Reload and verify inherit
    await page.reload();
    await page.waitForLoadState('networkidle');
    await expect(page.getByRole('button', { name: 'Add Node', exact: true })).toBeVisible({ timeout: 10000 });
    await selectTask(page, 'Sand Edges');
    await expect(page.getByTestId('batch-size-value')).toHaveText('Inherit', { timeout: 5000 });
  });

  test('completion modal shows time entries and total', async ({ page }) => {
    // Create recipe, publish, roll to get a job task
    await createRecipeWithTask(page, 'Completion Modal Test', 'Finishing Step');
    await selectTask(page, 'Finishing Step');

    // Enter a flat formula
    const formulaInput = page.getByTestId('estimated-time-formula');
    await formulaInput.fill('15m');
    await formulaInput.blur();
    await page.waitForTimeout(500);

    // Publish
    const publishBtn = page.getByRole('button', { name: 'Publish' });
    await expect(publishBtn).toBeVisible({ timeout: 5000 });
    await publishBtn.click();
    await expect(page.getByTestId('recipe-status-tag').first()).toHaveText('Published', { timeout: 10000 });

    // Roll
    const rollBtn = page.getByRole('button', { name: 'Roll Job' });
    await expect(rollBtn).toBeVisible({ timeout: 5000 });
    await rollBtn.click();
    const rollConfirmBtn = page.getByRole('button', { name: 'Roll', exact: true });
    if (await rollConfirmBtn.isVisible({ timeout: 2000 }).catch(() => false)) {
      await rollConfirmBtn.click();
    }
    await page.waitForURL(/\/spaces\/[^/]+\/nori-/, { timeout: 10000 });
    await page.waitForLoadState('networkidle');
    await expect(page.locator('.svelte-flow__node')).toHaveCount(2, { timeout: 10000 });

    // Select the task in the job
    const taskNode = page.locator('.svelte-flow__node', { hasText: 'Finishing Step' });
    await taskNode.click();
    await expect(page.getByRole('heading', { name: 'Finishing Step', level: 2 }).first()).toBeVisible({ timeout: 5000 });

    // Start the task (which also starts a timer)
    const startBtn = page.getByRole('button', { name: 'Start' });
    await expect(startBtn).toBeVisible({ timeout: 5000 });
    await startBtn.click();

    // Wait for task to become active
    await expect(page.getByText('Active')).toBeVisible({ timeout: 5000 });

    // Click Complete to open the modal
    const completeBtn = page.getByRole('button', { name: 'Complete' });
    await expect(completeBtn).toBeVisible({ timeout: 5000 });
    await completeBtn.click();

    // Modal should be visible with time entry list or empty message
    await expect(page.getByText('Complete Task')).toBeVisible({ timeout: 5000 });

    // Total should be visible
    const total = page.getByTestId('completion-total');
    await expect(total).toBeVisible({ timeout: 5000 });
    await expect(total).toContainText('Total');

    // "Add Time Entry" button should be present
    await expect(page.getByRole('button', { name: 'Add Time Entry' })).toBeVisible();

    // Confirm & Complete button should be present
    await expect(page.getByRole('button', { name: 'Confirm & Complete' })).toBeVisible();

    // Complete the task
    await page.getByRole('button', { name: 'Confirm & Complete' }).click();

    // Should show completed phase
    await expect(page.getByText('Task Completed')).toBeVisible({ timeout: 10000 });
  });

  test('rolled job task shows recipe estimate as read-only', async ({ page }) => {
    // Create recipe with formula, publish, and roll
    await createRecipeWithTask(page, 'Roll Estimate Test', 'Assembly Step');
    await selectTask(page, 'Assembly Step');

    // Enter a flat formula
    const formulaInput = page.getByTestId('estimated-time-formula');
    await formulaInput.fill('30m');
    await formulaInput.blur();
    await page.waitForTimeout(500);

    // Publish the recipe
    const publishBtn = page.getByRole('button', { name: 'Publish' });
    await expect(publishBtn).toBeVisible({ timeout: 5000 });
    await publishBtn.click();

    // Wait for publish confirmation
    await expect(page.getByTestId('recipe-status-tag').first()).toHaveText('Published', { timeout: 10000 });

    // Roll the recipe
    const rollBtn = page.getByRole('button', { name: 'Roll Job' });
    await expect(rollBtn).toBeVisible({ timeout: 5000 });
    await rollBtn.click();

    // Fill in roll dialog if it appears
    const rollConfirmBtn = page.getByRole('button', { name: 'Roll', exact: true });
    if (await rollConfirmBtn.isVisible({ timeout: 2000 }).catch(() => false)) {
      await rollConfirmBtn.click();
    }

    // Should navigate to the job page
    await page.waitForURL(/\/spaces\/[^/]+\/nori-/, { timeout: 10000 });
    await page.waitForLoadState('networkidle');

    // Wait for graph to load
    await expect(page.locator('.svelte-flow__node')).toHaveCount(2, { timeout: 10000 });

    // Click the Assembly Step task node
    const taskNode = page.locator('.svelte-flow__node', { hasText: 'Assembly Step' });
    await taskNode.click();

    // Should show "Recipe Estimate" with the evaluated value (30m = 1800s → "30m")
    const recipeEstimate = page.getByTestId('recipe-estimate');
    await expect(recipeEstimate).toBeVisible({ timeout: 5000 });
    await expect(recipeEstimate).toContainText('30m');

    // Should NOT show the formula input (this is a job task, not a recipe task)
    await expect(page.getByTestId('estimated-time-formula')).not.toBeVisible();
  });
});
