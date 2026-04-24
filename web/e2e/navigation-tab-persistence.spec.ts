import { test, expect } from '@playwright/test';
import { resetSpace } from './helpers/reset';

let spaceSlug = '';

test.describe('Space tab: last visited tab persistence', () => {
  test.beforeEach(async ({ page }) => {
    const result = await resetSpace(page);
    spaceSlug = result.slug;
    // Clear any persisted tab state for a clean baseline
    await page.evaluate((slug) => {
      localStorage.removeItem(`lastVisitedTab:${slug}`);
    }, spaceSlug);
  });

  test('/spaces/:slug redirects to /jobs by default', async ({ page }) => {
    await page.goto(`/spaces/${spaceSlug}`);
    await page.waitForURL(new RegExp(`/spaces/${spaceSlug}/jobs`), { timeout: 5_000 });
    expect(page.url()).toMatch(new RegExp(`/spaces/${spaceSlug}/jobs`));
  });

  test('switching to tasks tab persists selection, reload redirects to /tasks', async ({ page }) => {
    // Start on default (jobs)
    await page.goto(`/spaces/${spaceSlug}`);
    await page.waitForURL(new RegExp(`/spaces/${spaceSlug}/jobs`));

    // Click the Tasks tab
    await page.getByRole('tab', { name: 'Tasks' }).click();
    await page.waitForURL(new RegExp(`/spaces/${spaceSlug}/tasks`));

    // Navigate back to the space root
    await page.goto(`/spaces/${spaceSlug}`);

    // Should redirect to tasks (last visited)
    await page.waitForURL(new RegExp(`/spaces/${spaceSlug}/tasks`), { timeout: 5_000 });
    expect(page.url()).toMatch(new RegExp(`/spaces/${spaceSlug}/tasks`));
  });

  test('localStorage key is per-space slug', async ({ page }) => {
    await page.goto(`/spaces/${spaceSlug}/tasks`);
    await page.waitForURL(new RegExp(`/spaces/${spaceSlug}/tasks`));

    const stored = await page.evaluate(
      (slug) => localStorage.getItem(`lastVisitedTab:${slug}`),
      spaceSlug
    );
    expect(stored).toBe('tasks');
  });
});
