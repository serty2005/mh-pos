import { beforeEach, describe, expect, it, vi } from 'vitest';

import {
  addOrderLine,
  ApiError,
  getCurrentOrderByTable,
  getCurrentShift,
  listClosedOrders,
  listMenuItems,
  recordCheckCancellation,
  recordCheckRefund,
  refundPayment,
} from './api';

const authState = {
  clientDeviceId: 'client-1',
  nodeDeviceId: 'node-1',
  sessionId: 'session-1',
  actor: { employee_id: 'employee-1' },
};

vi.mock('../stores/auth', () => ({
  useAuthStore: () => authState,
}));

describe('api request helpers', () => {
  beforeEach(() => {
    vi.restoreAllMocks();
    vi.unstubAllGlobals();
  });

  it('sends required auth/device headers for runtime reads', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      text: async () => '[]',
    });
    vi.stubGlobal('fetch', fetchMock);

    await listMenuItems();

    expect(fetchMock).toHaveBeenCalledTimes(1);
    const init = fetchMock.mock.calls[0]?.[1] as RequestInit;
    const headers = new Headers(init.headers);
    expect(headers.get('X-Client-Device-ID')).toBe('client-1');
    expect(headers.get('X-Node-Device-ID')).toBe('node-1');
    expect(headers.get('X-Session-ID')).toBe('session-1');
    expect(headers.get('X-Actor-Employee-ID')).toBe('employee-1');
  });

  it('maps backend error payload into ApiError', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: false,
      status: 429,
      statusText: 'Too Many Requests',
      headers: new Headers({ 'X-Request-ID': 'req-1' }),
      text: async () => JSON.stringify({
        error: {
          code: 'RATE_LIMITED',
          message_key: 'errors.rateLimit',
          correlation_id: 'req-1',
        },
      }),
    });
    vi.stubGlobal('fetch', fetchMock);

    let thrown: unknown;
    try {
      await listMenuItems();
    } catch (error) {
      thrown = error;
    }
    expect(thrown).toBeInstanceOf(ApiError);
    expect(thrown).toMatchObject({
      status: 429,
      code: 'RATE_LIMITED',
      messageKey: 'errors.rateLimit',
      category: 'rate_limit',
      correlationId: 'req-1',
    });
  });

  it('keeps backend message_key ahead of generic conflict fallback', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: false,
      status: 409,
      headers: new Headers({ 'X-Request-ID': 'req-409' }),
      text: async () => JSON.stringify({
        error: {
          code: 'ACTIVE_PRECHECK_CONFLICT',
          message_key: 'errors.conflict_active_precheck',
          correlation_id: 'req-409',
        },
      }),
    });
    vi.stubGlobal('fetch', fetchMock);

    await expect(listMenuItems()).rejects.toMatchObject({
      status: 409,
      code: 'ACTIVE_PRECHECK_CONFLICT',
      messageKey: 'errors.conflict_active_precheck',
      category: 'conflict',
      correlationId: 'req-409',
    });
  });

  it('maps generic 409 fallback to the localized conflict category key', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: false,
      status: 409,
      headers: new Headers(),
      text: async () => JSON.stringify({ error: 'domain invariant violation' }),
    });
    vi.stubGlobal('fetch', fetchMock);

    await expect(listMenuItems()).rejects.toMatchObject({
      status: 409,
      code: 'CONFLICT',
      messageKey: 'errors.conflict',
      category: 'conflict',
      retryable: false,
    });
  });

  it('treats current shift and current order 404 as optional empty state', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: false,
      status: 404,
      headers: new Headers(),
      text: async () => JSON.stringify({
        error: {
          code: 'NOT_FOUND',
          message_key: 'errors.not_found',
        },
      }),
    });
    vi.stubGlobal('fetch', fetchMock);

    await expect(getCurrentShift()).resolves.toBeNull();
    await expect(getCurrentOrderByTable('table-1')).resolves.toBeNull();
    expect(fetchMock).toHaveBeenCalledTimes(2);
  });

  it('treats current shift 200 null as optional empty state', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      headers: new Headers(),
      text: async () => 'null',
    });
    vi.stubGlobal('fetch', fetchMock);

    await expect(getCurrentShift()).resolves.toBeNull();
    expect(fetchMock).toHaveBeenCalledTimes(1);
  });

  it('maps network failure without clearing client identity in api layer', async () => {
    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new TypeError('failed to fetch')));

    await expect(listMenuItems()).rejects.toMatchObject({
      code: 'NETWORK_ERROR',
      messageKey: 'errors.network.unavailable',
      category: 'network',
      retryable: true,
    });
  });

  it('sends selected modifiers when adding an order line', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      text: async () => JSON.stringify({
        id: 'line-1',
        order_id: 'order-1',
        menu_item_id: 'menu-1',
        catalog_item_id: 'catalog-1',
        name: 'Soup',
        quantity: 1,
        unit_price: 1000,
        total_price: 1250,
        currency_code: 'RUB',
        tax_profile_id: null,
        modifiers: [{
          id: 'line-modifier-1',
          order_line_id: 'line-1',
          modifier_group_id: 'group-1',
          modifier_option_id: 'option-1',
          name: 'Hot',
          quantity: 1,
          unit_price: 250,
          total_price: 250,
        }],
        status: 'active',
        created_at: '2026-05-09T10:00:00Z',
        updated_at: '2026-05-09T10:00:00Z',
      }),
    });
    vi.stubGlobal('fetch', fetchMock);

    await addOrderLine('order-1', 'menu-1', 1, [{ modifier_group_id: 'group-1', modifier_option_id: 'option-1', quantity: 1 }]);

    const init = fetchMock.mock.calls[0]?.[1] as RequestInit;
    expect(JSON.parse(String(init.body))).toMatchObject({
      menu_item_id: 'menu-1',
      quantity: 1,
      selected_modifiers: [{ modifier_group_id: 'group-1', modifier_option_id: 'option-1', quantity: 1 }],
    });
  });

  it('records full check cancellation through ledger endpoint', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      text: async () => JSON.stringify(financialOperationResponse('cancellation')),
    });
    vi.stubGlobal('fetch', fetchMock);

    await recordCheckCancellation('check-1', {
      commandId: 'cmd-ui-cancel-check',
      operationKind: 'full',
      inventoryDisposition: 'return_to_stock',
      reason: 'guest left',
    });

    expect(fetchMock).toHaveBeenCalledTimes(1);
    expect(fetchMock.mock.calls[0]?.[0]).toContain('/checks/check-1/cancellations');
    const init = fetchMock.mock.calls[0]?.[1] as RequestInit;
    expect(init.method).toBe('POST');
    expect(JSON.parse(String(init.body))).toEqual({
      command_id: 'cmd-ui-cancel-check',
      operation_kind: 'full',
      inventory_disposition: 'return_to_stock',
      reason: 'guest left',
    });
  });

  it('requests closed orders with bounded pagination and filters', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      text: async () => '[]',
    });
    vi.stubGlobal('fetch', fetchMock);

    await listClosedOrders({
      businessDateLocal: '2026-05-16',
      shiftId: 'shift-1',
      checkId: 'check-1',
      limit: 50,
      offset: 100,
    });

    expect(fetchMock).toHaveBeenCalledTimes(1);
    const url = String(fetchMock.mock.calls[0]?.[0]);
    expect(url).toContain('/orders/closed?');
    expect(url).toContain('business_date_local=2026-05-16');
    expect(url).toContain('shift_id=shift-1');
    expect(url).toContain('check_id=check-1');
    expect(url).toContain('limit=50');
    expect(url).toContain('offset=100');
  });

  it('records full check refund through ledger endpoint', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      text: async () => JSON.stringify(financialOperationResponse('refund')),
    });
    vi.stubGlobal('fetch', fetchMock);

    await recordCheckRefund('check-1', {
      commandId: 'cmd-ui-refund-check',
      operationKind: 'full',
      inventoryDisposition: 'write_off_waste',
      reason: 'guest refund',
    });

    expect(fetchMock).toHaveBeenCalledTimes(1);
    expect(fetchMock.mock.calls[0]?.[0]).toContain('/checks/check-1/refunds');
    const init = fetchMock.mock.calls[0]?.[1] as RequestInit;
    expect(init.method).toBe('POST');
    expect(JSON.parse(String(init.body))).toEqual({
      command_id: 'cmd-ui-refund-check',
      operation_kind: 'full',
      inventory_disposition: 'write_off_waste',
      reason: 'guest refund',
    });
  });

  it('records partial order-line cancellation payload through ledger endpoint', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      text: async () => JSON.stringify(financialOperationResponse('cancellation', 'partial', 'order_line')),
    });
    vi.stubGlobal('fetch', fetchMock);

    await recordCheckCancellation('check-1', {
      commandId: 'cmd-ui-cancel-line',
      operationKind: 'partial',
      inventoryDisposition: 'manual_review',
      reason: 'line issue',
      items: [{
        scope: 'order_line',
        orderLineId: 'line-1',
        quantity: 1,
        amount: 500,
        currency: 'RUB',
        taxAmount: 45,
      }],
    });

    expect(fetchMock.mock.calls[0]?.[0]).toContain('/checks/check-1/cancellations');
    const init = fetchMock.mock.calls[0]?.[1] as RequestInit;
    expect(JSON.parse(String(init.body))).toEqual({
      command_id: 'cmd-ui-cancel-line',
      operation_kind: 'partial',
      inventory_disposition: 'manual_review',
      reason: 'line issue',
      items: [{
        scope: 'order_line',
        order_line_id: 'line-1',
        quantity: 1,
        amount: 500,
        currency: 'RUB',
        tax_amount: 45,
      }],
    });
  });

  it('records partial order-line refund payload through ledger endpoint', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      text: async () => JSON.stringify(financialOperationResponse('refund', 'partial', 'order_line')),
    });
    vi.stubGlobal('fetch', fetchMock);

    await recordCheckRefund('check-1', {
      commandId: 'cmd-ui-refund-line',
      operationKind: 'partial',
      inventoryDisposition: 'return_to_stock',
      reason: 'line refund',
      items: [{
        scope: 'order_line',
        orderLineId: 'line-1',
        quantity: 2,
        amount: 1000,
        currency: 'RUB',
        taxAmount: 90,
      }],
    });

    expect(fetchMock.mock.calls[0]?.[0]).toContain('/checks/check-1/refunds');
    const init = fetchMock.mock.calls[0]?.[1] as RequestInit;
    expect(JSON.parse(String(init.body))).toEqual({
      command_id: 'cmd-ui-refund-line',
      operation_kind: 'partial',
      inventory_disposition: 'return_to_stock',
      reason: 'line refund',
      items: [{
        scope: 'order_line',
        order_line_id: 'line-1',
        quantity: 2,
        amount: 1000,
        currency: 'RUB',
        tax_amount: 90,
      }],
    });
  });

  it('requires reason for check ledger cancellation/refund clients', async () => {
    const fetchMock = vi.fn();
    vi.stubGlobal('fetch', fetchMock);

    let cancellationError: unknown;
    try {
      recordCheckCancellation('check-1', {
        commandId: 'cmd-empty-cancel',
        operationKind: 'full',
        inventoryDisposition: 'no_stock_effect',
        reason: '   ',
      });
    } catch (error) {
      cancellationError = error;
    }
    let refundError: unknown;
    try {
      recordCheckRefund('check-1', {
        commandId: 'cmd-empty-refund',
        operationKind: 'full',
        inventoryDisposition: 'no_stock_effect',
        reason: '',
      });
    } catch (error) {
      refundError = error;
    }
    expect(cancellationError).toBeInstanceOf(ApiError);
    expect(cancellationError).toMatchObject({
      code: 'VALIDATION_FAILED',
      messageKey: 'errors.validation',
      category: 'validation',
    });
    expect(refundError).toBeInstanceOf(ApiError);
    expect(refundError).toMatchObject({
      code: 'VALIDATION_FAILED',
      messageKey: 'errors.validation',
      category: 'validation',
    });
    expect(fetchMock).not.toHaveBeenCalled();
  });

  it('keeps compatibility payment refund available as fallback', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      text: async () => JSON.stringify({
        id: 'payment-1',
        edge_payment_id: 'edge-payment-1',
        restaurant_id: 'restaurant-1',
        device_id: 'node-1',
        shift_id: 'shift-1',
        precheck_id: 'precheck-1',
        method: 'cash',
        amount: 1000,
        currency: 'RUB',
        status: 'captured',
        business_date_local: '2026-05-16',
        provider_name: null,
        provider_transaction_id: null,
        provider_reference: null,
        fingerprint_hash: null,
        created_at: '2026-05-16T10:00:00Z',
        updated_at: '2026-05-16T10:00:00Z',
      }),
    });
    vi.stubGlobal('fetch', fetchMock);

    await refundPayment('payment-1', {
      commandId: 'cmd-ui-refund-payment',
      reason: 'legacy payment fallback',
    });

    expect(fetchMock).toHaveBeenCalledTimes(1);
    expect(fetchMock.mock.calls[0]?.[0]).toContain('/payments/payment-1/refund');
    const init = fetchMock.mock.calls[0]?.[1] as RequestInit;
    expect(init.method).toBe('POST');
    expect(JSON.parse(String(init.body))).toEqual({
      command_id: 'cmd-ui-refund-payment',
      reason: 'legacy payment fallback',
    });
  });
});

function financialOperationResponse(operationType: 'cancellation' | 'refund', operationKind: 'full' | 'partial' = 'full', scope: 'whole_check' | 'order_line' = 'whole_check') {
  return {
    id: `operation-${operationType}`,
    edge_operation_id: `edge-operation-${operationType}`,
    restaurant_id: 'restaurant-1',
    device_id: 'node-1',
    shift_id: 'shift-current',
    original_shift_id: 'shift-original',
    check_id: 'check-1',
    precheck_id: 'precheck-1',
    operation_type: operationType,
    operation_kind: operationKind,
    status: 'recorded',
    amount: 1000,
    currency: 'RUB',
    business_date_local: '2026-05-16',
    inventory_disposition: 'no_stock_effect',
    reason: 'guest request',
    created_by_employee_id: 'employee-1',
    approved_by_employee_id: null,
    snapshot: {},
    items: [{
      id: `item-${operationType}`,
      operation_id: `operation-${operationType}`,
      scope,
      order_line_id: scope === 'order_line' ? 'line-1' : null,
      payment_id: null,
      quantity: scope === 'order_line' ? 1 : null,
      amount: operationKind === 'partial' ? 500 : 1000,
      currency: 'RUB',
      tax_amount: 0,
      snapshot: {},
      created_at: '2026-05-16T10:00:00Z',
    }],
    created_at: '2026-05-16T10:00:00Z',
  };
}
