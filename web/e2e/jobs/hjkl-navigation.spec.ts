import { test, expect, type Page } from '@playwright/test';
import { resetSpace } from '../helpers/reset';

let spaceSlug = '';

/**
 * Create a job with N child tasks (default names).
 */
async function createJobWithTasks(page: Page, jobTitle: string, count: number): Promise<void> {
  await page.goto(`/spaces/${spaceSlug}/jobs`);
  await page.waitForLoadState('networkidle');
  await page.getByRole('button', { name: 'New Job' }).click();
  await page.locator('#job-title').fill(jobTitle);
  await page.getByRole('button', { name: 'Create Job' }).click();
  await page.waitForURL(new RegExp(`/spaces/${spaceSlug}/.+`));

  for (let i = 0; i < count; i++) {
    const addNodeBtn = page.getByRole('button', { name: 'Add Node' });
    await expect(addNodeBtn).toBeVisible({ timeout: 10000 });
    await addNodeBtn.click();
    await expect(page.locator('.svelte-flow__node')).toHaveCount(i + 1, { timeout: 10000 });
    await page.keyboard.press('Escape');
  }
}

test.describe('hjkl keyboard navigation', () => {
  test.beforeEach(async ({ page }) => {
    const result = await resetSpace(page);
    spaceSlug = result.slug;
  });

  test('board view: j/k moves between cards in a column', async ({ page }) => {
    await createJobWithTasks(page, 'HjklTest1', 3);

    await page.getByRole('button', { name: 'Board', exact: true }).click();
    await page.waitForLoadState('networkidle');

    // j selects first card
    await page.keyboard.press('j');
    const selected = page.locator('[data-kb-selected="true"]');
    await expect(selected).toHaveCount(1, { timeout: 5000 });
    const firstText = await selected.textContent();

    // j moves down
    await page.keyboard.press('j');
    await expect(selected).toHaveCount(1);
    const secondText = await selected.textContent();
    expect(firstText).not.toBe(secondText);

    // k moves back up
    await page.keyboard.press('k');
    await expect(selected).toHaveCount(1);
    const backUp = await selected.textContent();
    expect(backUp).toBe(firstText);
  });

  test('board view: h/l moves between columns', async ({ page }) => {
    await createJobWithTasks(page, 'HjklTest2', 2);

    // Start one task via API to move it to "In Progress" column
    // Get auth token and space ID from localStorage
    const { token, spaceId } = await page.evaluate(() => ({
      token: localStorage.getItem('accessToken') ?? '',
      spaceId: localStorage.getItem('activeSpaceId') ?? '',
    }));

    // Get the task ID of the first child node
    const taskId = await page.locator('.svelte-flow__node').first().getAttribute('data-id');

    // Start the task via API
    const apiURL = 'http://localhost:8081';
    const startRes = await fetch(`${apiURL}/api/v1/spaces/${spaceId}/tasks/${taskId}/start`, {
      method: 'POST',
      headers: { Cookie: `nori_token=${token}` },
    });
    expect(startRes.ok).toBeTruthy();

    // Reload to pick up the status change, then switch to board view
    await page.reload();
    await page.waitForLoadState('networkidle');
    await page.getByRole('button', { name: 'Board', exact: true }).click();
    await page.waitForLoadState('networkidle');

    // j selects first card
    await page.keyboard.press('j');
    const selected = page.locator('[data-kb-selected="true"]');
    await expect(selected).toHaveCount(1, { timeout: 5000 });
    const firstColText = await selected.textContent();

    // l moves to next column
    await page.keyboard.press('l');
    await expect(selected).toHaveCount(1);
    const nextColText = await selected.textContent();
    expect(firstColText).not.toBe(nextColText);

    // h moves back
    await page.keyboard.press('h');
    await expect(selected).toHaveCount(1);
    const backText = await selected.textContent();
    expect(backText).toBe(firstColText);
  });

  test('list view: j/k moves between rows', async ({ page }) => {
    await createJobWithTasks(page, 'HjklTest3', 3);

    await page.getByRole('button', { name: 'List', exact: true }).click();
    await page.waitForLoadState('networkidle');

    // j selects first row
    await page.keyboard.press('j');
    const selected = page.locator('[data-kb-selected="true"]');
    await expect(selected).toHaveCount(1, { timeout: 5000 });
    const firstRow = await selected.textContent();

    // j moves down
    await page.keyboard.press('j');
    await expect(selected).toHaveCount(1);
    const secondRow = await selected.textContent();
    expect(firstRow).not.toBe(secondRow);

    // k moves back up
    await page.keyboard.press('k');
    await expect(selected).toHaveCount(1);
    const backUp = await selected.textContent();
    expect(backUp).toBe(firstRow);
  });

  test('Enter selects task in detail panel from board view', async ({ page }) => {
    await createJobWithTasks(page, 'HjklTest4', 1);

    await page.getByRole('button', { name: 'Board', exact: true }).click();
    await page.waitForLoadState('networkidle');

    await page.keyboard.press('j');
    await expect(page.locator('[data-kb-selected="true"]')).toHaveCount(1, { timeout: 5000 });

    await page.keyboard.press('Enter');

    // Should show selected task in the detail panel (right pane)
    await expect(page.getByRole('heading', { level: 2 }).first()).toBeVisible({ timeout: 10000 });
  });

  test('keyboard navigation suppressed when focus is in an input', async ({ page }) => {
    await createJobWithTasks(page, 'HjklTest5', 2);

    await page.getByRole('button', { name: 'Board', exact: true }).click();
    await page.waitForLoadState('networkidle');

    // Verify j normally selects a card
    await page.keyboard.press('j');
    const selected = page.locator('[data-kb-selected="true"]');
    await expect(selected).toHaveCount(1, { timeout: 5000 });

    // Clear selection
    await page.keyboard.press('Escape');
    await expect(selected).toHaveCount(0, { timeout: 5000 });

    // Focus the search input
    const searchInput = page.locator('[role="searchbox"]');
    if (await searchInput.isVisible()) {
      await searchInput.focus();
      // j in input should NOT trigger card selection
      await page.keyboard.press('j');
      await expect(selected).toHaveCount(0);
    }
  });
});
