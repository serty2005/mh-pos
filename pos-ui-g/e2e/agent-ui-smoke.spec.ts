import { expect, test, type Page, type TestInfo } from '@playwright/test';

const now = '2026-06-01T00:00:00.000Z';

test.beforeEach(async ({ page }) => {
  const runtimeErrors: string[] = [];
  page.on('pageerror', (error) => runtimeErrors.push(error.message));
  page.on('console', (message) => {
    if (message.type() === 'error') runtimeErrors.push(message.text());
  });
  await page.exposeFunction('__agentRuntimeErrors', () => runtimeErrors);
});

test.afterEach(async ({ page }) => {
  const runtimeErrors = await page.evaluate(() => window.__agentRuntimeErrors());
  expect(runtimeErrors).toEqual([]);
});

test('agent smoke opens pairing UI, clicks controls, and keeps design invariants', async ({ page }, testInfo) => {
  await mockPosBackend(page, { paired: false });
  await page.goto('/');

  await expect(page).toHaveTitle('MyHoreca POS');
  await expect(page.locator('#root')).toContainText('MyHoreca POS');
  await expect(page.locator('#pair-mode-cloud-btn')).toBeVisible();

  await page.locator('#pair-mode-license-btn').click();
  await expect(page.locator('#pair-license-code-input')).toBeVisible();
  await page.locator('#pair-license-code-input').fill('mhpos-agent-check');
  await expect(page.locator('#pair-license-code-input')).toHaveValue('MHPOS-AGENT-CHECK');
  await expect(page.locator('#pair-license-submit-btn')).toBeEnabled();

  await page.locator('#pair-mode-cloud-btn').click();
  await expect(page.locator('#pair-register-cloud-btn')).toBeVisible();
  await expect(page.locator('#pair-refresh-status-btn')).toBeVisible();

  await expectFrontendDesignInvariants(page);
  await saveViewportScreenshot(page, testInfo, 'agent-pairing-desktop.png');
});

test('agent smoke opens POS shell, navigates by clicks, and validates desktop/mobile layout', async ({ page }, testInfo) => {
  await mockPosBackend(page, { paired: true });
  await seedUnlockedSession(page);
  await page.goto('/');

  await expect(page.locator('header')).toContainText('MyHoreca POS');
  await expect(page.locator('#nav-floor')).toBeVisible();

  await page.locator('#nav-cash').click();
  await expect(page.locator('main')).toContainText('Операции кассы и смены');

  await page.locator('#sidemenu-trigger-btn').click();
  await expect(page.locator('#drawer-mode-kds')).toBeVisible();
  await page.locator('#drawer-theme-light-btn').click();
  await expect(page.locator('#drawer-theme-light-btn')).toHaveAttribute('aria-pressed', 'true');
  await page.locator('#drawer-mode-kds').click();
  await expect(page.locator('#nav-kds-orders')).toBeVisible();
  await page.locator('#nav-kds-kitchen').click();
  await expect(page.locator('main')).toContainText('Кухня');

  await expectFrontendDesignInvariants(page);
  await saveViewportScreenshot(page, testInfo, 'agent-shell-desktop.png');

  await page.setViewportSize({ width: 390, height: 844 });
  await expect(page.locator('#waiter-mobile-runtime')).toBeVisible();
  await expectFrontendDesignInvariants(page);
  await saveViewportScreenshot(page, testInfo, 'agent-shell-mobile.png');
});

async function mockPosBackend(page: Page, options: { paired: boolean; entitlements?: Record<string, boolean> }) {
  await page.route('**/api/v1/**', async (route) => {
    const url = new URL(route.request().url());
    const path = url.pathname.replace(/^\/api\/v1/, '');
    if (route.request().method() === 'OPTIONS') {
      await route.fulfill({ status: 204, headers: corsHeaders() });
      return;
    }

    const body = responseFor(path, options);
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      headers: corsHeaders(),
      body: JSON.stringify(body),
    });
  });
}

async function seedUnlockedSession(page: Page) {
  await page.addInitScript(() => {
    localStorage.setItem('mh-pos.node_device_id', 'node-agent-e2e');
    localStorage.setItem('mh-pos.restaurant_id', 'rest-agent-e2e');
    localStorage.setItem('mh-pos.session_id', 'session-agent-e2e');
  });
}

function responseFor(path: string, options: { paired: boolean; entitlements?: Record<string, boolean> }) {
  const paired = options.paired;
  if (path === '/system/pairing-status') {
    return paired
      ? {
          paired: true,
          node_device_id: 'node-agent-e2e',
          restaurant_id: 'rest-agent-e2e',
          identity: {
            id: 'edge-agent-e2e',
            node_device_id: 'node-agent-e2e',
            restaurant_id: 'rest-agent-e2e',
            status: 'paired',
            paired_at: now,
          },
        }
      : { paired: false, node_device_id: 'node-agent-e2e' };
  }
  if (path === '/system/provisioning-status') return provisioningStatus(paired);
  if (path === '/license/entitlements') return entitlementSnapshot(options.entitlements);
  if (path === '/system/provisioning/register-cloud') return provisioningStatus(false);
  if (path === '/system/provisioning/pair-via-license') return provisioningStatus(true);
  if (path.startsWith('/auth/session') || path === '/auth/pin-login') return authResult();
  if (path === '/auth/logout') return {};
  if (path.startsWith('/employee-shifts/current')) return shift();
  if (path.startsWith('/employee-shifts/recent')) return [shift()];
  if (path.startsWith('/cash-shifts/current')) return cashSession();
  if (path.startsWith('/halls')) return halls();
  if (path.startsWith('/tables')) return tables();
  if (path === '/menu/items') return menuItems();
  if (path.startsWith('/orders/active')) return [];
  if (path.startsWith('/orders/closed')) return [];
  if (path === '/pricing/policies') return [];
  if (path === '/sync/status') return syncStatus();
  if (path === '/storage/status') return storageStatus();
  if (path.startsWith('/kitchen/order-queue')) return kitchenQueue();
  if (path.startsWith('/kitchen/tickets')) return kitchenQueue().orders.flatMap((order) => order.tickets);
  if (path === '/catalog/items') return catalogItems();
  if (path === '/kitchen/stop-list') return [];
  if (path.startsWith('/kitchen/proposals')) return [];
  if (path.includes('/recipe')) return kitchenRecipe();
  return {};
}

function entitlementSnapshot(entitlements: Record<string, boolean> = {
  'table-mode': true,
  'kitchen-space': true,
  'waiter-space': true,
  'warehouse-mode': true,
}) {
  return {
    tenant_id: 'tenant-agent-e2e',
    server_id: 'license-agent-e2e',
    version: 1,
    status: 'active',
    entitlements,
    issued_at: now,
    expires_at: '2099-01-01T00:00:00.000Z',
  };
}

function provisioningStatus(paired: boolean) {
  return {
    node_device_id: 'node-agent-e2e',
    cloud_url: 'http://cloud-agent-e2e',
    license_url: 'http://license-agent-e2e',
    restaurant_id: paired ? 'rest-agent-e2e' : '',
    status: paired ? 'paired' : 'not_configured',
    paired,
  };
}

function authResult() {
  return {
    session: {
      id: 'session-agent-e2e',
      restaurant_id: 'rest-agent-e2e',
      node_device_id: 'node-agent-e2e',
      client_device_id: 'client-agent-e2e',
      employee_id: 'employee-manager-e2e',
      status: 'active',
      started_at: now,
      last_seen_at: now,
    },
    actor: {
      employee_id: 'employee-manager-e2e',
      restaurant_id: 'rest-agent-e2e',
      role_id: 'role-manager-e2e',
      name: 'Agent Manager',
      permissions: [
        'pos.floor.view',
        'pos.menu.view',
        'pos.order.view',
        'pos.payment.cash',
        'pos.payment.refund',
        'pos.precheck.cancel',
        'pos.sync.view',
        'pos.kitchen.view',
        'pos.kitchen.catalog.view',
        'pos.kitchen.recipe.view',
      ],
    },
    permissions: [],
  };
}

function shift() {
  return {
    id: 'shift-agent-e2e',
    restaurant_id: 'rest-agent-e2e',
    device_id: 'node-agent-e2e',
    opened_by_employee_id: 'employee-manager-e2e',
    closed_by_employee_id: null,
    status: 'open',
    opened_at: now,
    closed_at: null,
    opening_cash_amount: 5000,
    closing_cash_amount: null,
    created_at: now,
    updated_at: now,
  };
}

function cashSession() {
  return {
    id: 'cash-agent-e2e',
    edge_cash_session_id: 'cash-agent-e2e',
    restaurant_id: 'rest-agent-e2e',
    device_id: 'node-agent-e2e',
    shift_id: 'shift-agent-e2e',
    opened_by_employee_id: 'employee-manager-e2e',
    closed_by_employee_id: null,
    status: 'open',
    opening_cash_amount: 5000,
    closing_cash_amount: null,
    opened_at: now,
    closed_at: null,
    created_at: now,
    updated_at: now,
  };
}

function halls() {
  return [{ id: 'hall-main', restaurant_id: 'rest-agent-e2e', name: 'Main Hall', active: true }];
}

function tables() {
  return [
    { id: 'table-1', restaurant_id: 'rest-agent-e2e', hall_id: 'hall-main', name: 'Table 1', seats: 4, active: true },
    { id: 'table-2', restaurant_id: 'rest-agent-e2e', hall_id: 'hall-main', name: 'Table 2', seats: 2, active: true },
  ];
}

function menuItems() {
  return [
    {
      id: 'menu-tea',
      catalog_item_id: 'catalog-tea',
      item_type: 'dish',
      name: 'Production Tea',
      price: 18000,
      currency: 'RUB',
      modifier_groups: [],
      active: true,
      created_at: now,
      updated_at: now,
    },
  ];
}

function syncStatus() {
  return { total: 0, pending: 0, processing: 0, sent: 0, failed: 0, suspended: 0, last_cloud_version: 42 };
}

function storageStatus() {
  return {
    generated_at: now,
    sqlite: {
      page_count: 1,
      page_size_bytes: 4096,
      freelist_count: 0,
      estimated_size_bytes: 4096,
      freelist_bytes: 0,
      journal_mode: 'wal',
    },
    tables: {},
    closed_order_business_date_range: {},
    closed_orders_by_business_date: [],
    outbox: [],
    blocking_outbox_messages: 0,
    retention: {
      mode: 'dev',
      destructive_apply_supported: false,
      financial_ledger_protected: true,
      immutable_snapshots_protected: true,
      reason: 'agent e2e smoke',
    },
    runtime_versions: [{ module_name: 'pos-backend', module_version: 'agent', schema_version: '1' }],
    schema_migrations: [],
  };
}

function kitchenQueue() {
  return {
    orders: [
      {
        order_id: 'order-kitchen-e2e',
        edge_order_id: 'order-kitchen-e2e',
        table_name: 'Table 1',
        kitchen_order_status: 'queued',
        created_at: now,
        elapsed_seconds: 180,
        tickets: [
          {
            id: 'ticket-agent-e2e',
            order_id: 'order-kitchen-e2e',
            order_line_id: 'line-agent-e2e',
            table_name: 'Table 1',
            name: 'Production Tea',
            quantity: 1,
            unit_code: 'PORTION',
            station_routing_key: 'hot',
            course: null,
            comment: null,
            status: 'new',
            created_at: now,
            updated_at: now,
          },
        ],
      },
    ],
    limit: 100,
    offset: 0,
  };
}

function catalogItems() {
  return [
    { id: 'catalog-tea', name: 'Production Tea', item_type: 'dish', sku: 'TEA-001', base_unit: 'PORTION', active: true },
    { id: 'catalog-lemon', name: 'Lemon', item_type: 'good', sku: 'LEM-001', base_unit: 'KG', active: true },
  ];
}

function kitchenRecipe() {
  return {
    catalog_item_id: 'catalog-tea',
    catalog_item: catalogItems()[0],
    recipe_version_id: 'recipe-agent-e2e',
    ingredients: [],
    lines: [],
    proposals: [],
  };
}

async function expectFrontendDesignInvariants(page: Page) {
  await expect(page.locator('#root')).not.toBeEmpty();
  await expect(page.locator('body')).not.toContainText(/vite|webpack|uncaught|stack trace/i);

  const horizontalOverflow = await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth);
  expect(horizontalOverflow).toBeLessThanOrEqual(2);

  const badTargets = await page.locator('button:visible, [role="button"]:visible').evaluateAll((items) => items
    .map((item) => {
      const rect = item.getBoundingClientRect();
      return { label: item.textContent?.trim() || item.getAttribute('aria-label') || item.id, width: rect.width, height: rect.height };
    })
    .filter((item) => item.width > 0 && item.height > 0 && (item.width < 40 || item.height < 40)));
  expect(badTargets).toEqual([]);

  const overflowingButtons = await page.locator('button:visible').evaluateAll((items) => items
    .filter((item) => item.scrollWidth - item.clientWidth > 2 || item.scrollHeight - item.clientHeight > 2)
    .map((item) => item.textContent?.trim() || item.getAttribute('aria-label') || item.id));
  expect(overflowingButtons).toEqual([]);
}

async function saveViewportScreenshot(page: Page, testInfo: TestInfo, name: string) {
  const path = testInfo.outputPath(name);
  await page.screenshot({ path, fullPage: false });
  await testInfo.attach(name, { path, contentType: 'image/png' });
}

function corsHeaders() {
  return {
    'Access-Control-Allow-Origin': '*',
    'Access-Control-Allow-Headers': '*',
    'Access-Control-Allow-Methods': 'GET,POST,PATCH,PUT,DELETE,OPTIONS',
  };
}

declare global {
  interface Window {
    __agentRuntimeErrors(): string[];
  }
}
