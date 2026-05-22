# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: payments-refunds.spec.ts >> полная оплата cash закрывает заказ и показывает оплату в закрытых заказах
- Location: e2e/payments-refunds.spec.ts:101:1

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
  4   | const clientDeviceId = process.env.POS_E2E_CLIENT_DEVICE_ID ?? 'playwright-e2e-client';
  5   | 
  6   | type DemoBootstrap = {
  7   |   restaurant_id: string;
  8   |   node_device_id: string;
  9   |   cashier_pin: string;
  10  |   manager_pin: string;
  11  |   table_ids: string[];
  12  |   menu_item_ids: string[];
  13  | };
  14  | 
  15  | const bootstrapJson = process.env.POS_E2E_BOOTSTRAP_JSON;
  16  | 
  17  | type AuthHeaders = Record<'X-Node-Device-ID' | 'X-Client-Device-ID' | 'X-Actor-Employee-ID' | 'X-Session-ID', string>;
  18  | 
  19  | type LoginResult = {
  20  |   session: { id: string };
  21  |   actor: { employee_id: string };
  22  | };
  23  | 
  24  | type Order = {
  25  |   id: string;
  26  |   status: string;
  27  |   total: number;
  28  |   check?: Check;
  29  | };
  30  | 
  31  | type Precheck = {
  32  |   id: string;
  33  |   status: string;
  34  |   total: number;
  35  |   paid_total: number;
  36  | };
  37  | 
  38  | type Payment = {
  39  |   id: string;
  40  |   precheck_id: string;
  41  |   amount: number;
  42  |   method: 'cash' | 'card' | 'other';
  43  |   currency: string;
  44  |   status: 'captured' | 'refunded' | 'failed';
  45  | };
  46  | 
  47  | type Check = {
  48  |   id: string;
  49  |   status: 'open' | 'paid' | 'refunded' | 'voided';
  50  |   total: number;
  51  |   paid_total: number;
  52  |   payments?: Payment[];
  53  | };
  54  | 
  55  | type Shift = {
  56  |   id: string;
  57  | };
  58  | 
  59  | type CashSession = {
  60  |   id: string;
  61  | };
  62  | 
  63  | type FinancialOperation = {
  64  |   id: string;
  65  |   operation_type: 'cancellation' | 'refund';
  66  |   operation_kind: 'full' | 'partial';
  67  |   status: 'recorded';
  68  |   check_id: string;
  69  |   amount: number;
  70  |   inventory_disposition: 'no_stock_effect' | 'return_to_stock' | 'write_off_waste' | 'manual_review';
  71  | };
  72  | 
  73  | test.describe.configure({ mode: 'serial' });
  74  | 
  75  | let demo: DemoBootstrap;
  76  | let headers: AuthHeaders;
  77  | let commandSequence = 0;
  78  | 
  79  | test.beforeAll(async ({ playwright }) => {
  80  |   const request = await playwright.request.newContext();
  81  |   try {
> 82  |     expect(bootstrapJson, 'Run scripts/bootstrap-production-way.ps1 and pass its JSON as POS_E2E_BOOTSTRAP_JSON').toBeTruthy();
      |                                                                                                                   ^ Error: Run scripts/bootstrap-production-way.ps1 and pass its JSON as POS_E2E_BOOTSTRAP_JSON
  83  |     demo = JSON.parse(bootstrapJson ?? '{}') as DemoBootstrap;
  84  |     const login = await post<LoginResult>(request, '/auth/pin-login', {
  85  |       node_device_id: demo.node_device_id,
  86  |       client_device_id: clientDeviceId,
  87  |       pin: demo.manager_pin,
  88  |     });
  89  |     headers = {
  90  |       'X-Node-Device-ID': demo.node_device_id,
  91  |       'X-Client-Device-ID': clientDeviceId,
  92  |       'X-Actor-Employee-ID': login.actor.employee_id,
  93  |       'X-Session-ID': login.session.id,
  94  |     };
  95  |     await ensureShiftAndCashSession(request);
  96  |   } finally {
  97  |     await request.dispose();
  98  |   }
  99  | });
  100 | 
  101 | test('полная оплата cash закрывает заказ и показывает оплату в закрытых заказах', async ({ request }) => {
  102 |   const { order, precheck } = await createOrderWithPrecheck(request, 0, 1);
  103 | 
  104 |   const payment = await capturePayment(request, precheck.id, 'cash', precheck.total);
  105 |   expect(payment.status).toBe('captured');
  106 |   expect(payment.amount).toBe(precheck.total);
  107 | 
  108 |   const paidOrder = await get<Order>(request, `/orders/${order.id}`);
  109 |   expect(paidOrder.status).toBe('closed');
  110 |   expect(paidOrder.check?.status).toBe('paid');
  111 |   expect(paidOrder.check?.paid_total).toBe(precheck.total);
  112 | 
  113 |   const closedOrders = await get<Order[]>(request, '/orders/closed?limit=20');
  114 |   const closed = closedOrders.find((item) => item.id === order.id);
  115 |   expect(closed?.check?.payments?.some((item) => item.id === payment.id && item.status === 'captured')).toBe(true);
  116 | });
  117 | 
  118 | test('частичные оплаты не создают финальный чек до полной оплаты', async ({ request }) => {
  119 |   const { order, precheck } = await createOrderWithPrecheck(request, 1, 2);
  120 |   const firstAmount = Math.floor(precheck.total / 2);
  121 | 
  122 |   const firstPayment = await capturePayment(request, precheck.id, 'cash', firstAmount);
  123 |   expect(firstPayment.status).toBe('captured');
  124 | 
  125 |   const afterFirstPayment = await get<Order>(request, `/orders/${order.id}`);
  126 |   expect(afterFirstPayment.status).toBe('locked');
  127 |   expect(afterFirstPayment.check).toBeUndefined();
  128 | 
  129 |   const secondPayment = await capturePayment(request, precheck.id, 'card', precheck.total - firstAmount);
  130 |   expect(secondPayment.status).toBe('captured');
  131 | 
  132 |   const paidOrder = await get<Order>(request, `/orders/${order.id}`);
  133 |   expect(paidOrder.status).toBe('closed');
  134 |   expect(paidOrder.check?.status).toBe('paid');
  135 |   expect(paidOrder.check?.paid_total).toBe(precheck.total);
  136 | });
  137 | 
  138 | test('compatibility refund records ledger operation without mutating finalized payment/check', async ({ request }) => {
  139 |   const { order, precheck } = await createOrderWithPrecheck(request, 0, 1);
  140 |   const payment = await capturePayment(request, precheck.id, 'cash', precheck.total);
  141 | 
  142 |   await closeCurrentShiftAndCashSession(request);
  143 |   await ensureShiftAndCashSession(request);
  144 | 
  145 |   const refunded = await post<Payment>(request, `/payments/${payment.id}/refund`, {
  146 |     command_id: nextCommandID('refund-payment'),
  147 |     reason: 'e2e refund',
  148 |   }, headers);
  149 |   expect(refunded.status).toBe('captured');
  150 | 
  151 |   const refundedOrder = await get<Order>(request, `/orders/${order.id}`);
  152 |   expect(refundedOrder.check?.status).toBe('paid');
  153 |   expect(refundedOrder.check?.paid_total).toBe(precheck.total);
  154 | 
  155 |   const closedOrders = await get<Order[]>(request, '/orders/closed?limit=20');
  156 |   const closed = closedOrders.find((item) => item.id === order.id);
  157 |   expect(closed?.check?.status).toBe('paid');
  158 |   expect(closed?.check?.payments?.some((item) => item.id === payment.id && item.status === 'captured')).toBe(true);
  159 | });
  160 | 
  161 | test('full check cancellation records ledger operation without UI status mutation expectations', async ({ request }) => {
  162 |   const { order, precheck } = await createOrderWithPrecheck(request, 0, 1);
  163 |   const payment = await capturePayment(request, precheck.id, 'cash', precheck.total);
  164 |   const paidOrder = await get<Order>(request, `/orders/${order.id}`);
  165 |   expect(paidOrder.check?.id).toBeTruthy();
  166 | 
  167 |   const operation = await post<FinancialOperation>(request, `/checks/${paidOrder.check?.id}/cancellations`, {
  168 |     command_id: nextCommandID('cancel-check'),
  169 |     operation_kind: 'full',
  170 |     inventory_disposition: 'manual_review',
  171 |     reason: 'e2e full check cancellation',
  172 |   }, headers);
  173 |   expect(operation.operation_type).toBe('cancellation');
  174 |   expect(operation.operation_kind).toBe('full');
  175 |   expect(operation.status).toBe('recorded');
  176 |   expect(operation.inventory_disposition).toBe('manual_review');
  177 |   expect(operation.amount).toBe(precheck.total);
  178 | 
  179 |   const afterCancellation = await get<Order>(request, `/orders/${order.id}`);
  180 |   expect(afterCancellation.check?.status).toBe('paid');
  181 |   expect(afterCancellation.check?.payments?.some((item) => item.id === payment.id && item.status === 'captured')).toBe(true);
  182 | });
```