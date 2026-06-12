import { test, expect, type Page } from '@playwright/test';
import { resetSpace } from './helpers/reset';

let spaceSlug = '';

/** Create a bare job (no child tasks) and return to the jobs view. */
async function createJob(page: Page, title: string): Promise<void> {
  await page.goto(`/spaces/${spaceSlug}/jobs`);
  await page.waitForLoadState('networkidle');
  await page.getByRole('button', { name: 'New Job' }).click();
  await page.locator('#job-title').fill(title);
  await page.getByRole('button', { name: 'Create Job' }).click();
  // Creating a job navigates to its detail page.
  await page.waitForURL(new RegExp(`/spaces/${spaceSlug}/.+`));
}

test.describe('Space landing: Jobs tab is the default and is highlighted', () => {
  test.beforeEach(async ({ page }) => {
    const result = await resetSpace(page);
    spaceSlug = result.slug;
    // Clear persisted tab state so we exercise the default (Jobs) path.
    await page.evaluate((slug) => {
      localStorage.removeItem(`lastVisitedTab:${slug}`);
    }, spaceSlug);
  });

  test('opening a bare space URL lands on /jobs with the Jobs tab selected', async ({ page }) => {
    await page.goto(`/spaces/${spaceSlug}`);

    await page.waitForURL(new RegExp(`/spaces/${spaceSlug}/jobs`), { timeout: 10_000 });
    expect(page.url()).toMatch(new RegExp(`/spaces/${spaceSlug}/jobs`));

    const jobsTab = page.getByRole('tab', { name: 'Jobs' });
    await expect(jobsTab).toHaveAttribute('aria-selected', 'true');

    // No other tab should be selected.
    await expect(page.getByRole('tab', { name: 'Tasks' })).toHaveAttribute('aria-selected', 'false');
  });

  test('clicking a job card (board view) navigates to the job detail page', async ({ page }) => {
    await createJob(page, 'BoardNavJob');

    await page.goto(`/spaces/${spaceSlug}/jobs`);
    await page.waitForLoadState('networkidle');

    const card = page.locator('a[href^="/spaces/"]').filter({ hasText: 'BoardNavJob' }).first();
    const href = await card.getAttribute('href');
    expect(href).toMatch(new RegExp(`^/spaces/${spaceSlug}/[^/]+$`));

    await card.click();
    await page.waitForURL(`**${href}`, { timeout: 10_000 });
    expect(page.url()).toContain(href!);
  });

  test('clicking a job row (list view) navigates to the job detail page', async ({ page }) => {
    await createJob(page, 'ListNavJob');

    await page.goto(`/spaces/${spaceSlug}/jobs?view=list`);
    await page.waitForLoadState('networkidle');

    const row = page.locator('[data-testid="jobs-list-table"] tr[role="link"]').filter({
      hasText: 'ListNavJob',
    });
    await expect(row).toHaveCount(1);

    // The ID cell carries the full job id in its title attribute.
    const jobId = await row.locator('td').first().getAttribute('title');
    expect(jobId).toBeTruthy();

    await row.click();
    await page.waitForURL((url) => url.pathname === `/spaces/${spaceSlug}/${jobId}`, {
      timeout: 10_000,
    });
  });
});
