/**
 * Playwright global setup — runs once before all tests.
 *
 * Flow:
 *   1. Verify the backend is reachable.
 *   2. Log in as the E2E test user (seeded by server on startup).
 *   3. Inject the auth cookie on localhost and save storage state.
 *
 * The E2E test user is created by the Go server when E2E_ACCOUNT_ENABLED=true.
 * It has its own isolated account, space, and stations — completely separate
 * from the admin account.
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

  // ── 2. Log in as the seeded E2E test user ──────────────────────────
  const loginRes = await fetch(`${env.apiURL}/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email: env.email, password: env.password }),
  });

  if (!loginRes.ok) {
    throw new Error(
      `E2E test user login failed (${loginRes.status}). ` +
        `Ensure E2E_ACCOUNT_ENABLED=true is set in docker/.env ` +
        `and the server has been restarted.`,
    );
  }

  // ── 3. Save storage state ──────────────────────────────────────────
  const loginData = await loginRes.json();
  const token = extractToken(loginRes);
  if (!token) {
    throw new Error('E2E test user login succeeded but no nori_token cookie was returned.');
  }

  // Resolve activeSpaceId: use login response, or fall back to /auth/me
  let activeSpaceId: string = loginData.activeSpaceId ?? '';
  if (!activeSpaceId) {
    const meRes = await fetch(`${env.apiURL}/auth/me`, {
      headers: { Cookie: `nori_token=${token}` },
    });
    if (meRes.ok) {
      const meData = await meRes.json();
      if (meData.activeSpaceId) {
        activeSpaceId = meData.activeSpaceId;
      } else if (meData.accessibleSpaces?.length > 0) {
        activeSpaceId = meData.accessibleSpaces[0].id;
      }
    }
  }

  if (!activeSpaceId) {
    throw new Error(
      'Could not determine activeSpaceId from login or /auth/me. ' +
        'Ensure E2E_ACCOUNT_ENABLED=true so the server seeds a space for the test user.',
    );
  }

  // Record a space visit so the test user has at least one recent space.
  await fetch(`${env.apiURL}/api/spaces/${activeSpaceId}/visit`, {
    method: 'POST',
    headers: { Cookie: `nori_token=${token}` },
  });

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
      localStorage.setItem('activeSpaceId', activeSpaceId);
    },
    {
      accessToken: loginData.accessToken,
      activeSpaceId,
    },
  );

  await context.storageState({ path: AUTH_FILE });
  await browser.close();
}

export default globalSetup;
