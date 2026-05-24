# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: payments-refunds.spec.ts >> полная оплата cash закрывает заказ и показывает оплату в закрытых заказах
- Location: e2e/payments-refunds.spec.ts:103:1

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
  6   | const clientDeviceId = process.env.POS_E2E_CLIENT_DEVICE_ID ?? 'playwright-e2e-client';
  7   | 
  8   | type DemoBootstrap = {
  9   |   restaurant_id: string;
  10  |   node_device_id: string;
  11  |   cashier_pin: string;
  12  |   manager_pin: string;
  13  |   table_ids: string[];
  14  |   menu_item_ids: string[];
  15  | };
  16  | 
  17  | const bootstrapJson = loadBootstrapJson();
  18  | 
  19  | type AuthHeaders = Record<'X-Node-Device-ID' | 'X-Client-Device-ID' | 'X-Actor-Employee-ID' | 'X-Session-ID', string>;
  20  | 
  21  | type LoginResult = {
  22  |   session: { id: string };
  23  |   actor: { employee_id: string };
  24  | };
  25  | 
  26  | type Order = {
  27  |   id: string;
  28  |   status: string;
  29  |   total: number;
  30  |   check?: Check;
  31  | };
  32  | 
  33  | type Precheck = {
  34  |   id: string;
  35  |   status: string;
  36  |   total: number;
  37  |   paid_total: number;
  38  | };
  39  | 
  40  | type Payment = {
  41  |   id: string;
  42  |   precheck_id: string;
  43  |   amount: number;
  44  |   method: 'cash' | 'card' | 'other';
  45  |   currency: string;
  46  |   status: 'captured' | 'refunded' | 'failed';
  47  | };
  48  | 
  49  | type Check = {
  50  |   id: string;
  51  |   status: 'open' | 'paid' | 'refunded' | 'voided';
  52  |   total: number;
  53  |   paid_total: number;
  54  |   payments?: Payment[];
  55  | };
  56  | 
  57  | type Shift = {
  58  |   id: string;
  59  | };
  60  | 
  61  | type CashSession = {
  62  |   id: string;
  63  | };
  64  | 
  65  | type FinancialOperation = {
  66  |   id: string;
  67  |   operation_type: 'cancellation' | 'refund';
  68  |   operation_kind: 'full' | 'partial';
  69  |   status: 'recorded';
  70  |   check_id: string;
  71  |   amount: number;
  72  |   inventory_disposition: 'no_stock_effect' | 'return_to_stock' | 'write_off_waste' | 'manual_review';
  73  | };
  74  | 
  75  | test.describe.configure({ mode: 'serial' });
  76  | 
  77  | let demo: DemoBootstrap;
  78  | let headers: AuthHeaders;
  79  | let commandSequence = 0;
  80  | 
  81  | test.beforeAll(async ({ playwright }) => {
  82  |   const request = await playwright.request.newContext();
  83  |   try {
> 84  |     expect(bootstrapJson, bootstrapRequiredMessage()).toBeTruthy();
      |                                                       ^ Error: Run stack bootstrap and set POS_E2E_BOOTSTRAP_JSON to JSON content or /workspace/myhoreca-pos/.e2e/bootstrap.json
  85  |     demo = JSON.parse(bootstrapJson ?? '{}') as DemoBootstrap;
  86  |     const login = await post<LoginResult>(request, '/auth/pin-login', {
  87  |       node_device_id: demo.node_device_id,
  88  |       client_device_id: clientDeviceId,
  89  |       pin: demo.manager_pin,
  90  |     });
  91  |     headers = {
  92  |       'X-Node-Device-ID': demo.node_device_id,
  93  |       'X-Client-Device-ID': clientDeviceId,
  94  |       'X-Actor-Employee-ID': login.actor.employee_id,
  95  |       'X-Session-ID': login.session.id,
  96  |     };
  97  |     await ensureShiftAndCashSession(request);
  98  |   } finally {
  99  |     await request.dispose();
  100 |   }
  101 | });
  102 | 
  103 | test('полная оплата cash закрывает заказ и показывает оплату в закрытых заказах', async ({ request }) => {
  104 |   const { order, precheck } = await createOrderWithPrecheck(request, 0, 1);
  105 | 
  106 |   const payment = await capturePayment(request, precheck.id, 'cash', precheck.total);
  107 |   expect(payment.status).toBe('captured');
  108 |   expect(payment.amount).toBe(precheck.total);
  109 | 
  110 |   const paidOrder = await get<Order>(request, `/orders/${order.id}`);
  111 |   expect(paidOrder.status).toBe('closed');
  112 |   expect(paidOrder.check?.status).toBe('paid');
  113 |   expect(paidOrder.check?.paid_total).toBe(precheck.total);
  114 | 
  115 |   const closedOrders = await get<Order[]>(request, '/orders/closed?limit=20');
  116 |   const closed = closedOrders.find((item) => item.id === order.id);
  117 |   expect(closed?.check?.payments?.some((item) => item.id === payment.id && item.status === 'captured')).toBe(true);
  118 | });
  119 | 
  120 | test('частичные оплаты не создают финальный чек до полной оплаты', async ({ request }) => {
  121 |   const { order, precheck } = await createOrderWithPrecheck(request, 1, 2);
  122 |   const firstAmount = Math.floor(precheck.total / 2);
  123 | 
  124 |   const firstPayment = await capturePayment(request, precheck.id, 'cash', firstAmount);
  125 |   expect(firstPayment.status).toBe('captured');
  126 | 
  127 |   const afterFirstPayment = await get<Order>(request, `/orders/${order.id}`);
  128 |   expect(afterFirstPayment.status).toBe('locked');
  129 |   expect(afterFirstPayment.check).toBeUndefined();
  130 | 
  131 |   const secondPayment = await capturePayment(request, precheck.id, 'card', precheck.total - firstAmount);
  132 |   expect(secondPayment.status).toBe('captured');
  133 | 
  134 |   const paidOrder = await get<Order>(request, `/orders/${order.id}`);
  135 |   expect(paidOrder.status).toBe('closed');
  136 |   expect(paidOrder.check?.status).toBe('paid');
  137 |   expect(paidOrder.check?.paid_total).toBe(precheck.total);
  138 | });
  139 | 
  140 | test('compatibility refund records ledger operation without mutating finalized payment/check', async ({ request }) => {
  141 |   const { order, precheck } = await createOrderWithPrecheck(request, 0, 1);
  142 |   const payment = await capturePayment(request, precheck.id, 'cash', precheck.total);
  143 | 
  144 |   await closeCurrentShiftAndCashSession(request);
  145 |   await ensureShiftAndCashSession(request);
  146 | 
  147 |   const refunded = await post<Payment>(request, `/payments/${payment.id}/refund`, {
  148 |     command_id: nextCommandID('refund-payment'),
  149 |     reason: 'e2e refund',
  150 |   }, headers);
  151 |   expect(refunded.status).toBe('captured');
  152 | 
  153 |   const refundedOrder = await get<Order>(request, `/orders/${order.id}`);
  154 |   expect(refundedOrder.check?.status).toBe('paid');
  155 |   expect(refundedOrder.check?.paid_total).toBe(precheck.total);
  156 | 
  157 |   const closedOrders = await get<Order[]>(request, '/orders/closed?limit=20');
  158 |   const closed = closedOrders.find((item) => item.id === order.id);
  159 |   expect(closed?.check?.status).toBe('paid');
  160 |   expect(closed?.check?.payments?.some((item) => item.id === payment.id && item.status === 'captured')).toBe(true);
  161 | });
  162 | 
  163 | test('full check cancellation records ledger operation without UI status mutation expectations', async ({ request }) => {
  164 |   const { order, precheck } = await createOrderWithPrecheck(request, 0, 1);
  165 |   const payment = await capturePayment(request, precheck.id, 'cash', precheck.total);
  166 |   const paidOrder = await get<Order>(request, `/orders/${order.id}`);
  167 |   expect(paidOrder.check?.id).toBeTruthy();
  168 | 
  169 |   const operation = await post<FinancialOperation>(request, `/checks/${paidOrder.check?.id}/cancellations`, {
  170 |     command_id: nextCommandID('cancel-check'),
  171 |     operation_kind: 'full',
  172 |     inventory_disposition: 'manual_review',
  173 |     reason: 'e2e full check cancellation',
  174 |   }, headers);
  175 |   expect(operation.operation_type).toBe('cancellation');
  176 |   expect(operation.operation_kind).toBe('full');
  177 |   expect(operation.status).toBe('recorded');
  178 |   expect(operation.inventory_disposition).toBe('manual_review');
  179 |   expect(operation.amount).toBe(precheck.total);
  180 | 
  181 |   const afterCancellation = await get<Order>(request, `/orders/${order.id}`);
  182 |   expect(afterCancellation.check?.status).toBe('paid');
  183 |   expect(afterCancellation.check?.payments?.some((item) => item.id === payment.id && item.status === 'captured')).toBe(true);
  184 | });
```