import { expect, test, type APIRequestContext } from '@playwright/test';
import { execFileSync } from 'node:child_process';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const cloudBase = (process.env.POS_E2E_CLOUD_BASE ?? 'http://127.0.0.1:8090/api/v1').replace(/\/$/, '');
const edgeBase = (process.env.POS_E2E_API_BASE ?? 'http://127.0.0.1:8080/api/v1').replace(/\/$/, '');
const repoRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..', '..');
const clientDeviceId = 'sync-provisioning-e2e-client';

type AuthHeaders = Record<string, string>;

const managerPermissions = [
  'pos.employee_shift.open',
  'pos.employee_shift.close',
  'pos.employee_shift.view_current',
  'pos.employee_shift.recent',
  'pos.cash_session.open',
  'pos.cash_session.close',
  'pos.cash_session.view_current',
  'pos.cash_drawer.record_event',
  'pos.catalog.view',
  'pos.floor.view',
  'pos.menu.view',
  'pos.order.create',
  'pos.order.view',
  'pos.order.add_line',
  'pos.order.change_quantity',
  'pos.order.void_line',
  'pos.order.close',
  'pos.precheck.issue',
  'pos.precheck.view',
  'pos.precheck.reprint',
  'pos.precheck.cancel.request',
  'pos.precheck.cancel',
  'pos.payment.cash',
  'pos.payment.card.manual',
  'pos.payment.other',
  'pos.payment.refund',
  'pos.check.view',
  'pos.check.reprint',
  'pos.sync.view',
  'pos.sync.retry_failed',
];

function commandId(prefix: string) {
  return `cmd-sync-provisioning-${prefix}-${Date.now()}-${Math.random().toString(16).slice(2)}`;
}

function permissionsJSON(values: string[]) {
  return JSON.stringify(Object.fromEntries([...new Set(values)].sort().map((item) => [item, true])));
}

async function postJSON(request: APIRequestContext, url: string, data: unknown, expected = 201) {
  const resp = await request.post(url, { data });
  expect(resp.status(), await resp.text()).toBe(expected);
  return resp.json();
}

async function getJSON(request: APIRequestContext, url: string, expected = 200) {
  const resp = await request.get(url);
  expect(resp.status(), await resp.text()).toBe(expected);
  return resp.json();
}

async function login(request: APIRequestContext, nodeDeviceId: string, pin: string): Promise<AuthHeaders> {
  const resp = await request.post(`${edgeBase}/auth/pin-login`, {
    data: { node_device_id: nodeDeviceId, client_device_id: clientDeviceId, pin },
  });
  expect(resp.status(), await resp.text()).toBe(201);
  const body = await resp.json();
  return {
    'X-Node-Device-ID': nodeDeviceId,
    'X-Client-Device-ID': clientDeviceId,
    'X-Actor-Employee-ID': body.actor.employee_id,
    'X-Session-ID': body.session.id,
  };
}

async function ensureOperationalState(request: APIRequestContext, headers: AuthHeaders, restaurantId: string, actorEmployeeId: string) {
  const currentShift = await request.get(`${edgeBase}/employee-shifts/current`, { headers });
  if (currentShift.status() !== 200 || (await currentShift.text()).trim() === 'null') {
    const openShift = await request.post(`${edgeBase}/employee-shifts/open`, {
      headers,
      data: {
        command_id: commandId('open-shift'),
        restaurant_id: restaurantId,
        opened_by_employee_id: actorEmployeeId,
      },
    });
    expect([201, 409]).toContain(openShift.status());
  }

  const currentCash = await request.get(`${edgeBase}/cash-shifts/current`, { headers });
  if (currentCash.status() !== 200) {
    const openCash = await request.post(`${edgeBase}/cash-shifts/open`, {
      headers,
      data: {
        command_id: commandId('open-cash'),
        restaurant_id: restaurantId,
        opened_by_employee_id: actorEmployeeId,
        opening_cash_amount: 0,
      },
    });
    expect([201, 409]).toContain(openCash.status());
  }
}

function cloudLogsSince(markerISO: string) {
  try {
    return execFileSync('docker', ['compose', '-f', 'docker-compose.local.yml', 'logs', 'cloud-api', '--since', markerISO], {
      cwd: repoRoot,
      encoding: 'utf-8',
    });
  } catch {
    return null;
  }
}

test('pairing opens exchange sync and stops repeated Cloud registration polling', async ({ request }) => {
  const suffix = Math.random().toString(16).slice(2, 10);
  const managerPin = '2222';

  const initialStatus = await getJSON(request, `${edgeBase}/system/provisioning-status`);
  test.skip(initialStatus.paired === true, 'Provisioning e2e requires an unpaired POS Edge; seeded stack is already paired');
  const nodeDeviceId = initialStatus.node_device_id as string;
  expect(nodeDeviceId).toBeTruthy();

  const restaurant = await postJSON(request, `${cloudBase}/restaurants`, {
    name: `Sync Pairing Bistro ${suffix}`,
    timezone: 'Europe/Moscow',
    currency: 'RUB',
    business_day_mode: 'standard',
    business_day_boundary_local_time: '04:00',
  });
  const managerRole = await postJSON(request, `${cloudBase}/roles`, {
    restaurant_id: restaurant.id,
    name: `manager-${suffix}`,
    permissions_json: permissionsJSON(managerPermissions),
  });
  const manager = await postJSON(request, `${cloudBase}/employees`, {
    restaurant_id: restaurant.id,
    role_id: managerRole.id,
    name: 'Sync Manager',
    pin: managerPin,
  });
  const hall = await postJSON(request, `${cloudBase}/halls`, {
    restaurant_id: restaurant.id,
    name: 'Main Hall',
  });
  const table = await postJSON(request, `${cloudBase}/tables`, {
    restaurant_id: restaurant.id,
    hall_id: hall.id,
    name: 'T1',
    seats: 2,
  });
  const catalog = await postJSON(request, `${cloudBase}/catalog/items`, {
    restaurant_id: restaurant.id,
    type: 'dish',
    name: 'Sync Soup',
    sku: `SYNC-SOUP-${suffix}`,
    base_unit: 'portion',
  });
  const menuItem = await postJSON(request, `${cloudBase}/menu/items`, {
    restaurant_id: restaurant.id,
    catalog_item_id: catalog.id,
    name: 'Sync Soup',
    price: 25000,
    currency: 'RUB',
    availability_json: '{}',
  });

  await postJSON(request, `${cloudBase}/restaurants/${restaurant.id}/master-data/publish`, {
    published_by: 'sync-provisioning-e2e',
    node_device_id: nodeDeviceId,
  });
  const pairing = await postJSON(request, `${cloudBase}/restaurants/${restaurant.id}/devices/generate-pairing-code`, {
    node_device_id: nodeDeviceId,
    display_name: 'POS Terminal E2E',
    expires_in_minutes: 30,
  });

  const paired = await request.post(`${edgeBase}/system/provisioning/pair-via-license`, {
    data: { pairing_code: pairing.pairing_code },
  });
  expect(paired.status(), await paired.text()).toBe(200);

  await expect.poll(async () => {
    const status = await request.get(`${edgeBase}/system/provisioning-status`);
    if (status.status() !== 200) {
      return false;
    }
    const body = await status.json();
    return body.paired === true && body.status === 'paired';
  }, { timeout: 30_000 }).toBe(true);

  const marker = new Date().toISOString();
  await new Promise((resolve) => setTimeout(resolve, 7_000));
  const logs = cloudLogsSince(marker);
  if (logs !== null) {
    expect(logs).not.toContain('"action":"POST /api/v1/devices/register"');
    expect(logs).not.toContain('/master-data/snapshot');
    const emptyExchangePosts = logs.match(/"action":"POST \/api\/v1\/sync\/exchange"/g)?.length ?? 0;
    expect(emptyExchangePosts).toBeLessThanOrEqual(1);
  }

  const headers = await login(request, nodeDeviceId, managerPin);
  await ensureOperationalState(request, headers, restaurant.id, manager.id);
  const orderCommandID = commandId('create-order');
  const order = await request.post(`${edgeBase}/orders`, {
    headers,
    data: {
      command_id: orderCommandID,
      restaurant_id: restaurant.id,
      table_id: table.id,
      table_name: 'T1',
      guest_count: 1,
    },
  });
  expect(order.status(), await order.text()).toBe(201);
  const orderBody = await order.json();

  const lineCommandID = commandId('add-line');
  const line = await request.post(`${edgeBase}/orders/${orderBody.id}/lines`, {
    headers,
    data: {
      command_id: lineCommandID,
      menu_item_id: menuItem.id,
      quantity: 1,
    },
  });
  expect(line.status(), await line.text()).toBe(201);

  await expect.poll(async () => {
    const events = await request.get(`${cloudBase}/sync/edge-events?restaurant_id=${restaurant.id}&event_type=OrderLineAdded&limit=50`);
    if (events.status() !== 200) {
      return false;
    }
    const items = await events.json();
    return Array.isArray(items) && items.some((item) => item.command_id === lineCommandID);
  }, { timeout: 60_000 }).toBe(true);
});
