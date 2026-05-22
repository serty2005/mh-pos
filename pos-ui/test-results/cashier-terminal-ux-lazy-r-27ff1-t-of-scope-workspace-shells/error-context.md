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
  45  |   for (const path of ['/pos/waiter', '/pos/kitchen', '/pos/manager']) {
  46  |     await page.goto(path);
  47  |     await expect(page.getByText('Вне текущего объема')).toBeVisible();
  48  |     await expect(page.getByText(/runtime-сценарий не реализован/i)).toBeVisible();
  49  |     await expect(page.getByRole('link', { name: /Терминал кассира/i })).toBeVisible();
  50  |     await expect(page.locator('.q-layout')).not.toContainText(/готовый runtime|runtime готов|реализовано сейчас/i);
  51  |   }
  52  | });
  53  | 
  54  | test('redesigned POS shell supports section navigation and cashier flow', async ({ page }, testInfo) => {
  55  |   await page.setViewportSize({ width: 1440, height: 900 });
  56  |   await loginAsManager(page);
  57  |   await page.goto('/pos');
  58  | 
  59  |   await expectRedesignedShell(page);
  60  |   await expect(page.getByText('Production Manager')).toBeVisible();
  61  |   await expectTouchTargets(page);
  62  | 
  63  |   await page.keyboard.press('Tab');
  64  |   await expectVisibleFocusRing(page);
  65  | 
  66  |   await assertSectionMenuNavigation(page, testInfo);
  67  |   await ensureOperationsReady(page);
  68  |   await saveViewportScreenshot(page, testInfo, 'redesign-cash-ready-desktop.png');
  69  | 
  70  |   await openSection(page, 'Залы и столы');
  71  |   const hall = page.locator('.hall-chip').first();
  72  |   await expect(hall).toHaveAttribute('aria-pressed', 'true');
  73  | 
  74  |   const table = page.locator('.floor-table-tile').first();
  75  |   await table.click();
  76  |   await expect(page.locator('.floor-order-rail')).toBeVisible();
  77  |   await expect(page.locator('.floor-order-rail')).toContainText(/На выбранном столе нет активного заказа|Активные заказы/i);
  78  |   await saveViewportScreenshot(page, testInfo, 'redesign-order-empty-desktop.png');
  79  | 
  80  |   const createOrder = page.getByRole('button', { name: /Создать заказ/i }).last();
  81  |   if (await createOrder.isVisible()) {
  82  |     await expect(createOrder).toBeEnabled();
  83  |     await createOrder.click();
  84  |   }
  85  |   await expect(page.locator('.pos-menu-area')).toBeVisible();
  86  |   await expect(page.locator('.pos-order-rail')).toBeVisible();
  87  |   await expect(page.locator('.rail-summary')).toBeVisible();
  88  | 
  89  |   await cancelIssuedPrecheckIfPresent(page);
  90  |   const lineCountBeforeAdd = await page.locator('.rail-line').count();
  91  |   const menuTile = page.locator('.menu-tile:not([disabled])').filter({ hasNotText: /Есть модификаторы/i }).first();
  92  |   await expect(menuTile).toBeVisible();
  93  |   await menuTile.click();
  94  |   await expect.poll(() => page.locator('.rail-line').count()).toBeGreaterThan(lineCountBeforeAdd);
  95  |   await saveViewportScreenshot(page, testInfo, 'redesign-order-line.png');
  96  | 
  97  |   await page.getByRole('button', { name: /Выпустить пречек/i }).click();
  98  |   await expect(page.locator('.pos-order-rail')).toContainText('Пречек выпущен');
  99  | 
  100 |   await page.getByRole('button', { name: /Отмена пречека/i }).click();
  101 |   const cancelDialog = page.locator('.q-dialog').filter({ hasText: 'Отмена пречека' });
  102 |   await expect(cancelDialog).toBeVisible();
  103 |   await expect(cancelDialog.getByLabel(/PIN менеджера/i)).toHaveAttribute('type', 'password');
  104 |   await cancelDialog.getByLabel(/ID менеджера/i).fill(demo.manager_employee_id);
  105 |   await cancelDialog.getByLabel(/PIN менеджера/i).fill('0000');
  106 |   await cancelDialog.getByLabel(/Причина отмены/i).fill('playwright invalid manager override');
  107 |   await cancelDialog.getByRole('button', { name: /Отмена пречека/i }).click();
  108 |   await expect(page.getByRole('heading', { name: /Операция не выполнена|Недостаточно прав|Проверьте данные/i })).toBeVisible();
  109 |   await expectNoSensitiveText(page, ['0000', 'manager_pin', 'managerPin', 'sql:', 'stack trace', 'panic:']);
  110 |   await page.getByRole('button', { name: /Закрыть/i }).last().click();
  111 | 
  112 |   await cancelDialog.getByLabel(/PIN менеджера/i).fill(demo.manager_pin);
  113 |   await cancelDialog.getByLabel(/Причина отмены/i).fill('playwright cancel precheck');
  114 |   await cancelDialog.getByRole('button', { name: /Отмена пречека/i }).click();
  115 |   await expect(cancelDialog).toBeHidden();
```