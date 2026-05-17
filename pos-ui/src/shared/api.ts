import { z } from 'zod';

import { useAuthStore } from '../stores/auth';
import {
  cashSessionSchema,
  cashDrawerEventSchema,
  checkSchema,
  closedOrderSchema,
  financialOperationSchema,
  hallSchema,
  localEventSchema,
  menuItemSchema,
  orderLineSchema,
  orderSchema,
  outboxMessageSchema,
  pairingStatusSchema,
  provisioningStatusSchema,
  paymentSchema,
  pinLoginResultSchema,
  precheckSchema,
  reprintDocumentSchema,
  retryFailedOutboxResultSchema,
  shiftSchema,
  syncStatusSchema,
  tableSchema,
  type PinLoginResult,
} from './schemas';

export type SelectedModifierPayload = {
  modifier_group_id: string;
  modifier_option_id: string;
  quantity: number;
};

export type InventoryDisposition = 'no_stock_effect' | 'return_to_stock' | 'write_off_waste' | 'manual_review';
export type FinancialOperationKind = 'full' | 'partial';
export type FinancialOperationItemScope = 'whole_check' | 'order_line' | 'modifier_line' | 'service_charge' | 'tip' | 'payment';

export type FinancialOperationItemPayload = {
  scope: FinancialOperationItemScope;
  orderLineId?: string;
  paymentId?: string;
  quantity?: number;
  amount: number;
  currency?: string;
  taxAmount?: number;
  snapshot?: unknown;
};

export type CheckLedgerOperationPayload = {
  commandId?: string;
  reason: string;
  inventoryDisposition?: InventoryDisposition;
  operationKind?: FinancialOperationKind;
  items?: FinancialOperationItemPayload[];
};

export type PaymentRefundPayload = {
  commandId?: string;
  reason?: string;
};

let commandSequence = 0;

function nextCommandId(prefix: string) {
  commandSequence += 1;
  return `cmd-pos-ui-${Date.now()}-${commandSequence}-${prefix}`;
}

function defaultApiBase() {
  const hostname = globalThis.location?.hostname;
  if (hostname === 'host.docker.internal') {
    return 'http://host.docker.internal:8080/api/v1';
  }
  return 'http://localhost:8080/api/v1';
}

const apiBase = (import.meta.env.VITE_POS_API_BASE ?? defaultApiBase()).replace(/\/$/, '');
const defaultTimeoutMs = 15_000;

const backendErrorSchema = z.object({
  error: z.union([
    z.string(),
    z.object({
      code: z.string().optional(),
      message_key: z.string().optional(),
      details: z.record(z.string(), z.string()).optional(),
      correlation_id: z.string().optional(),
    }),
  ]),
});

/** Категория ошибки API определяет UX-поток без разбора сырого backend text в компонентах. */
export type ApiErrorCategory =
  | 'auth'
  | 'permission'
  | 'validation'
  | 'not_found'
  | 'conflict'
  | 'rate_limit'
  | 'server'
  | 'network'
  | 'timeout'
  | 'unexpected';

/** Параметры безопасной API-ошибки после нормализации backend/network ответа. */
export type ApiErrorOptions = {
  status: number;
  code: string;
  messageKey: string;
  category: ApiErrorCategory;
  details?: Record<string, string>;
  correlationId?: string;
  retryable?: boolean;
};

/** ApiError хранит стабильные code/message_key/correlation id без stack trace и секретов для UI. */
export class ApiError extends Error {
  status: number;
  code: string;
  messageKey: string;
  category: ApiErrorCategory;
  details: Record<string, string>;
  correlationId: string;
  retryable: boolean;

  constructor(options: ApiErrorOptions) {
    super(options.code);
    this.name = 'ApiError';
    this.status = options.status;
    this.code = options.code;
    this.messageKey = options.messageKey;
    this.category = options.category;
    this.details = options.details ?? {};
    this.correlationId = options.correlationId ?? '';
    this.retryable = options.retryable ?? false;
  }
}

async function request<T>(path: string, schema: z.ZodType<T>, init: RequestInit = {}) {
  const auth = useAuthStore();
  const headers = new Headers(init.headers);
  const hasBody = init.body !== undefined && init.body !== null;
  if (hasBody && !headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json');
  }
  headers.set('X-Client-Device-ID', auth.clientDeviceId);
  if (auth.nodeDeviceId) headers.set('X-Node-Device-ID', auth.nodeDeviceId);
  if (auth.sessionId) headers.set('X-Session-ID', auth.sessionId);
  if (auth.actor?.employee_id) headers.set('X-Actor-Employee-ID', auth.actor.employee_id);

  const controller = new AbortController();
  const timeout = globalThis.setTimeout(() => controller.abort(), defaultTimeoutMs);
  let response: Response;
  try {
    response = await fetch(`${apiBase}${path}`, { ...init, headers, signal: controller.signal });
  } catch (error) {
    throw networkApiError(error);
  } finally {
    globalThis.clearTimeout(timeout);
  }

  const data = await parseResponseBody(response);
  if (!response.ok) {
    throw apiErrorFromResponse(response, data);
  }
  try {
    return schema.parse(data);
  } catch {
    throw new ApiError({
      status: 0,
      code: 'INVALID_RESPONSE',
      messageKey: 'errors.response.invalid',
      category: 'unexpected',
      retryable: false,
    });
  }
}

async function requestOptional<T>(path: string, schema: z.ZodType<T>, init: RequestInit = {}) {
  try {
    return await request(path, schema.nullable(), init);
  } catch (error) {
    if (error instanceof ApiError && error.status === 404) {
      return null;
    }
    throw error;
  }
}

async function parseResponseBody(response: Response) {
  const text = await response.text();
  if (!text.trim()) return null;
  try {
    return JSON.parse(text) as unknown;
  } catch {
    if (!response.ok) {
      return null;
    }
    throw new ApiError({
      status: response.status,
      code: 'INVALID_JSON',
      messageKey: 'errors.response.invalid',
      category: 'unexpected',
      correlationId: response.headers.get('X-Request-ID') ?? '',
    });
  }
}

function apiErrorFromResponse(response: Response, data: unknown) {
  const parsed = backendErrorSchema.safeParse(data);
  const errorValue = parsed.success ? parsed.data.error : null;
  const structured = typeof errorValue === 'object' && errorValue !== null ? errorValue : null;
  const code = structured?.code ?? codeForStatus(response.status);
  return new ApiError({
    status: response.status,
    code,
    messageKey: structured?.message_key ?? messageKeyForStatus(response.status),
    category: categoryForStatusAndCode(response.status, code),
    details: structured?.details,
    correlationId: structured?.correlation_id ?? response.headers.get('X-Request-ID') ?? '',
    retryable: isRetryable(response.status),
  });
}

function networkApiError(error: unknown) {
  const aborted = typeof DOMException !== 'undefined' && error instanceof DOMException && error.name === 'AbortError';
  return new ApiError({
    status: 0,
    code: aborted ? 'REQUEST_TIMEOUT' : 'NETWORK_ERROR',
    messageKey: aborted ? 'errors.network.timeout' : 'errors.network.unavailable',
    category: aborted ? 'timeout' : 'network',
    retryable: true,
  });
}

function codeForStatus(status: number) {
  switch (status) {
    case 400:
    case 422:
      return 'VALIDATION_FAILED';
    case 401:
      return 'SESSION_REQUIRED';
    case 403:
      return 'PERMISSION_DENIED';
    case 404:
      return 'NOT_FOUND';
    case 409:
      return 'CONFLICT';
    case 429:
      return 'RATE_LIMITED';
    default:
      return status >= 500 ? 'INTERNAL_ERROR' : 'UNKNOWN_ERROR';
  }
}

function messageKeyForStatus(status: number) {
  switch (status) {
    case 400:
    case 422:
      return 'errors.validation';
    case 401:
      return 'errors.session.required';
    case 403:
      return 'errors.permission';
    case 404:
      return 'errors.not_found';
    case 409:
      return 'errors.conflict';
    case 429:
      return 'errors.rateLimit';
    default:
      return status >= 500 ? 'errors.server' : 'errors.unknown';
  }
}

function categoryForStatusAndCode(status: number, code: string): ApiErrorCategory {
  if (status === 401 || code.startsWith('SESSION_')) return 'auth';
  if (status === 403) return 'permission';
  if (status === 400 || status === 422) return 'validation';
  if (status === 404) return 'not_found';
  if (status === 409) return 'conflict';
  if (status === 429) return 'rate_limit';
  if (status >= 500) return 'server';
  return 'unexpected';
}

function isRetryable(status: number) {
  return status === 0 || status === 429 || status >= 500;
}

function actorId() {
  const auth = useAuthStore();
  return auth.actor?.employee_id ?? '';
}

function requireLedgerReason(reason: string) {
  const trimmed = reason.trim();
  if (!trimmed) {
    throw new ApiError({
      status: 0,
      code: 'VALIDATION_FAILED',
      messageKey: 'errors.validation',
      category: 'validation',
      retryable: false,
    });
  }
  return trimmed;
}

function mapFinancialOperationItems(items?: FinancialOperationItemPayload[]) {
  if (!items?.length) return undefined;
  return items.map((item) => ({
    scope: item.scope,
    order_line_id: item.orderLineId,
    payment_id: item.paymentId,
    quantity: item.quantity,
    amount: item.amount,
    currency: item.currency,
    tax_amount: item.taxAmount,
    snapshot: item.snapshot,
  }));
}

export function getPairingStatus() {
  return request('/system/pairing-status', pairingStatusSchema);
}

export function getProvisioningStatus() {
  return request('/system/provisioning-status', provisioningStatusSchema);
}

export function registerCloudProvisioning(cloudUrl?: string) {
  return request('/system/provisioning/register-cloud', provisioningStatusSchema, {
    method: 'POST',
    body: JSON.stringify({
      cloud_url: cloudUrl ?? '',
      display_name: 'POS Terminal',
      app_version: import.meta.env.VITE_APP_VERSION ?? 'pos-ui',
    }),
  });
}

export function pairViaLicense(pairingCode: string) {
  return request('/system/provisioning/pair-via-license', provisioningStatusSchema, {
    method: 'POST',
    body: JSON.stringify({ pairing_code: pairingCode }),
  });
}

export async function pairEdgeNodeAndRefresh(pairingCode: string) {
  await request('/system/pair', z.unknown(), {
    method: 'POST',
    body: JSON.stringify({ pairing_code: pairingCode }),
  });
  return getPairingStatus();
}

export function pinLogin(pin: string) {
  const auth = useAuthStore();
  return request<PinLoginResult>('/auth/pin-login', pinLoginResultSchema, {
    method: 'POST',
    body: JSON.stringify({
      node_device_id: auth.nodeDeviceId,
      client_device_id: auth.clientDeviceId,
      pin,
    }),
  });
}

export function getAuthSession() {
  const auth = useAuthStore();
  const query = new URLSearchParams({
    session_id: auth.sessionId,
    node_device_id: auth.nodeDeviceId,
    client_device_id: auth.clientDeviceId,
  });
  return request(`/auth/session?${query}`, pinLoginResultSchema);
}

export function logout() {
  const auth = useAuthStore();
  return request('/auth/logout', z.unknown(), {
    method: 'POST',
    body: JSON.stringify({
      node_device_id: auth.nodeDeviceId,
      client_device_id: auth.clientDeviceId,
      session_id: auth.sessionId,
    }),
  });
}

export function getCurrentShift() {
  const auth = useAuthStore();
  const query = new URLSearchParams({ node_device_id: auth.nodeDeviceId });
  return requestOptional(`/employee-shifts/current?${query}`, shiftSchema);
}

export function listRecentShifts() {
  return request('/employee-shifts/recent?limit=5', z.array(shiftSchema));
}

export function openShift() {
  const auth = useAuthStore();
  return request('/employee-shifts/open', shiftSchema, {
    method: 'POST',
    body: JSON.stringify({
      restaurant_id: auth.restaurantId,
      opened_by_employee_id: actorId(),
    }),
  });
}

export function closeShift(shiftId: string) {
  return request(`/employee-shifts/${encodeURIComponent(shiftId)}/close`, shiftSchema, {
    method: 'POST',
    body: JSON.stringify({
      closed_by_employee_id: actorId(),
    }),
  });
}

export function getCurrentCashSession() {
  const auth = useAuthStore();
  const query = new URLSearchParams({ node_device_id: auth.nodeDeviceId });
  return requestOptional(`/cash-shifts/current?${query}`, cashSessionSchema);
}

export function openCashSession(openingCashAmount: number) {
  const auth = useAuthStore();
  return request('/cash-shifts/open', cashSessionSchema, {
    method: 'POST',
    body: JSON.stringify({
      restaurant_id: auth.restaurantId,
      opened_by_employee_id: actorId(),
      opening_cash_amount: openingCashAmount,
    }),
  });
}

export function closeCashSession(cashSessionId: string, closingCashAmount: number) {
  return request(`/cash-shifts/${encodeURIComponent(cashSessionId)}/close`, cashSessionSchema, {
    method: 'POST',
    body: JSON.stringify({
      closed_by_employee_id: actorId(),
      closing_cash_amount: closingCashAmount,
    }),
  });
}

export type CashDrawerEventType = 'cash_in' | 'cash_out' | 'no_sale' | 'cash_count';

export function recordCashDrawerEvent(cashSessionId: string, eventType: CashDrawerEventType, amount: number, reason = '', note = '') {
  return request('/cash-drawer-events', cashDrawerEventSchema, {
    method: 'POST',
    body: JSON.stringify({
      cash_session_id: cashSessionId,
      created_by_employee_id: actorId(),
      event_type: eventType,
      amount,
      reason,
      note,
    }),
  });
}

export function listHalls(restaurantId: string) {
  return request(`/halls?restaurant_id=${encodeURIComponent(restaurantId)}`, z.array(hallSchema));
}

export function listTables(restaurantId: string, hallId: string) {
  const query = new URLSearchParams({ restaurant_id: restaurantId, hall_id: hallId });
  return request(`/tables?${query}`, z.array(tableSchema));
}

export function listMenuItems() {
  return request('/menu/items', z.array(menuItemSchema));
}

export function getCurrentOrderByTable(tableId: string) {
  const query = new URLSearchParams({ table_id: tableId });
  return requestOptional(`/orders/current?${query}`, orderSchema);
}

export function listActiveOrdersByHall(hallId: string) {
  const query = new URLSearchParams({ hall_id: hallId });
  return request(`/orders/active?${query}`, z.array(orderSchema));
}

export function getOrder(orderId: string) {
  return request(`/orders/${encodeURIComponent(orderId)}`, orderSchema);
}

export function createOrder(tableId: string, tableName: string, guestCount: number) {
  const auth = useAuthStore();
  return request('/orders', orderSchema, {
    method: 'POST',
    body: JSON.stringify({
      restaurant_id: auth.restaurantId,
      table_id: tableId,
      table_name: tableName,
      guest_count: guestCount,
    }),
  });
}

export function addOrderLine(orderId: string, menuItemId: string, quantity = 1, selectedModifiers: SelectedModifierPayload[] = []) {
  return request(`/orders/${encodeURIComponent(orderId)}/lines`, orderLineSchema, {
    method: 'POST',
    body: JSON.stringify({
      menu_item_id: menuItemId,
      quantity,
      selected_modifiers: selectedModifiers,
    }),
  });
}

export function changeOrderLineQuantity(orderId: string, lineId: string, quantity: number) {
  return request(`/orders/${encodeURIComponent(orderId)}/lines/${encodeURIComponent(lineId)}`, orderLineSchema, {
    method: 'PATCH',
    body: JSON.stringify({ quantity }),
  });
}

export function updateOrderLineDetails(orderId: string, lineId: string, course: string, comment: string) {
  return request(`/orders/${encodeURIComponent(orderId)}/lines/${encodeURIComponent(lineId)}/details`, orderLineSchema, {
    method: 'PATCH',
    body: JSON.stringify({ course, comment }),
  });
}

export function voidOrderLine(orderId: string, lineId: string, reason: string) {
  return request(`/orders/${encodeURIComponent(orderId)}/lines/${encodeURIComponent(lineId)}/void`, orderLineSchema, {
    method: 'POST',
    body: JSON.stringify({ reason }),
  });
}

export function closeOrder(orderId: string) {
  return request(`/orders/${encodeURIComponent(orderId)}/close`, orderSchema, {
    method: 'POST',
    body: JSON.stringify({}),
  });
}

export function listPrechecksByOrder(orderId: string) {
  return request(`/orders/${encodeURIComponent(orderId)}/prechecks`, z.array(precheckSchema));
}

export function issuePrecheck(orderId: string) {
  return request(`/orders/${encodeURIComponent(orderId)}/precheck`, precheckSchema, {
    method: 'POST',
    body: JSON.stringify({}),
  });
}

export function cancelPrecheck(precheckId: string, managerEmployeeId: string, managerPin: string, cancellationReason: string) {
  return request(`/prechecks/${encodeURIComponent(precheckId)}/cancel`, precheckSchema, {
    method: 'POST',
    body: JSON.stringify({
      manager_employee_id: managerEmployeeId,
      manager_pin: managerPin,
      cancellation_reason: cancellationReason,
    }),
  });
}

export function reprintPrecheck(precheckId: string) {
  return request(`/prechecks/${encodeURIComponent(precheckId)}/reprint`, reprintDocumentSchema, {
    method: 'POST',
    body: JSON.stringify({}),
  });
}

export function capturePrecheckPayment(precheckId: string, method: 'cash' | 'card', amount: number, currency: string) {
  return request(`/prechecks/${encodeURIComponent(precheckId)}/payments`, paymentSchema, {
    method: 'POST',
    body: JSON.stringify({
      method,
      amount,
      currency,
      provider_name: method === 'card' ? 'trusted_manual' : undefined,
    }),
  });
}

export function getCheck(checkId: string) {
  return request(`/checks/${encodeURIComponent(checkId)}`, checkSchema);
}

export function reprintCheck(checkId: string) {
  return request(`/checks/${encodeURIComponent(checkId)}/reprint`, reprintDocumentSchema, {
    method: 'POST',
    body: JSON.stringify({}),
  });
}

export function getSyncStatus() {
  return request('/sync/status', syncStatusSchema);
}

export function listSyncOutbox(limit = 5) {
  return request(`/sync/outbox?limit=${limit}`, z.array(outboxMessageSchema));
}

export function listLocalEvents(limit = 5, eventType = '') {
  const query = new URLSearchParams({ limit: String(limit) });
  if (eventType.trim()) query.set('event_type', eventType.trim());
  return request(`/sync/local-events?${query}`, z.array(localEventSchema));
}

export function retryFailedOutbox() {
  return request('/sync/retry-failed', retryFailedOutboxResultSchema, {
    method: 'POST',
    body: JSON.stringify({}),
  });
}

export function listClosedOrders(limit = 50) {
  return request(`/orders/closed?limit=${limit}`, z.array(closedOrderSchema));
}

export function refundPayment(paymentId: string, payload: string | PaymentRefundPayload = '') {
  const body = typeof payload === 'string'
    ? { command_id: nextCommandId('payment-refund'), reason: payload }
    : { command_id: payload.commandId ?? nextCommandId('payment-refund'), reason: payload.reason ?? '' };
  return request(`/payments/${encodeURIComponent(paymentId)}/refund`, paymentSchema, {
    method: 'POST',
    body: JSON.stringify(body),
  });
}

export function recordCheckCancellation(checkId: string, payload: string | CheckLedgerOperationPayload, inventoryDisposition: InventoryDisposition = 'no_stock_effect') {
  const body = typeof payload === 'string'
    ? {
      command_id: nextCommandId('check-cancellation'),
      operation_kind: 'full' satisfies FinancialOperationKind,
      inventory_disposition: inventoryDisposition,
      reason: requireLedgerReason(payload),
    }
    : {
      command_id: payload.commandId ?? nextCommandId('check-cancellation'),
      operation_kind: payload.operationKind ?? 'full',
      inventory_disposition: payload.inventoryDisposition ?? 'no_stock_effect',
      reason: requireLedgerReason(payload.reason),
      items: mapFinancialOperationItems(payload.items),
    };
  return request(`/checks/${encodeURIComponent(checkId)}/cancellations`, financialOperationSchema, {
    method: 'POST',
    body: JSON.stringify(body),
  });
}

export function recordCheckRefund(checkId: string, payload: string | CheckLedgerOperationPayload, inventoryDisposition: InventoryDisposition = 'no_stock_effect') {
  const body = typeof payload === 'string'
    ? {
      command_id: nextCommandId('check-refund'),
      operation_kind: 'full' satisfies FinancialOperationKind,
      inventory_disposition: inventoryDisposition,
      reason: requireLedgerReason(payload),
    }
    : {
      command_id: payload.commandId ?? nextCommandId('check-refund'),
      operation_kind: payload.operationKind ?? 'full',
      inventory_disposition: payload.inventoryDisposition ?? 'no_stock_effect',
      reason: requireLedgerReason(payload.reason),
      items: mapFinancialOperationItems(payload.items),
    };
  return request(`/checks/${encodeURIComponent(checkId)}/refunds`, financialOperationSchema, {
    method: 'POST',
    body: JSON.stringify(body),
  });
}
