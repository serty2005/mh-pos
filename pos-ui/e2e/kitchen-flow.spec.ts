import { expect, test, type Page } from '@playwright/test';

test.beforeEach(async ({ page }) => {
  const runtimeErrors: string[] = [];
  page.on('pageerror', (error) => runtimeErrors.push(error.message));
  page.on('console', (message) => {
    if (message.type() !== 'error') return;
    runtimeErrors.push(message.text());
  });
  await page.exposeFunction('__posAssertNoRuntimeErrors', () => runtimeErrors);
});

test.afterEach(async ({ page }) => {
  const runtimeErrors = await page.evaluate(() => window.__posAssertNoRuntimeErrors());
  expect(runtimeErrors).toEqual([]);
});

test('kitchen route is an honest readiness screen while KDS endpoints are absent', async ({ page }) => {
  await loginAsManager(page);
  await page.goto('/pos/kitchen');

  await expect(page.getByRole('heading', { name: /Кухонный экран/i })).toBeVisible();
  await expect(page.getByText('запланировано далее')).toBeVisible();
  await expect(page.getByText(/нет routes для kitchen tickets/i)).toBeVisible();
  await expect(page.getByText(/GET для kitchen tickets/i)).toBeVisible();
  await expect(page.getByText(/new/)).toBeVisible();
  await expect(page.getByText(/accepted/)).toBeVisible();
  await expect(page.getByText(/ready/)).toBeVisible();
  await expect(page.getByRole('button', { name: /accepted|in_progress|ready|served|recall|cancelled/i })).toHaveCount(0);
  await expect(page.locator('.kitchen-page')).not.toContainText(/готовый runtime|реализовано сейчас/i);
});

async function loginAsManager(page: Page) {
  await page.goto('/login');
  await expect(page.getByRole('heading', { name: /Вход по PIN/i })).toBeVisible();
  await page.getByLabel(/^PIN$/i).fill('2222');
  await page.getByRole('button', { name: /Войти/i }).click();
  await expect(page.locator('.pos-bottom-bar')).toBeVisible();
}
