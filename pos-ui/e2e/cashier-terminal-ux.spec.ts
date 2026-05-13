import { expect, test, type Page, type TestInfo } from '@playwright/test';

const bootstrapJson = process.env.POS_E2E_BOOTSTRAP_JSON;

type DemoBootstrap = {
  manager_employee_id: string;
  manager_pin: string;
};

test.describe.configure({ mode: 'serial' });

let demo: DemoBootstrap;

test.beforeAll(() => {
  expect(bootstrapJson, 'Run scripts/bootstrap-production-way.ps1 and pass its JSON as POS_E2E_BOOTSTRAP_JSON').toBeTruthy();
  demo = JSON.parse(bootstrapJson ?? '{}') as DemoBootstrap;
});

test.beforeEach(async ({ page }) => {
  const runtimeErrors: string[] = [];
  page.on('pageerror', (error) => runtimeErrors.push(error.message));
  page.on('console', (message) => {
    if (message.type() !== 'error') return;
    if (/Failed to load resource: the server responded with a status of (403|404|500)/i.test(message.text())) return;
    runtimeErrors.push(message.text());
  });
  await page.exposeFunction('__posAssertNoRuntimeErrors', () => runtimeErrors);
});

test.afterEach(async ({ page }) => {
  const runtimeErrors = await page.evaluate(() => window.__posAssertNoRuntimeErrors());
  expect(runtimeErrors).toEqual([]);
});

test('lazy routes load cashier terminal and out-of-scope workspace shells', async ({ page }, testInfo) => {
  await loginAsManager(page);

  for (const path of ['/pos', '/pos/cashier']) {
    await page.goto(path);
    await expect(page.locator('.cashier-status-bar')).toBeVisible();
    await expect(page.locator('.terminal-grid')).toBeVisible();
    await expect(page.getByRole('heading', { name: /Активный заказ/i })).toBeVisible();
    await expect(page.locator('.q-layout')).not.toContainText(/vite|webpack|uncaught|stack trace/i);
    await saveViewportScreenshot(page, testInfo, `route-${path.replaceAll('/', '-') || 'root'}.png`);
  }

  for (const path of ['/pos/waiter', '/pos/kitchen', '/pos/manager']) {
    await page.goto(path);
    await expect(page.getByText('Вне текущего объема')).toBeVisible();
    await expect(page.getByText(/runtime-сценарий не реализован/i)).toBeVisible();
    await expect(page.getByRole('link', { name: /Терминал кассира/i })).toBeVisible();
    await expect(page.locator('.q-layout')).not.toContainText(/готовый runtime|runtime готов|реализовано сейчас/i);
  }
});

test('cashier terminal supports floor, order, checkout, drawers and safe manager override UX', async ({ page }, testInfo) => {
  await page.setViewportSize({ width: 1440, height: 900 });
  await loginAsManager(page);
  await page.goto('/pos');

  await expect(page.locator('.cashier-status-bar')).toBeVisible();
  await expect(page.getByText('Production Manager')).toBeVisible();
  await expect(page.getByText('Личная смена')).toBeVisible();
  await expect(page.getByText('Кассовая смена')).toBeVisible();
  await expect(page.locator('.cashier-status-bar').getByText('Синхронизация')).toBeVisible();
  await expect(page.locator('.cashier-status-bar').getByRole('button', { name: /Заблокировать/i })).toBeVisible();
  await expectNoStatusOverlap(page);
  await expectTouchTargets(page);

  await page.keyboard.press('Tab');
  await expectVisibleFocusRing(page);

  const hall = page.locator('.hall-chip').first();
  await expect(hall).toHaveAttribute('aria-pressed', 'true');

  const table = page.locator('.table-button').first();
  await table.click();
  await expect(table).toHaveAttribute('aria-pressed', 'true');
  await expect(page.getByText(/На выбранном столе нет активного заказа|В заказе нет активных позиций/i)).toBeVisible();

  await expect(page.locator('.checkout-dock')).toBeInViewport();
  await expect(page.locator('.pane-scroll').last()).toHaveCSS('overflow-y', 'auto');
  await saveViewportScreenshot(page, testInfo, 'cashier-empty-desktop.png');

  await page.getByRole('button', { name: /Создать заказ/i }).click();
  await expect(page.locator('.order-summary')).toBeVisible();

  await page.locator('.menu-button').first().click();
  await expect(page.locator('.line-row')).toHaveCount(1);
  await saveViewportScreenshot(page, testInfo, 'cashier-order-line.png');

  await page.getByRole('button', { name: /Выпустить пречек/i }).click();
  await expect(page.getByText('Пречек выпущен')).toBeVisible();

  await page.getByRole('button', { name: /Отмена пречека/i }).click();
  const cancelDialog = page.locator('.q-dialog').filter({ hasText: 'Отмена пречека' });
  await expect(cancelDialog).toBeVisible();
  await expect(cancelDialog.getByLabel(/PIN менеджера/i)).toHaveAttribute('type', 'password');
  await cancelDialog.getByLabel(/ID менеджера/i).fill(demo.manager_employee_id);
  await cancelDialog.getByLabel(/PIN менеджера/i).fill('0000');
  await cancelDialog.getByLabel(/Причина отмены/i).fill('playwright invalid manager override');
  await cancelDialog.getByRole('button', { name: /Отмена пречека/i }).click();
  await expect(page.getByRole('heading', { name: /Операция не выполнена|Недостаточно прав|Проверьте данные/i })).toBeVisible();
  await expectNoSensitiveText(page, ['0000', 'manager_pin', 'managerPin', 'sql:', 'stack trace', 'panic:']);
  await page.getByRole('button', { name: /Закрыть/i }).last().click();

  await cancelDialog.getByLabel(/PIN менеджера/i).fill(demo.manager_pin);
  await cancelDialog.getByLabel(/Причина отмены/i).fill('playwright cancel precheck');
  await cancelDialog.getByRole('button', { name: /Отмена пречека/i }).click();
  await expect(cancelDialog).toBeHidden();
  await expect(page.getByRole('button', { name: /Выпустить пречек/i })).toBeEnabled();

  await page.getByRole('button', { name: /Выпустить пречек/i }).click();
  await expect(page.getByText('Пречек выпущен')).toBeVisible();
  await page.getByRole('button', { name: /Наличные/i }).click();
  await expect(page.getByText(/Финальный чек создан/i)).toBeVisible();

  await page.getByRole('button', { name: /Кассовый ящик/i }).click();
  await expect(page.locator('.q-dialog').filter({ hasText: 'Кассовый ящик' })).toBeVisible();
  await saveViewportScreenshot(page, testInfo, 'cash-drawer-dialog.png');
  await page.getByRole('button', { name: /^Отменить$/i }).click();

  await page.getByRole('button', { name: /Синхронизация/i }).last().click();
  const syncDrawer = page.locator('.q-drawer').filter({ hasText: 'Синхронизация' });
  await expect(syncDrawer).toBeVisible();
  await saveViewportScreenshot(page, testInfo, 'sync-drawer.png');
  await syncDrawer.getByRole('button', { name: /^Закрыть$/i }).click();

  await page.getByRole('button', { name: /Закрытые заказы/i }).click();
  const closedOrdersDrawer = page.locator('.q-drawer').filter({ hasText: 'Закрытые заказы' });
  await expect(closedOrdersDrawer).toBeVisible();
  const refundButton = closedOrdersDrawer.getByRole('button', { name: /Вернуть оплату/i }).first();
  await expect(refundButton).toBeVisible();
  await refundButton.click();
  await expect(page.locator('.q-dialog').filter({ hasText: 'Вернуть оплату' })).toBeVisible();
  await saveViewportScreenshot(page, testInfo, 'refund-dialog-supported-flow.png');
});

test('cashier terminal remains usable on tablet and mobile viewports', async ({ page }, testInfo) => {
  await loginAsManager(page);

  for (const viewport of [
    { name: 'tablet-1024x768', width: 1024, height: 768 },
    { name: 'mobile-390x844', width: 390, height: 844 },
  ]) {
    await page.setViewportSize(viewport);
    await page.goto('/pos');
    await expect(page.locator('.cashier-status-bar')).toBeVisible();
    await expect(page.locator('.terminal-grid')).toBeVisible();
    await expect(page.getByRole('heading', { name: /Активный заказ/i })).toBeVisible();
    await expectNoHorizontalOverflow(page);
    await expectTouchTargets(page);
    await saveViewportScreenshot(page, testInfo, `${viewport.name}-cashier.png`);

    await page.getByRole('button', { name: /Закрытые заказы/i }).click();
    await expect(page.locator('.q-drawer').filter({ hasText: 'Закрытые заказы' })).toBeVisible();
    await expectOverlayFitsViewport(page, '.q-drawer');
    await saveViewportScreenshot(page, testInfo, `${viewport.name}-closed-orders-drawer.png`);
    await page.locator('.q-drawer').filter({ hasText: 'Закрытые заказы' }).getByRole('button', { name: /^Закрыть$/i }).click();

    await page.getByRole('button', { name: /Кассовый ящик/i }).click();
    await expect(page.locator('.q-dialog').filter({ hasText: 'Кассовый ящик' })).toBeVisible();
    await expectOverlayFitsViewport(page, '.q-dialog .dialog-card');
    await saveViewportScreenshot(page, testInfo, `${viewport.name}-cash-drawer-dialog.png`);
    await page.getByRole('button', { name: /^Отменить$/i }).click();
  }
});

test('loading and error states are readable without raw backend details', async ({ page }, testInfo) => {
  await loginAsManager(page);
  await page.route('**/api/v1/orders/current?**', async (route) => {
    await route.fulfill({
      status: 500,
      contentType: 'application/json',
      body: JSON.stringify({
        error: {
          code: 'INTERNAL_ERROR',
          message_key: 'errors.server',
          correlation_id: 'ux-smoke-correlation',
        },
      }),
    });
  });

  await page.goto('/pos');
  await page.locator('.table-button').first().click();
  await expect(page.locator('.order-skeleton')).toBeVisible();
  await expect(page.getByText(/Ошибка backend|POS Edge backend вернул внутреннюю ошибку|Не удалось загрузить данные/i)).toBeVisible();
  await expectNoSensitiveText(page, ['SELECT ', 'sqlite', 'panic:', 'stack trace', 'manager_pin', 'token']);
  await saveViewportScreenshot(page, testInfo, 'cashier-safe-error-state.png');
});

async function loginAsManager(page: Page) {
  await page.goto('/login');
  await expect(page.getByRole('heading', { name: /Вход по PIN/i })).toBeVisible();
  await page.getByLabel(/^PIN$/i).fill(demo.manager_pin);
  await page.getByRole('button', { name: /Войти/i }).click();
  await expect(page.locator('.cashier-status-bar')).toBeVisible();
}

async function saveViewportScreenshot(page: Page, testInfo: TestInfo, name: string) {
  const path = testInfo.outputPath(name);
  await page.screenshot({ path, fullPage: false });
  await testInfo.attach(name, { path, contentType: 'image/png' });
}

async function expectNoStatusOverlap(page: Page) {
  const boxes = await page.evaluate(() => {
    const status = document.querySelector('.cashier-status-bar')?.getBoundingClientRect();
    const grid = document.querySelector('.terminal-grid')?.getBoundingClientRect();
    return status && grid ? { statusBottom: status.bottom, gridTop: grid.top } : null;
  });
  expect(boxes).not.toBeNull();
  expect(boxes!.statusBottom).toBeLessThanOrEqual(boxes!.gridTop);
}

async function expectNoHorizontalOverflow(page: Page) {
  const overflow = await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth);
  expect(overflow).toBeLessThanOrEqual(2);
}

async function expectTouchTargets(page: Page) {
  const tooSmall = await page.locator('button:visible, .q-btn:visible').evaluateAll((items) => items
    .map((item) => {
      const rect = item.getBoundingClientRect();
      return { text: item.textContent?.trim() ?? item.getAttribute('aria-label') ?? '', width: rect.width, height: rect.height };
    })
    .filter((item) => item.width > 0 && item.height > 0 && (item.width < 48 || item.height < 40)));
  expect(tooSmall).toEqual([]);
}

async function expectVisibleFocusRing(page: Page) {
  const focused = await page.evaluate(() => {
    const element = document.activeElement;
    if (!element) return null;
    const style = window.getComputedStyle(element);
    return {
      outlineStyle: style.outlineStyle,
      outlineWidth: Number.parseFloat(style.outlineWidth || '0'),
      boxShadow: style.boxShadow,
    };
  });
  expect(focused).not.toBeNull();
  expect(focused!.outlineStyle !== 'none' || focused!.outlineWidth > 0 || focused!.boxShadow !== 'none').toBe(true);
}

async function expectOverlayFitsViewport(page: Page, selector: string) {
  const fits = await page.locator(selector).last().evaluate((element) => {
    const rect = element.getBoundingClientRect();
    return rect.left >= -1 && rect.top >= -1 && rect.right <= window.innerWidth + 1 && rect.bottom <= window.innerHeight + 1;
  });
  expect(fits).toBe(true);
}

async function expectNoSensitiveText(page: Page, needles: string[]) {
  const bodyText = await page.locator('body').innerText();
  const lower = bodyText.toLocaleLowerCase();
  for (const needle of needles) {
    expect(lower).not.toContain(needle.toLocaleLowerCase());
  }
}

declare global {
  interface Window {
    __posAssertNoRuntimeErrors(): string[];
  }
}
