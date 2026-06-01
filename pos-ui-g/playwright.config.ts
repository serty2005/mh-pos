import { defineConfig } from '@playwright/test';

export default defineConfig({
  testDir: './tests',
  testMatch: '**/*.e2e.ts',
  timeout: 60_000,
  use: {
    baseURL: process.env.POS_UI_URL ?? 'http://localhost:3000',
    trace: 'retain-on-failure',
  },
});
