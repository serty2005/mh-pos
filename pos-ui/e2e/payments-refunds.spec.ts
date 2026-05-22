import { expect, test, type APIRequestContext } from '@playwright/test';

import { bootstrapRequiredMessage, loadBootstrapJson } from './support/bootstrap';

const apiBase = (process.env.POS_E2E_API_BASE ?? 'http://localhost:8080/api/v1').replace(/\/$/, '');
const clientDeviceId = process.env.POS_E2E_CLIENT_DEVICE_ID ?? 'playwright-e2e-client';

type DemoBootstrap = {
  restaurant_id: string;
  node_device_id: string;
  cashier_pin: string;
  manager_pin: string;
  table_ids: string[];
  menu_item_ids: string[];
};

const bootstrapJson = loadBootstrapJson();

type AuthHeaders = Record<'X-Node-Device-ID' | 'X-Client-Device-ID' | 'X-Actor-Employee-ID' | 'X-Session-ID', string>;

type LoginResult = {
  session: { id: string };
  actor: { employee_id: string };
};

type Order = {
  id: string;
  status: string;
  total: number;
  check?: Check;
};

type Precheck = {
  id: string;
  status: string;
  total: number;
  paid_total: number;
};

type Payment = {
  id: string;
  precheck_id: string;
  amount: number;
  method: 'cash' | 'card' | 'other';
  currency: string;
  status: 'captured' | 'refunded' | 'failed';
};

type Check = {
  id: string;
  status: 'open' | 'paid' | 'refunded' | 'voided';
  total: number;
  paid_total: number;
  payments?: Payment[];
};

type Shift = {
  id: string;
};

type CashSession = {
  id: string;
};

type FinancialOperation = {
  id: string;
  operation_type: 'cancellation' | 'refund';
  operation_kind: 'full' | 'partial';
  status: 'recorded';
  check_id: string;
  amount: number;
  inventory_disposition: 'no_stock_effect' | 'return_to_stock' | 'write_off_waste' | 'manual_review';
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

test('полная оплата cash закрывает заказ и показывает оплату в закрытых заказах', async ({ request }) => {
  const { order, precheck } = await createOrderWithPrecheck(request, 0, 1);

  const payment = await capturePayment(request, precheck.id, 'cash', precheck.total);
  expect(payment.status).toBe('captured');
  expect(payment.amount).toBe(precheck.total);

  const paidOrder = await get<Order>(request, `/orders/${order.id}`);
  expect(paidOrder.status).toBe('closed');
  expect(paidOrder.check?.status).toBe('paid');
  expect(paidOrder.check?.paid_total).toBe(precheck.total);

  const closedOrders = await get<Order[]>(request, '/orders/closed?limit=20');
  const closed = closedOrders.find((item) => item.id === order.id);
  expect(closed?.check?.payments?.some((item) => item.id === payment.id && item.status === 'captured')).toBe(true);
});

test('частичные оплаты не создают финальный чек до полной оплаты', async ({ request }) => {
  const { order, precheck } = await createOrderWithPrecheck(request, 1, 2);
  const firstAmount = Math.floor(precheck.total / 2);

  const firstPayment = await capturePayment(request, precheck.id, 'cash', firstAmount);
  expect(firstPayment.status).toBe('captured');

  const afterFirstPayment = await get<Order>(request, `/orders/${order.id}`);
  expect(afterFirstPayment.status).toBe('locked');
  expect(afterFirstPayment.check).toBeUndefined();

  const secondPayment = await capturePayment(request, precheck.id, 'card', precheck.total - firstAmount);
  expect(secondPayment.status).toBe('captured');

  const paidOrder = await get<Order>(request, `/orders/${order.id}`);
  expect(paidOrder.status).toBe('closed');
  expect(paidOrder.check?.status).toBe('paid');
  expect(paidOrder.check?.paid_total).toBe(precheck.total);
});

test('compatibility refund records ledger operation without mutating finalized payment/check', async ({ request }) => {
  const { order, precheck } = await createOrderWithPrecheck(request, 0, 1);
  const payment = await capturePayment(request, precheck.id, 'cash', precheck.total);

  await closeCurrentShiftAndCashSession(request);
  await ensureShiftAndCashSession(request);

  const refunded = await post<Payment>(request, `/payments/${payment.id}/refund`, {
    command_id: nextCommandID('refund-payment'),
    reason: 'e2e refund',
  }, headers);
  expect(refunded.status).toBe('captured');

  const refundedOrder = await get<Order>(request, `/orders/${order.id}`);
  expect(refundedOrder.check?.status).toBe('paid');
  expect(refundedOrder.check?.paid_total).toBe(precheck.total);

  const closedOrders = await get<Order[]>(request, '/orders/closed?limit=20');
  const closed = closedOrders.find((item) => item.id === order.id);
  expect(closed?.check?.status).toBe('paid');
  expect(closed?.check?.payments?.some((item) => item.id === payment.id && item.status === 'captured')).toBe(true);
});

test('full check cancellation records ledger operation without UI status mutation expectations', async ({ request }) => {
  const { order, precheck } = await createOrderWithPrecheck(request, 0, 1);
  const payment = await capturePayment(request, precheck.id, 'cash', precheck.total);
  const paidOrder = await get<Order>(request, `/orders/${order.id}`);
  expect(paidOrder.check?.id).toBeTruthy();

  const operation = await post<FinancialOperation>(request, `/checks/${paidOrder.check?.id}/cancellations`, {
    command_id: nextCommandID('cancel-check'),
    operation_kind: 'full',
    inventory_disposition: 'manual_review',
    reason: 'e2e full check cancellation',
  }, headers);
  expect(operation.operation_type).toBe('cancellation');
  expect(operation.operation_kind).toBe('full');
  expect(operation.status).toBe('recorded');
  expect(operation.inventory_disposition).toBe('manual_review');
  expect(operation.amount).toBe(precheck.total);

  const afterCancellation = await get<Order>(request, `/orders/${order.id}`);
  expect(afterCancellation.check?.status).toBe('paid');
  expect(afterCancellation.check?.payments?.some((item) => item.id === payment.id && item.status === 'captured')).toBe(true);
});

test('full check refund records ledger operation without mutating finalized payment/check', async ({ request }) => {
  const { order, precheck } = await createOrderWithPrecheck(request, 1, 1);
  const payment = await capturePayment(request, precheck.id, 'card', precheck.total);
  const paidOrder = await get<Order>(request, `/orders/${order.id}`);
  expect(paidOrder.check?.id).toBeTruthy();

  await closeCurrentShiftAndCashSession(request);
  await ensureShiftAndCashSession(request);

  const operation = await post<FinancialOperation>(request, `/checks/${paidOrder.check?.id}/refunds`, {
    command_id: nextCommandID('refund-check'),
    operation_kind: 'full',
    inventory_disposition: 'return_to_stock',
    reason: 'e2e full check refund',
  }, headers);
  expect(operation.operation_type).toBe('refund');
  expect(operation.operation_kind).toBe('full');
  expect(operation.status).toBe('recorded');
  expect(operation.inventory_disposition).toBe('return_to_stock');
  expect(operation.amount).toBe(precheck.total);

  const afterRefund = await get<Order>(request, `/orders/${order.id}`);
  expect(afterRefund.check?.status).toBe('paid');
  expect(afterRefund.check?.payments?.some((item) => item.id === payment.id && item.status === 'captured')).toBe(true);
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

async function closeCurrentShiftAndCashSession(request: APIRequestContext) {
  const cashSession = await get<CashSession>(request, '/cash-shifts/current');
  await post<CashSession>(request, `/cash-shifts/${cashSession.id}/close`, {
    command_id: nextCommandID('close-cash-session'),
    closed_by_employee_id: headers['X-Actor-Employee-ID'],
    closing_cash_amount: 0,
  }, headers, [200]);

  const shift = await get<Shift>(request, '/employee-shifts/current');
  await post<Shift>(request, `/employee-shifts/${shift.id}/close`, {
    command_id: nextCommandID('close-shift'),
    closed_by_employee_id: headers['X-Actor-Employee-ID'],
  }, headers, [200]);
}

async function createOrderWithPrecheck(request: APIRequestContext, tableIndex: number, quantity: number) {
  const tableId = demo.table_ids[tableIndex % demo.table_ids.length];
  const menuItemId = demo.menu_item_ids[0];
  const order = await post<Order>(request, '/orders', {
    command_id: nextCommandID('create-order'),
    restaurant_id: demo.restaurant_id,
    table_id: tableId,
    table_name: `E2E-${tableIndex + 1}`,
    guest_count: 1,
  }, headers);

  await post(request, `/orders/${order.id}/lines`, {
    command_id: nextCommandID('add-line'),
    menu_item_id: menuItemId,
    quantity,
  }, headers);

  const precheck = await post<Precheck>(request, `/orders/${order.id}/precheck`, {
    command_id: nextCommandID('issue-precheck'),
  }, headers);
  expect(precheck.status).toBe('issued');
  expect(precheck.total).toBeGreaterThan(0);
  return { order, precheck };
}

async function capturePayment(request: APIRequestContext, precheckId: string, method: 'cash' | 'card', amount: number) {
  return post<Payment>(request, `/prechecks/${precheckId}/payments`, {
    command_id: nextCommandID(`capture-${method}`),
    method,
    amount,
    currency: 'RUB',
    provider_name: method === 'card' ? 'trusted_manual' : undefined,
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

function nextCommandID(prefix: string) {
  commandSequence += 1;
  return `cmd-e2e-${Date.now()}-${commandSequence}-${prefix}`;
}
