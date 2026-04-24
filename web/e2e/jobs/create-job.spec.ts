import { test, expect } from '@playwright/test';
import { resetSpace } from '../helpers/reset';

let spaceSlug = '';

test.describe('Create Job Dialog', () => {
  test.beforeEach(async ({ page }) => {
    const result = await resetSpace(page);
    spaceSlug = result.slug;
  });

  test('open create job dialog, fill in title, submit, verify redirect to job detail page', async ({
    page,
  }) => {
    await page.goto(`/spaces/${spaceSlug}/jobs`);
    await page.waitForLoadState('networkidle');

    const newJobButton = page.getByRole('button', { name: 'New Job' });
    await expect(newJobButton).toBeVisible();
    await newJobButton.click();

    await expect(page.getByText('Create New Job')).toBeVisible();

    await page.locator('#job-title').fill('Oak Dining Table');
    await page.getByRole('button', { name: 'Create Job' }).click();

    await page.waitForURL(new RegExp(`/spaces/${spaceSlug}/.+`));
    await expect(page.locator('h1', { hasText: 'Oak Dining Table' })).toBeVisible();
  });

  test('job appears in jobs list after creation', async ({ page }) => {
    await page.goto(`/spaces/${spaceSlug}/jobs`);
    await page.waitForLoadState('networkidle');

    await page.getByRole('button', { name: 'New Job' }).click();
    await expect(page.getByText('Create New Job')).toBeVisible();
    await page.locator('#job-title').fill('Walnut Bookshelf');
    await page.getByRole('button', { name: 'Create Job' }).click();

    await page.waitForURL(new RegExp(`/spaces/${spaceSlug}/.+`));

    await page.goto(`/spaces/${spaceSlug}/jobs`);
    await expect(page.getByText('Walnut Bookshelf')).toBeVisible();
  });

  test('cancel button closes dialog without creating a job', async ({ page }) => {
    await page.goto(`/spaces/${spaceSlug}/jobs`);
    await page.waitForLoadState('networkidle');

    await page.getByRole('button', { name: 'New Job' }).click();
    await expect(page.getByText('Create New Job')).toBeVisible();

    await page.locator('#job-title').fill('Should Not Be Created');
    await page.getByRole('button', { name: 'Cancel' }).click();

    await expect(page.getByText('Create New Job')).not.toBeVisible();
    await expect(page.getByText('Should Not Be Created')).not.toBeVisible();
  });

  test('title is required — submit button disabled when title is empty', async ({ page }) => {
    await page.goto(`/spaces/${spaceSlug}/jobs`);
    await page.waitForLoadState('networkidle');

    await page.getByRole('button', { name: 'New Job' }).click();
    await expect(page.getByText('Create New Job')).toBeVisible();

    await expect(page.getByRole('button', { name: 'Create Job' })).toBeDisabled();

    await page.locator('#job-title').fill('Valid Title');
    await expect(page.getByRole('button', { name: 'Create Job' })).toBeEnabled();
  });
});
