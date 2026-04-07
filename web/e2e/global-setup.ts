/**
 * Playwright global setup — runs once before all tests.
 *
 * 1. Verifies the backend is reachable.
 * 2. Logs in as the admin user.
 * 3. Creates a dedicated e2e test user (idempotent — skips if already exists).
 * 4. Logs in as the test user via the API, injects the auth cookie into a
 *    browser context for both :5173 and :8080, and saves the storage state
 *    to `e2e/.auth/user.json` for reuse by tests.
 *
 * NOTE: In local dev, SvelteKit (:5173) and the Go backend (:8080) run on
 * different origins. The Go backend sets the nori_token cookie on :8080,
 * but SvelteKit's server-side hooks need it on :5173. This setup injects
 * the cookie on both origins so SSR auth works correctly.
 */
import { chromium, type FullConfig } from '@playwright/test';
import { env } from './helpers/env';
import * as fs from 'node:fs';
import * as path from 'node:path';
import { fileURLToPath } from 'node:url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const AUTH_DIR = path.join(__dirname, '.auth');
export const AUTH_FILE = path.join(AUTH_DIR, 'user.json');

async function globalSetup(_config: FullConfig) {
  fs.mkdirSync(AUTH_DIR, { recursive: true });

  // ── 1. Verify backend health ───────────────────────────────────────
  const healthRes = await fetch(`${env.apiURL}/auth/me`).catch(() => null);
  if (!healthRes) {
    throw new Error(
      `Backend not reachable at ${env.apiURL}. ` +
        `Start the Go server before running e2e tests.`,
    );
  }

  // ── 2. Log in as admin ─────────────────────────────────────────────
  const adminLoginRes = await fetch(`${env.apiURL}/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      email: env.adminEmail,
      password: env.adminPassword,
    }),
  });

  if (!adminLoginRes.ok) {
    throw new Error(
      `Admin login failed (${adminLoginRes.status}). ` +
        `Set NORI_ADMIN_EMAIL and NORI_ADMIN_PASSWORD env vars (or .env.test) ` +
        `to match your docker/.env.`,
    );
  }

  const adminCookieHeader = adminLoginRes.headers
    .getSetCookie?.()
    ?.find((c: string) => c.startsWith('nori_token='));
  if (!adminCookieHeader) {
    throw new Error(
      'Admin login succeeded but no nori_token cookie was returned.',
    );
  }
  const adminToken = adminCookieHeader.split(';')[0].split('=')[1];

  // ── 3. Create test user (idempotent) ───────────────────────────────
  const tempPassword = 'TempE2E_' + Date.now();

  const createRes = await fetch(`${env.apiURL}/admin/users`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Cookie: `nori_token=${adminToken}`,
    },
    body: JSON.stringify({
      email: env.email,
      firstName: 'E2E',
      lastName: 'Test',
      tempPassword,
      role: 'admin', // admin so tests can access /admin routes
    }),
  });

  const createBody = await createRes.text();
  const userAlreadyExists =
    !createRes.ok && createBody.toLowerCase().includes('already exists');

  if (createRes.ok) {
    // New user — log in with temp password and change it.
    const testLoginRes = await fetch(`${env.apiURL}/auth/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email: env.email, password: tempPassword }),
    });

    if (!testLoginRes.ok) {
      throw new Error(
        `Test user login failed after creation (${testLoginRes.status}).`,
      );
    }

    const testCookieHeader = testLoginRes.headers
      .getSetCookie?.()
      ?.find((c: string) => c.startsWith('nori_token='));
    const testToken = testCookieHeader!.split(';')[0].split('=')[1];

    const changePwRes = await fetch(`${env.apiURL}/auth/change-password`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Cookie: `nori_token=${testToken}`,
      },
      body: JSON.stringify({
        currentPassword: tempPassword,
        newPassword: env.password,
      }),
    });

    if (!changePwRes.ok) {
      throw new Error(
        `Failed to change test user password (${changePwRes.status}).`,
      );
    }
  } else if (!userAlreadyExists) {
    throw new Error(
      `Failed to create test user (${createRes.status}): ${createBody}`,
    );
  }

  // ── 4. Login as test user and save storage state ───────────────────
  const loginRes = await fetch(`${env.apiURL}/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email: env.email, password: env.password }),
  });

  if (!loginRes.ok) {
    throw new Error(
      `Test user login failed (${loginRes.status}). ` +
        `If the user was created in a previous run, ensure NORI_TEST_PASSWORD ` +
        `matches what was set then (default: TestPass123!).`,
    );
  }

  const loginData = await loginRes.json();
  const cookieHeader = loginRes.headers
    .getSetCookie?.()
    ?.find((c: string) => c.startsWith('nori_token='));
  const token = cookieHeader!.split(';')[0].split('=')[1];

  // Create a browser context with the auth cookie injected on BOTH origins
  // (Go backend :8080 and SvelteKit :5173) so SSR auth checks pass.
  const browser = await chromium.launch();
  const context = await browser.newContext();

  const cookieBase = {
    name: 'nori_token',
    value: token,
    path: '/',
    httpOnly: true,
    secure: false,
    sameSite: 'Lax' as const,
    expires: Math.floor(Date.now() / 1000) + 30 * 24 * 60 * 60, // 30 days
  };

  await context.addCookies([
    { ...cookieBase, domain: 'localhost' },
  ]);

  // Also set localStorage values that the client-side auth store expects
  const page = await context.newPage();
  await page.goto(env.baseURL);
  await page.evaluate(
    ({ accessToken, activeSpaceId }) => {
      localStorage.setItem('accessToken', accessToken);
      if (activeSpaceId) {
        localStorage.setItem('activeSpaceId', activeSpaceId);
      }
    },
    {
      accessToken: loginData.accessToken,
      activeSpaceId: loginData.activeSpaceId ?? '',
    },
  );

  await context.storageState({ path: AUTH_FILE });
  await browser.close();
}

export default globalSetup;
