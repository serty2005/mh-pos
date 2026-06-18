import { expect, test, type Page } from '@playwright/test';

const now = '2026-06-16T10:00:00.000Z';

test('mobile waiter runs order and precheck flow without financial endpoints', async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  const backend = new WaiterBackendMock();
  const forbiddenRequests: string[] = [];
  const addLinePayloads: unknown[] = [];

  await page.route('**/api/v1/**', async (route) => {
    const request = route.request();
    const url = new URL(request.url());
    const path = url.pathname.replace(/^\/api\/v1/, '');
    if (isForbiddenWaiterEndpoint(path)) {
      forbiddenRequests.push(`${request.method()} ${path}`);
    }
    if (request.method() === 'POST' && /\/orders\/[^/]+\/lines$/.test(path)) {
      addLinePayloads.push(JSON.parse(request.postData() || '{}'));
    }
    const response = backend.responseFor(path, request.method(), request.postData() || '');
    await route.fulfill({
      status: response.status ?? 200,
      contentType: 'application/json',
      headers: { 'Access-Control-Allow-Origin': '*' },
      body: JSON.stringify(response.body),
    });
  });

  await page.addInitScript(() => {
    localStorage.removeItem('mh-pos.session_id');
  });

  await page.goto('/');
  await page.locator('#pin-btn-1').waitFor();
  for (const digit of ['#pin-btn-1', '#pin-btn-1', '#pin-btn-1', '#pin-btn-1']) {
    await page.locator(digit).click();
  }
  await page.locator('#pin-submit-btn').click();

  await expect(page.locator('#waiter-mobile-runtime')).toBeVisible({ timeout: 15_000 });
  await expect(page.getByText('Доступ к Waiter-экрану')).toHaveCount(0);
  await expect(page.getByText('Выберите зал')).toBeVisible();
  await expect(page.locator('#waiter-table-table-1')).toBeVisible();

  await page.locator('#waiter-table-table-1').click();
  await expect(page.locator('#waiter-create-order-btn')).toBeEnabled();
  await page.locator('#waiter-create-order-btn').click();
  await expect(page.getByRole('heading', { name: /Table 1 #/ })).toBeVisible();
  await expect(page.locator('#waiter-menu-search')).toBeVisible();

  await page.locator('#waiter-menu-search').fill('Latte');
  await page.locator('#waiter-menu-item-menu-latte').click();
  await expect(page.locator('#modifier-submit-btn')).toBeVisible();
  await page.locator('#mod-opt-mod-oat').click();
  await page.locator('#modifier-submit-btn').click();

  await expect.poll(() => addLinePayloads.length).toBe(1);
  expect(addLinePayloads[0]).toMatchObject({
    selected_modifiers: [
      {
        modifier_group_id: 'mod-milk',
        modifier_option_id: 'mod-oat',
        quantity: 1,
      },
    ],
  });
  await expect(page.locator('#waiter-order-line-line-latte').getByText('Oat milk')).toBeVisible();

  await page.locator('#waiter-menu-search').fill('Soup');
  await page.locator('#waiter-menu-item-menu-soup').click();
  await expect.poll(() => backend.activeOrder()?.lines.length ?? 0).toBe(2);

  await expect(page.locator('#waiter-line-qty-line-latte-increase')).toBeEnabled();
  await page.locator('#waiter-line-qty-line-latte-increase').click();
  await expect.poll(() => backend.lineQuantity('line-latte')).toBe(2);

  await expect(page.locator('#waiter-void-line-line-soup')).toBeEnabled();
  await page.locator('#waiter-void-line-line-soup').click();
  await page.locator('#waiter-void-reason-input').fill('guest changed order');
  await page.locator('#waiter-confirm-void-btn').click();
  await expect.poll(() => backend.activeOrder()?.lines.filter((line) => line.status === 'active').length ?? 0).toBe(1);

  await expect(page.locator('#waiter-issue-precheck-btn')).toBeEnabled();
  await page.locator('#waiter-issue-precheck-btn').click();
  await expect(page.getByText('Выпущен активный пречек. Добавление, изменение количества и списание позиций недоступны.').first()).toBeVisible();
  await expect(page.locator('#waiter-line-qty-line-latte-increase')).toBeDisabled();
  await expect(page.locator('#waiter-void-line-line-latte')).toBeDisabled();
  await expect(page.locator('#waiter-menu-item-menu-soup')).toBeDisabled();

  await expect(page.locator('#waiter-reprint-precheck-btn')).toBeVisible();
  await page.locator('#waiter-reprint-precheck-btn').click();
  await expect.poll(() => backend.reprintCount).toBe(1);

  await expect(page.getByRole('button', { name: /оплата|возврат|кассовый|фиск/i })).toHaveCount(0);
  expect(forbiddenRequests).toEqual([]);
});

function isForbiddenWaiterEndpoint(path: string) {
  return path.includes('/payments')
    || path.includes('/refund')
    || path.includes('/financial-operations')
    || path.includes('/cash-drawer-events')
    || path.includes('/cash-shifts');
}

type MockLine = {
  id: string;
  order_id: string;
  menu_item_id: string;
  catalog_item_id: string;
  name: string;
  quantity: number;
  unit_price: number;
  total_price: number;
  currency_code: string;
  tax_profile_id: string | null;
  course: string | null;
  comment: string | null;
  modifiers: Array<{
    id: string;
    order_line_id: string;
    modifier_group_id: string;
    modifier_option_id: string;
    name: string;
    quantity: number;
    unit_price: number;
    total_price: number;
  }>;
  status: 'active' | 'cancelled' | 'voided';
  created_at: string;
  updated_at: string;
};

type MockOrder = {
  id: string;
  edge_order_id: string;
  restaurant_id: string;
  device_id: string;
  shift_id: string;
  status: 'open' | 'locked' | 'closed' | 'cancelled';
  table_id: string;
  table_name: string;
  guest_count: number;
  subtotal: number;
  discount_total: number;
  tax_total: number;
  total: number;
  opened_at: string;
  closed_at: string | null;
  created_at: string;
  updated_at: string;
  lines: MockLine[];
};

class WaiterBackendMock {
  orders: MockOrder[] = [];
  prechecks: unknown[] = [];
  reprintCount = 0;

  responseFor(path: string, method: string, postData: string) {
    if (method === 'OPTIONS') return { status: 204, body: {} };
    if (path === '/system/pairing-status') return { body: pairingStatus() };
    if (path === '/system/provisioning-status') return { body: provisioningStatus() };
    if (path === '/auth/pin-login' || path.startsWith('/auth/session')) return { body: authResult() };
    if (path === '/auth/logout') return { body: {} };
    if (path.startsWith('/employee-shifts/current')) return { body: shift() };
    if (path.startsWith('/employee-shifts/recent')) return { body: [shift()] };
    if (path.startsWith('/halls')) return { body: halls() };
    if (path.startsWith('/tables')) return { body: tables() };
    if (path === '/menu/items') return { body: menuItems() };
    if (path === '/pricing/policies') return { body: [] };
    if (path.startsWith('/orders/closed')) return { body: [] };
    if (path.startsWith('/orders/active')) return { body: this.orders };
    if (path === '/orders' && method === 'POST') return { status: 201, body: this.createOrder(postData) };

    const precheckMatch = path.match(/^\/orders\/([^/]+)\/precheck$/);
    if (precheckMatch && method === 'POST') return { status: 201, body: this.issuePrecheck(precheckMatch[1]) };

    const prechecksMatch = path.match(/^\/orders\/([^/]+)\/prechecks$/);
    if (prechecksMatch) return { body: this.prechecks };

    const pricingMatch = path.match(/^\/orders\/([^/]+)\/pricing$/);
    if (pricingMatch) return { body: this.pricing(pricingMatch[1]) };

    const addLineMatch = path.match(/^\/orders\/([^/]+)\/lines$/);
    if (addLineMatch && method === 'POST') return { status: 201, body: this.addLine(addLineMatch[1], postData) };

    const quantityMatch = path.match(/^\/orders\/([^/]+)\/lines\/([^/]+)$/);
    if (quantityMatch && method === 'PATCH') return { body: this.changeQuantity(quantityMatch[1], quantityMatch[2], postData) };

    const voidMatch = path.match(/^\/orders\/([^/]+)\/lines\/([^/]+)\/void$/);
    if (voidMatch && method === 'POST') return { body: this.voidLine(voidMatch[1], voidMatch[2]) };

    const reprintMatch = path.match(/^\/prechecks\/([^/]+)\/reprint$/);
    if (reprintMatch && method === 'POST') {
      this.reprintCount += 1;
      return { body: reprintDocument(reprintMatch[1]) };
    }

    return { body: {} };
  }

  activeOrder() {
    return this.orders[0] ?? null;
  }

  lineQuantity(lineId: string) {
    return this.activeOrder()?.lines.find((line) => line.id === lineId)?.quantity ?? 0;
  }

  emptyOrder(tableId: string, tableName: string, guestCount: number): MockOrder {
    return {
      id: 'order-waiter-1',
      edge_order_id: 'edge-order-waiter-1',
      restaurant_id: 'rest-waiter-e2e',
      device_id: 'node-waiter-e2e',
      shift_id: 'shift-waiter-e2e',
      status: 'open',
      table_id: tableId,
      table_name: tableName,
      guest_count: guestCount,
      subtotal: 0,
      discount_total: 0,
      tax_total: 0,
      total: 0,
      opened_at: now,
      closed_at: null,
      created_at: now,
      updated_at: now,
      lines: [] as MockLine[],
    };
  }

  createOrder(postData: string) {
    const body = JSON.parse(postData || '{}');
    const order = this.emptyOrder(body.table_id, body.table_name, body.guest_count);
    this.orders = [order];
    return order;
  }

  addLine(orderId: string, postData: string) {
    const order = this.orders.find((item) => item.id === orderId);
    if (!order) return {};
    const body = JSON.parse(postData || '{}');
    const item = menuItems().find((menuItem) => menuItem.id === body.menu_item_id) ?? menuItems()[0];
    const selectedModifiers = (body.selected_modifiers ?? []) as Array<{ modifier_group_id: string; modifier_option_id: string; quantity: number }>;
    const modifiers = selectedModifiers.map((modifier) => {
      const group = item.modifier_groups.find((candidate) => candidate.id === modifier.modifier_group_id);
      const option = group?.options.find((candidate) => candidate.id === modifier.modifier_option_id);
      return {
        id: `line-mod-${modifier.modifier_option_id}`,
        order_line_id: '',
        modifier_group_id: modifier.modifier_group_id,
        modifier_option_id: modifier.modifier_option_id,
        name: option?.name ?? modifier.modifier_option_id,
        quantity: modifier.quantity,
        unit_price: option?.price_minor ?? 0,
        total_price: (option?.price_minor ?? 0) * modifier.quantity,
      };
    });
    const modifierTotal = modifiers.reduce((sum, modifier) => sum + modifier.total_price, 0);
    const line: MockLine = {
      id: item.id === 'menu-latte' ? 'line-latte' : 'line-soup',
      order_id: order.id,
      menu_item_id: item.id,
      catalog_item_id: item.catalog_item_id,
      name: item.name,
      quantity: body.quantity ?? 1,
      unit_price: item.price,
      total_price: item.price * (body.quantity ?? 1) + modifierTotal,
      currency_code: item.currency,
      tax_profile_id: null,
      course: null,
      comment: null,
      modifiers: modifiers.map((modifier) => ({ ...modifier, order_line_id: item.id === 'menu-latte' ? 'line-latte' : 'line-soup' })),
      status: 'active',
      created_at: now,
      updated_at: now,
    };
    order.lines.push(line);
    this.recalculate(order);
    return line;
  }

  changeQuantity(orderId: string, lineId: string, postData: string) {
    const order = this.orders.find((item) => item.id === orderId);
    const line = order?.lines.find((item) => item.id === lineId);
    if (!order || !line) return {};
    const body = JSON.parse(postData || '{}');
    line.quantity = body.quantity;
    const modifierTotal = line.modifiers.reduce((sum, modifier) => sum + modifier.total_price, 0);
    line.total_price = line.unit_price * line.quantity + modifierTotal;
    this.recalculate(order);
    return line;
  }

  voidLine(orderId: string, lineId: string) {
    const order = this.orders.find((item) => item.id === orderId);
    const line = order?.lines.find((item) => item.id === lineId);
    if (!order || !line) return {};
    line.status = 'voided';
    this.recalculate(order);
    return line;
  }

  issuePrecheck(orderId: string) {
    const order = this.orders.find((item) => item.id === orderId);
    if (order) order.status = 'locked';
    const precheck = {
      id: 'precheck-waiter-1',
      order_id: orderId,
      status: 'issued',
      version: 1,
      supersedes_precheck_id: null,
      currency_code: 'RUB',
      subtotal: order?.subtotal ?? 0,
      discount_total: 0,
      surcharge_total: 0,
      tax_total: order?.tax_total ?? 0,
      total: order?.total ?? 0,
      paid_total: 0,
      remaining_total: order?.total ?? 0,
      snapshot: {},
      created_at: now,
      issued_at: now,
      closed_at: null,
      cancelled_by_employee_id: null,
      cancellation_reason: null,
    };
    this.prechecks = [precheck];
    return precheck;
  }

  pricing(orderId: string) {
    const order = this.orders.find((item) => item.id === orderId);
    return {
      subtotal_minor: order?.subtotal ?? 0,
      discount_total_minor: 0,
      surcharge_total_minor: 0,
      tax_total_minor: order?.tax_total ?? 0,
      grand_total_minor: order?.total ?? 0,
      lines: order?.lines.filter((line) => line.status === 'active').map((line) => ({
        order_line_id: line.id,
        subtotal_minor: line.total_price,
        discount_total_minor: 0,
        surcharge_total_minor: 0,
        tax_total_minor: 0,
        total_minor: line.total_price,
      })) ?? [],
      discounts: [],
      surcharges: [],
    };
  }

  recalculate(order: MockOrder) {
    const activeLines = order.lines.filter((line) => line.status === 'active');
    order.subtotal = activeLines.reduce((sum, line) => sum + line.total_price, 0);
    order.tax_total = 0;
    order.total = order.subtotal;
  }
}

function pairingStatus() {
  return {
    paired: true,
    node_device_id: 'node-waiter-e2e',
    restaurant_id: 'rest-waiter-e2e',
    identity: {
      id: 'edge-waiter-e2e',
      node_device_id: 'node-waiter-e2e',
      restaurant_id: 'rest-waiter-e2e',
      status: 'paired',
      paired_at: now,
    },
  };
}

function provisioningStatus() {
  return {
    node_device_id: 'node-waiter-e2e',
    restaurant_id: 'rest-waiter-e2e',
    status: 'paired',
    paired: true,
  };
}

function authResult() {
  return {
    session: {
      id: 'session-waiter-e2e',
      restaurant_id: 'rest-waiter-e2e',
      node_device_id: 'node-waiter-e2e',
      client_device_id: 'client-waiter-e2e',
      employee_id: 'employee-waiter-e2e',
      status: 'active',
      started_at: now,
      last_seen_at: now,
    },
    actor: {
      employee_id: 'employee-waiter-e2e',
      restaurant_id: 'rest-waiter-e2e',
      role_id: 'role-waiter-e2e',
      name: 'Waiter E2E',
      permissions: [
        'pos.employee_shift.open',
        'pos.employee_shift.close',
        'pos.employee_shift.view_current',
        'pos.employee_shift.recent',
        'pos.catalog.view',
        'pos.floor.view',
        'pos.menu.view',
        'pos.order.create',
        'pos.order.view',
        'pos.order.add_line',
        'pos.order.change_quantity',
        'pos.order.void_line',
        'pos.precheck.issue',
        'pos.precheck.view',
        'pos.precheck.reprint',
      ],
    },
    permissions: [],
  };
}

function shift() {
  return {
    id: 'shift-waiter-e2e',
    restaurant_id: 'rest-waiter-e2e',
    device_id: 'node-waiter-e2e',
    opened_by_employee_id: 'employee-waiter-e2e',
    closed_by_employee_id: null,
    status: 'open',
    opened_at: now,
    closed_at: null,
    opening_cash_amount: 0,
    closing_cash_amount: null,
    created_at: now,
    updated_at: now,
  };
}

function halls() {
  return [{ id: 'hall-main', restaurant_id: 'rest-waiter-e2e', name: 'Main Hall', active: true }];
}

function tables() {
  return [
    { id: 'table-1', restaurant_id: 'rest-waiter-e2e', hall_id: 'hall-main', name: 'Table 1', seats: 4, active: true },
    { id: 'table-2', restaurant_id: 'rest-waiter-e2e', hall_id: 'hall-main', name: 'Table 2', seats: 2, active: true },
  ];
}

function menuItems() {
  return [
    {
      id: 'menu-latte',
      catalog_item_id: 'catalog-latte',
      item_type: 'dish',
      name: 'Latte',
      price: 22000,
      currency: 'RUB',
      modifier_groups: [
        {
          id: 'mod-milk',
          restaurant_id: 'rest-waiter-e2e',
          name: 'Milk',
          required: false,
          min_count: 0,
          max_count: 1,
          active: true,
          options: [
            {
              id: 'mod-oat',
              restaurant_id: 'rest-waiter-e2e',
              modifier_group_id: 'mod-milk',
              name: 'Oat milk',
              price_minor: 5000,
              active: true,
            },
          ],
        },
      ],
      active: true,
      created_at: now,
      updated_at: now,
    },
    {
      id: 'menu-soup',
      catalog_item_id: 'catalog-soup',
      item_type: 'dish',
      name: 'Soup',
      price: 18000,
      currency: 'RUB',
      modifier_groups: [],
      active: true,
      created_at: now,
      updated_at: now,
    },
  ];
}

function reprintDocument(precheckId: string) {
  return {
    document_type: 'precheck',
    source_id: precheckId,
    copy_marker: 'copy',
    actor_employee_id: 'employee-waiter-e2e',
    reprinted_at: now,
    snapshot: {},
  };
}
