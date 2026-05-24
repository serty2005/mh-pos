# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: waiter-mobile-flow.spec.ts >> waiter mobile can select a table, create an order, add a line and issue precheck
- Location: e2e/waiter-mobile-flow.spec.ts:29:1

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
  18 |     if (/Failed to load resource: the server responded with a status of (403|404|500)/i.test(message.text())) return;
  19 |     runtimeErrors.push(message.text());
  20 |   });
  21 |   await page.exposeFunction('__posAssertNoRuntimeErrors', () => runtimeErrors);
  22 | });
  23 | 
  24 | test.afterEach(async ({ page }) => {
  25 |   const runtimeErrors = await page.evaluate(() => window.__posAssertNoRuntimeErrors());
  26 |   expect(runtimeErrors).toEqual([]);
  27 | });
  28 | 
  29 | test('waiter mobile can select a table, create an order, add a line and issue precheck', async ({ page }) => {
  30 |   await page.setViewportSize({ width: 390, height: 844 });
  31 |   await loginAsManager(page);
  32 |   await page.goto('/pos/waiter');
  33 | 
  34 |   await expect(page.getByRole('heading', { name: /Столы, заказ и пречек/i })).toBeVisible();
  35 |   await expect(page.getByText(/не принимает финансовые решения/i)).toBeVisible();
  36 |   await expect(page.getByRole('button', { name: /Наличные|Карта|Кассовый ящик|Вернуть оплату/i })).toHaveCount(0);
  37 | 
  38 |   const freeTable = page.locator('.waiter-table-card:not(.occupied)').first();
  39 |   await expect(freeTable).toBeVisible();
  40 |   await freeTable.click();
  41 | 
  42 |   const createOrder = page.getByRole('button', { name: /Создать заказ/i });
  43 |   if (await createOrder.isEnabled()) {
  44 |     await createOrder.click();
  45 |   }
  46 | 
  47 |   const menuItem = page.locator('.waiter-menu-item:not([disabled])').first();
  48 |   await expect(menuItem).toBeVisible();
  49 |   await menuItem.click();
  50 | 
  51 |   const modifierDialog = page.locator('.q-dialog').filter({ hasText: 'Модификаторы' });
  52 |   if (await modifierDialog.isVisible().catch(() => false)) {
  53 |     const firstModifierAdd = modifierDialog.getByRole('button', { name: /Добавить/i }).first();
  54 |     if (await firstModifierAdd.isVisible()) await firstModifierAdd.click();
  55 |     await modifierDialog.locator('.dialog-actions').getByRole('button', { name: /^Добавить$/i }).click();
  56 |   }
  57 | 
  58 |   await expect(page.locator('.waiter-line-row').first()).toBeVisible();
  59 |   await page.getByRole('button', { name: /Выпустить пречек/i }).click();
  60 |   await expect(page.getByText(/Пречек выпущен|Заказ заблокирован активным пречеком/i)).toBeVisible();
  61 |   await expect(page.locator('.waiter-menu-item:not([disabled])')).toHaveCount(0);
  62 |   await expect(page.getByRole('button', { name: /Наличные|Карта|Кассовый ящик|Вернуть оплату/i })).toHaveCount(0);
  63 | });
  64 | 
  65 | test('waiter mobile keeps payment, refund and cash drawer controls out of the default surface', async ({ page }) => {
  66 |   await page.setViewportSize({ width: 390, height: 844 });
  67 |   await loginAsManager(page);
  68 |   await page.goto('/pos/waiter');
  69 | 
  70 |   await expect(page.locator('.waiter-page')).toBeVisible();
  71 |   await expect(page.getByText(/не принимает финансовые решения/i)).toBeVisible();
  72 |   await expect(page.getByRole('button', { name: /Наличные|Карта|Вернуть оплату|Кассовый ящик/i })).toHaveCount(0);
  73 |   await expect(page.locator('.waiter-page')).not.toContainText(/refund|payment refund|cash drawer/i);
  74 | });
  75 | 
  76 | async function loginAsManager(page: Page) {
  77 |   await page.goto('/login');
  78 |   await expect(page.getByRole('heading', { name: /Вход по PIN/i })).toBeVisible();
  79 |   await page.getByLabel(/^PIN$/i).fill('2222');
  80 |   await page.getByRole('button', { name: /Войти/i }).click();
  81 |   await expect(page.locator('.pos-bottom-bar, .waiter-page')).toBeVisible();
  82 | }
  83 | 
```