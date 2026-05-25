import { expect, test, type APIRequestContext } from '@playwright/test';
import { execFileSync } from 'node:child_process';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

import { bootstrapRequiredMessage, loadBootstrapJson } from './support/bootstrap';

const cloudBase = (process.env.POS_E2E_CLOUD_BASE ?? 'http://localhost:8090/api/v1').replace(/\/$/, '');
const edgeBase = (process.env.POS_E2E_API_BASE ?? 'http://localhost:8080/api/v1').replace(/\/$/, '');
const bootstrapJson = loadBootstrapJson();
const nodeToken = process.env.POS_E2E_NODE_TOKEN?.trim();
const repoRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..', '..');

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
let actorEmployeeId: string;

test.beforeAll(async ({ playwright }) => {
  test.skip(!bootstrapJson, bootstrapRequiredMessage());
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
  actorEmployeeId = body.actor.employee_id;
  headers = {
    'X-Node-Device-ID': demo.node_device_id,
    'X-Client-Device-ID': 'sync-exchange-e2e-client',
    'X-Actor-Employee-ID': actorEmployeeId,
    'X-Session-ID': body.session.id,
  };
  await request.dispose();
});

async function ensureOperationalState(request: APIRequestContext) {
  const currentShift = await request.get(`${edgeBase}/employee-shifts/current`, { headers });
  if (currentShift.status() === 200 && (await currentShift.text()).trim() !== 'null') {
    return;
  }
  const openShift = await request.post(`${edgeBase}/employee-shifts/open`, {
    headers,
    data: {
      command_id: `cmd-sync-exchange-open-shift-${Date.now()}`,
      restaurant_id: demo.restaurant_id,
      opened_by_employee_id: actorEmployeeId,
    },
  });
  expect([201, 409]).toContain(openShift.status());

  const currentCash = await request.get(`${edgeBase}/cash-shifts/current`, { headers });
  if (currentCash.status() === 200) {
    return;
  }
  const openCash = await request.post(`${edgeBase}/cash-shifts/open`, {
    headers,
    data: {
      command_id: `cmd-sync-exchange-open-cash-${Date.now()}`,
      restaurant_id: demo.restaurant_id,
      opened_by_employee_id: actorEmployeeId,
      opening_cash_amount: 0,
    },
  });
  expect([201, 409]).toContain(openCash.status());
}

function loadCurrentNodeToken(): string {
  if (!dockerCLIAvailable()) return nodeToken ?? '';
  const tokenDir = path.join(os.tmpdir(), 'mh-pos-sync-exchange-e2e');
  const posEdgeContainer = process.env.POS_E2E_POS_EDGE_CONTAINER ?? 'mh-pos-local-pos-edge-1';
  try {
    fs.mkdirSync(tokenDir, { recursive: true });
    for (const suffix of ['', '-wal', '-shm']) {
      execFileSync('docker', [
        'cp',
        `${posEdgeContainer}:/app/data/pos-edge.db${suffix}`,
        path.join(tokenDir, `pos-edge.db${suffix}`),
      ], { stdio: 'ignore' });
    }
    const dbPath = path.join(tokenDir, 'pos-edge.db');
    const script = [
      'import sqlite3',
      `con = sqlite3.connect(${JSON.stringify(dbPath)})`,
      'row = con.execute("select credentials_token from edge_provisioning_state where id = \'local\'").fetchone()',
      'print(row[0] if row else "")',
    ].join('\n');
    return execFileSync('python', ['-c', script], { encoding: 'utf-8' }).trim();
  } catch {
    return nodeToken ?? '';
  }
}

function dockerCLIAvailable(): boolean {
  try {
    execFileSync('docker', ['--version'], { stdio: 'ignore' });
    return true;
  } catch {
    return false;
  }
}

async function createOutboxMutation(request: APIRequestContext, prefix: string): Promise<{ commandId: string; eventType: string }> {
  await ensureOperationalState(request);

  let commandId = `cmd-sync-exchange-${prefix}-${Date.now()}`;
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

  return { commandId, eventType };
}

async function cloudReceiptExists(request: APIRequestContext, commandId: string, eventType: string): Promise<boolean> {
  try {
    const events = await request.get(`${cloudBase}/sync/edge-events?restaurant_id=${demo.restaurant_id}&event_type=${eventType}&limit=50`);
    if (events.status() !== 200) {
      return false;
    }
    const items = await events.json();
    return Array.isArray(items) && items.some((item) => item.command_id === commandId);
  } catch {
    return false;
  }
}

test('online worker exchange drains Edge outbox into Cloud receipt log', async ({ request }) => {
  const { commandId, eventType } = await createOutboxMutation(request, 'online');

  await expect.poll(async () => {
    return cloudReceiptExists(request, commandId, eventType);
  }, { timeout: 45_000 }).toBe(true);
});

test('direct exchange returns partial ACK and idempotent replay for the same event', async ({ request }) => {
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
  const exchange = async () => {
    const currentToken = loadCurrentNodeToken();
    test.skip(!currentToken, 'Set POS_E2E_NODE_TOKEN or run docker compose stack to load node token');
    return request.post(`${cloudBase}/sync/exchange`, {
      headers: { Authorization: `Bearer ${currentToken}` },
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
  };

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

test('worker exchange recovers and catches up after Cloud outage', async ({ request }) => {
  test.skip(!dockerCLIAvailable(), 'Docker CLI is required for the Cloud outage recovery e2e');
  const composeFile = path.join(repoRoot, 'docker-compose.local.yml');
  execFileSync('docker', ['compose', '-f', composeFile, 'stop', 'cloud-api'], { cwd: repoRoot, stdio: 'ignore' });

  let commandId = '';
  let eventType = '';
  try {
    const mutation = await createOutboxMutation(request, 'recovery');
    commandId = mutation.commandId;
    eventType = mutation.eventType;

    await expect.poll(async () => {
      return cloudReceiptExists(request, commandId, eventType);
    }, { timeout: 5_000 }).toBe(false);
  } finally {
    execFileSync('docker', ['compose', '-f', composeFile, 'start', 'cloud-api'], { cwd: repoRoot, stdio: 'ignore' });
  }

  await expect.poll(async () => {
    return cloudReceiptExists(request, commandId, eventType);
  }, { timeout: 90_000 }).toBe(true);
});
