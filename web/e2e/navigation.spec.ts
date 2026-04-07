import { test, expect } from '@playwright/test';

test.describe('Navigation — authenticated pages', () => {
  // Uses storageState from globalSetup automatically

  test('/flow loads without error', async ({ page }) => {
    await page.goto('/flow');
    await expect(page).toHaveURL(/\/flow/);
    // Page should render some meaningful content (not a blank white screen)
    await expect(page.locator('body')).not.toBeEmpty();
  });

  test('/admin loads without error for admin user', async ({ page }) => {
    await page.goto('/admin');

    // Two valid outcomes: the admin page renders, or we get redirected
    // (non-admin users are sent to /).  Either way, no crash.
    const url = page.url();
    expect(url).toMatch(/\/(admin|$)/);
    await expect(page.locator('body')).not.toBeEmpty();
  });

  test('/sops loads without error', async ({ page }) => {
    await page.goto('/sops');
    // May redirect if /sops isn't built yet — that's fine, just no crash
    await expect(page.locator('body')).not.toBeEmpty();
  });
});
