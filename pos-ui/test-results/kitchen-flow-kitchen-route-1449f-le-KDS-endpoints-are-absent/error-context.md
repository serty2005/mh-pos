# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: kitchen-flow.spec.ts >> kitchen route is an honest readiness screen while KDS endpoints are absent
- Location: e2e/kitchen-flow.spec.ts:28:1

# Error details

```
Error: Run stack bootstrap and set POS_E2E_BOOTSTRAP_JSON to JSON content or /workspace/myhoreca-pos/.e2e/bootstrap.json

expect(received).toBeTruthy()

Received: ""
```

# Test source

```ts
  1  | import { expect, test, type Page } from '@playwright/test';
  2  | 
  3  | import { bootstrapRequiredMessage, loadBootstrapJson } from './support/bootstrap';
  4  | 
  5  | const bootstrapJson = loadBootstrapJson();
  6  | 
  7  | test.describe.configure({ mode: 'serial' });
  8  | 
  9  | test.beforeAll(() => {
> 10 |   expect(bootstrapJson, bootstrapRequiredMessage()).toBeTruthy();
     |                                                     ^ Error: Run stack bootstrap and set POS_E2E_BOOTSTRAP_JSON to JSON content or /workspace/myhoreca-pos/.e2e/bootstrap.json
  11 | });
  12 | 
  13 | test.beforeEach(async ({ page }) => {
  14 |   const runtimeErrors: string[] = [];
  15 |   page.on('pageerror', (error) => runtimeErrors.push(error.message));
  16 |   page.on('console', (message) => {
  17 |     if (message.type() !== 'error') return;
  18 |     runtimeErrors.push(message.text());
  19 |   });
  20 |   await page.exposeFunction('__posAssertNoRuntimeErrors', () => runtimeErrors);
  21 | });
  22 | 
  23 | test.afterEach(async ({ page }) => {
  24 |   const runtimeErrors = await page.evaluate(() => window.__posAssertNoRuntimeErrors());
  25 |   expect(runtimeErrors).toEqual([]);
  26 | });
  27 | 
  28 | test('kitchen route is an honest readiness screen while KDS endpoints are absent', async ({ page }) => {
  29 |   await loginAsManager(page);
  30 |   await page.goto('/pos/kitchen');
  31 | 
  32 |   await expect(page.getByRole('heading', { name: /Кухонный экран/i })).toBeVisible();
  33 |   await expect(page.getByText('запланировано далее')).toBeVisible();
  34 |   await expect(page.getByText(/нет routes для kitchen tickets/i)).toBeVisible();
  35 |   await expect(page.getByText('Только readiness')).toBeVisible();
  36 |   await expect(page.getByText('Lifecycle-команды отключены')).toBeVisible();
  37 |   await expect(page.getByText(/GET для kitchen tickets/i)).toBeVisible();
  38 |   await expect(page.getByText(/new/)).toBeVisible();
  39 |   await expect(page.getByText(/accepted/)).toBeVisible();
  40 |   await expect(page.getByText(/ready/)).toBeVisible();
  41 |   await expect(page.getByRole('button', { name: /accepted|in_progress|ready|served|recall|cancelled/i })).toHaveCount(0);
  42 |   await expect(page.locator('.kitchen-page')).not.toContainText(/готовый runtime|реализовано сейчас/i);
  43 | });
  44 | 
  45 | async function loginAsManager(page: Page) {
  46 |   await page.goto('/login');
  47 |   await expect(page.getByRole('heading', { name: /Вход по PIN/i })).toBeVisible();
  48 |   await page.getByLabel(/^PIN$/i).fill('2222');
  49 |   await page.getByRole('button', { name: /Войти/i }).click();
  50 |   await expect(page.locator('.pos-bottom-bar')).toBeVisible();
  51 | }
  52 | 
```