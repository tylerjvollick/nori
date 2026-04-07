import { defineConfig, devices } from '@playwright/test';
import dotenv from 'dotenv';

// Load .env.test if it exists — provides admin creds for globalSetup
dotenv.config({ path: '.env.test' });

export default defineConfig({
  testDir: './e2e',
  globalSetup: './e2e/global-setup.ts',
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 1 : 0,
  workers: process.env.CI ? 1 : undefined,
  reporter: 'html',
  timeout: 15_000,
  expect: { timeout: 5_000 },

  use: {
    baseURL: 'http://localhost:5173',
    trace: 'on-first-retry',
    // Use auth state from globalSetup by default (tests that need
    // unauthenticated access override this with storageState: undefined).
    storageState: './e2e/.auth/user.json',
  },

  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],
});
