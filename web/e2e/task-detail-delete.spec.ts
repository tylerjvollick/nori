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

  const addNodeBtn = page.getByRole('button', { name: 'Add Node', exact: true });
  await expect(addNodeBtn).toBeVisible({ timeout: 10_000 });
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

// nori-j7k: discoverable Delete action on the individual task detail page.
test.describe('Task detail: delete', () => {
  test.beforeEach(async ({ page }) => {
    const result = await resetSpace(page);
    spaceSlug = result.slug;
  });

  test('deleting a child task removes it and navigates back to the parent job', async ({
    page,
  }) => {
    const { jobUrl, childId } = await createJobWithChild(page, 'Delete Parent Job', 'Doomed Task');

    // Open the child task's own detail page.
    await page.goto(`/spaces/${spaceSlug}/${childId}`);
    await expect(page.getByRole('heading', { level: 2, name: 'Doomed Task' })).toBeVisible({
      timeout: 10_000,
    });

    // Trigger delete and confirm.
    await page.getByTestId('delete-task').click();
    const dialog = page.getByRole('dialog');
    await expect(dialog).toBeVisible({ timeout: 5_000 });
    await expect(dialog).toContainText('Delete task?');
    await page.getByTestId('delete-task-confirm').click();

    // Navigates back to the parent job's detail page.
    await page.waitForURL((url) => url.pathname === new URL(jobUrl).pathname, { timeout: 10_000 });

    // The deleted task no longer appears in the job graph.
    await expect(page.locator('.svelte-flow__node', { hasText: 'Doomed Task' })).toHaveCount(0, {
      timeout: 10_000,
    });
  });

  test('the Delete control is not offered on a job-root detail page', async ({ page }) => {
    await createJobWithChild(page, 'Undeletable Job', 'Some Task');
    // After creation we are on the job-root detail page — Delete must be absent.
    await expect(page.getByTestId('delete-task')).toHaveCount(0);
  });
});
