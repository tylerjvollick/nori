import { test, expect } from '@playwright/test';

test.describe('Navigation — authenticated pages', () => {
  // Uses storageState from globalSetup automatically

  test('/flow redirects to a space', async ({ page }) => {
    const response = await page.goto('/flow');
    expect(response?.status()).toBeLessThan(400);
    // /flow now redirects to /spaces/:slug (slug = 2-5 uppercase letters/digits)
    await page.waitForURL(/\/spaces\/[A-Z][A-Z0-9]{1,4}/, { timeout: 5_000 });
    // Page should render some meaningful content (not a blank white screen)
    await expect(page.locator('body')).not.toBeEmpty();
    // Should not show SvelteKit error page
    await expect(page.locator('text=Internal Error')).not.toBeVisible();
    await expect(page.locator('text=500')).not.toBeVisible();
  });

  test('/admin loads without error for admin user', async ({ page }) => {
    const response = await page.goto('/admin');

    // Two valid outcomes: the admin page renders, or we get redirected
    // (non-admin users are sent to /).  Either way, no crash.
    expect(response?.status()).toBeLessThan(400);
    const url = page.url();
    expect(url).toMatch(/\/(admin|$)/);
    await expect(page.locator('body')).not.toBeEmpty();
    await expect(page.locator('text=Internal Error')).not.toBeVisible();
  });

});
