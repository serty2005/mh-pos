import { z } from 'zod';

import {
  catalogItemSchema,
  cashDrawerEventSchema,
  cashSessionSchema,
  closedOrderSchema,
  financialOperationSchema,
  hallSchema,
  kitchenOrderQueueResponseSchema,
  kitchenProposalSchema,
  kitchenRecipeSchema,
  kitchenStopListStateSchema,
  kitchenTicketSchema,
  menuItemSchema,
  orderLineSchema,
  orderSchema,
  pricingCalculationSchema,
  outboxMessageSchema,
  pairingStatusSchema,
  paymentSchema,
  pinLoginResultSchema,
  precheckSchema,
  pricingPolicySchema,
  orderDiscountSchema,
  orderSurchargeSchema,
  provisioningStatusSchema,
  reprintDocumentSchema,
  retryFailedOutboxResultSchema,
  shiftSchema,
  storageStatusSchema,
  syncStatusSchema,
  tableSchema,
} from './schemas';

export type AuthSnapshot = {
  clientDeviceId: string;
  nodeDeviceId: string;
  restaurantId: string;
  sessionId: string;
  actorEmployeeId: string;
};

export type SelectedModifierPayload = {
  modifier_group_id: string;
  modifier_option_id: string;
  quantity: number;
};

export type CashDrawerEventType = 'cash_in' | 'cash_out' | 'no_sale' | 'cash_count';
export type InventoryDisposition = 'no_stock_effect' | 'return_to_stock' | 'write_off_waste' | 'manual_review';
export type FinancialOperationKind = 'full' | 'partial';
export type FinancialOperationItemScope = 'whole_check' | 'order_line' | 'modifier_line' | 'service_charge' | 'tip' | 'payment';
export type KitchenTicketAction = 'accept' | 'start' | 'hold' | 'ready' | 'serve' | 'recall' | 'cancel';
export type KitchenStopListUpdatePayload = {
  command_id?: string;
  stop_list_id?: string;
  warehouse_id?: string;
  catalog_item_id: string;
  available_quantity?: number;
  active: boolean;
  reason?: string;
};

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

export type ClosedOrdersQuery = {
  businessDateLocal?: string;
  fromBusinessDateLocal?: string;
  toBusinessDateLocal?: string;
  shiftId?: string;
  deviceId?: string;
  checkId?: string;
  limit?: number;
  offset?: number;
};

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

export type ApiErrorOptions = {
  status: number;
  code: string;
  messageKey: string;
  category: ApiErrorCategory;
  details?: Record<string, string>;
  correlationId?: string;
  retryable?: boolean;
};

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

let commandSequence = 0;

function defaultApiBase() {
  const hostname = globalThis.location?.hostname;
  if (hostname === 'host.docker.internal') {
    return 'http://host.docker.internal:8080/api/v1';
  }
  return 'http://localhost:8080/api/v1';
}

function nextCommandId(prefix: string) {
  commandSequence += 1;
  return `cmd-pos-ui-g-${Date.now()}-${commandSequence}-${prefix}`;
}

function appendQueryParam(params: URLSearchParams, key: string, value: string | undefined) {
  const normalized = value?.trim();
  if (normalized) params.set(key, normalized);
}

const viteEnv = import.meta as unknown as { env?: Record<string, string | undefined> };

export function createApiClient(getAuth: () => AuthSnapshot, base = (viteEnv.env?.VITE_POS_API_BASE ?? defaultApiBase()).replace(/\/$/, '')) {
  async function request<T>(path: string, schema: z.ZodType<T>, init: RequestInit = {}) {
    const auth = getAuth();
    const headers = new Headers(init.headers);
    const hasBody = init.body !== undefined && init.body !== null;
    if (hasBody && !headers.has('Content-Type')) {
      headers.set('Content-Type', 'application/json');
    }
    headers.set('X-Client-Device-ID', auth.clientDeviceId);
    if (auth.nodeDeviceId) headers.set('X-Node-Device-ID', auth.nodeDeviceId);
    if (auth.sessionId) headers.set('X-Session-ID', auth.sessionId);
    if (auth.actorEmployeeId) headers.set('X-Actor-Employee-ID', auth.actorEmployeeId);

    let response: Response;
    try {
      response = await fetch(`${base}${path}`, { ...init, headers });
    } catch (error) {
      throw networkApiError(error);
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
      });
    }
  }

  async function requestOptional<T>(path: string, schema: z.ZodType<T>, init: RequestInit = {}) {
    try {
      return await request(path, schema.nullable(), init);
    } catch (error) {
      if (error instanceof ApiError && error.status === 404) return null;
      throw error;
    }
  }

  return {
    getPairingStatus: () => request('/system/pairing-status', pairingStatusSchema),
    getProvisioningStatus: () => request('/system/provisioning-status', provisioningStatusSchema),
    registerCloudProvisioning: (cloudUrl = '') => request('/system/provisioning/register-cloud', provisioningStatusSchema, {
      method: 'POST',
      body: JSON.stringify({
        cloud_url: cloudUrl,
        display_name: 'POS Terminal',
        app_version: viteEnv.env?.VITE_APP_VERSION ?? 'pos-ui-g',
      }),
    }),
    pairViaLicense: (pairingCode: string) => request('/system/provisioning/pair-via-license', provisioningStatusSchema, {
      method: 'POST',
      body: JSON.stringify({ pairing_code: pairingCode }),
    }),
    pinLogin: (pin: string) => request('/auth/pin-login', pinLoginResultSchema, {
      method: 'POST',
      body: JSON.stringify({
        node_device_id: getAuth().nodeDeviceId,
        client_device_id: getAuth().clientDeviceId,
        pin,
      }),
    }),
    getAuthSession: () => {
      const auth = getAuth();
      const query = new URLSearchParams({
        session_id: auth.sessionId,
        node_device_id: auth.nodeDeviceId,
        client_device_id: auth.clientDeviceId,
      });
      return request(`/auth/session?${query}`, pinLoginResultSchema);
    },
    logout: () => {
      const auth = getAuth();
      return request('/auth/logout', z.unknown(), {
        method: 'POST',
        body: JSON.stringify({
          node_device_id: auth.nodeDeviceId,
          client_device_id: auth.clientDeviceId,
          session_id: auth.sessionId,
        }),
      });
    },
    getCurrentShift: () => requestOptional(`/employee-shifts/current?${new URLSearchParams({ node_device_id: getAuth().nodeDeviceId })}`, shiftSchema),
    listRecentShifts: () => request('/employee-shifts/recent?limit=5', z.array(shiftSchema)),
    openShift: () => request('/employee-shifts/open', shiftSchema, {
      method: 'POST',
      body: JSON.stringify({
        restaurant_id: getAuth().restaurantId,
        opened_by_employee_id: getAuth().actorEmployeeId,
      }),
    }),
    closeShift: (shiftId: string) => request(`/employee-shifts/${encodeURIComponent(shiftId)}/close`, shiftSchema, {
      method: 'POST',
      body: JSON.stringify({ closed_by_employee_id: getAuth().actorEmployeeId }),
    }),
    getCurrentCashSession: () => requestOptional(`/cash-shifts/current?${new URLSearchParams({ node_device_id: getAuth().nodeDeviceId })}`, cashSessionSchema),
    openCashSession: (openingCashAmount: number) => request('/cash-shifts/open', cashSessionSchema, {
      method: 'POST',
      body: JSON.stringify({
        restaurant_id: getAuth().restaurantId,
        opened_by_employee_id: getAuth().actorEmployeeId,
        opening_cash_amount: openingCashAmount,
      }),
    }),
    closeCashSession: (cashSessionId: string, closingCashAmount: number) => request(`/cash-shifts/${encodeURIComponent(cashSessionId)}/close`, cashSessionSchema, {
      method: 'POST',
      body: JSON.stringify({
        closed_by_employee_id: getAuth().actorEmployeeId,
        closing_cash_amount: closingCashAmount,
      }),
    }),
    recordCashDrawerEvent: (cashSessionId: string, eventType: CashDrawerEventType, amount: number, reason = '', note = '') => request('/cash-drawer-events', cashDrawerEventSchema, {
      method: 'POST',
      body: JSON.stringify({
        cash_session_id: cashSessionId,
        created_by_employee_id: getAuth().actorEmployeeId,
        event_type: eventType,
        amount,
        reason,
        note,
      }),
    }),
    listHalls: () => request(`/halls?${new URLSearchParams({ restaurant_id: getAuth().restaurantId })}`, z.array(hallSchema)),
    listTables: (hallId: string) => request(`/tables?${new URLSearchParams({ restaurant_id: getAuth().restaurantId, hall_id: hallId })}`, z.array(tableSchema)),
    listMenuItems: () => request('/menu/items', z.array(menuItemSchema)),
    listCatalogItems: () => request('/catalog/items', z.array(catalogItemSchema)),
    getCurrentOrderByTable: (tableId: string) => requestOptional(`/orders/current?${new URLSearchParams({ table_id: tableId })}`, orderSchema),
    listActiveOrdersByHall: (hallId: string) => request(`/orders/active?${new URLSearchParams({ hall_id: hallId })}`, z.array(orderSchema)),
    getOrder: (orderId: string) => request(`/orders/${encodeURIComponent(orderId)}`, orderSchema),
    getOrderPricing: (orderId: string) => request(`/orders/${encodeURIComponent(orderId)}/pricing`, pricingCalculationSchema),
    createOrder: (tableId: string, tableName: string, guestCount: number) => request('/orders', orderSchema, {
      method: 'POST',
      body: JSON.stringify({
        restaurant_id: getAuth().restaurantId,
        table_id: tableId,
        table_name: tableName,
        guest_count: guestCount,
      }),
    }),
    addOrderLine: (orderId: string, menuItemId: string, quantity = 1, selectedModifiers: SelectedModifierPayload[] = []) => request(`/orders/${encodeURIComponent(orderId)}/lines`, orderLineSchema, {
      method: 'POST',
      body: JSON.stringify({ menu_item_id: menuItemId, quantity, selected_modifiers: selectedModifiers }),
    }),
    changeOrderLineQuantity: (orderId: string, lineId: string, quantity: number) => request(`/orders/${encodeURIComponent(orderId)}/lines/${encodeURIComponent(lineId)}`, orderLineSchema, {
      method: 'PATCH',
      body: JSON.stringify({ quantity }),
    }),
    updateOrderLineDetails: (orderId: string, lineId: string, course: string, comment: string) => request(`/orders/${encodeURIComponent(orderId)}/lines/${encodeURIComponent(lineId)}/details`, orderLineSchema, {
      method: 'PATCH',
      body: JSON.stringify({ course, comment }),
    }),
    updateOrderLineModifiers: (orderId: string, lineId: string, selectedModifiers: SelectedModifierPayload[]) => request(`/orders/${encodeURIComponent(orderId)}/lines/${encodeURIComponent(lineId)}/modifiers`, orderLineSchema, {
      method: 'PATCH',
      body: JSON.stringify({ selected_modifiers: selectedModifiers }),
    }),
    voidOrderLine: (orderId: string, lineId: string, reason: string) => request(`/orders/${encodeURIComponent(orderId)}/lines/${encodeURIComponent(lineId)}/void`, orderLineSchema, {
      method: 'POST',
      body: JSON.stringify({ reason }),
    }),
    listActivePricingPolicies: () => request('/pricing/policies', z.array(pricingPolicySchema)),
    applyDiscountPolicy: (orderId: string, pricingPolicyId: string, orderLineId = '', reason = '') => request(`/orders/${encodeURIComponent(orderId)}/discounts`, orderDiscountSchema, {
      method: 'POST',
      body: JSON.stringify({ pricing_policy_id: pricingPolicyId, order_line_id: orderLineId, reason }),
    }),
    applySurchargePolicy: (orderId: string, pricingPolicyId: string, reason = '') => request(`/orders/${encodeURIComponent(orderId)}/surcharges`, orderSurchargeSchema, {
      method: 'POST',
      body: JSON.stringify({ pricing_policy_id: pricingPolicyId, reason }),
    }),
    listPrechecksByOrder: (orderId: string) => request(`/orders/${encodeURIComponent(orderId)}/prechecks`, z.array(precheckSchema)),
    issuePrecheck: (orderId: string) => request(`/orders/${encodeURIComponent(orderId)}/precheck`, precheckSchema, {
      method: 'POST',
      body: JSON.stringify({}),
    }),
    cancelPrecheck: (precheckId: string, managerPin: string, cancellationReason: string) => request(`/prechecks/${encodeURIComponent(precheckId)}/cancel`, precheckSchema, {
      method: 'POST',
      body: JSON.stringify({ manager_pin: managerPin, cancellation_reason: cancellationReason }),
    }),
    reprintPrecheck: (precheckId: string) => request(`/prechecks/${encodeURIComponent(precheckId)}/reprint`, reprintDocumentSchema, {
      method: 'POST',
      body: JSON.stringify({}),
    }),
    capturePrecheckPayment: (precheckId: string, method: 'cash' | 'card', amount: number, currency: string) => request(`/prechecks/${encodeURIComponent(precheckId)}/payments`, paymentSchema, {
      method: 'POST',
      body: JSON.stringify({ method, amount, currency, provider_name: method === 'card' ? 'trusted_manual' : undefined }),
    }),
    getCheck: (checkId: string) => request(`/checks/${encodeURIComponent(checkId)}`, z.unknown()),
    listFinancialOperationsByCheck: (checkId: string, limit = 50, offset = 0) => request(`/checks/${encodeURIComponent(checkId)}/financial-operations?${new URLSearchParams({ limit: String(limit), offset: String(offset) })}`, z.array(financialOperationSchema)),
    reprintCheck: (checkId: string) => request(`/checks/${encodeURIComponent(checkId)}/reprint`, reprintDocumentSchema, {
      method: 'POST',
      body: JSON.stringify({}),
    }),
    listClosedOrders: (query: number | ClosedOrdersQuery = 50, offset = 0) => {
      const params = new URLSearchParams();
      if (typeof query === 'number') {
        params.set('limit', String(query));
        params.set('offset', String(offset));
      } else {
        appendQueryParam(params, 'business_date_local', query.businessDateLocal);
        appendQueryParam(params, 'from_business_date_local', query.fromBusinessDateLocal);
        appendQueryParam(params, 'to_business_date_local', query.toBusinessDateLocal);
        appendQueryParam(params, 'shift_id', query.shiftId);
        appendQueryParam(params, 'device_id', query.deviceId);
        appendQueryParam(params, 'check_id', query.checkId);
        if (query.limit !== undefined) params.set('limit', String(query.limit));
        if (query.offset !== undefined) params.set('offset', String(query.offset));
      }
      const suffix = params.toString();
      return request(`/orders/closed${suffix ? `?${suffix}` : ''}`, z.array(closedOrderSchema));
    },
    refundPayment: (paymentId: string, reason = '') => request(`/payments/${encodeURIComponent(paymentId)}/refund`, paymentSchema, {
      method: 'POST',
      body: JSON.stringify({ command_id: nextCommandId('payment-refund'), reason }),
    }),
    recordCheckRefund: (checkId: string, payload: CheckLedgerOperationPayload) => request(`/checks/${encodeURIComponent(checkId)}/refunds`, financialOperationSchema, {
      method: 'POST',
      body: JSON.stringify(mapLedgerPayload(payload, 'check-refund')),
    }),
    recordCheckCancellation: (checkId: string, payload: CheckLedgerOperationPayload) => request(`/checks/${encodeURIComponent(checkId)}/cancellations`, financialOperationSchema, {
      method: 'POST',
      body: JSON.stringify(mapLedgerPayload(payload, 'check-cancellation')),
    }),
    getSyncStatus: () => request('/sync/status', syncStatusSchema),
    listSyncOutbox: (limit = 5) => request(`/sync/outbox?limit=${limit}`, z.array(outboxMessageSchema)),
    getStorageStatus: () => request('/storage/status', storageStatusSchema),
    retryFailedOutbox: () => request('/sync/retry-failed', retryFailedOutboxResultSchema, {
      method: 'POST',
      body: JSON.stringify({}),
    }),
    listKitchenOrderQueue: (query: { status?: string; station?: string; limit?: number; offset?: number } = {}) => {
      const params = new URLSearchParams();
      if (query.status) params.set('status', query.status);
      if (query.station) params.set('station', query.station);
      params.set('limit', String(query.limit ?? 50));
      params.set('offset', String(query.offset ?? 0));
      return request(`/kitchen/order-queue?${params}`, kitchenOrderQueueResponseSchema);
    },
    listKitchenTickets: (query: { status?: string; station?: string; limit?: number; offset?: number } = {}) => {
      const params = new URLSearchParams();
      if (query.status) params.set('status', query.status);
      if (query.station) params.set('station', query.station);
      params.set('limit', String(query.limit ?? 50));
      params.set('offset', String(query.offset ?? 0));
      return request(`/kitchen/tickets?${params}`, z.array(kitchenTicketSchema));
    },
    changeKitchenTicketStatus: (ticketId: string, action: KitchenTicketAction) => request(`/kitchen/tickets/${encodeURIComponent(ticketId)}/${action}`, kitchenTicketSchema, {
      method: 'POST',
      body: JSON.stringify({ command_id: nextCommandId(`kitchen-${action}`) }),
    }),
    submitKitchenStockReceipt: (payload: unknown) => request('/kitchen/stock-receipts', z.unknown(), {
      method: 'POST',
      body: JSON.stringify(payload),
    }),
    submitKitchenInventoryCount: (payload: unknown) => request('/kitchen/inventory-counts', z.unknown(), {
      method: 'POST',
      body: JSON.stringify(payload),
    }),
    submitKitchenWriteOff: (payload: unknown) => request('/kitchen/stock-write-offs', z.unknown(), {
      method: 'POST',
      body: JSON.stringify(payload),
    }),
    submitKitchenProduction: (payload: unknown) => request('/kitchen/productions', z.unknown(), {
      method: 'POST',
      body: JSON.stringify(payload),
    }),
    listKitchenStopList: () => request('/kitchen/stop-list', z.array(kitchenStopListStateSchema)),
    submitKitchenStopListUpdate: (payload: KitchenStopListUpdatePayload) => request('/kitchen/stop-list-updates', z.unknown(), {
      method: 'POST',
      body: JSON.stringify({
        ...payload,
        command_id: payload.command_id ?? nextCommandId('stop-list-update'),
      }),
    }),
    getKitchenRecipe: (catalogItemId: string) => request(`/kitchen/catalog/items/${encodeURIComponent(catalogItemId)}/recipe`, kitchenRecipeSchema),
    submitCatalogSuggestion: (payload: unknown) => request('/kitchen/catalog-suggestions', z.unknown(), {
      method: 'POST',
      body: JSON.stringify(payload),
    }),
    submitRecipeSuggestion: (payload: unknown) => request('/kitchen/recipe-suggestions', z.unknown(), {
      method: 'POST',
      body: JSON.stringify(payload),
    }),
    listKitchenProposals: (query: { kind?: string; status?: string; limit?: number; offset?: number } = {}) => {
      const params = new URLSearchParams();
      if (query.kind) params.set('kind', query.kind);
      if (query.status) params.set('status', query.status);
      params.set('limit', String(query.limit ?? 50));
      params.set('offset', String(query.offset ?? 0));
      return request(`/kitchen/proposals?${params}`, z.array(kitchenProposalSchema));
    },
  };
}

async function parseResponseBody(response: Response) {
  const text = await response.text();
  if (!text.trim()) return null;
  try {
    return JSON.parse(text) as unknown;
  } catch {
    if (!response.ok) return null;
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

function mapLedgerPayload(payload: CheckLedgerOperationPayload, prefix: string) {
  return {
    command_id: payload.commandId ?? nextCommandId(prefix),
    operation_kind: payload.operationKind ?? 'full',
    inventory_disposition: payload.inventoryDisposition ?? 'no_stock_effect',
    reason: payload.reason.trim(),
    items: payload.items?.map((item) => ({
      scope: item.scope,
      order_line_id: item.orderLineId,
      payment_id: item.paymentId,
      quantity: item.quantity,
      amount: item.amount,
      currency: item.currency,
      tax_amount: item.taxAmount,
      snapshot: item.snapshot,
    })),
  };
}

export type PosApiClient = ReturnType<typeof createApiClient>;
