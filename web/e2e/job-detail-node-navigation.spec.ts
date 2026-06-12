import { test, expect, type Page } from '@playwright/test';
import { resetSpace } from './helpers/reset';

let spaceSlug = '';

/** Create a job with one child task; return the job URL and the child's node id. */
async function createJobWithChild(
  page: Page,
  jobTitle: string,
  childTitle: string,
): Promise<{ jobUrl: string; childId: string }> {
  await page.goto(`/spaces/${spaceSlug}/jobs`);
  await page.waitForLoadState('networkidle');
  await page.getByRole('button', { name: 'New Job' }).click();
  await page.locator('#job-title').fill(jobTitle);
  await page.getByRole('button', { name: 'Create Job' }).click();
  // Creating a job navigates to its detail page; Add Node only exists there,
  // so waiting for it guarantees we're past the jobs-list page.
  const addNodeBtn = page.getByRole('button', { name: 'Add Node', exact: true });
  await expect(addNodeBtn).toBeVisible({ timeout: 10_000 });
  // Now on the job detail page — capture its URL (without any view query).
  const jobUrl = page.url().split('?')[0];

  await addNodeBtn.click();
  const inlineInput = page.locator('.svelte-flow__node input');
  await expect(inlineInput).toBeVisible({ timeout: 10_000 });
  await inlineInput.fill(childTitle);
  await page.keyboard.press('Enter');

  const childNode = page.locator('.svelte-flow__node', { hasText: childTitle });
  await expect(childNode).toBeVisible({ timeout: 10_000 });
  const childId = await childNode.getAttribute('data-id');
  expect(childId).toBeTruthy();
  return { jobUrl, childId: childId! };
}

test.describe('Job detail: selecting a task node navigates to its detail page', () => {
  test.beforeEach(async ({ page }) => {
    const result = await resetSpace(page);
    spaceSlug = result.slug;
  });

  test('graph view: single-clicking a child node navigates to that task', async ({ page }) => {
    const { childId } = await createJobWithChild(page, 'Graph Nav Job', 'Graph Child');

    // Dismiss any lingering selection/inline edit by clicking empty canvas.
    await page.locator('.svelte-flow').click({ position: { x: 12, y: 12 } });

    const node = page.locator('.svelte-flow__node', { hasText: 'Graph Child' });
    await node.click();

    await page.waitForURL((url) => url.pathname === `/spaces/${spaceSlug}/${childId}`, {
      timeout: 10_000,
    });
    await expect(page.getByRole('heading', { level: 2, name: 'Graph Child' })).toBeVisible({
      timeout: 10_000,
    });
  });

  test('list view: clicking a task row navigates to that task', async ({ page }) => {
    const { jobUrl, childId } = await createJobWithChild(page, 'List Nav Job', 'List Child');

    await page.goto(`${jobUrl}?view=list`);
    await page.waitForLoadState('networkidle');

    const row = page.locator('tr[role="link"]').filter({ hasText: 'List Child' });
    await expect(row).toBeVisible({ timeout: 10_000 });
    await row.click();

    await page.waitForURL((url) => url.pathname === `/spaces/${spaceSlug}/${childId}`, {
      timeout: 10_000,
    });
    await expect(page.getByRole('heading', { level: 2, name: 'List Child' })).toBeVisible({
      timeout: 10_000,
    });
  });

  test('board view: clicking a task card navigates to that task', async ({ page }) => {
    const { childId } = await createJobWithChild(page, 'Board Nav Job', 'Board Child');

    await page.getByRole('button', { name: 'Board', exact: true }).click();

    const card = page.locator(`a[href$="/spaces/${spaceSlug}/${childId}"]`);
    await expect(card).toBeVisible({ timeout: 10_000 });
    await card.click();

    await page.waitForURL((url) => url.pathname === `/spaces/${spaceSlug}/${childId}`, {
      timeout: 10_000,
    });
    await expect(page.getByRole('heading', { level: 2, name: 'Board Child' })).toBeVisible({
      timeout: 10_000,
    });
  });
});
