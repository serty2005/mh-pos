# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: cashier-terminal-ux.spec.ts >> lazy routes load redesigned POS shell and out-of-scope workspace shells
- Location: e2e/cashier-terminal-ux.spec.ts:37:1

# Error details

```
Error: Run stack bootstrap and set POS_E2E_BOOTSTRAP_JSON to JSON content or /workspace/myhoreca-pos/.e2e/bootstrap.json

expect(received).toBeTruthy()

Received: ""
```

# Test source

```ts
  1   | import { expect, test, type Page, type TestInfo } from '@playwright/test';
  2   | 
  3   | import { bootstrapRequiredMessage, loadBootstrapJson } from './support/bootstrap';
  4   | 
  5   | const bootstrapJson = loadBootstrapJson();
  6   | 
  7   | type DemoBootstrap = {
  8   |   manager_employee_id: string;
  9   |   manager_pin: string;
  10  | };
  11  | 
  12  | test.describe.configure({ mode: 'serial' });
  13  | 
  14  | let demo: DemoBootstrap;
  15  | 
  16  | test.beforeAll(() => {
> 17  |   expect(bootstrapJson, bootstrapRequiredMessage()).toBeTruthy();
      |                                                     ^ Error: Run stack bootstrap and set POS_E2E_BOOTSTRAP_JSON to JSON content or /workspace/myhoreca-pos/.e2e/bootstrap.json
  18  |   demo = JSON.parse(bootstrapJson ?? '{}') as DemoBootstrap;
  19  | });
  20  | 
  21  | test.beforeEach(async ({ page }) => {
  22  |   const runtimeErrors: string[] = [];
  23  |   page.on('pageerror', (error) => runtimeErrors.push(error.message));
  24  |   page.on('console', (message) => {
  25  |     if (message.type() !== 'error') return;
  26  |     if (/Failed to load resource: the server responded with a status of (403|404|500)/i.test(message.text())) return;
  27  |     runtimeErrors.push(message.text());
  28  |   });
  29  |   await page.exposeFunction('__posAssertNoRuntimeErrors', () => runtimeErrors);
  30  | });
  31  | 
  32  | test.afterEach(async ({ page }) => {
  33  |   const runtimeErrors = await page.evaluate(() => window.__posAssertNoRuntimeErrors());
  34  |   expect(runtimeErrors).toEqual([]);
  35  | });
  36  | 
  37  | test('lazy routes load redesigned POS shell and out-of-scope workspace shells', async ({ page }, testInfo) => {
  38  |   await loginAsManager(page);
  39  | 
  40  |   for (const path of ['/pos', '/pos/cashier']) {
  41  |     await page.goto(path);
  42  |     await expectRedesignedShell(page);
  43  |     await expect(page.locator('.q-layout')).not.toContainText(/vite|webpack|uncaught|stack trace/i);
  44  |     await saveViewportScreenshot(page, testInfo, `route-${path.replaceAll('/', '-') || 'root'}.png`);
  45  |   }
  46  | 
  47  |   await page.setViewportSize({ width: 390, height: 844 });
  48  |   await page.goto('/pos/waiter');
  49  |   await expect(page.locator('.waiter-page')).toBeVisible();
  50  |   await expect(page.getByText(/не принимает финансовые решения/i)).toBeVisible();
  51  |   await expect(page.getByRole('button', { name: /Наличные|Карта|Кассовый ящик|Вернуть оплату/i })).toHaveCount(0);
  52  | 
  53  |   for (const path of ['/pos/kitchen', '/pos/manager']) {
  54  |     await page.goto(path);
  55  |     if (path === '/pos/kitchen') {
  56  |       await expect(page.getByText('запланировано далее')).toBeVisible();
  57  |       await expect(page.getByText(/нет routes для kitchen tickets/i)).toBeVisible();
  58  |     } else {
  59  |       await expect(page.getByText('Вне текущего объема')).toBeVisible();
  60  |       await expect(page.getByText(/runtime-сценарий не реализован/i)).toBeVisible();
  61  |       await expect(page.getByRole('link', { name: /Терминал кассира/i })).toBeVisible();
  62  |     }
  63  |     await expect(page.locator('.q-layout')).not.toContainText(/готовый runtime|runtime готов|реализовано сейчас/i);
  64  |   }
  65  | });
  66  | 
  67  | test('redesigned POS shell supports section navigation and cashier flow', async ({ page }, testInfo) => {
  68  |   await page.setViewportSize({ width: 1440, height: 900 });
  69  |   await loginAsManager(page);
  70  |   await page.goto('/pos');
  71  | 
  72  |   await expectRedesignedShell(page);
  73  |   await expect(page.getByText('Production Manager')).toBeVisible();
  74  |   await expectTouchTargets(page);
  75  | 
  76  |   await page.keyboard.press('Tab');
  77  |   await expectVisibleFocusRing(page);
  78  | 
  79  |   await assertSectionMenuNavigation(page, testInfo);
  80  |   await ensureOperationsReady(page);
  81  |   await saveViewportScreenshot(page, testInfo, 'redesign-cash-ready-desktop.png');
  82  | 
  83  |   await openSection(page, 'Залы и столы');
  84  |   const hall = page.locator('.hall-chip').first();
  85  |   await expect(hall).toHaveAttribute('aria-pressed', 'true');
  86  | 
  87  |   const table = page.locator('.floor-table-tile').first();
  88  |   await table.click();
  89  |   await expect(page.locator('.floor-order-rail')).toBeVisible();
  90  |   await expect(page.locator('.floor-order-rail')).toContainText(/На выбранном столе нет активного заказа|Активные заказы/i);
  91  |   await saveViewportScreenshot(page, testInfo, 'redesign-order-empty-desktop.png');
  92  | 
  93  |   const createOrder = page.getByRole('button', { name: /Создать заказ/i }).last();
  94  |   if (await createOrder.isVisible()) {
  95  |     await expect(createOrder).toBeEnabled();
  96  |     await createOrder.click();
  97  |   }
  98  |   await expect(page.locator('.pos-menu-area')).toBeVisible();
  99  |   await expect(page.locator('.pos-order-rail')).toBeVisible();
  100 |   await expect(page.locator('.rail-summary')).toBeVisible();
  101 | 
  102 |   await cancelIssuedPrecheckIfPresent(page);
  103 |   const lineCountBeforeAdd = await page.locator('.rail-line').count();
  104 |   const menuTile = page.locator('.menu-tile:not([disabled])').filter({ hasNotText: /Есть модификаторы/i }).first();
  105 |   await expect(menuTile).toBeVisible();
  106 |   await menuTile.click();
  107 |   await expect.poll(() => page.locator('.rail-line').count()).toBeGreaterThan(lineCountBeforeAdd);
  108 |   await saveViewportScreenshot(page, testInfo, 'redesign-order-line.png');
  109 | 
  110 |   await page.getByRole('button', { name: /Выпустить пречек/i }).click();
  111 |   await expect(page.locator('.pos-order-rail')).toContainText('Пречек выпущен');
  112 | 
  113 |   await page.getByRole('button', { name: /Отмена пречека/i }).click();
  114 |   const cancelDialog = page.locator('.q-dialog').filter({ hasText: 'Отмена пречека' });
  115 |   await expect(cancelDialog).toBeVisible();
  116 |   await expect(cancelDialog.getByLabel(/PIN менеджера/i)).toHaveAttribute('type', 'password');
  117 |   await cancelDialog.getByLabel(/ID менеджера/i).fill(demo.manager_employee_id);
```