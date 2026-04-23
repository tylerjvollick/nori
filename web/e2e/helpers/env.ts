/**
 * Test environment configuration.
 *
 * Reads credentials from env vars. Falls back to defaults that work with
 * the standard `nori init` setup:
 *
 *   NORI_ADMIN_EMAIL     — admin account email  (default: from docker/.env)
 *   NORI_ADMIN_PASSWORD  — admin account password
 *   NORI_TEST_EMAIL      — dedicated e2e test user (default: e2e-test@nori.dev)
 *   NORI_TEST_PASSWORD   — password for the test user  (default: TestPass123!)
 *   NORI_TEST_BASE_URL   — SvelteKit dev server  (default: http://localhost:5173)
 *   NORI_TEST_API_URL    — Go backend API         (default: http://localhost:8081)
 */
export const env = {
  /** Admin email — needed to bootstrap the test user. */
  adminEmail: process.env.NORI_ADMIN_EMAIL ?? 'admin@nori.dev',
  /** Admin password. */
  adminPassword: process.env.NORI_ADMIN_PASSWORD ?? 'admin',

  /** Email for the dedicated e2e test user. */
  email: process.env.NORI_TEST_EMAIL ?? 'e2e-test@nori.dev',
  /** Password for the e2e test user. */
  password: process.env.NORI_TEST_PASSWORD ?? 'TestPass123!',

  /** Base URL of the SvelteKit dev server. */
  baseURL: process.env.NORI_TEST_BASE_URL ?? 'http://localhost:5173',
  /** Base URL of the Go backend API. */
  apiURL: process.env.NORI_TEST_API_URL ?? 'http://localhost:8081',
};
