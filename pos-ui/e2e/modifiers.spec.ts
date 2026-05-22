import { expect, test, type APIRequestContext } from '@playwright/test';

import { bootstrapRequiredMessage, loadBootstrapJson } from './support/bootstrap';

const apiBase = (process.env.POS_E2E_API_BASE ?? 'http://localhost:8080/api/v1').replace(/\/$/, '');
const clientDeviceId = process.env.POS_E2E_CLIENT_DEVICE_ID ?? 'playwright-modifiers-client';
const bootstrapJson = loadBootstrapJson();

type DemoBootstrap = {
  restaurant_id: string;
  node_device_id: string;
  manager_pin: string;
  table_id: string;
  menu_item_ids: string[];
  modifier_group_id: string;
  modifier_option_id: string;
};

type AuthHeaders = Record<'X-Node-Device-ID' | 'X-Client-Device-ID' | 'X-Actor-Employee-ID' | 'X-Session-ID', string>;

type LoginResult = {
  session: { id: string };
  actor: { employee_id: string };
};

type OrderLineModifier = {
  modifier_group_id: string;
  modifier_option_id: string;
  name: string;
  quantity: number;
  unit_price: number;
  total_price: number;
};

type OrderLine = {
  id: string;
  menu_item_id: string;
  quantity: number;
  unit_price: number;
  total_price: number;
  modifiers: OrderLineModifier[];
};

type Order = {
  id: string;
  status: string;
  total: number;
  lines?: OrderLine[];
};

type Precheck = {
  id: string;
  status: string;
  total: number;
  snapshot?: unknown;
};

test.describe.configure({ mode: 'serial' });

let demo: DemoBootstrap;
let headers: AuthHeaders;
let commandSequence = 0;

test.beforeAll(async ({ playwright }) => {
  const request = await playwright.request.newContext();
  try {
    expect(bootstrapJson, bootstrapRequiredMessage()).toBeTruthy();
    demo = JSON.parse(bootstrapJson ?? '{}') as DemoBootstrap;
    const login = await post<LoginResult>(request, '/auth/pin-login', {
      node_device_id: demo.node_device_id,
      client_device_id: clientDeviceId,
      pin: demo.manager_pin,
    });
    headers = {
      'X-Node-Device-ID': demo.node_device_id,
      'X-Client-Device-ID': clientDeviceId,
      'X-Actor-Employee-ID': login.actor.employee_id,
      'X-Session-ID': login.session.id,
    };
    await ensureShiftAndCashSession(request);
  } finally {
    await request.dispose();
  }
});

test('API edit modifiers reprices line, persists pricing snapshot and rejects locked order edits', async ({ request }) => {
  const order = await createOrder(request);
  const line = await post<OrderLine>(request, `/orders/${order.id}/lines`, {
    command_id: nextCommandID('add-line-with-modifier'),
    menu_item_id: demo.menu_item_ids[0],
    quantity: 1,
    selected_modifiers: [{
      modifier_group_id: demo.modifier_group_id,
      modifier_option_id: demo.modifier_option_id,
      quantity: 1,
    }],
  }, headers);

  expect(line.modifiers).toHaveLength(1);
  expect(line.modifiers[0]).toMatchObject({
    modifier_group_id: demo.modifier_group_id,
    modifier_option_id: demo.modifier_option_id,
    quantity: 1,
    total_price: 3000,
  });
  expect(line.total_price).toBe(18000);

  const edited = await patch<OrderLine>(request, `/orders/${order.id}/lines/${line.id}/modifiers`, {
    command_id: nextCommandID('edit-line-modifier'),
    selected_modifiers: [{
      modifier_group_id: demo.modifier_group_id,
      modifier_option_id: demo.modifier_option_id,
      quantity: 2,
    }],
  });
  expect(edited.modifiers).toHaveLength(1);
  expect(edited.modifiers[0].quantity).toBe(2);
  expect(edited.modifiers[0].total_price).toBe(6000);
  expect(edited.total_price).toBe(21000);

  const updatedOrder = await get<Order>(request, `/orders/${order.id}`);
  expect(updatedOrder.total).toBe(21000);
  expect(updatedOrder.lines?.find((item) => item.id === line.id)?.modifiers[0]).toMatchObject({
    modifier_option_id: demo.modifier_option_id,
    quantity: 2,
    total_price: 6000,
  });

  const precheck = await post<Precheck>(request, `/orders/${order.id}/precheck`, {
    command_id: nextCommandID('issue-precheck-with-modifier'),
  }, headers);
  expect(precheck.status).toBe('issued');
  expect(precheck.total).toBe(21000);
  expect(JSON.stringify(precheck.snapshot)).toContain(demo.modifier_option_id);

  const rejected = await request.patch(`${apiBase}/orders/${order.id}/lines/${line.id}/modifiers`, {
    data: {
      command_id: nextCommandID('edit-locked-modifier'),
      selected_modifiers: [],
    },
    headers,
  });
  expect(rejected.status(), await rejected.text()).toBe(409);
});

async function ensureShiftAndCashSession(request: APIRequestContext) {
  await post(request, '/employee-shifts/open', {
    command_id: nextCommandID('open-shift'),
    restaurant_id: demo.restaurant_id,
    opened_by_employee_id: headers['X-Actor-Employee-ID'],
  }, headers, [201, 409]);

  await post(request, '/cash-shifts/open', {
    command_id: nextCommandID('open-cash-session'),
    restaurant_id: demo.restaurant_id,
    opened_by_employee_id: headers['X-Actor-Employee-ID'],
    opening_cash_amount: 0,
  }, headers, [201, 409]);
}

async function createOrder(request: APIRequestContext) {
  return post<Order>(request, '/orders', {
    command_id: nextCommandID('create-modifier-order'),
    restaurant_id: demo.restaurant_id,
    table_id: demo.table_id,
    table_name: 'E2E modifiers',
    guest_count: 1,
  }, headers);
}

async function get<T>(request: APIRequestContext, path: string, expectedStatus = 200): Promise<T> {
  const response = await request.get(`${apiBase}${path}`, { headers });
  expect(response.status(), await response.text()).toBe(expectedStatus);
  return response.json() as Promise<T>;
}

async function post<T>(
  request: APIRequestContext,
  path: string,
  data?: unknown,
  requestHeaders?: Record<string, string>,
  expectedStatuses: number[] = [201],
): Promise<T> {
  const response = await request.post(`${apiBase}${path}`, {
    data,
    headers: requestHeaders,
  });
  expect(expectedStatuses, await response.text()).toContain(response.status());
  return response.json() as Promise<T>;
}

async function patch<T>(request: APIRequestContext, path: string, data?: unknown): Promise<T> {
  const response = await request.patch(`${apiBase}${path}`, { data, headers });
  expect(response.status(), await response.text()).toBe(200);
  return response.json() as Promise<T>;
}

function nextCommandID(prefix: string) {
  commandSequence += 1;
  return `cmd-e2e-modifiers-${Date.now()}-${commandSequence}-${prefix}`;
}
