/**
 * Test environment configuration.
 *
 * The E2E test user is seeded by the Go server on startup when
 * E2E_ACCOUNT_ENABLED=true. No admin credentials are needed here.
 *
 *   NORI_TEST_EMAIL      — e2e test user email    (default: e2e-test@nori.dev)
 *   NORI_TEST_PASSWORD   — e2e test user password  (default: TestPass123!)
 *   NORI_TEST_BASE_URL   — SvelteKit dev server    (default: http://localhost:5173)
 *   NORI_TEST_API_URL    — Go backend API           (default: http://localhost:8081)
 */
export const env = {
  /** Email for the dedicated e2e test user (seeded by server). */
  email: process.env.NORI_TEST_EMAIL ?? 'e2e-test@nori.dev',
  /** Password for the e2e test user. */
  password: process.env.NORI_TEST_PASSWORD ?? 'TestPass123!',

  /** Base URL of the SvelteKit dev server. */
  baseURL: process.env.NORI_TEST_BASE_URL ?? 'http://localhost:5173',
  /** Base URL of the Go backend API. */
  apiURL: process.env.NORI_TEST_API_URL ?? 'http://localhost:8081',
};
