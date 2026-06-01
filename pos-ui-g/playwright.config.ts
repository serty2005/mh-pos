import { defineConfig, devices } from '@playwright/test';
import os from 'node:os';
import path from 'node:path';

process.env.NO_PROXY = [
  process.env.NO_PROXY,
  '127.0.0.1',
  'localhost',
  '::1',
].filter(Boolean).join(',');

const port = Number(process.env.POS_UI_G_E2E_PORT ?? 3000);
const host = process.env.POS_UI_G_E2E_HOST ?? '127.0.0.1';
const baseURL = process.env.POS_UI_G_E2E_BASE_URL ?? process.env.POS_UI_URL ?? `http://${host}:${port}`;

process.env.POS_UI_URL ??= baseURL;

export default defineConfig({
  testDir: '.',
  testMatch: ['e2e/**/*.spec.ts', 'tests/**/*.e2e.ts'],
  outputDir: process.env.POS_UI_G_E2E_OUTPUT_DIR ?? path.join(os.tmpdir(), 'mh-pos-ui-g-playwright'),
  fullyParallel: false,
  workers: 1,
  timeout: 60_000,
  expect: {
    timeout: 5_000,
  },
  reporter: [['list']],
  use: {
    baseURL,
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
  },
  webServer: {
    command: `npx vite --host=${host} --port=${port} --strictPort`,
    url: baseURL,
    reuseExistingServer: true,
    timeout: 60_000,
    env: {
      DISABLE_HMR: 'true',
      VITE_POS_API_BASE: process.env.POS_UI_G_E2E_API_BASE ?? 'http://localhost:8080/api/v1',
    },
  },
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'], viewport: { width: 1440, height: 900 } },
    },
  ],
});
