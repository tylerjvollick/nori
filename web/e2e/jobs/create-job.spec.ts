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

    // "New Job" button should be visible in the toolbar
    const newJobButton = page.getByRole('button', { name: 'New Job' });
    await expect(newJobButton).toBeVisible();

    // Open the dialog
    await newJobButton.click();

    // Dialog should be visible
    await expect(page.getByRole('dialog')).toBeVisible();
    await expect(page.getByText('Create New Job')).toBeVisible();

    // Fill in the title
    await page.locator('#job-title').fill('Oak Dining Table');

    // Submit the form
    await page.getByRole('button', { name: 'Create Job' }).click();

    // Should navigate to the job detail page
    await page.waitForURL(new RegExp(`/spaces/${spaceSlug}/.+`));

    // Job title should be visible on the detail page
    await expect(page.getByRole('heading', { name: 'Oak Dining Table' })).toBeVisible();
  });

  test('job appears in jobs list after creation', async ({ page }) => {
    await page.goto(`/spaces/${spaceSlug}/jobs`);

    // Create a job
    await page.getByRole('button', { name: 'New Job' }).click();
    await page.locator('#job-title').fill('Walnut Bookshelf');
    await page.getByRole('button', { name: 'Create Job' }).click();

    // Wait for navigation to the detail page
    await page.waitForURL(new RegExp(`/spaces/${spaceSlug}/.+`));

    // Navigate back to the jobs list
    await page.goto(`/spaces/${spaceSlug}/jobs`);

    // The job should appear in the board
    await expect(page.getByText('Walnut Bookshelf')).toBeVisible();
  });

  test('cancel button closes dialog without creating a job', async ({ page }) => {
    await page.goto(`/spaces/${spaceSlug}/jobs`);

    await page.getByRole('button', { name: 'New Job' }).click();
    await expect(page.getByRole('dialog')).toBeVisible();

    await page.locator('#job-title').fill('Should Not Be Created');
    await page.getByRole('button', { name: 'Cancel' }).click();

    // Dialog should be closed
    await expect(page.getByRole('dialog')).not.toBeVisible();

    // The job should not be in the board
    await expect(page.getByText('Should Not Be Created')).not.toBeVisible();
  });

  test('title is required — submit button disabled when title is empty', async ({ page }) => {
    await page.goto(`/spaces/${spaceSlug}/jobs`);

    await page.getByRole('button', { name: 'New Job' }).click();
    await expect(page.getByRole('dialog')).toBeVisible();

    // Submit button should be disabled with no title
    await expect(page.getByRole('button', { name: 'Create Job' })).toBeDisabled();

    // After filling in the title it becomes enabled
    await page.locator('#job-title').fill('Valid Title');
    await expect(page.getByRole('button', { name: 'Create Job' })).toBeEnabled();
  });
});
