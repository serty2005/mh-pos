# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: modifiers.spec.ts >> API edit modifiers reprices line, persists pricing snapshot and rejects locked order edits
- Location: e2e/modifiers.spec.ts:86:1

# Error details

```
Error: Run stack bootstrap and set POS_E2E_BOOTSTRAP_JSON to JSON content or /workspace/myhoreca-pos/.e2e/bootstrap.json

expect(received).toBeTruthy()

Received: ""
```

# Test source

```ts
  1   | import { expect, test, type APIRequestContext } from '@playwright/test';
  2   | 
  3   | import { bootstrapRequiredMessage, loadBootstrapJson } from './support/bootstrap';
  4   | 
  5   | const apiBase = (process.env.POS_E2E_API_BASE ?? 'http://localhost:8080/api/v1').replace(/\/$/, '');
  6   | const clientDeviceId = process.env.POS_E2E_CLIENT_DEVICE_ID ?? 'playwright-modifiers-client';
  7   | const bootstrapJson = loadBootstrapJson();
  8   | 
  9   | type DemoBootstrap = {
  10  |   restaurant_id: string;
  11  |   node_device_id: string;
  12  |   manager_pin: string;
  13  |   table_id: string;
  14  |   menu_item_ids: string[];
  15  |   modifier_group_id: string;
  16  |   modifier_option_id: string;
  17  | };
  18  | 
  19  | type AuthHeaders = Record<'X-Node-Device-ID' | 'X-Client-Device-ID' | 'X-Actor-Employee-ID' | 'X-Session-ID', string>;
  20  | 
  21  | type LoginResult = {
  22  |   session: { id: string };
  23  |   actor: { employee_id: string };
  24  | };
  25  | 
  26  | type OrderLineModifier = {
  27  |   modifier_group_id: string;
  28  |   modifier_option_id: string;
  29  |   name: string;
  30  |   quantity: number;
  31  |   unit_price: number;
  32  |   total_price: number;
  33  | };
  34  | 
  35  | type OrderLine = {
  36  |   id: string;
  37  |   menu_item_id: string;
  38  |   quantity: number;
  39  |   unit_price: number;
  40  |   total_price: number;
  41  |   modifiers: OrderLineModifier[];
  42  | };
  43  | 
  44  | type Order = {
  45  |   id: string;
  46  |   status: string;
  47  |   total: number;
  48  |   lines?: OrderLine[];
  49  | };
  50  | 
  51  | type Precheck = {
  52  |   id: string;
  53  |   status: string;
  54  |   total: number;
  55  |   snapshot?: unknown;
  56  | };
  57  | 
  58  | test.describe.configure({ mode: 'serial' });
  59  | 
  60  | let demo: DemoBootstrap;
  61  | let headers: AuthHeaders;
  62  | let commandSequence = 0;
  63  | 
  64  | test.beforeAll(async ({ playwright }) => {
  65  |   const request = await playwright.request.newContext();
  66  |   try {
> 67  |     expect(bootstrapJson, bootstrapRequiredMessage()).toBeTruthy();
      |                                                       ^ Error: Run stack bootstrap and set POS_E2E_BOOTSTRAP_JSON to JSON content or /workspace/myhoreca-pos/.e2e/bootstrap.json
  68  |     demo = JSON.parse(bootstrapJson ?? '{}') as DemoBootstrap;
  69  |     const login = await post<LoginResult>(request, '/auth/pin-login', {
  70  |       node_device_id: demo.node_device_id,
  71  |       client_device_id: clientDeviceId,
  72  |       pin: demo.manager_pin,
  73  |     });
  74  |     headers = {
  75  |       'X-Node-Device-ID': demo.node_device_id,
  76  |       'X-Client-Device-ID': clientDeviceId,
  77  |       'X-Actor-Employee-ID': login.actor.employee_id,
  78  |       'X-Session-ID': login.session.id,
  79  |     };
  80  |     await ensureShiftAndCashSession(request);
  81  |   } finally {
  82  |     await request.dispose();
  83  |   }
  84  | });
  85  | 
  86  | test('API edit modifiers reprices line, persists pricing snapshot and rejects locked order edits', async ({ request }) => {
  87  |   const order = await createOrder(request);
  88  |   const line = await post<OrderLine>(request, `/orders/${order.id}/lines`, {
  89  |     command_id: nextCommandID('add-line-with-modifier'),
  90  |     menu_item_id: demo.menu_item_ids[0],
  91  |     quantity: 1,
  92  |     selected_modifiers: [{
  93  |       modifier_group_id: demo.modifier_group_id,
  94  |       modifier_option_id: demo.modifier_option_id,
  95  |       quantity: 1,
  96  |     }],
  97  |   }, headers);
  98  | 
  99  |   expect(line.modifiers).toHaveLength(1);
  100 |   expect(line.modifiers[0]).toMatchObject({
  101 |     modifier_group_id: demo.modifier_group_id,
  102 |     modifier_option_id: demo.modifier_option_id,
  103 |     quantity: 1,
  104 |     total_price: 3000,
  105 |   });
  106 |   expect(line.total_price).toBe(18000);
  107 | 
  108 |   const edited = await patch<OrderLine>(request, `/orders/${order.id}/lines/${line.id}/modifiers`, {
  109 |     command_id: nextCommandID('edit-line-modifier'),
  110 |     selected_modifiers: [{
  111 |       modifier_group_id: demo.modifier_group_id,
  112 |       modifier_option_id: demo.modifier_option_id,
  113 |       quantity: 2,
  114 |     }],
  115 |   });
  116 |   expect(edited.modifiers).toHaveLength(1);
  117 |   expect(edited.modifiers[0].quantity).toBe(2);
  118 |   expect(edited.modifiers[0].total_price).toBe(6000);
  119 |   expect(edited.total_price).toBe(21000);
  120 | 
  121 |   const updatedOrder = await get<Order>(request, `/orders/${order.id}`);
  122 |   expect(updatedOrder.total).toBe(21000);
  123 |   expect(updatedOrder.lines?.find((item) => item.id === line.id)?.modifiers[0]).toMatchObject({
  124 |     modifier_option_id: demo.modifier_option_id,
  125 |     quantity: 2,
  126 |     total_price: 6000,
  127 |   });
  128 | 
  129 |   const precheck = await post<Precheck>(request, `/orders/${order.id}/precheck`, {
  130 |     command_id: nextCommandID('issue-precheck-with-modifier'),
  131 |   }, headers);
  132 |   expect(precheck.status).toBe('issued');
  133 |   expect(precheck.total).toBe(21000);
  134 |   expect(JSON.stringify(precheck.snapshot)).toContain(demo.modifier_option_id);
  135 | 
  136 |   const rejected = await request.patch(`${apiBase}/orders/${order.id}/lines/${line.id}/modifiers`, {
  137 |     data: {
  138 |       command_id: nextCommandID('edit-locked-modifier'),
  139 |       selected_modifiers: [],
  140 |     },
  141 |     headers,
  142 |   });
  143 |   expect(rejected.status(), await rejected.text()).toBe(409);
  144 | });
  145 | 
  146 | async function ensureShiftAndCashSession(request: APIRequestContext) {
  147 |   await post(request, '/employee-shifts/open', {
  148 |     command_id: nextCommandID('open-shift'),
  149 |     restaurant_id: demo.restaurant_id,
  150 |     opened_by_employee_id: headers['X-Actor-Employee-ID'],
  151 |   }, headers, [201, 409]);
  152 | 
  153 |   await post(request, '/cash-shifts/open', {
  154 |     command_id: nextCommandID('open-cash-session'),
  155 |     restaurant_id: demo.restaurant_id,
  156 |     opened_by_employee_id: headers['X-Actor-Employee-ID'],
  157 |     opening_cash_amount: 0,
  158 |   }, headers, [201, 409]);
  159 | }
  160 | 
  161 | async function createOrder(request: APIRequestContext) {
  162 |   return post<Order>(request, '/orders', {
  163 |     command_id: nextCommandID('create-modifier-order'),
  164 |     restaurant_id: demo.restaurant_id,
  165 |     table_id: demo.table_id,
  166 |     table_name: 'E2E modifiers',
  167 |     guest_count: 1,
```