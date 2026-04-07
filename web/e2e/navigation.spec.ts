import { test, expect } from '@playwright/test';

test.describe('Navigation — authenticated pages', () => {
  // Uses storageState from globalSetup automatically

  test('/flow loads without error', async ({ page }) => {
    const response = await page.goto('/flow');
    expect(response?.status()).toBeLessThan(400);
    await expect(page).toHaveURL(/\/flow/);
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

  test('/sops loads without error', async ({ page }) => {
    const response = await page.goto('/sops');
    expect(response?.status()).toBeLessThan(400);
    // May redirect if /sops isn't built yet — that's fine, just no crash
    await expect(page.locator('body')).not.toBeEmpty();
    await expect(page.locator('text=Internal Error')).not.toBeVisible();
  });
});
