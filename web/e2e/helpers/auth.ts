import { type Page } from '@playwright/test';
import { env } from './env';

/**
 * Log in via the API and inject auth cookies + localStorage into the browser.
 *
 * In local dev, SvelteKit (:5173) and the Go backend (:8080) run on different
 * origins. The Go backend's Set-Cookie only applies to :8080, so we inject
 * the nori_token cookie on `localhost` (covers both ports) manually.
 */
export async function login(
  page: Page,
  { email, password }: { email?: string; password?: string } = {},
): Promise<void> {
  const loginEmail = email ?? env.email;
  const loginPassword = password ?? env.password;

  // Call the backend API directly
  const res = await fetch(`${env.apiURL}/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email: loginEmail, password: loginPassword }),
  });

  if (!res.ok) {
    throw new Error(`Login failed for ${loginEmail} (${res.status})`);
  }

  const data = await res.json();
  const cookieHeader = res.headers
    .getSetCookie?.()
    ?.find((c: string) => c.startsWith('nori_token='));
  const token = cookieHeader?.split(';')[0].split('=')[1] ?? data.accessToken;

  // Inject cookie on localhost (covers both :5173 and :8080)
  await page.context().addCookies([
    {
      name: 'nori_token',
      value: token,
      domain: 'localhost',
      path: '/',
      httpOnly: true,
      secure: false,
      sameSite: 'Lax',
      expires: Math.floor(Date.now() / 1000) + 30 * 24 * 60 * 60,
    },
  ]);

  // Set localStorage values the client-side auth store expects
  await page.goto(env.baseURL);
  await page.evaluate(
    ({ accessToken, activeSpaceId }) => {
      localStorage.setItem('accessToken', accessToken);
      if (activeSpaceId) {
        localStorage.setItem('activeSpaceId', activeSpaceId);
      }
    },
    {
      accessToken: data.accessToken,
      activeSpaceId: data.activeSpaceId ?? '',
    },
  );

  // Navigate to home — should now be authenticated
  await page.goto('/');
  await page.waitForURL((url) => !url.pathname.startsWith('/login'), {
    timeout: 10_000,
  });
}
