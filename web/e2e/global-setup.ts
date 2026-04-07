/**
 * Playwright global setup — runs once before all tests.
 *
 * Flow:
 *   1. Verify the backend is reachable.
 *   2. Try to log in as the dedicated test user (e2e-test@nori.dev).
 *   3. If that fails, bootstrap: log in as admin, create the test user,
 *      change its password, then log in again.
 *   4. Inject the auth cookie on localhost and save storage state.
 *
 * Admin credentials (NORI_ADMIN_EMAIL / NORI_ADMIN_PASSWORD) are only
 * used on the very first run or after a database wipe.
 *
 * NOTE: In local dev, SvelteKit (:5173) and the Go backend (:8080) run on
 * different origins. The Go backend sets the nori_token cookie on :8080,
 * but SvelteKit's server-side hooks need it on :5173. This setup injects
 * the cookie on localhost (covers both ports) so SSR auth works correctly.
 */
import { chromium, type FullConfig } from '@playwright/test';
import { env } from './helpers/env';
import * as fs from 'node:fs';
import * as path from 'node:path';
import { fileURLToPath } from 'node:url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const AUTH_DIR = path.join(__dirname, '.auth');
export const AUTH_FILE = path.join(AUTH_DIR, 'user.json');

/** Extract the nori_token value from a fetch Response's Set-Cookie headers. */
function extractToken(res: Response): string | null {
  const header = res.headers
    .getSetCookie?.()
    ?.find((c: string) => c.startsWith('nori_token='));
  return header ? header.split(';')[0].split('=')[1] : null;
}

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

  // ── 2. Try logging in as the test user directly ────────────────────
  let loginRes = await fetch(`${env.apiURL}/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email: env.email, password: env.password }),
  });

  // ── 3. Bootstrap if test user doesn't exist yet ────────────────────
  if (!loginRes.ok) {
    // Log in as admin
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
        `Test user login failed and admin login also failed (${adminLoginRes.status}). ` +
          `Set NORI_ADMIN_EMAIL and NORI_ADMIN_PASSWORD env vars (or .env.test) ` +
          `to match your docker/.env.`,
      );
    }

    const adminToken = extractToken(adminLoginRes);
    if (!adminToken) {
      throw new Error(
        'Admin login succeeded but no nori_token cookie was returned.',
      );
    }

    // Create the test user
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
      const tempLoginRes = await fetch(`${env.apiURL}/auth/login`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email: env.email, password: tempPassword }),
      });

      if (!tempLoginRes.ok) {
        throw new Error(
          `Test user login failed after creation (${tempLoginRes.status}).`,
        );
      }

      const tempToken = extractToken(tempLoginRes);

      const changePwRes = await fetch(`${env.apiURL}/auth/change-password`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Cookie: `nori_token=${tempToken}`,
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

    // Log in as the test user with the permanent password
    loginRes = await fetch(`${env.apiURL}/auth/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email: env.email, password: env.password }),
    });

    if (!loginRes.ok) {
      throw new Error(
        `Test user login failed after bootstrap (${loginRes.status}). ` +
          `If the user existed from a previous run with a different password, ` +
          `delete it via the admin API and re-run.`,
      );
    }
  }

  // ── 4. Save storage state ──────────────────────────────────────────
  const loginData = await loginRes.json();
  const token = extractToken(loginRes);
  if (!token) {
    throw new Error('Test user login succeeded but no nori_token cookie was returned.');
  }

  const browser = await chromium.launch();
  const context = await browser.newContext();

  await context.addCookies([
    {
      name: 'nori_token',
      value: token,
      domain: 'localhost',
      path: '/',
      httpOnly: true,
      secure: false,
      sameSite: 'Lax',
      expires: Math.floor(Date.now() / 1000) + 30 * 24 * 60 * 60, // 30 days
    },
  ]);

  // Set localStorage values the client-side auth store expects
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
