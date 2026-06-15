import { test, expect } from '@playwright/test';

const SPACE_URL = /\/spaces\/[A-Z][A-Z0-9]{1,4}/;

test.describe('Home (/) forwards authenticated users into a space', () => {
  // Uses storageState from globalSetup automatically (authenticated user).

  test('visiting / redirects to a space and never renders the landing cards', async ({ page }) => {
    // Clear any persisted last-visited space for a clean default-space baseline.
    await page.goto('/');
    await page.evaluate(() => localStorage.removeItem('lastVisitedSpace'));

    await page.goto('/');
    await page.waitForURL(SPACE_URL, { timeout: 10_000 });
    expect(page.url()).toMatch(SPACE_URL);

    // The old marketing landing page must never be shown.
    await expect(page.getByText('Welcome back')).toHaveCount(0);
    await expect(page.getByText('A thin layer that holds everything together')).toHaveCount(0);
  });

  test('a stored lastVisitedSpace is honored on the next visit to /', async ({ page }) => {
    // Land on a space first — opening it persists lastVisitedSpace.
    await page.goto('/');
    await page.waitForURL(SPACE_URL, { timeout: 10_000 });

    const slug = page.url().match(/\/spaces\/([A-Z][A-Z0-9]{1,4})/)?.[1];
    expect(slug).toBeTruthy();

    // The space slug should now be persisted.
    const stored = await page.evaluate(() => localStorage.getItem('lastVisitedSpace'));
    expect(stored).toBe(slug);

    // Returning to / forwards back to the stored space.
    await page.goto('/');
    await page.waitForURL(new RegExp(`/spaces/${slug}`), { timeout: 10_000 });
    expect(page.url()).toMatch(new RegExp(`/spaces/${slug}`));
  });
});
