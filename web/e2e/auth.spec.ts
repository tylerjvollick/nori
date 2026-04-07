import { test, expect } from '@playwright/test';
import { login } from './helpers/auth';
import { env } from './helpers/env';

test.describe('Authentication', () => {
  test.describe('unauthenticated flows', () => {
    // These tests must NOT use the stored auth state
    test.use({ storageState: { cookies: [], origins: [] } });

    test('unauthenticated user is redirected to /login (no loop)', async ({
      page,
    }) => {
      await page.goto('/');

      // Should land on /login
      await expect(page).toHaveURL(/\/login/);

      // Wait a beat and confirm we're still on /login (no infinite loop)
      await page.waitForTimeout(1_500);
      await expect(page).toHaveURL(/\/login/);
    });

    test('login with valid credentials succeeds', async ({ page }) => {
      await login(page);

      // After login we should be somewhere other than /login
      expect(page.url()).not.toContain('/login');
    });

    test('login form shows error for invalid credentials', async ({ page }) => {
      await page.goto('/login', { waitUntil: 'networkidle' });

      await page.getByPlaceholder('Email address').fill('bad@example.com');
      await page.getByPlaceholder('Password').fill('wrong');
      await page.getByRole('button', { name: 'Sign in' }).click();

      // Wait for the API call to return and error state to render.
      // The error message "Unauthorized" is rendered in a .text-destructive div.
      await expect(
        page.locator('.text-destructive'),
      ).toBeVisible({ timeout: 5_000 });

      // Should still be on /login
      await expect(page).toHaveURL(/\/login/);
    });

    test('protected route redirects to /login when unauthenticated', async ({
      page,
    }) => {
      await page.goto('/flow');
      await expect(page).toHaveURL(/\/login/);
    });
  });

  test.describe('authenticated flows', () => {
    test('clearing auth state redirects to /login', async ({ page }) => {
      // Starts authenticated via storageState from globalSetup

      // Verify we can access an authenticated page
      await page.goto('/');
      await page.waitForURL((url) => !url.pathname.startsWith('/login'), {
        timeout: 5_000,
      });

      // Clear all auth state (simulates what logout does)
      await page.context().clearCookies();
      await page.evaluate(() => {
        localStorage.removeItem('accessToken');
        localStorage.removeItem('activeSpaceId');
      });

      // Navigate to a protected route — should redirect to /login
      await page.goto('/flow');
      await expect(page).toHaveURL(/\/login/);
    });

    test('logout button is visible when authenticated', async ({ page }) => {
      await page.goto('/');
      await page.waitForURL((url) => !url.pathname.startsWith('/login'), {
        timeout: 5_000,
      });

      // The Logout button should be visible in the header
      await expect(
        page.getByRole('button', { name: /logout/i }),
      ).toBeVisible({ timeout: 3_000 });
    });
  });
});
