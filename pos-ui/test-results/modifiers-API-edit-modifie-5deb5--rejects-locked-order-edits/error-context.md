# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: modifiers.spec.ts >> API edit modifiers reprices line, persists pricing snapshot and rejects locked order edits
- Location: e2e/modifiers.spec.ts:84:1

# Error details

```
Error: Run scripts/bootstrap-production-way.ps1 and pass its JSON as POS_E2E_BOOTSTRAP_JSON

expect(received).toBeTruthy()

Received: undefined
```

# Test source

```ts
  1   | import { expect, test, type APIRequestContext } from '@playwright/test';
  2   | 
  3   | const apiBase = (process.env.POS_E2E_API_BASE ?? 'http://localhost:8080/api/v1').replace(/\/$/, '');
  4   | const clientDeviceId = process.env.POS_E2E_CLIENT_DEVICE_ID ?? 'playwright-modifiers-client';
  5   | const bootstrapJson = process.env.POS_E2E_BOOTSTRAP_JSON;
  6   | 
  7   | type DemoBootstrap = {
  8   |   restaurant_id: string;
  9   |   node_device_id: string;
  10  |   manager_pin: string;
  11  |   table_id: string;
  12  |   menu_item_ids: string[];
  13  |   modifier_group_id: string;
  14  |   modifier_option_id: string;
  15  | };
  16  | 
  17  | type AuthHeaders = Record<'X-Node-Device-ID' | 'X-Client-Device-ID' | 'X-Actor-Employee-ID' | 'X-Session-ID', string>;
  18  | 
  19  | type LoginResult = {
  20  |   session: { id: string };
  21  |   actor: { employee_id: string };
  22  | };
  23  | 
  24  | type OrderLineModifier = {
  25  |   modifier_group_id: string;
  26  |   modifier_option_id: string;
  27  |   name: string;
  28  |   quantity: number;
  29  |   unit_price: number;
  30  |   total_price: number;
  31  | };
  32  | 
  33  | type OrderLine = {
  34  |   id: string;
  35  |   menu_item_id: string;
  36  |   quantity: number;
  37  |   unit_price: number;
  38  |   total_price: number;
  39  |   modifiers: OrderLineModifier[];
  40  | };
  41  | 
  42  | type Order = {
  43  |   id: string;
  44  |   status: string;
  45  |   total: number;
  46  |   lines?: OrderLine[];
  47  | };
  48  | 
  49  | type Precheck = {
  50  |   id: string;
  51  |   status: string;
  52  |   total: number;
  53  |   snapshot?: unknown;
  54  | };
  55  | 
  56  | test.describe.configure({ mode: 'serial' });
  57  | 
  58  | let demo: DemoBootstrap;
  59  | let headers: AuthHeaders;
  60  | let commandSequence = 0;
  61  | 
  62  | test.beforeAll(async ({ playwright }) => {
  63  |   const request = await playwright.request.newContext();
  64  |   try {
> 65  |     expect(bootstrapJson, 'Run scripts/bootstrap-production-way.ps1 and pass its JSON as POS_E2E_BOOTSTRAP_JSON').toBeTruthy();
      |                                                                                                                   ^ Error: Run scripts/bootstrap-production-way.ps1 and pass its JSON as POS_E2E_BOOTSTRAP_JSON
  66  |     demo = JSON.parse(bootstrapJson ?? '{}') as DemoBootstrap;
  67  |     const login = await post<LoginResult>(request, '/auth/pin-login', {
  68  |       node_device_id: demo.node_device_id,
  69  |       client_device_id: clientDeviceId,
  70  |       pin: demo.manager_pin,
  71  |     });
  72  |     headers = {
  73  |       'X-Node-Device-ID': demo.node_device_id,
  74  |       'X-Client-Device-ID': clientDeviceId,
  75  |       'X-Actor-Employee-ID': login.actor.employee_id,
  76  |       'X-Session-ID': login.session.id,
  77  |     };
  78  |     await ensureShiftAndCashSession(request);
  79  |   } finally {
  80  |     await request.dispose();
  81  |   }
  82  | });
  83  | 
  84  | test('API edit modifiers reprices line, persists pricing snapshot and rejects locked order edits', async ({ request }) => {
  85  |   const order = await createOrder(request);
  86  |   const line = await post<OrderLine>(request, `/orders/${order.id}/lines`, {
  87  |     command_id: nextCommandID('add-line-with-modifier'),
  88  |     menu_item_id: demo.menu_item_ids[0],
  89  |     quantity: 1,
  90  |     selected_modifiers: [{
  91  |       modifier_group_id: demo.modifier_group_id,
  92  |       modifier_option_id: demo.modifier_option_id,
  93  |       quantity: 1,
  94  |     }],
  95  |   }, headers);
  96  | 
  97  |   expect(line.modifiers).toHaveLength(1);
  98  |   expect(line.modifiers[0]).toMatchObject({
  99  |     modifier_group_id: demo.modifier_group_id,
  100 |     modifier_option_id: demo.modifier_option_id,
  101 |     quantity: 1,
  102 |     total_price: 3000,
  103 |   });
  104 |   expect(line.total_price).toBe(18000);
  105 | 
  106 |   const edited = await patch<OrderLine>(request, `/orders/${order.id}/lines/${line.id}/modifiers`, {
  107 |     command_id: nextCommandID('edit-line-modifier'),
  108 |     selected_modifiers: [{
  109 |       modifier_group_id: demo.modifier_group_id,
  110 |       modifier_option_id: demo.modifier_option_id,
  111 |       quantity: 2,
  112 |     }],
  113 |   });
  114 |   expect(edited.modifiers).toHaveLength(1);
  115 |   expect(edited.modifiers[0].quantity).toBe(2);
  116 |   expect(edited.modifiers[0].total_price).toBe(6000);
  117 |   expect(edited.total_price).toBe(21000);
  118 | 
  119 |   const updatedOrder = await get<Order>(request, `/orders/${order.id}`);
  120 |   expect(updatedOrder.total).toBe(21000);
  121 |   expect(updatedOrder.lines?.find((item) => item.id === line.id)?.modifiers[0]).toMatchObject({
  122 |     modifier_option_id: demo.modifier_option_id,
  123 |     quantity: 2,
  124 |     total_price: 6000,
  125 |   });
  126 | 
  127 |   const precheck = await post<Precheck>(request, `/orders/${order.id}/precheck`, {
  128 |     command_id: nextCommandID('issue-precheck-with-modifier'),
  129 |   }, headers);
  130 |   expect(precheck.status).toBe('issued');
  131 |   expect(precheck.total).toBe(21000);
  132 |   expect(JSON.stringify(precheck.snapshot)).toContain(demo.modifier_option_id);
  133 | 
  134 |   const rejected = await request.patch(`${apiBase}/orders/${order.id}/lines/${line.id}/modifiers`, {
  135 |     data: {
  136 |       command_id: nextCommandID('edit-locked-modifier'),
  137 |       selected_modifiers: [],
  138 |     },
  139 |     headers,
  140 |   });
  141 |   expect(rejected.status(), await rejected.text()).toBe(409);
  142 | });
  143 | 
  144 | async function ensureShiftAndCashSession(request: APIRequestContext) {
  145 |   await post(request, '/employee-shifts/open', {
  146 |     command_id: nextCommandID('open-shift'),
  147 |     restaurant_id: demo.restaurant_id,
  148 |     opened_by_employee_id: headers['X-Actor-Employee-ID'],
  149 |   }, headers, [201, 409]);
  150 | 
  151 |   await post(request, '/cash-shifts/open', {
  152 |     command_id: nextCommandID('open-cash-session'),
  153 |     restaurant_id: demo.restaurant_id,
  154 |     opened_by_employee_id: headers['X-Actor-Employee-ID'],
  155 |     opening_cash_amount: 0,
  156 |   }, headers, [201, 409]);
  157 | }
  158 | 
  159 | async function createOrder(request: APIRequestContext) {
  160 |   return post<Order>(request, '/orders', {
  161 |     command_id: nextCommandID('create-modifier-order'),
  162 |     restaurant_id: demo.restaurant_id,
  163 |     table_id: demo.table_id,
  164 |     table_name: 'E2E modifiers',
  165 |     guest_count: 1,
```