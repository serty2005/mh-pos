# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: sync-exchange.spec.ts >> online worker exchange drains Edge outbox into Cloud receipt log
- Location: e2e\sync-exchange.spec.ts:42:1

# Error details

```
Error: {"error":{"code":"CONFLICT","message_key":"errors.conflict","correlation_id":"e29af06fe66b/H9Dgm6pGJ2-000013"}}


expect(received).toBe(expected) // Object.is equality

Expected: 201
Received: 409
```

# Test source

```ts
  1   | import { expect, test } from '@playwright/test';
  2   | 
  3   | const cloudBase = (process.env.POS_E2E_CLOUD_BASE ?? 'http://localhost:8090/api/v1').replace(/\/$/, '');
  4   | const edgeBase = (process.env.POS_E2E_API_BASE ?? 'http://localhost:8080/api/v1').replace(/\/$/, '');
  5   | const bootstrapJson = process.env.POS_E2E_BOOTSTRAP_JSON;
  6   | const nodeToken = process.env.POS_E2E_NODE_TOKEN;
  7   | 
  8   | type Bootstrap = {
  9   |   restaurant_id: string;
  10  |   node_device_id: string;
  11  |   manager_pin: string;
  12  |   table_ids?: string[];
  13  | };
  14  | 
  15  | test.describe.configure({ mode: 'serial' });
  16  | 
  17  | let demo: Bootstrap;
  18  | let headers: Record<string, string>;
  19  | 
  20  | test.beforeAll(async ({ playwright }) => {
  21  |   test.skip(!bootstrapJson, 'Run scripts/bootstrap-production-way.ps1 and pass POS_E2E_BOOTSTRAP_JSON');
  22  |   demo = JSON.parse(bootstrapJson ?? '{}') as Bootstrap;
  23  |   const request = await playwright.request.newContext();
  24  |   const login = await request.post(`${edgeBase}/auth/pin-login`, {
  25  |     data: {
  26  |       node_device_id: demo.node_device_id,
  27  |       client_device_id: 'sync-exchange-e2e-client',
  28  |       pin: demo.manager_pin,
  29  |     },
  30  |   });
  31  |   expect(login.status(), await login.text()).toBe(201);
  32  |   const body = await login.json();
  33  |   headers = {
  34  |     'X-Node-Device-ID': demo.node_device_id,
  35  |     'X-Client-Device-ID': 'sync-exchange-e2e-client',
  36  |     'X-Actor-Employee-ID': body.actor.employee_id,
  37  |     'X-Session-ID': body.session.id,
  38  |   };
  39  |   await request.dispose();
  40  | });
  41  | 
  42  | test('online worker exchange drains Edge outbox into Cloud receipt log', async ({ request }) => {
  43  |   const commandId = `cmd-sync-exchange-e2e-${Date.now()}`;
  44  |   const order = await request.post(`${edgeBase}/orders`, {
  45  |     headers,
  46  |     data: {
  47  |       command_id: commandId,
  48  |       restaurant_id: demo.restaurant_id,
  49  |       table_id: demo.table_ids?.[0],
  50  |       table_name: 'Sync E2E',
  51  |       guest_count: 1,
  52  |     },
  53  |   });
> 54  |   expect(order.status(), await order.text()).toBe(201);
      |                                              ^ Error: {"error":{"code":"CONFLICT","message_key":"errors.conflict","correlation_id":"e29af06fe66b/H9Dgm6pGJ2-000013"}}
  55  | 
  56  |   await expect.poll(async () => {
  57  |     const events = await request.get(`${cloudBase}/sync/edge-events?restaurant_id=${demo.restaurant_id}&event_type=OrderCreated&limit=50`);
  58  |     if (events.status() !== 200) {
  59  |       return false;
  60  |     }
  61  |     const items = await events.json();
  62  |     return Array.isArray(items) && items.some((item) => item.command_id === commandId);
  63  |   }, { timeout: 45_000 }).toBe(true);
  64  | });
  65  | 
  66  | test('direct exchange returns partial ACK and idempotent replay for the same event', async ({ request }) => {
  67  |   test.skip(!nodeToken, 'Set POS_E2E_NODE_TOKEN to run direct authenticated exchange checks');
  68  |   const eventId = `event-sync-exchange-${Date.now()}`;
  69  |   const payload = {
  70  |     version: '1',
  71  |     event_id: eventId,
  72  |     command_id: `cmd-${eventId}`,
  73  |     event_type: 'OrderCreated',
  74  |     aggregate_type: 'Order',
  75  |     aggregate_id: `order-${eventId}`,
  76  |     restaurant_id: demo.restaurant_id,
  77  |     device_id: demo.node_device_id,
  78  |     node_device_id: demo.node_device_id,
  79  |     occurred_at: new Date().toISOString(),
  80  |     payload: {
  81  |       origin: 'edge_device',
  82  |       data: {
  83  |         id: `order-${eventId}`,
  84  |         edge_order_id: `edge-order-${eventId}`,
  85  |         restaurant_id: demo.restaurant_id,
  86  |         device_id: demo.node_device_id,
  87  |         shift_id: 'sync-exchange-e2e-shift',
  88  |         status: 'open',
  89  |         table_name: 'Sync E2E',
  90  |         guest_count: 1,
  91  |         opened_at: new Date().toISOString(),
  92  |         created_at: new Date().toISOString(),
  93  |         updated_at: new Date().toISOString(),
  94  |       },
  95  |     },
  96  |   };
  97  |   const exchange = async () => request.post(`${cloudBase}/sync/exchange`, {
  98  |     headers: { Authorization: `Bearer ${nodeToken}` },
  99  |     data: {
  100 |       protocol_version: 'sync_exchange.v1',
  101 |       node_device_id: demo.node_device_id,
  102 |       restaurant_id: demo.restaurant_id,
  103 |       edge_events: [
  104 |         { client_item_id: 'e2e-valid', payload },
  105 |         { client_item_id: 'e2e-invalid', payload: { version: '1' } },
  106 |       ],
  107 |       streams: [{ stream_name: 'catalog', last_cloud_version: 0 }],
  108 |     },
  109 |   });
  110 | 
  111 |   const first = await exchange();
  112 |   expect(first.status(), await first.text()).toBe(202);
  113 |   const firstBody = await first.json();
  114 |   expect(firstBody.status).toBe('partial');
  115 |   expect(firstBody.edge_acks.find((item) => item.client_item_id === 'e2e-valid')?.status).toBe('accepted');
  116 |   expect(firstBody.edge_acks.find((item) => item.client_item_id === 'e2e-invalid')?.status).toBe('rejected');
  117 | 
  118 |   const replay = await exchange();
  119 |   expect(replay.status(), await replay.text()).toBe(202);
  120 |   const replayBody = await replay.json();
  121 |   expect(replayBody.edge_acks.find((item) => item.client_item_id === 'e2e-valid')?.ack.cloud_receipt_id)
  122 |     .toBe(firstBody.edge_acks.find((item) => item.client_item_id === 'e2e-valid')?.ack.cloud_receipt_id);
  123 | });
  124 | 
```