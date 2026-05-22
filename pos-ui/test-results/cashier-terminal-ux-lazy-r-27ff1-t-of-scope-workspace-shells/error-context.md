# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: cashier-terminal-ux.spec.ts >> lazy routes load redesigned POS shell and out-of-scope workspace shells
- Location: e2e/cashier-terminal-ux.spec.ts:35:1

# Error details

```
Error: Run scripts/bootstrap-production-way.ps1 and pass its JSON as POS_E2E_BOOTSTRAP_JSON

expect(received).toBeTruthy()

Received: undefined
```

# Test source

```ts
  1   | import { expect, test, type Page, type TestInfo } from '@playwright/test';
  2   | 
  3   | const bootstrapJson = process.env.POS_E2E_BOOTSTRAP_JSON;
  4   | 
  5   | type DemoBootstrap = {
  6   |   manager_employee_id: string;
  7   |   manager_pin: string;
  8   | };
  9   | 
  10  | test.describe.configure({ mode: 'serial' });
  11  | 
  12  | let demo: DemoBootstrap;
  13  | 
  14  | test.beforeAll(() => {
> 15  |   expect(bootstrapJson, 'Run scripts/bootstrap-production-way.ps1 and pass its JSON as POS_E2E_BOOTSTRAP_JSON').toBeTruthy();
      |                                                                                                                 ^ Error: Run scripts/bootstrap-production-way.ps1 and pass its JSON as POS_E2E_BOOTSTRAP_JSON
  16  |   demo = JSON.parse(bootstrapJson ?? '{}') as DemoBootstrap;
  17  | });
  18  | 
  19  | test.beforeEach(async ({ page }) => {
  20  |   const runtimeErrors: string[] = [];
  21  |   page.on('pageerror', (error) => runtimeErrors.push(error.message));
  22  |   page.on('console', (message) => {
  23  |     if (message.type() !== 'error') return;
  24  |     if (/Failed to load resource: the server responded with a status of (403|404|500)/i.test(message.text())) return;
  25  |     runtimeErrors.push(message.text());
  26  |   });
  27  |   await page.exposeFunction('__posAssertNoRuntimeErrors', () => runtimeErrors);
  28  | });
  29  | 
  30  | test.afterEach(async ({ page }) => {
  31  |   const runtimeErrors = await page.evaluate(() => window.__posAssertNoRuntimeErrors());
  32  |   expect(runtimeErrors).toEqual([]);
  33  | });
  34  | 
  35  | test('lazy routes load redesigned POS shell and out-of-scope workspace shells', async ({ page }, testInfo) => {
  36  |   await loginAsManager(page);
  37  | 
  38  |   for (const path of ['/pos', '/pos/cashier']) {
  39  |     await page.goto(path);
  40  |     await expectRedesignedShell(page);
  41  |     await expect(page.locator('.q-layout')).not.toContainText(/vite|webpack|uncaught|stack trace/i);
  42  |     await saveViewportScreenshot(page, testInfo, `route-${path.replaceAll('/', '-') || 'root'}.png`);
  43  |   }
  44  | 
  45  |   await page.setViewportSize({ width: 390, height: 844 });
  46  |   await page.goto('/pos/waiter');
  47  |   await expect(page.locator('.waiter-page')).toBeVisible();
  48  |   await expect(page.getByText(/не принимает финансовые решения/i)).toBeVisible();
  49  |   await expect(page.getByRole('button', { name: /Наличные|Карта|Кассовый ящик|Вернуть оплату/i })).toHaveCount(0);
  50  | 
  51  |   for (const path of ['/pos/kitchen', '/pos/manager']) {
  52  |     await page.goto(path);
  53  |     if (path === '/pos/kitchen') {
  54  |       await expect(page.getByText('запланировано далее')).toBeVisible();
  55  |       await expect(page.getByText(/нет routes для kitchen tickets/i)).toBeVisible();
  56  |     } else {
  57  |       await expect(page.getByText('Вне текущего объема')).toBeVisible();
  58  |       await expect(page.getByText(/runtime-сценарий не реализован/i)).toBeVisible();
  59  |       await expect(page.getByRole('link', { name: /Терминал кассира/i })).toBeVisible();
  60  |     }
  61  |     await expect(page.locator('.q-layout')).not.toContainText(/готовый runtime|runtime готов|реализовано сейчас/i);
  62  |   }
  63  | });
  64  | 
  65  | test('redesigned POS shell supports section navigation and cashier flow', async ({ page }, testInfo) => {
  66  |   await page.setViewportSize({ width: 1440, height: 900 });
  67  |   await loginAsManager(page);
  68  |   await page.goto('/pos');
  69  | 
  70  |   await expectRedesignedShell(page);
  71  |   await expect(page.getByText('Production Manager')).toBeVisible();
  72  |   await expectTouchTargets(page);
  73  | 
  74  |   await page.keyboard.press('Tab');
  75  |   await expectVisibleFocusRing(page);
  76  | 
  77  |   await assertSectionMenuNavigation(page, testInfo);
  78  |   await ensureOperationsReady(page);
  79  |   await saveViewportScreenshot(page, testInfo, 'redesign-cash-ready-desktop.png');
  80  | 
  81  |   await openSection(page, 'Залы и столы');
  82  |   const hall = page.locator('.hall-chip').first();
  83  |   await expect(hall).toHaveAttribute('aria-pressed', 'true');
  84  | 
  85  |   const table = page.locator('.floor-table-tile').first();
  86  |   await table.click();
  87  |   await expect(page.locator('.floor-order-rail')).toBeVisible();
  88  |   await expect(page.locator('.floor-order-rail')).toContainText(/На выбранном столе нет активного заказа|Активные заказы/i);
  89  |   await saveViewportScreenshot(page, testInfo, 'redesign-order-empty-desktop.png');
  90  | 
  91  |   const createOrder = page.getByRole('button', { name: /Создать заказ/i }).last();
  92  |   if (await createOrder.isVisible()) {
  93  |     await expect(createOrder).toBeEnabled();
  94  |     await createOrder.click();
  95  |   }
  96  |   await expect(page.locator('.pos-menu-area')).toBeVisible();
  97  |   await expect(page.locator('.pos-order-rail')).toBeVisible();
  98  |   await expect(page.locator('.rail-summary')).toBeVisible();
  99  | 
  100 |   await cancelIssuedPrecheckIfPresent(page);
  101 |   const lineCountBeforeAdd = await page.locator('.rail-line').count();
  102 |   const menuTile = page.locator('.menu-tile:not([disabled])').filter({ hasNotText: /Есть модификаторы/i }).first();
  103 |   await expect(menuTile).toBeVisible();
  104 |   await menuTile.click();
  105 |   await expect.poll(() => page.locator('.rail-line').count()).toBeGreaterThan(lineCountBeforeAdd);
  106 |   await saveViewportScreenshot(page, testInfo, 'redesign-order-line.png');
  107 | 
  108 |   await page.getByRole('button', { name: /Выпустить пречек/i }).click();
  109 |   await expect(page.locator('.pos-order-rail')).toContainText('Пречек выпущен');
  110 | 
  111 |   await page.getByRole('button', { name: /Отмена пречека/i }).click();
  112 |   const cancelDialog = page.locator('.q-dialog').filter({ hasText: 'Отмена пречека' });
  113 |   await expect(cancelDialog).toBeVisible();
  114 |   await expect(cancelDialog.getByLabel(/PIN менеджера/i)).toHaveAttribute('type', 'password');
  115 |   await cancelDialog.getByLabel(/ID менеджера/i).fill(demo.manager_employee_id);
```