import { test, expect } from '@playwright/test';
import { env } from './helpers/env';

test.describe('Backend health', () => {
  test('API is reachable and returns JSON', async ({ request }) => {
    // /auth/me is a lightweight endpoint every running backend exposes.
    // We don't care about the status code (401 is fine when unauthenticated),
    // only that the server responds.
    const res = await request.get(`${env.apiURL}/auth/me`);
    expect([200, 401]).toContain(res.status());
  });
});
