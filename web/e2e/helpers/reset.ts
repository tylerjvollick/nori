import { type Page } from '@playwright/test';
import { env } from './env';

/**
 * Call the dev-only POST /api/test/reset endpoint to wipe all work data
 * (recipes, tasks, etc.) from the active space, then re-seed baseline stations.
 *
 * Requires:
 *   - NORI_ENV=development on the backend
 *   - A valid auth token (cookie from storageState)
 *   - activeSpaceId in localStorage
 *
 * Intended to be called in test beforeEach/beforeAll hooks so each test
 * starts from a clean slate.
 */
export async function resetSpace(page: Page): Promise<void> {
  // Ensure localStorage is populated (storageState injects on first navigation)
  await page.goto(env.baseURL);

  const { token, spaceId } = await page.evaluate(() => {
    const cookies = document.cookie;
    return {
      token: localStorage.getItem('accessToken') ?? '',
      spaceId: localStorage.getItem('activeSpaceId') ?? '',
    };
  });

  if (!spaceId) {
    throw new Error(
      'resetSpace: activeSpaceId not found in localStorage. ' +
        'Ensure the test user has logged in and selected a space.',
    );
  }

  const res = await fetch(`${env.apiURL}/api/test/reset`, {
    method: 'POST',
    headers: {
      Cookie: `nori_token=${token}`,
      'X-Space-ID': spaceId,
    },
  });

  if (!res.ok) {
    const body = await res.text();
    throw new Error(`POST /api/test/reset failed (${res.status}): ${body}`);
  }
}
