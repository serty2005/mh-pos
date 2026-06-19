import { beforeEach, describe, expect, it, vi } from 'vitest';
import type React from 'react';

import type { PosApiClient } from '../shared/api';
import type {
  BackendActorContext,
  BackendCashSession,
  BackendClosedOrder,
  BackendHall,
  BackendOrder,
  BackendPayment,
  BackendPrecheck,
  BackendShift,
  BackendSyncStatus,
  BackendTable,
} from '../shared/schemas';
import { permissions } from '../types';
import { POSProvider } from './POSContext';

const hookRuntime = vi.hoisted(() => {
  const missing = Symbol('missing-state-slot');
  let stateSlots: unknown[] = [];
  let stateIndex = 0;
  let refIndex = 0;
  let actorRefValue: unknown = null;

  return {
    missing,
    reset(nextSlots: unknown[], nextActorRefValue: unknown) {
      stateSlots = nextSlots;
      stateIndex = 0;
      refIndex = 0;
      actorRefValue = nextActorRefValue;
    },
    stateAt<T>(index: number) {
      return stateSlots[index] as T;
    },
    useState<T>(initialValue: T | (() => T)): [T, (next: T | ((previous: T) => T)) => void] {
      const index = stateIndex;
      stateIndex += 1;
      if (stateSlots[index] === missing) {
        stateSlots[index] = typeof initialValue === 'function'
          ? (initialValue as () => T)()
          : initialValue;
      }
      return [
        stateSlots[index] as T,
        (next) => {
          const previous = stateSlots[index] as T;
          stateSlots[index] = typeof next === 'function'
            ? (next as (current: T) => T)(previous)
            : next;
        },
      ];
    },
    useRef<T>(initialValue: T): { current: T } {
      const index = refIndex;
      refIndex += 1;
      if (index === 1) return { current: actorRefValue as T };
      return { current: initialValue };
    },
  };
});

const apiRuntime = vi.hoisted(() => ({
  client: {} as MockPosApiClient,
  createApiClient: vi.fn(),
}));

vi.mock('react', async (importOriginal) => {
  const actual = await importOriginal<typeof import('react')>();
  return {
    ...actual,
    default: actual,
    useCallback: (callback: unknown) => callback,
    useEffect: () => undefined,
    useMemo: (factory: () => unknown) => factory(),
    useRef: hookRuntime.useRef,
    useState: hookRuntime.useState,
  };
});

vi.mock('../shared/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../shared/api')>();
  return {
    ...actual,
    createApiClient: apiRuntime.createApiClient,
  };
});

type MockPosApiClient = Record<keyof PosApiClient, ReturnType<typeof vi.fn>>;
type POSWorkflowContext = React.ComponentProps<typeof POSProvider> & {
  payOrder: (method: 'cash' | 'card', inputAmount: number) => Promise<{ success: boolean; change: number; errorKey?: string }>;
  refundCheck: (checkId: string, reason: string, disposition: 'waste' | 'return') => Promise<void>;
  partialRefundCheck: (checkId: string, lineId: string, qtyToRefund: number, reason: string, disposition: 'waste' | 'return') => Promise<void>;
  cancelPrecheck: (managerPin: string, reason: string) => Promise<boolean>;
  reprintPrecheck: () => Promise<void>;
  reprintCheck: (checkId: string) => Promise<void>;
  openCashSession: (initialAmount: number) => Promise<void>;
  closeCashSession: () => Promise<void>;
  syncOutbox: () => Promise<void>;
  closedOrders: Array<{ id: string; lines: Array<{ id: string; quantity: number }> }>;
};

const safeAuth = {
  clientDeviceId: 'client-1',
  nodeDeviceId: 'node-1',
  restaurantId: 'restaurant-1',
  sessionId: 'session-1',
  actorEmployeeId: 'employee-1',
};

const workflowPermissions = [
  permissions.CASH_SESSION_VIEW,
  permissions.CASH_SESSION_OPEN,
  permissions.CASH_SESSION_CLOSE,
  permissions.PAYMENT_CASH,
  permissions.PAYMENT_CARD,
  permissions.PAYMENT_REFUND,
  permissions.PRECHECK_CANCEL,
  permissions.SYNC_VIEW,
  permissions.SYNC_RETRY,
];

describe('POSContext workflow orchestration', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    apiRuntime.client = createApiMock();
    apiRuntime.createApiClient.mockImplementation(() => apiRuntime.client);
  });

  it.each(['cash', 'card'] as const)('captures %s payment once and refreshes displayed state after success', async (method) => {
    const context = renderPOSContext();

    const result = await context.payOrder(method, 1500);

    expect(apiRuntime.client.capturePrecheckPayment).toHaveBeenCalledTimes(1);
    expect(apiRuntime.client.capturePrecheckPayment).toHaveBeenCalledWith('precheck-1', method, 1500, 'RUB');
    expect(result).toEqual({ success: true, change: 500 });
    expect(apiRuntime.client.listTables).toHaveBeenCalledWith('hall-1');
    expect(apiRuntime.client.listActiveOrdersByHall).toHaveBeenCalledWith('hall-1');
    expect(apiRuntime.client.listClosedOrders).toHaveBeenCalled();
    expect(apiRuntime.client.getSyncStatus).toHaveBeenCalledTimes(1);
  });

  it('returns a safe payment failure without auto-retrying the financial mutation', async () => {
    apiRuntime.client.capturePrecheckPayment.mockRejectedValueOnce(rawInternalError());
    const context = renderPOSContext();

    const result = await context.payOrder('cash', 1000);
    await flushMicrotasks();

    expect(result).toEqual({ success: false, change: 0, errorKey: 'errors.paymentFailed' });
    expect(apiRuntime.client.capturePrecheckPayment).toHaveBeenCalledTimes(1);
    expect(apiRuntime.client.listTables).not.toHaveBeenCalled();
    expectNoRawLeakage();
  });

  it('cancels the active precheck once and refreshes floor and prechecks', async () => {
    const context = renderPOSContext();
    const managerPin = '****';

    const result = await context.cancelPrecheck(managerPin, 'guest request');

    expect(result).toBe(true);
    expect(apiRuntime.client.cancelPrecheck).toHaveBeenCalledTimes(1);
    expect(apiRuntime.client.cancelPrecheck.mock.calls[0]).toEqual(['precheck-1', managerPin, 'guest request']);
    expect(apiRuntime.client.listTables).toHaveBeenCalledWith('hall-1');
    expect(apiRuntime.client.listActiveOrdersByHall).toHaveBeenCalledWith('hall-1');
    expect(apiRuntime.client.listPrechecksByOrder).toHaveBeenCalledWith('order-1');
  });

  it('handles denied precheck cancellation safely without auto-retry', async () => {
    apiRuntime.client.cancelPrecheck.mockRejectedValueOnce(rawInternalError());
    const context = renderPOSContext();

    const result = await context.cancelPrecheck('****', 'permission check');
    await flushMicrotasks();

    expect(result).toBe(false);
    expect(apiRuntime.client.cancelPrecheck).toHaveBeenCalledTimes(1);
    expect(apiRuntime.client.listPrechecksByOrder).not.toHaveBeenCalled();
    expectNoRawLeakage();
  });

  it('reprints active prechecks and closed checks through the backend document endpoints', async () => {
    const context = renderPOSContext();

    await context.reprintPrecheck();
    await context.reprintCheck('check-1');

    expect(apiRuntime.client.reprintPrecheck).toHaveBeenCalledTimes(1);
    expect(apiRuntime.client.reprintPrecheck).toHaveBeenCalledWith('precheck-1');
    expect(apiRuntime.client.reprintCheck).toHaveBeenCalledTimes(1);
    expect(apiRuntime.client.reprintCheck).toHaveBeenCalledWith('check-1');
  });

  it.each([
    ['waste', 'write_off_waste'],
    ['return', 'return_to_stock'],
  ] as const)('records full refund with %s inventory disposition and refreshes activity', async (uiDisposition, backendDisposition) => {
    const context = renderPOSContext();

    await context.refundCheck('check-1', 'guest refund', uiDisposition);

    expect(apiRuntime.client.recordCheckRefund).toHaveBeenCalledTimes(1);
    expect(apiRuntime.client.recordCheckRefund).toHaveBeenCalledWith('check-1', {
      reason: 'guest refund',
      operationKind: 'full',
      inventoryDisposition: backendDisposition,
    });
    expect(apiRuntime.client.listClosedOrders).toHaveBeenCalled();
  });

  it('handles full refund errors safely without auto-retry', async () => {
    apiRuntime.client.recordCheckRefund.mockRejectedValueOnce(rawInternalError());
    const context = renderPOSContext();

    await context.refundCheck('check-1', 'conflict', 'waste');
    await flushMicrotasks();

    expect(apiRuntime.client.recordCheckRefund).toHaveBeenCalledTimes(1);
    expect(apiRuntime.client.listClosedOrders).not.toHaveBeenCalled();
    expectNoRawLeakage();
  });

  it('records partial refund for one order line without creating a full-check payload', async () => {
    const context = renderPOSContext();

    await context.partialRefundCheck('check-1', 'line-1', 2, 'one item returned', 'return');

    expect(apiRuntime.client.recordCheckRefund).toHaveBeenCalledTimes(1);
    expect(apiRuntime.client.recordCheckRefund).toHaveBeenCalledWith('check-1', {
      reason: 'one item returned',
      operationKind: 'partial',
      inventoryDisposition: 'return_to_stock',
      items: [{
        scope: 'order_line',
        orderLineId: 'line-1',
        quantity: 2,
        amount: 1000,
        currency: 'RUB',
        taxAmount: 0,
      }],
    });
  });

  it('keeps partial refund state safe and does not auto-retry on API failure', async () => {
    apiRuntime.client.recordCheckRefund.mockRejectedValueOnce(rawInternalError());
    const context = renderPOSContext();
    const closedOrdersBefore = context.closedOrders;

    await context.partialRefundCheck('check-1', 'line-1', 1, 'conflict', 'waste');
    await flushMicrotasks();

    expect(apiRuntime.client.recordCheckRefund).toHaveBeenCalledTimes(1);
    expect(context.closedOrders).toEqual(closedOrdersBefore);
    expectNoRawLeakage();
  });

  it('opens cash session once and refreshes operational state', async () => {
    const context = renderPOSContext({ cashSession: null });

    await context.openCashSession(5000);

    expect(apiRuntime.client.openCashSession).toHaveBeenCalledTimes(1);
    expect(apiRuntime.client.openCashSession).toHaveBeenCalledWith(5000);
    expectRefreshAllAfterSuccessfulOperation();
  });

  it('handles cash session open errors safely without duplicate mutation calls', async () => {
    apiRuntime.client.openCashSession.mockRejectedValueOnce(rawInternalError());
    const context = renderPOSContext({ cashSession: null });

    await context.openCashSession(5000);
    await flushMicrotasks();

    expect(apiRuntime.client.openCashSession).toHaveBeenCalledTimes(1);
    expect(apiRuntime.client.getCurrentShift).not.toHaveBeenCalled();
    expectNoRawLeakage();
  });

  it('closes cash session with the current contract amount and refreshes operational state', async () => {
    const context = renderPOSContext();

    await context.closeCashSession();

    expect(apiRuntime.client.closeCashSession).toHaveBeenCalledTimes(1);
    expect(apiRuntime.client.closeCashSession).toHaveBeenCalledWith('cash-1', 5000);
    expectRefreshAllAfterSuccessfulOperation();
  });

  it('handles cash session close errors safely without auto-retry', async () => {
    apiRuntime.client.closeCashSession.mockRejectedValueOnce(rawInternalError());
    const context = renderPOSContext();

    await context.closeCashSession();
    await flushMicrotasks();

    expect(apiRuntime.client.closeCashSession).toHaveBeenCalledTimes(1);
    expect(apiRuntime.client.getCurrentShift).not.toHaveBeenCalled();
    expectNoRawLeakage();
  });

  it('retries sync outbox once and refreshes sync-visible activity state', async () => {
    const context = renderPOSContext();

    await context.syncOutbox();

    expect(apiRuntime.client.retryFailedOutbox).toHaveBeenCalledTimes(1);
    expect(apiRuntime.client.listClosedOrders).toHaveBeenCalled();
    expect(apiRuntime.client.getSyncStatus).toHaveBeenCalledTimes(1);
    expect(apiRuntime.client.getStorageStatus).toHaveBeenCalledTimes(1);
  });

  it('keeps sync errors safe and does not spin in retry loop', async () => {
    apiRuntime.client.retryFailedOutbox.mockRejectedValueOnce(rawInternalError());
    const context = renderPOSContext();

    await context.syncOutbox();
    await flushMicrotasks();

    expect(apiRuntime.client.retryFailedOutbox).toHaveBeenCalledTimes(1);
    expect(apiRuntime.client.listClosedOrders).not.toHaveBeenCalled();
    expectNoRawLeakage();
  });

  it('respects sync view permission when refreshing activity', async () => {
    const context = renderPOSContext({
      actor: actorFixture([permissions.PAYMENT_CASH]),
    });

    await context.syncOutbox();

    expect(apiRuntime.client.retryFailedOutbox).toHaveBeenCalledTimes(1);
    expect(apiRuntime.client.listClosedOrders).toHaveBeenCalled();
    expect(apiRuntime.client.getSyncStatus).not.toHaveBeenCalled();
    expect(apiRuntime.client.getStorageStatus).not.toHaveBeenCalled();
  });
});

function renderPOSContext(overrides: Partial<RenderState> = {}) {
  const state = createRenderState(overrides);
  hookRuntime.reset(state.slots, state.actor);
  const element = POSProvider({ children: null }) as React.ReactElement<{ value: POSWorkflowContext }>;
  return element.props.value;
}

type RenderState = {
  actor: BackendActorContext;
  shift: BackendShift;
  cashSession: BackendCashSession | null;
  activeHallId: string;
  selectedTableId: string;
  activeOrders: BackendOrder[];
  prechecks: BackendPrecheck[];
  closedOrders: BackendClosedOrder[];
  syncStatus: BackendSyncStatus;
};

function createRenderState(overrides: Partial<RenderState>) {
  const actor = overrides.actor ?? actorFixture(workflowPermissions);
  const activeHallId = overrides.activeHallId ?? 'hall-1';
  const selectedTableId = overrides.selectedTableId ?? 'table-1';
  const slots: unknown[] = Array.from({ length: 30 }, () => hookRuntime.missing);
  slots[0] = 'floor';
  slots[1] = activeHallId;
  slots[3] = false;
  slots[4] = selectedTableId;
  slots[5] = safeAuth;
  slots[6] = actor;
  slots[7] = overrides.shift ?? shiftFixture();
  slots[8] = [];
  slots[9] = overrides.cashSession === undefined ? cashSessionFixture() : overrides.cashSession;
  slots[10] = [];
  slots[11] = [hallFixture(activeHallId)];
  slots[12] = [tableFixture(selectedTableId, activeHallId)];
  slots[13] = overrides.activeOrders ?? [activeOrderFixture(selectedTableId)];
  slots[14] = [];
  slots[15] = [];
  slots[16] = overrides.prechecks ?? [precheckFixture()];
  slots[17] = null;
  slots[18] = overrides.closedOrders ?? [closedOrderFixture()];
  slots[19] = 1;
  slots[20] = false;
  slots[21] = false;
  slots[22] = '';
  slots[23] = '2026-06-19';
  slots[24] = overrides.syncStatus ?? syncStatusFixture();
  slots[25] = null;
  slots[26] = null;
  slots[27] = false;
  slots[28] = '';
  slots[29] = [];
  return { actor, slots };
}

function createApiMock(): MockPosApiClient {
  const api = new Proxy({}, {
    get(target, prop: string) {
      if (!(prop in target)) {
        (target as Record<string, ReturnType<typeof vi.fn>>)[prop] = vi.fn().mockResolvedValue(undefined);
      }
      return (target as Record<string, ReturnType<typeof vi.fn>>)[prop];
    },
  }) as MockPosApiClient;

  api.getCurrentShift.mockResolvedValue(shiftFixture());
  api.listRecentShifts.mockResolvedValue([]);
  api.getCurrentCashSession.mockResolvedValue(cashSessionFixture());
  api.listHalls.mockResolvedValue([hallFixture('hall-1')]);
  api.listMenuItems.mockResolvedValue([]);
  api.listActivePricingPolicies.mockResolvedValue([]);
  api.listTables.mockResolvedValue([tableFixture('table-1', 'hall-1')]);
  api.listActiveOrdersByHall.mockResolvedValue([activeOrderFixture('table-1')]);
  api.listPrechecksByOrder.mockResolvedValue([precheckFixture()]);
  api.getOrderPricing.mockResolvedValue(null);
  api.listClosedOrders.mockResolvedValue([closedOrderFixture()]);
  api.getSyncStatus.mockResolvedValue(syncStatusFixture());
  api.getStorageStatus.mockResolvedValue(storageStatusFixture());
  api.openCashSession.mockResolvedValue(cashSessionFixture());
  api.closeCashSession.mockResolvedValue({ ...cashSessionFixture(), status: 'closed', closed_at: '2026-06-19T12:00:00Z' });
  api.capturePrecheckPayment.mockResolvedValue(paymentFixture());
  api.cancelPrecheck.mockResolvedValue({ ...precheckFixture(), status: 'cancelled' });
  api.reprintPrecheck.mockResolvedValue({ document_type: 'precheck', updated_at: '2026-06-19T10:00:00Z' });
  api.reprintCheck.mockResolvedValue({ document_type: 'check', updated_at: '2026-06-19T10:00:00Z' });
  api.recordCheckRefund.mockResolvedValue(financialOperationFixture());
  api.retryFailedOutbox.mockResolvedValue({ retried: 1 });
  return api;
}

function expectRefreshAllAfterSuccessfulOperation() {
  expect(apiRuntime.client.getCurrentShift).toHaveBeenCalledTimes(1);
  expect(apiRuntime.client.listRecentShifts).toHaveBeenCalledTimes(1);
  expect(apiRuntime.client.getCurrentCashSession).toHaveBeenCalledTimes(1);
  expect(apiRuntime.client.listHalls).toHaveBeenCalledTimes(1);
  expect(apiRuntime.client.listMenuItems).toHaveBeenCalledTimes(1);
  expect(apiRuntime.client.listTables).toHaveBeenCalledWith('hall-1');
  expect(apiRuntime.client.listActiveOrdersByHall).toHaveBeenCalledWith('hall-1');
  expect(apiRuntime.client.listPrechecksByOrder).toHaveBeenCalledWith('order-1');
  expect(apiRuntime.client.getOrderPricing).toHaveBeenCalledWith('order-1');
  expect(apiRuntime.client.listClosedOrders).toHaveBeenCalled();
}

function expectNoRawLeakage() {
  const logs = hookRuntime.stateAt<Array<{ msg: string }>>(29);
  const renderedLogText = logs.map((event) => event.msg).join('\n');
  for (const marker of rawMarkers) {
    expect(renderedLogText.includes(marker)).toBe(false);
  }
  expect(logs.length).toBeGreaterThan(0);
}

const rawMarkers = [
  'raw sql marker',
  'panic stack marker',
  'manager-pin-secret-marker',
  'payment-sensitive-marker',
];

function rawInternalError() {
  return new Error(rawMarkers.join(' | '));
}

async function flushMicrotasks() {
  await Promise.resolve();
  await Promise.resolve();
}

function actorFixture(actorPermissions: string[]): BackendActorContext {
  return {
    employee_id: 'employee-1',
    restaurant_id: 'restaurant-1',
    role_id: 'role-1',
    name: 'Operator',
    permissions: actorPermissions,
  };
}

function shiftFixture(): BackendShift {
  return {
    id: 'shift-1',
    restaurant_id: 'restaurant-1',
    device_id: 'node-1',
    opened_by_employee_id: 'employee-1',
    closed_by_employee_id: null,
    status: 'open',
    opened_at: '2026-06-19T08:00:00Z',
    closed_at: null,
    opening_cash_amount: 0,
    closing_cash_amount: null,
    created_at: '2026-06-19T08:00:00Z',
    updated_at: '2026-06-19T08:00:00Z',
  };
}

function cashSessionFixture(): BackendCashSession {
  return {
    id: 'cash-1',
    edge_cash_session_id: 'edge-cash-1',
    restaurant_id: 'restaurant-1',
    device_id: 'node-1',
    shift_id: 'shift-1',
    opened_by_employee_id: 'employee-1',
    closed_by_employee_id: null,
    status: 'open',
    opening_cash_amount: 5000,
    closing_cash_amount: null,
    opened_at: '2026-06-19T08:10:00Z',
    closed_at: null,
    created_at: '2026-06-19T08:10:00Z',
    updated_at: '2026-06-19T08:10:00Z',
  };
}

function hallFixture(id: string): BackendHall {
  return {
    id,
    restaurant_id: 'restaurant-1',
    name: 'Main hall',
    active: true,
  };
}

function tableFixture(id: string, hallId: string): BackendTable {
  return {
    id,
    restaurant_id: 'restaurant-1',
    hall_id: hallId,
    name: 'Table 1',
    seats: 4,
    active: true,
  };
}

function activeOrderFixture(tableId: string): BackendOrder {
  return {
    id: 'order-1',
    edge_order_id: 'edge-order-1',
    restaurant_id: 'restaurant-1',
    device_id: 'node-1',
    shift_id: 'shift-1',
    status: 'locked',
    table_id: tableId,
    table_name: 'Table 1',
    guest_count: 2,
    subtotal: 1000,
    discount_total: 0,
    tax_total: 0,
    total: 1000,
    opened_at: '2026-06-19T09:00:00Z',
    closed_at: null,
    created_at: '2026-06-19T09:00:00Z',
    updated_at: '2026-06-19T09:00:00Z',
    lines: [],
  };
}

function precheckFixture(): BackendPrecheck {
  return {
    id: 'precheck-1',
    order_id: 'order-1',
    status: 'issued',
    version: 1,
    supersedes_precheck_id: null,
    currency_code: 'RUB',
    subtotal: 1000,
    discount_total: 0,
    surcharge_total: 0,
    tax_total: 0,
    total: 1000,
    paid_total: 0,
    remaining_total: 1000,
    created_at: '2026-06-19T09:30:00Z',
    issued_at: '2026-06-19T09:30:00Z',
    closed_at: null,
    cancelled_by_employee_id: null,
    cancellation_reason: null,
  };
}

function paymentFixture(): BackendPayment {
  return {
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
    business_date_local: '2026-06-19',
    provider_name: null,
    provider_transaction_id: null,
    provider_reference: null,
    fingerprint_hash: null,
    created_at: '2026-06-19T10:00:00Z',
    updated_at: '2026-06-19T10:00:00Z',
  };
}

function closedOrderFixture(): BackendClosedOrder {
  return {
    id: 'order-closed-1',
    table_name: 'Table 1',
    opened_at: '2026-06-19T09:00:00Z',
    closed_at: '2026-06-19T10:00:00Z',
    total: 1000,
    status: 'closed',
    check: {
      id: 'check-1',
      order_id: 'order-closed-1',
      status: 'paid',
      currency_code: 'RUB',
      subtotal: 1000,
      discount_total: 0,
      surcharge_total: 0,
      tax_total: 0,
      total: 1000,
      paid_total: 1000,
      remaining_total: 0,
      business_date_local: '2026-06-19',
      closed_at: '2026-06-19T10:00:00Z',
      snapshot: {
        precheck_snapshot: {
          lines: [{
            order_line_id: 'line-1',
            menu_item_id: 'menu-1',
            name: 'Dish',
            quantity: 2,
            unit_price_minor: 500,
            total_price_minor: 1000,
          }],
        },
      },
      payments: [paymentFixture()],
      created_at: '2026-06-19T10:00:00Z',
      updated_at: '2026-06-19T10:00:00Z',
    },
  };
}

function financialOperationFixture() {
  return {
    id: 'operation-1',
    edge_operation_id: 'edge-operation-1',
    restaurant_id: 'restaurant-1',
    device_id: 'node-1',
    shift_id: 'shift-1',
    original_shift_id: 'shift-1',
    check_id: 'check-1',
    precheck_id: 'precheck-1',
    operation_type: 'refund',
    operation_kind: 'full',
    status: 'recorded',
    amount: 1000,
    currency: 'RUB',
    business_date_local: '2026-06-19',
    inventory_disposition: 'write_off_waste',
    reason: 'guest refund',
    created_by_employee_id: 'employee-1',
    approved_by_employee_id: null,
    items: [],
    created_at: '2026-06-19T10:10:00Z',
  };
}

function syncStatusFixture(): BackendSyncStatus {
  return {
    total: 3,
    pending: 1,
    processing: 0,
    sent: 1,
    failed: 1,
    suspended: 0,
    oldest_pending_sequence_no: 1,
    last_cloud_version: 7,
  };
}

function storageStatusFixture() {
  return {
    generated_at: '2026-06-19T10:00:00Z',
    sqlite: {
      page_count: 1,
      page_size_bytes: 4096,
      freelist_count: 0,
      estimated_size_bytes: 4096,
      freelist_bytes: 0,
      journal_mode: 'wal',
    },
    tables: {},
    closed_order_business_date_range: {},
    closed_orders_by_business_date: [],
    outbox: [],
    blocking_outbox_messages: 0,
    retention: {
      mode: 'archive_apply',
      destructive_apply_supported: true,
      financial_ledger_protected: true,
      immutable_snapshots_protected: true,
      reason: 'ready',
    },
    runtime_versions: [],
    schema_migrations: [],
  };
}
