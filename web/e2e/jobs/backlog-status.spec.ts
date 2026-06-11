import { test, expect, type Page } from '@playwright/test';
import { resetSpace } from '../helpers/reset';

let spaceSlug = '';

/** Locate a kanban column on the jobs board by its header title. */
function column(page: Page, title: string) {
  return page.locator('div.min-w-\\[280px\\]', {
    has: page.getByRole('heading', { name: title, exact: true }),
  });
}

/** Create a job through the New Job dialog, optionally putting it in the backlog. */
async function createJob(page: Page, title: string, opts?: { backlog?: boolean }) {
  await page.goto(`/spaces/${spaceSlug}/jobs`);
  await page.waitForLoadState('networkidle');

  await page.getByRole('button', { name: 'New Job' }).click();
  await expect(page.getByText('Create New Job')).toBeVisible();
  await page.locator('#job-title').fill(title);

  if (opts?.backlog) {
    await page.getByTestId('job-status-select').click();
    await page.getByRole('option', { name: /Backlog/ }).click();
  }

  await page.getByRole('button', { name: 'Create Job' }).click();
  await page.waitForURL(new RegExp(`/spaces/${spaceSlug}/.+`));
}

test.describe('Backlog job status', () => {
  test.beforeEach(async ({ page }) => {
    const result = await resetSpace(page);
    spaceSlug = result.slug;
  });

  test('jobs board shows a Backlog column', async ({ page }) => {
    await page.goto(`/spaces/${spaceSlug}/jobs`);
    await page.waitForLoadState('networkidle');

    await expect(page.getByRole('heading', { name: 'Backlog', exact: true })).toBeVisible();
    await expect(page.getByRole('heading', { name: 'Ready', exact: true })).toBeVisible();
  });

  test('backlog job appears in the Backlog column, visually distinct from open jobs', async ({
    page,
  }) => {
    await createJob(page, 'Open Shelf Job');
    await createJob(page, 'Backlog Table Job', { backlog: true });

    await page.goto(`/spaces/${spaceSlug}/jobs`);
    await page.waitForLoadState('networkidle');

    // The backlog job lives in the Backlog column.
    const backlogColumn = column(page, 'Backlog');
    await expect(backlogColumn.getByText('Backlog Table Job')).toBeVisible();

    // The open job lives in the Ready column, not the Backlog column.
    const readyColumn = column(page, 'Ready');
    await expect(readyColumn.getByText('Open Shelf Job')).toBeVisible();
    await expect(backlogColumn.getByText('Open Shelf Job')).not.toBeVisible();

    // The backlog job card carries a distinct "Backlog" badge; open jobs say "Ready".
    const backlogCard = page.getByRole('link', { name: /Backlog Table Job/ });
    await expect(backlogCard.getByTestId('job-status-badge')).toHaveText('Backlog');

    const openCard = page.getByRole('link', { name: /Open Shelf Job/ });
    await expect(openCard.getByTestId('job-status-badge')).toHaveText('Ready');
  });

  test('list view shows backlog status for backlog jobs', async ({ page }) => {
    await createJob(page, 'Backlog List Job', { backlog: true });

    await page.goto(`/spaces/${spaceSlug}/jobs?view=list`);
    await page.waitForLoadState('networkidle');

    const table = page.locator('[data-testid="jobs-list-table"]');
    await expect(table).toBeVisible();

    const row = table.locator('tr', { hasText: 'Backlog List Job' });
    await expect(row.getByText('backlog', { exact: true })).toBeVisible();
  });
});
