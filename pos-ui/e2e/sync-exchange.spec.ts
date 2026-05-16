import { expect, test } from '@playwright/test';

const cloudBase = (process.env.POS_E2E_CLOUD_BASE ?? 'http://localhost:8090/api/v1').replace(/\/$/, '');
const edgeBase = (process.env.POS_E2E_API_BASE ?? 'http://localhost:8080/api/v1').replace(/\/$/, '');
const bootstrapJson = process.env.POS_E2E_BOOTSTRAP_JSON;
const nodeToken = process.env.POS_E2E_NODE_TOKEN;

type Bootstrap = {
  restaurant_id: string;
  node_device_id: string;
  manager_pin: string;
  table_ids?: string[];
  menu_item_ids?: string[];
};

test.describe.configure({ mode: 'serial' });

let demo: Bootstrap;
let headers: Record<string, string>;

test.beforeAll(async ({ playwright }) => {
  test.skip(!bootstrapJson, 'Run scripts/bootstrap-production-way.ps1 and pass POS_E2E_BOOTSTRAP_JSON');
  demo = JSON.parse(bootstrapJson ?? '{}') as Bootstrap;
  const request = await playwright.request.newContext();
  const login = await request.post(`${edgeBase}/auth/pin-login`, {
    data: {
      node_device_id: demo.node_device_id,
      client_device_id: 'sync-exchange-e2e-client',
      pin: demo.manager_pin,
    },
  });
  expect(login.status(), await login.text()).toBe(201);
  const body = await login.json();
  headers = {
    'X-Node-Device-ID': demo.node_device_id,
    'X-Client-Device-ID': 'sync-exchange-e2e-client',
    'X-Actor-Employee-ID': body.actor.employee_id,
    'X-Session-ID': body.session.id,
  };
  await request.dispose();
});

test('online worker exchange drains Edge outbox into Cloud receipt log', async ({ request }) => {
  let commandId = `cmd-sync-exchange-e2e-${Date.now()}`;
  let eventType = 'OrderCreated';
  const order = await request.post(`${edgeBase}/orders`, {
    headers,
    data: {
      command_id: commandId,
      restaurant_id: demo.restaurant_id,
      table_id: demo.table_ids?.[0],
      table_name: 'Sync E2E',
      guest_count: 1,
    },
  });
  if (order.status() === 409) {
    const current = await request.get(`${edgeBase}/orders/current?table_id=${demo.table_ids?.[0]}`, { headers });
    expect(current.status(), await current.text()).toBe(200);
    const currentOrder = await current.json();
    commandId = `cmd-sync-exchange-line-e2e-${Date.now()}`;
    eventType = 'OrderLineAdded';
    const line = await request.post(`${edgeBase}/orders/${currentOrder.id}/lines`, {
      headers,
      data: {
        command_id: commandId,
        menu_item_id: demo.menu_item_ids?.[0],
        quantity: 1,
      },
    });
    expect(line.status(), await line.text()).toBe(201);
  } else {
    expect(order.status(), await order.text()).toBe(201);
  }

  await expect.poll(async () => {
    const events = await request.get(`${cloudBase}/sync/edge-events?restaurant_id=${demo.restaurant_id}&event_type=${eventType}&limit=50`);
    if (events.status() !== 200) {
      return false;
    }
    const items = await events.json();
    return Array.isArray(items) && items.some((item) => item.command_id === commandId);
  }, { timeout: 45_000 }).toBe(true);
});

test('direct exchange returns partial ACK and idempotent replay for the same event', async ({ request }) => {
  test.skip(!nodeToken, 'Set POS_E2E_NODE_TOKEN to run direct authenticated exchange checks');
  const eventId = `event-sync-exchange-${Date.now()}`;
  const payload = {
    version: '1',
    event_id: eventId,
    command_id: `cmd-${eventId}`,
    event_type: 'OrderCreated',
    aggregate_type: 'Order',
    aggregate_id: `order-${eventId}`,
    restaurant_id: demo.restaurant_id,
    device_id: demo.node_device_id,
    node_device_id: demo.node_device_id,
    occurred_at: new Date().toISOString(),
    payload: {
      origin: 'edge_device',
      data: {
        id: `order-${eventId}`,
        edge_order_id: `edge-order-${eventId}`,
        restaurant_id: demo.restaurant_id,
        device_id: demo.node_device_id,
        shift_id: 'sync-exchange-e2e-shift',
        status: 'open',
        table_name: 'Sync E2E',
        guest_count: 1,
        opened_at: new Date().toISOString(),
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
      },
    },
  };
  const exchange = async () => request.post(`${cloudBase}/sync/exchange`, {
    headers: { Authorization: `Bearer ${nodeToken}` },
    data: {
      protocol_version: 'sync_exchange.v1',
      node_device_id: demo.node_device_id,
      restaurant_id: demo.restaurant_id,
      edge_events: [
        { client_item_id: 'e2e-valid', payload },
        { client_item_id: 'e2e-invalid', payload: { version: '1' } },
      ],
      streams: [{ stream_name: 'catalog', last_cloud_version: 0 }],
    },
  });

  const first = await exchange();
  expect(first.status(), await first.text()).toBe(202);
  const firstBody = await first.json();
  expect(firstBody.status).toBe('partial');
  expect(firstBody.edge_acks.find((item) => item.client_item_id === 'e2e-valid')?.status).toBe('accepted');
  expect(firstBody.edge_acks.find((item) => item.client_item_id === 'e2e-invalid')?.status).toBe('rejected');

  const replay = await exchange();
  expect(replay.status(), await replay.text()).toBe(202);
  const replayBody = await replay.json();
  expect(replayBody.edge_acks.find((item) => item.client_item_id === 'e2e-valid')?.ack.cloud_receipt_id)
    .toBe(firstBody.edge_acks.find((item) => item.client_item_id === 'e2e-valid')?.ack.cloud_receipt_id);
});
