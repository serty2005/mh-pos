import { expect, test, type Page, type TestInfo } from '@playwright/test';

import { bootstrapRequiredMessage, loadBootstrapJson } from './support/bootstrap';

const bootstrapJson = loadBootstrapJson();

type DemoBootstrap = {
  manager_pin: string;
};

test.describe.configure({ mode: 'serial' });

let demo: DemoBootstrap;

test.beforeAll(() => {
  expect(bootstrapJson, bootstrapRequiredMessage()).toBeTruthy();
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

test('lazy routes load redesigned POS shell and out-of-scope workspace shells', async ({ page }, testInfo) => {
  await loginAsManager(page);

  for (const path of ['/pos', '/pos/cashier']) {
    await page.goto(path);
    await expectRedesignedShell(page);
    await expect(page.locator('.q-layout')).not.toContainText(/vite|webpack|uncaught|stack trace/i);
    await saveViewportScreenshot(page, testInfo, `route-${path.replaceAll('/', '-') || 'root'}.png`);
  }

  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto('/pos/waiter');
  await expect(page.locator('.waiter-page')).toBeVisible();
  await expect(page.getByText(/не принимает финансовые решения/i)).toBeVisible();
  await expect(page.getByRole('button', { name: /Наличные|Карта|Кассовый ящик|Вернуть оплату/i })).toHaveCount(0);

  for (const path of ['/pos/kitchen', '/pos/manager']) {
    await page.goto(path);
    if (path === '/pos/kitchen') {
      await expect(page.getByText('запланировано далее').first()).toBeVisible();
      await expect(page.getByText(/нет routes для kitchen tickets/i)).toBeVisible();
    } else {
      await expect(page.getByText('Вне текущего объема')).toBeVisible();
      await expect(page.getByText(/runtime-сценарий не реализован/i)).toBeVisible();
      await expect(page.getByRole('link', { name: /Терминал кассира/i })).toBeVisible();
    }
    await expect(page.locator('.q-layout')).not.toContainText(/готовый runtime|runtime готов|реализовано сейчас/i);
  }
});

test('redesigned POS shell supports section navigation and cashier flow', async ({ page }, testInfo) => {
  await page.setViewportSize({ width: 1440, height: 900 });
  await loginAsManager(page);
  await page.goto('/pos');

  await expectRedesignedShell(page);
  await expect(page.getByText(/Demo Manager|Manager/i).first()).toBeVisible();
  await expectTouchTargets(page);

  await page.keyboard.press('Tab');
  await expectVisibleFocusRing(page);

  await assertSectionMenuNavigation(page, testInfo);
  await ensureOperationsReady(page);
  await saveViewportScreenshot(page, testInfo, 'redesign-cash-ready-desktop.png');

  await openSection(page, 'Залы и столы');
  const hall = page.locator('.hall-chip').first();
  await expect(hall).toHaveAttribute('aria-pressed', 'true');

  const table = page.locator('.floor-table-tile').first();
  await table.click();
  await expect(page.locator('.floor-order-rail')).toBeVisible();
  await expect(page.locator('.floor-order-rail')).toContainText(/На выбранном столе нет активного заказа|Активные заказы/i);
  await saveViewportScreenshot(page, testInfo, 'redesign-order-empty-desktop.png');

  const createOrder = page.getByRole('button', { name: /Создать заказ/i }).last();
  if (await createOrder.isVisible()) {
    await expect(createOrder).toBeEnabled();
    await createOrder.click();
  }
  await expect(page.locator('.pos-menu-area')).toBeVisible();
  await expect(page.locator('.pos-order-rail')).toBeVisible();
  await expect(page.locator('.rail-summary')).toBeVisible();

  await cancelIssuedPrecheckIfPresent(page);
  const lineCountBeforeAdd = await page.locator('.rail-line').count();
  const menuTile = page.locator('.menu-tile:not([disabled])').filter({ hasNotText: /Есть модификаторы/i }).first();
  await expect(menuTile).toBeVisible();
  await menuTile.click();
  await expect.poll(() => page.locator('.rail-line').count()).toBeGreaterThan(lineCountBeforeAdd);
  await saveViewportScreenshot(page, testInfo, 'redesign-order-line.png');

  await page.getByRole('button', { name: /Выпустить пречек/i }).click();
  await expect(page.locator('.pos-order-rail')).toContainText('Пречек выпущен');

  await page.getByRole('button', { name: /Отмена пречека/i }).click();
  const cancelDialog = page.locator('.q-dialog').filter({ hasText: 'Отмена пречека' });
  await expect(cancelDialog).toBeVisible();
  await expect(cancelDialog.getByLabel(/PIN менеджера/i)).toHaveAttribute('type', 'password');
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
  await expect(page.locator('.pos-order-rail')).toContainText('Пречек выпущен');
  await page.getByRole('button', { name: /^Касса$/i }).click();
  const paymentDialog = page.locator('.q-dialog').filter({ hasText: 'Оплата' });
  await expect(paymentDialog).toBeVisible();
  await paymentDialog.getByRole('button', { name: /Наличные/i }).click();
  await expect(page.locator('.pos-order-rail')).toContainText(/Финальный чек создан/i);
  await paymentDialog.getByRole('button', { name: /Закрыть/i }).click();
  await expect(paymentDialog).toBeHidden();

  await openSection(page, 'Активность');
  await expect(page.locator('.activity-workspace')).toBeVisible();
  const closedOrder = page.locator('.activity-order-item').first();
  await expect(closedOrder).toBeVisible();
  await closedOrder.click();
  await expect(page.locator('.activity-detail-rail')).toBeVisible();
  await expect(page.getByRole('button', { name: /Копия чека/i })).toBeVisible();
  const refundButton = page.getByRole('button', { name: /Вернуть оплату/i }).first();
  await expect(refundButton).toBeVisible();
  await refundButton.click();
  await expect(page.locator('.q-dialog').filter({ hasText: 'Вернуть оплату' })).toBeVisible();
  await saveViewportScreenshot(page, testInfo, 'refund-dialog-supported-flow.png');

  await openSection(page, 'Касса');
  await expect(page.locator('.cash-workspace')).toBeVisible();
  await expect(page.getByRole('button', { name: /Кассовый ящик/i })).toBeVisible();
  await expect(page.getByRole('button', { name: /Заблокировать/i })).toBeVisible();
});

test('redesigned POS shell remains usable on tablet and mobile viewports', async ({ page }, testInfo) => {
  await loginAsManager(page);

  for (const viewport of [
    { name: 'tablet-1024x768', width: 1024, height: 768 },
    { name: 'mobile-390x844', width: 390, height: 844 },
  ]) {
    await page.setViewportSize(viewport);
    await page.goto('/pos');
    await expectRedesignedShell(page);
    await expectNoHorizontalOverflow(page);
    await expectTouchTargets(page);
    await saveViewportScreenshot(page, testInfo, `${viewport.name}-redesign-shell.png`);

    await openSection(page, 'Активность');
    await expect(page.locator('.activity-workspace')).toBeVisible();
    await saveViewportScreenshot(page, testInfo, `${viewport.name}-activity-section.png`);

    await openSection(page, 'Касса');
    await expect(page.locator('.cash-workspace')).toBeVisible();
    await expect(page.locator('.pos-bottom-bar')).toBeInViewport();
    await saveViewportScreenshot(page, testInfo, `${viewport.name}-cash-section.png`);
  }
});

test('cashier can add and edit selected modifiers on an open order line', async ({ page }, testInfo) => {
  await page.setViewportSize({ width: 1440, height: 900 });
  await loginAsManager(page);
  await page.goto('/pos');
  await ensureOperationsReady(page);
  await openSection(page, 'Залы и столы');
  await page.locator('.floor-table-tile').first().click();

  await openSection(page, 'Заказы');
  await cancelIssuedPrecheckIfPresent(page);

  const createOrder = page.getByRole('button', { name: /Создать заказ/i }).last();
  if (await createOrder.isVisible().catch(() => false)) {
    await createOrder.click();
  }
  await expect(page.getByRole('region', { name: /Меню/i })).toBeVisible();

  const lineCountBeforeAdd = await page.locator('.order-item-row').count();
  const menuTile = page.getByRole('region', { name: /Меню/i }).getByRole('button', { name: /Production Tea/i }).first();
  await expect(menuTile).toBeVisible();
  await menuTile.click();

  const addDialog = page.locator('.q-dialog').filter({ hasText: 'Модификаторы' });
  await expect(addDialog).toBeVisible();
  const lemonOption = addDialog.locator('.modifier-option').filter({ hasText: 'Lemon' });
  await lemonOption.getByRole('button', { name: /Добавить/i }).click();
  await addDialog.locator('.dialog-actions').getByRole('button', { name: /^Добавить$/i }).click();

  await expect.poll(() => page.locator('.order-item-row').count()).toBeGreaterThan(lineCountBeforeAdd);
  const editedLine = page.locator('.order-item-row').last();
  await expect(editedLine).toContainText('Lemon');
  await expect(editedLine).toContainText('180,00');
  const editedQuantity = page.locator('.quantity-control').last();
  await expect(editedQuantity).toContainText('1 шт');

  const editButton = editedQuantity.getByRole('button', { name: /Изменить модификаторы/i });
  await expect(editButton).toBeEnabled();
  await editButton.click();

  const editDialog = page.locator('.q-dialog').filter({ hasText: 'Изменение модификаторов' });
  await expect(editDialog).toBeVisible();
  await expect(editDialog.locator('.modifier-option').filter({ hasText: 'Lemon' })).toContainText('1');
  await expect(editDialog.locator('.dialog-actions').getByRole('button', { name: /^Сохранить$/i })).toBeVisible();
  await editDialog.locator('.modifier-option').filter({ hasText: 'Lemon' }).getByRole('button', { name: /Добавить/i }).click();
  await editDialog.locator('.dialog-actions').getByRole('button', { name: /^Сохранить$/i }).click();

  await expect(editDialog).toBeHidden();
  await expect(editedLine).toContainText('210,00');
  await saveViewportScreenshot(page, testInfo, 'modifier-edit-supported-flow.png');
});

test('loading and error states are readable without raw backend details', async ({ page }, testInfo) => {
  await loginAsManager(page);
  await ensureOperationsReady(page);
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

  await openSection(page, 'Залы и столы');
  await page.locator('.floor-table-tile').first().click();
  await expect(page.locator('.order-skeleton')).toBeVisible();
  await expect(page.getByText(/Ошибка backend|POS Edge backend вернул внутреннюю ошибку|Не удалось загрузить данные/i)).toBeVisible();
  await expectNoSensitiveText(page, ['SELECT ', 'sqlite', 'panic:', 'stack trace', 'manager_pin', 'token']);
  await saveViewportScreenshot(page, testInfo, 'cashier-safe-error-state.png');
});

async function loginAsManager(page: Page) {
  await page.goto('/login');
  await expect(page.getByRole('heading', { name: /Вход по PIN/i })).toBeVisible();
  await page.getByLabel(/^PIN$/i).fill('2222');
  await page.getByRole('button', { name: /Войти/i }).click();
  await expect(page.locator('.pos-bottom-bar')).toBeVisible();
}

async function expectRedesignedShell(page: Page) {
  await expect(page.locator('.pos-app-shell')).toBeVisible();
  await expect(page.locator('.pos-bottom-bar')).toBeVisible();
  await expect(page.locator('.bottom-section-button')).toBeVisible();
}

async function assertSectionMenuNavigation(page: Page, testInfo: TestInfo) {
  for (const section of ['Залы / столы', 'Заказы', 'Активность', 'Отчеты', 'Касса']) {
    await openSection(page, section);
    await expect(page.locator('.bottom-section-button')).toContainText(section);
    await expect(page.locator('.pos-section-menu')).toBeHidden();
  }
  await page.locator('.bottom-section-button').click();
  await expect(page.locator('.pos-section-menu')).toBeVisible();
  await expect(page.locator('.section-menu-item')).toHaveCount(5);
  await saveViewportScreenshot(page, testInfo, 'section-menu-open.png');
  await page.keyboard.press('Escape');
  await expect(page.locator('.pos-section-menu')).toBeHidden();
}

async function openSection(page: Page, section: string) {
  const sectionAliases: Record<string, string> = {
    'Залы и столы': 'Залы / столы',
    Заказы: 'Заказы',
    Активность: 'Активность',
    Отчеты: 'Отчеты',
    Касса: 'Касса',
  };
  const label = sectionAliases[section] ?? section;
  await page.locator('.bottom-section-button').click();
  await expect(page.locator('.pos-section-menu')).toBeVisible();
  await page.locator('.section-menu-item').filter({ hasText: label }).click();
}

async function ensureOperationsReady(page: Page) {
  await openSection(page, 'Касса');
  const openShift = page.getByRole('button', { name: /Открыть личную смену/i });
  if (await openShift.isVisible().catch(() => false)) {
    await expect(openShift).toBeEnabled();
    await openShift.click();
    await expect(openShift).toBeHidden();
  }

  const openCash = page.getByRole('button', { name: /Открыть кассовую смену/i });
  if (await openCash.isVisible().catch(() => false)) {
    await expect(openCash).toBeEnabled();
    await openCash.click();
    await expect(openCash).toBeHidden();
  }

  await expect(page.locator('.cash-workspace')).toBeVisible();
  await expect(openShift).toBeHidden();
  await expect(openCash).toBeHidden();
}

async function cancelIssuedPrecheckIfPresent(page: Page) {
  const cancelPrecheck = page.getByRole('button', { name: /Отмена пречека/i });
  if (!(await cancelPrecheck.isVisible().catch(() => false)) || !(await cancelPrecheck.isEnabled().catch(() => false))) {
    return;
  }
  await cancelPrecheck.click();
  const cancelDialog = page.locator('.q-dialog').filter({ hasText: 'Отмена пречека' });
  await expect(cancelDialog).toBeVisible();
  await cancelDialog.getByLabel(/PIN менеджера/i).fill(demo.manager_pin);
  await cancelDialog.getByLabel(/Причина отмены/i).fill('playwright prepare editable order');
  await cancelDialog.getByRole('button', { name: /Отмена пречека/i }).click();
  await expect(cancelDialog).toBeHidden();
  await expect(page.getByRole('region', { name: /Меню/i })).toBeVisible();
  await expect(page.getByRole('region', { name: /Меню/i }).getByRole('button').first()).toBeVisible();
}

async function saveViewportScreenshot(page: Page, testInfo: TestInfo, name: string) {
  const path = testInfo.outputPath(name);
  await page.screenshot({ path, fullPage: false });
  await testInfo.attach(name, { path, contentType: 'image/png' });
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
