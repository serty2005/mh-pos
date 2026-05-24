# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: sync-provisioning-flow.spec.ts >> pairing opens exchange sync and stops repeated Cloud registration polling
- Location: e2e/sync-provisioning-flow.spec.ts:116:1

# Error details

```
Error: apiRequestContext.get: connect ECONNREFUSED 127.0.0.1:8080
Call log:
  - → GET http://127.0.0.1:8080/api/v1/system/provisioning-status
    - user-agent: Playwright/1.59.1 (x64; debian 12) node/26.2
    - accept: */*
    - accept-encoding: gzip,deflate,br

```

# Test source

```ts
  1   | import { expect, test, type APIRequestContext } from '@playwright/test';
  2   | import { execFileSync } from 'node:child_process';
  3   | import path from 'node:path';
  4   | import { fileURLToPath } from 'node:url';
  5   | 
  6   | const cloudBase = (process.env.POS_E2E_CLOUD_BASE ?? 'http://127.0.0.1:8090/api/v1').replace(/\/$/, '');
  7   | const edgeBase = (process.env.POS_E2E_API_BASE ?? 'http://127.0.0.1:8080/api/v1').replace(/\/$/, '');
  8   | const repoRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..', '..');
  9   | const clientDeviceId = 'sync-provisioning-e2e-client';
  10  | 
  11  | type AuthHeaders = Record<string, string>;
  12  | 
  13  | const managerPermissions = [
  14  |   'pos.employee_shift.open',
  15  |   'pos.employee_shift.close',
  16  |   'pos.employee_shift.view_current',
  17  |   'pos.employee_shift.recent',
  18  |   'pos.cash_session.open',
  19  |   'pos.cash_session.close',
  20  |   'pos.cash_session.view_current',
  21  |   'pos.cash_drawer.record_event',
  22  |   'pos.catalog.view',
  23  |   'pos.floor.view',
  24  |   'pos.menu.view',
  25  |   'pos.order.create',
  26  |   'pos.order.view',
  27  |   'pos.order.add_line',
  28  |   'pos.order.change_quantity',
  29  |   'pos.order.void_line',
  30  |   'pos.order.close',
  31  |   'pos.precheck.issue',
  32  |   'pos.precheck.view',
  33  |   'pos.precheck.reprint',
  34  |   'pos.precheck.cancel.request',
  35  |   'pos.precheck.cancel',
  36  |   'pos.payment.cash',
  37  |   'pos.payment.card.manual',
  38  |   'pos.payment.other',
  39  |   'pos.payment.refund',
  40  |   'pos.check.view',
  41  |   'pos.check.reprint',
  42  |   'pos.sync.view',
  43  |   'pos.sync.retry_failed',
  44  | ];
  45  | 
  46  | function commandId(prefix: string) {
  47  |   return `cmd-sync-provisioning-${prefix}-${Date.now()}-${Math.random().toString(16).slice(2)}`;
  48  | }
  49  | 
  50  | function permissionsJSON(values: string[]) {
  51  |   return JSON.stringify(Object.fromEntries([...new Set(values)].sort().map((item) => [item, true])));
  52  | }
  53  | 
  54  | async function postJSON(request: APIRequestContext, url: string, data: unknown, expected = 201) {
  55  |   const resp = await request.post(url, { data });
  56  |   expect(resp.status(), await resp.text()).toBe(expected);
  57  |   return resp.json();
  58  | }
  59  | 
  60  | async function getJSON(request: APIRequestContext, url: string, expected = 200) {
> 61  |   const resp = await request.get(url);
      |                              ^ Error: apiRequestContext.get: connect ECONNREFUSED 127.0.0.1:8080
  62  |   expect(resp.status(), await resp.text()).toBe(expected);
  63  |   return resp.json();
  64  | }
  65  | 
  66  | async function login(request: APIRequestContext, nodeDeviceId: string, pin: string): Promise<AuthHeaders> {
  67  |   const resp = await request.post(`${edgeBase}/auth/pin-login`, {
  68  |     data: { node_device_id: nodeDeviceId, client_device_id: clientDeviceId, pin },
  69  |   });
  70  |   expect(resp.status(), await resp.text()).toBe(201);
  71  |   const body = await resp.json();
  72  |   return {
  73  |     'X-Node-Device-ID': nodeDeviceId,
  74  |     'X-Client-Device-ID': clientDeviceId,
  75  |     'X-Actor-Employee-ID': body.actor.employee_id,
  76  |     'X-Session-ID': body.session.id,
  77  |   };
  78  | }
  79  | 
  80  | async function ensureOperationalState(request: APIRequestContext, headers: AuthHeaders, restaurantId: string, actorEmployeeId: string) {
  81  |   const currentShift = await request.get(`${edgeBase}/employee-shifts/current`, { headers });
  82  |   if (currentShift.status() !== 200 || (await currentShift.text()).trim() === 'null') {
  83  |     const openShift = await request.post(`${edgeBase}/employee-shifts/open`, {
  84  |       headers,
  85  |       data: {
  86  |         command_id: commandId('open-shift'),
  87  |         restaurant_id: restaurantId,
  88  |         opened_by_employee_id: actorEmployeeId,
  89  |       },
  90  |     });
  91  |     expect([201, 409]).toContain(openShift.status());
  92  |   }
  93  | 
  94  |   const currentCash = await request.get(`${edgeBase}/cash-shifts/current`, { headers });
  95  |   if (currentCash.status() !== 200) {
  96  |     const openCash = await request.post(`${edgeBase}/cash-shifts/open`, {
  97  |       headers,
  98  |       data: {
  99  |         command_id: commandId('open-cash'),
  100 |         restaurant_id: restaurantId,
  101 |         opened_by_employee_id: actorEmployeeId,
  102 |         opening_cash_amount: 0,
  103 |       },
  104 |     });
  105 |     expect([201, 409]).toContain(openCash.status());
  106 |   }
  107 | }
  108 | 
  109 | function cloudLogsSince(markerISO: string) {
  110 |   return execFileSync('docker', ['compose', '-f', 'docker-compose.local.yml', 'logs', 'cloud-api', '--since', markerISO], {
  111 |     cwd: repoRoot,
  112 |     encoding: 'utf-8',
  113 |   });
  114 | }
  115 | 
  116 | test('pairing opens exchange sync and stops repeated Cloud registration polling', async ({ request }) => {
  117 |   const suffix = Math.random().toString(16).slice(2, 10);
  118 |   const managerPin = '2222';
  119 | 
  120 |   const initialStatus = await getJSON(request, `${edgeBase}/system/provisioning-status`);
  121 |   const nodeDeviceId = initialStatus.node_device_id as string;
  122 |   expect(nodeDeviceId).toBeTruthy();
  123 | 
  124 |   const restaurant = await postJSON(request, `${cloudBase}/restaurants`, {
  125 |     name: `Sync Pairing Bistro ${suffix}`,
  126 |     timezone: 'Europe/Moscow',
  127 |     currency: 'RUB',
  128 |     business_day_mode: 'standard',
  129 |     business_day_boundary_local_time: '04:00',
  130 |   });
  131 |   const managerRole = await postJSON(request, `${cloudBase}/roles`, {
  132 |     restaurant_id: restaurant.id,
  133 |     name: `manager-${suffix}`,
  134 |     permissions_json: permissionsJSON(managerPermissions),
  135 |   });
  136 |   const manager = await postJSON(request, `${cloudBase}/employees`, {
  137 |     restaurant_id: restaurant.id,
  138 |     role_id: managerRole.id,
  139 |     name: 'Sync Manager',
  140 |     pin: managerPin,
  141 |   });
  142 |   const hall = await postJSON(request, `${cloudBase}/halls`, {
  143 |     restaurant_id: restaurant.id,
  144 |     name: 'Main Hall',
  145 |   });
  146 |   const table = await postJSON(request, `${cloudBase}/tables`, {
  147 |     restaurant_id: restaurant.id,
  148 |     hall_id: hall.id,
  149 |     name: 'T1',
  150 |     seats: 2,
  151 |   });
  152 |   const catalog = await postJSON(request, `${cloudBase}/catalog/items`, {
  153 |     restaurant_id: restaurant.id,
  154 |     type: 'dish',
  155 |     name: 'Sync Soup',
  156 |     sku: `SYNC-SOUP-${suffix}`,
  157 |     base_unit: 'portion',
  158 |   });
  159 |   const menuItem = await postJSON(request, `${cloudBase}/menu/items`, {
  160 |     restaurant_id: restaurant.id,
  161 |     catalog_item_id: catalog.id,
```