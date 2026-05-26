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
    runtimeErrors.push(message.text());
  });
  await page.exposeFunction('__posAssertNoRuntimeErrors', () => runtimeErrors);
});

test.afterEach(async ({ page }) => {
  const runtimeErrors = await page.evaluate(() => window.__posAssertNoRuntimeErrors());
  expect(runtimeErrors).toEqual([]);
});

test('kitchen route reads backend KDS tickets without fake runtime', async ({ page }) => {
  await loginAsKitchen(page);
  await page.goto('/pos/kitchen');

  await expect(page.getByRole('heading', { name: /Кухонный экран/i })).toBeVisible();
  await expect(page.getByText('реализовано сейчас').first()).toBeVisible();
  await expect(page.getByText('KDS runtime активен')).toBeVisible();
  await expect(page.getByText('Backend authoritative')).toBeVisible();
  await expect(page.getByText('new', { exact: true })).toBeVisible();
  await expect(page.getByText('accepted', { exact: true })).toBeVisible();
  await expect(page.getByText('ready', { exact: true })).toBeVisible();
  await expect(page.locator('.kitchen-page')).not.toContainText(/нет routes для kitchen tickets/i);
  await expect(page.locator('.kitchen-page')).not.toContainText(/Только readiness/i);
});

async function loginAsKitchen(page: Page) {
  await page.goto('/login');
  await expect(page.getByRole('heading', { name: /Вход по PIN/i })).toBeVisible();
  await page.getByLabel(/^PIN$/i).fill('5555');
  await page.getByRole('button', { name: /Войти/i }).click();
  await expect(page.locator('.pos-bottom-bar')).toBeVisible();
}
