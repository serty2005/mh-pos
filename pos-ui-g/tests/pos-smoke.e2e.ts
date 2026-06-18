import { expect, test } from '@playwright/test';

const appUrl = process.env.POS_UI_URL ?? 'http://localhost:3000';

test('cashier can add an order line with selected modifier payload', async ({ page }) => {
  test.setTimeout(60_000);
  const consoleMessages: string[] = [];
  const unexpectedHttpErrors: string[] = [];
  const addLinePayloads: unknown[] = [];

  page.on('console', (message) => {
    if (message.type() === 'error' || message.type() === 'warning') {
      if (message.text().startsWith('Failed to load resource:')) return;
      consoleMessages.push(`${message.type()}: ${message.text()}`);
    }
  });

  page.on('response', (response) => {
    if (response.status() < 400) return;
    const url = new URL(response.url());
    if (response.status() === 404 && url.pathname === '/api/v1/cash-shifts/current') return;
    unexpectedHttpErrors.push(`${response.status()} ${response.request().method()} ${url.pathname}`);
  });

  page.on('request', (request) => {
    if (request.method() === 'POST' && /\/api\/v1\/orders\/[^/]+\/lines$/.test(request.url())) {
      addLinePayloads.push(JSON.parse(request.postData() || '{}'));
    }
  });

  await page.addInitScript(() => {
    localStorage.removeItem('mh-pos.session_id');
  });

  await page.goto(appUrl);
  await page.locator('#pin-btn-1').waitFor();
  for (const digit of ['#pin-btn-1', '#pin-btn-1', '#pin-btn-1', '#pin-btn-1']) {
    await page.locator(digit).click();
  }
  await page.locator('#pin-submit-btn').click();
  await expect(page.locator('header')).toBeVisible({ timeout: 15_000 });

  const openShiftButton = page.locator('#cash-open-shift-btn');
  if (await openShiftButton.isVisible({ timeout: 500 }).catch(() => false)) {
    await openShiftButton.click();
    await page.waitForTimeout(1_000);
  } else {
    await page.evaluate(() => document.querySelector<HTMLElement>('#nav-cash')?.click());
  }

  const openCashSessionButton = page.locator('#cash-open-session-btn');
  if (await openCashSessionButton.isEnabled({ timeout: 500 }).catch(() => false)) {
    await openCashSessionButton.click();
    await page.waitForTimeout(1_000);
  }

  await page.evaluate(() => document.querySelector<HTMLElement>('#nav-floor')?.click());
  await expect(page.getByText('Нет столов')).toBeHidden();
  await page.locator('[id^="table-card-"]').first().click();

  const confirmGuests = page.locator('#confirm-guests-btn');
  if (await confirmGuests.isVisible().catch(() => false)) {
    await confirmGuests.click();
  }

  await expect(page.locator('#dish-search-input')).toBeVisible();
  await page.locator('[id^="menu-tile-"]').filter({ hasText: '+Мод' }).first().click();

  const submitButton = page.locator('#modifier-submit-btn');
  await expect(submitButton).toBeVisible();
  const beforeTotal = await submitButton.innerText();
  const paidOption = page.locator('button[id^="mod-opt-"]').filter({ hasText: /\+\d+\s*₽/ }).first();
  const optionName = (await paidOption.locator('span').first().innerText()).trim();

  await paidOption.click();
  await expect(submitButton).not.toHaveText(beforeTotal);
  await submitButton.click();

  await expect.poll(() => addLinePayloads.length).toBeGreaterThan(0);
  expect(addLinePayloads.at(-1)).toMatchObject({
    selected_modifiers: [
      {
        modifier_group_id: expect.any(String),
        modifier_option_id: expect.any(String),
        quantity: 1,
      },
    ],
  });
  await expect(page.getByText(optionName, { exact: false }).first()).toBeVisible();
  expect(unexpectedHttpErrors).toEqual([]);
  expect(consoleMessages).toEqual([]);
});

test('mobile viewport opens waiter runtime instead of desktop POS', async ({ page }) => {
  test.setTimeout(60_000);
  await page.setViewportSize({ width: 390, height: 844 });
  await page.addInitScript(() => {
    localStorage.removeItem('mh-pos.session_id');
  });

  await page.goto(appUrl);
  await page.locator('#pin-btn-1').waitFor();
  for (const digit of ['#pin-btn-1', '#pin-btn-1', '#pin-btn-1', '#pin-btn-1']) {
    await page.locator(digit).click();
  }
  await page.locator('#pin-submit-btn').click();

  await expect(page.locator('#waiter-mobile-runtime')).toBeVisible();
  await expect(page.getByText('Доступ к Waiter-экрану')).toHaveCount(0);
});
