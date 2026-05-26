import { expect, test, type APIRequestContext, type Page } from '@playwright/test';

import { bootstrapRequiredMessage, loadBootstrapJson } from './support/bootstrap';

const apiBase = (process.env.POS_E2E_API_BASE ?? 'http://localhost:8080/api/v1').replace(/\/$/, '');
const clientDeviceId = process.env.POS_E2E_CLIENT_DEVICE_ID ?? 'playwright-kitchen-client';
const bootstrapJson = loadBootstrapJson();

type DemoBootstrap = {
  restaurant_id: string;
  node_device_id: string;
  cashier_pin: string;
  kitchen_pin: string;
  manager_pin: string;
  table_ids: string[];
  menu_item_ids: string[];
};

type AuthHeaders = Record<'X-Node-Device-ID' | 'X-Client-Device-ID' | 'X-Actor-Employee-ID' | 'X-Session-ID', string>;

type LoginResult = {
  session: { id: string };
  actor: { employee_id: string };
};

type Order = { id: string };

type OrderLine = {
  id: string;
  name: string;
};

test.describe.configure({ mode: 'serial' });

let demo: DemoBootstrap;
let managerHeaders: AuthHeaders;
let commandSequence = 0;

test.beforeAll(async ({ playwright }) => {
  const request = await playwright.request.newContext();
  try {
    expect(bootstrapJson, bootstrapRequiredMessage()).toBeTruthy();
    demo = JSON.parse(bootstrapJson ?? '{}') as DemoBootstrap;
    managerHeaders = await loginAPI(request, demo.manager_pin, 'manager');
    await ensureShift(request, managerHeaders);
  } finally {
    await request.dispose();
  }
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

test('kitchen route performs a backend-backed KDS transition without fake runtime', async ({ page, request }) => {
  const line = await createKitchenTicket(request);
  let ticketReads = 0;
  await page.route('**/api/v1/kitchen/tickets**', async (route) => {
    ticketReads += 1;
    await route.continue();
  });

  await loginAsKitchen(page);
  await page.goto('/pos/kitchen');

  await expect(page.getByRole('heading', { name: /Кухонный экран/i })).toBeVisible();
  await expect(page.getByText('реализовано сейчас').first()).toBeVisible();
  await expect(page.getByText('KDS runtime активен')).toBeVisible();
  await expect(page.getByText('Backend authoritative')).toBeVisible();
  const ticketCard = page.locator('.kds-ticket').filter({ hasText: line.name }).first();
  await expect(ticketCard).toBeVisible();
  await ticketCard.getByRole('button', { name: /Принять/i }).click();
  await expect.poll(() => ticketReads).toBeGreaterThanOrEqual(2);
  await expect(ticketCard.getByRole('button', { name: /Начать/i })).toBeVisible();
  await ticketCard.getByRole('button', { name: /Начать/i }).click();
  await expect(ticketCard.getByRole('button', { name: /Готово/i })).toBeVisible();
  await ticketCard.getByRole('button', { name: /Готово/i }).click();
  await expect(ticketCard.getByRole('button', { name: /Выдать/i })).toBeVisible();
  await ticketCard.getByRole('button', { name: /Выдать/i }).click();
  await expect(ticketCard).toContainText(/Для текущего статуса нет доступных переходов/i);
  await expect(ticketCard.getByRole('button')).toHaveCount(0);

  await expect(page.locator('.kitchen-page')).not.toContainText(/нет routes для kitchen tickets/i);
  await expect(page.locator('.kitchen-page')).not.toContainText(/Только readiness/i);
  await expect(page.getByRole('button', { name: /Наличные|Карта|Кассовый ящик|Вернуть оплату/i })).toHaveCount(0);
  await expect(page.locator('.kitchen-page')).not.toContainText(/refund|payment refund|cash drawer/i);
});

test('kitchen route shows safe no-permission state for cashier role', async ({ page }) => {
  await loginWithPin(page, demo.cashier_pin);
  await page.goto('/pos/kitchen');

  await expect(page.getByText(/Нужно право pos\.kitchen\.view/i)).toBeVisible();
  await expect(page.getByRole('button', { name: /Наличные|Карта|Кассовый ящик|Вернуть оплату/i })).toHaveCount(0);
});

async function createKitchenTicket(request: APIRequestContext) {
  const tableId = demo.table_ids[0];
  const menuItemId = demo.menu_item_ids[0];
  const order = await post<Order>(request, '/orders', {
    command_id: nextCommandID('create-kitchen-order'),
    restaurant_id: demo.restaurant_id,
    table_id: tableId,
    table_name: 'KDS E2E',
    guest_count: 1,
  }, managerHeaders);
  const line = await post<OrderLine>(request, `/orders/${order.id}/lines`, {
    command_id: nextCommandID('add-kitchen-line'),
    menu_item_id: menuItemId,
    quantity: 1,
  }, managerHeaders);
  return line;
}

async function loginAPI(request: APIRequestContext, pin: string, suffix: string): Promise<AuthHeaders> {
  const login = await post<LoginResult>(request, '/auth/pin-login', {
    command_id: nextCommandID(`login-${suffix}`),
    node_device_id: demo.node_device_id,
    client_device_id: `${clientDeviceId}-${suffix}`,
    pin,
  });
  return {
    'X-Node-Device-ID': demo.node_device_id,
    'X-Client-Device-ID': `${clientDeviceId}-${suffix}`,
    'X-Actor-Employee-ID': login.actor.employee_id,
    'X-Session-ID': login.session.id,
  };
}

async function ensureShift(request: APIRequestContext, headers: AuthHeaders) {
  await post(request, '/employee-shifts/open', {
    command_id: nextCommandID('open-manager-shift'),
    restaurant_id: demo.restaurant_id,
    opened_by_employee_id: headers['X-Actor-Employee-ID'],
    opening_cash_amount: 0,
  }, headers, [201, 409]);
}

async function loginAsKitchen(page: Page) {
  await loginWithPin(page, demo.kitchen_pin);
}

async function loginWithPin(page: Page, pin: string) {
  await page.goto('/login');
  await expect(page.getByRole('heading', { name: /Вход по PIN/i })).toBeVisible();
  await page.getByLabel(/^PIN$/i).fill(pin);
  await page.getByRole('button', { name: /Войти/i }).click();
  await expect(page.locator('.pos-bottom-bar')).toBeVisible();
}

async function post<T>(
  request: APIRequestContext,
  path: string,
  data?: unknown,
  headers?: AuthHeaders,
  expectedStatuses: number[] = [201],
): Promise<T> {
  const response = await request.post(`${apiBase}${path}`, { data, headers });
  expect(expectedStatuses, await response.text()).toContain(response.status());
  return response.json() as Promise<T>;
}

function nextCommandID(prefix: string) {
  commandSequence += 1;
  return `cmd-kitchen-e2e-${Date.now()}-${commandSequence}-${prefix}`;
}
