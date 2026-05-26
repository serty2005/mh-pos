import { expect, test, type Page } from '@playwright/test';

import { bootstrapRequiredMessage, loadBootstrapJson } from './support/bootstrap';

const bootstrapJson = loadBootstrapJson();

test.describe.configure({ mode: 'serial' });

test.beforeAll(() => {
  expect(bootstrapJson, bootstrapRequiredMessage()).toBeTruthy();
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

test('waiter mobile can select a table, create an order, add a line and issue precheck', async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await loginAsManager(page);
  await page.goto('/pos/waiter');

  await expect(page.getByRole('heading', { name: /Столы, заказ и пречек/i })).toBeVisible();
  await expect(page.getByText(/не принимает финансовые решения/i)).toBeVisible();
  await expect(page.getByRole('button', { name: /Наличные|Карта|Кассовый ящик|Вернуть оплату/i })).toHaveCount(0);

  const freeTable = page.locator('.waiter-table-card:not(.occupied)').first();
  await expect(freeTable).toBeVisible();
  const tableName = (await freeTable.locator('strong').innerText()).trim();
  await freeTable.click();
  await expect(page.locator('.waiter-sticky-context')).toContainText(tableName);

  const createOrder = page.getByRole('button', { name: /Создать заказ/i });
  if (await createOrder.isEnabled()) {
    await createOrder.click();
  }

  const menuItem = page.locator('.waiter-menu-item:not([disabled])').first();
  await expect(menuItem).toBeVisible();
  await menuItem.click();

  const modifierDialog = page.locator('.q-dialog').filter({ hasText: 'Модификаторы' });
  if (await modifierDialog.isVisible().catch(() => false)) {
    const firstModifierAdd = modifierDialog.getByRole('button', { name: /Добавить/i }).first();
    if (await firstModifierAdd.isVisible()) await firstModifierAdd.click();
    await modifierDialog.locator('.dialog-actions').getByRole('button', { name: /^Добавить$/i }).click();
  }

  await expect(page.locator('.waiter-line-row').first()).toBeVisible();
  await page.getByRole('button', { name: /Выпустить пречек/i }).click();
  await expect(page.getByText(/Пречек выпущен|Заказ заблокирован активным пречеком/i)).toBeVisible();
  await expect(page.locator('.waiter-sticky-context.locked')).toBeVisible();
  await expect(page.locator('.waiter-menu-item:not([disabled])')).toHaveCount(0);
  await expect(page.getByRole('button', { name: /Наличные|Карта|Кассовый ящик|Вернуть оплату/i })).toHaveCount(0);
});

test('waiter mobile keeps payment, refund and cash drawer controls out of the default surface', async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await loginAsManager(page);
  await page.goto('/pos/waiter');

  await expect(page.locator('.waiter-page')).toBeVisible();
  await expect(page.getByText(/не принимает финансовые решения/i)).toBeVisible();
  await expect(page.getByRole('button', { name: /Наличные|Карта|Вернуть оплату|Кассовый ящик/i })).toHaveCount(0);
  await expect(page.locator('.waiter-page')).not.toContainText(/refund|payment refund|cash drawer/i);
});

async function loginAsManager(page: Page) {
  await page.goto('/login');
  await expect(page.getByRole('heading', { name: /Вход по PIN/i })).toBeVisible();
  await page.getByLabel(/^PIN$/i).fill('2222');
  await page.getByRole('button', { name: /Войти/i }).click();
  await expect(page.locator('.pos-bottom-bar, .waiter-page')).toBeVisible();
}
