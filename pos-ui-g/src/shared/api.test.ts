import { afterEach, describe, expect, it, vi } from 'vitest';

import { ApiError, createApiClient, type AuthSnapshot } from './api';

const auth: AuthSnapshot = {
  clientDeviceId: 'client-1',
  nodeDeviceId: 'node-1',
  restaurantId: 'restaurant-1',
  sessionId: 'session-1',
  actorEmployeeId: 'employee-1',
};

describe('POS API client', () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('sends POS identity headers with requests', async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response('[]', { status: 200 }));
    vi.stubGlobal('fetch', fetchMock);

    const api = createApiClient(() => auth, 'http://pos.local/api/v1');
    await api.listMenuItems();

    const [, init] = fetchMock.mock.calls[0];
    const headers = init.headers as Headers;
    expect(fetchMock.mock.calls[0][0]).toBe('http://pos.local/api/v1/menu/items');
    expect(headers.get('X-Client-Device-ID')).toBe('client-1');
    expect(headers.get('X-Node-Device-ID')).toBe('node-1');
    expect(headers.get('X-Session-ID')).toBe('session-1');
    expect(headers.get('X-Actor-Employee-ID')).toBe('employee-1');
  });

  it('normalizes backend safe error response', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(JSON.stringify({
      error: {
        code: 'PERMISSION_DENIED',
        message_key: 'errors.permission',
        correlation_id: 'req-1',
      },
    }), { status: 403 })));

    const api = createApiClient(() => auth, 'http://pos.local/api/v1');

    await expect(api.listMenuItems()).rejects.toMatchObject({
      code: 'PERMISSION_DENIED',
      messageKey: 'errors.permission',
      category: 'permission',
      correlationId: 'req-1',
    });
  });

  it('uses backend pricing policy endpoints without manual amount payloads', async () => {
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(new Response(JSON.stringify([{
        id: 'policy-1',
        restaurant_id: 'restaurant-1',
        kind: 'discount',
        name: 'Скидка сотрудника',
        scope: 'order',
        amount_kind: 'percentage',
        amount_minor: 0,
        value_basis_points: 1000,
        application_index: 10,
        requires_permission: '',
        manual: false,
        active: true,
        created_at: '2026-05-24T10:00:00Z',
        updated_at: '2026-05-24T10:00:00Z',
      }]), { status: 200 }))
      .mockResolvedValueOnce(new Response(JSON.stringify({
        id: 'discount-1',
        order_id: 'order-1',
        order_line_id: null,
        pricing_policy_id: 'policy-1',
        scope: 'order',
        application_index: 10,
        amount_kind: 'percentage',
        amount_minor: 0,
        value_basis_points: 1000,
        reason: 'manager',
        created_at: '2026-05-24T10:01:00Z',
      }), { status: 201 }))
      .mockResolvedValueOnce(new Response(JSON.stringify({
        id: 'surcharge-1',
        order_id: 'order-1',
        pricing_policy_id: 'policy-2',
        kind: 'manual',
        application_index: 20,
        amount_kind: 'fixed',
        amount_minor: 150,
        value_basis_points: 0,
        reason: '',
        created_at: '2026-05-24T10:02:00Z',
      }), { status: 201 }));
    vi.stubGlobal('fetch', fetchMock);

    const api = createApiClient(() => auth, 'http://pos.local/api/v1');
    await expect(api.listActivePricingPolicies()).resolves.toHaveLength(1);
    await api.applyDiscountPolicy('order-1', 'policy-1', '', 'manager');
    await api.applySurchargePolicy('order-1', 'policy-2');

    expect(fetchMock.mock.calls[0][0]).toBe('http://pos.local/api/v1/pricing/policies');
    expect(JSON.parse(String((fetchMock.mock.calls[1][1] as RequestInit).body))).toEqual({
      pricing_policy_id: 'policy-1',
      order_line_id: '',
      reason: 'manager',
    });
    expect(JSON.parse(String((fetchMock.mock.calls[2][1] as RequestInit).body))).toEqual({
      pricing_policy_id: 'policy-2',
      reason: '',
    });
  });

  it('updates order-line modifiers through backend endpoint', async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify({
      id: 'line-1',
      order_id: 'order-1',
      menu_item_id: 'menu-1',
      catalog_item_id: 'catalog-1',
      name: 'Стейк',
      quantity: 1,
      unit_price: 1000,
      total_price: 1250,
      currency_code: 'RUB',
      tax_profile_id: null,
      course: null,
      comment: null,
      modifiers: [],
      status: 'active',
      created_at: '2026-05-24T10:00:00Z',
      updated_at: '2026-05-24T10:00:00Z',
    }), { status: 200 }));
    vi.stubGlobal('fetch', fetchMock);

    const api = createApiClient(() => auth, 'http://pos.local/api/v1');
    await api.updateOrderLineModifiers('order-1', 'line-1', [{ modifier_group_id: 'group-1', modifier_option_id: 'option-1', quantity: 2 }]);

    expect(fetchMock.mock.calls[0][0]).toBe('http://pos.local/api/v1/orders/order-1/lines/line-1/modifiers');
    expect((fetchMock.mock.calls[0][1] as RequestInit).method).toBe('PATCH');
    expect(JSON.parse(String((fetchMock.mock.calls[0][1] as RequestInit).body))).toEqual({
      selected_modifiers: [{ modifier_group_id: 'group-1', modifier_option_id: 'option-1', quantity: 2 }],
    });
  });

  it('sends selected modifiers when adding a new order line', async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify({
      id: 'line-1',
      order_id: 'order-1',
      menu_item_id: 'menu-1',
      catalog_item_id: 'catalog-1',
      name: 'Espresso',
      quantity: 1,
      unit_price: 12900,
      total_price: 14900,
      currency_code: 'RUB',
      tax_profile_id: null,
      course: null,
      comment: null,
      modifiers: [{
        id: 'line-mod-1',
        order_line_id: 'line-1',
        modifier_group_id: 'milk',
        modifier_option_id: 'lactose-free',
        name: 'Lactose Free',
        quantity: 1,
        unit_price: 2000,
        total_price: 2000,
      }],
      status: 'active',
      created_at: '2026-06-01T10:00:00Z',
      updated_at: '2026-06-01T10:00:00Z',
    }), { status: 201 }));
    vi.stubGlobal('fetch', fetchMock);

    const api = createApiClient(() => auth, 'http://pos.local/api/v1');
    await api.addOrderLine('order-1', 'menu-1', 1, [{ modifier_group_id: 'milk', modifier_option_id: 'lactose-free', quantity: 1 }]);

    expect(fetchMock.mock.calls[0][0]).toBe('http://pos.local/api/v1/orders/order-1/lines');
    expect(JSON.parse(String((fetchMock.mock.calls[0][1] as RequestInit).body))).toEqual({
      menu_item_id: 'menu-1',
      quantity: 1,
      selected_modifiers: [{ modifier_group_id: 'milk', modifier_option_id: 'lactose-free', quantity: 1 }],
    });
  });

  it('returns null for optional current cash session 404 without logging an error', async () => {
    const consoleError = vi.spyOn(console, 'error').mockImplementation(() => undefined);
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(JSON.stringify({
      error: {
        code: 'NOT_FOUND',
        message_key: 'errors.not_found',
        correlation_id: 'req-cash-missing',
      },
    }), { status: 404 })));

    const api = createApiClient(() => auth, 'http://pos.local/api/v1');

    await expect(api.getCurrentCashSession()).resolves.toBeNull();
    expect(consoleError).not.toHaveBeenCalled();
  });

  it('accepts current cash session 200 response from backend contract', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(JSON.stringify({
      id: 'cash-1',
      edge_cash_session_id: 'edge-cash-1',
      restaurant_id: 'restaurant-1',
      device_id: 'node-1',
      shift_id: 'shift-1',
      opened_by_employee_id: 'employee-1',
      closed_by_employee_id: null,
      status: 'open',
      business_date_local: '2026-06-01',
      opening_cash_amount: 5000,
      closing_cash_amount: null,
      opened_at: '2026-06-01T10:00:00Z',
      closed_at: null,
      created_at: '2026-06-01T10:00:00Z',
      updated_at: '2026-06-01T10:00:00Z',
    }), { status: 200 })));

    const api = createApiClient(() => auth, 'http://pos.local/api/v1');

    await expect(api.getCurrentCashSession()).resolves.toMatchObject({
      id: 'cash-1',
      edge_cash_session_id: 'edge-cash-1',
      opening_cash_amount: 5000,
    });
  });

  it('reads storage runtime metadata for the displayed schema version', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(JSON.stringify({
      generated_at: '2026-05-24T10:00:00Z',
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
      runtime_versions: [{
        module_name: 'pos-backend',
        module_version: '0.1.4',
        schema_version: '001_init.sql',
        status: 'applied',
      }],
      schema_migrations: [{ version: '001_init.sql', status: 'applied' }],
    }), { status: 200 })));

    const api = createApiClient(() => auth, 'http://pos.local/api/v1');
    await expect(api.getStorageStatus()).resolves.toMatchObject({
      runtime_versions: [{ module_name: 'pos-backend', module_version: '0.1.4' }],
    });
  });

  it('builds bounded closed order query with business date and offset', async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify([{
      id: 'order-closed-1',
      table_name: 'T1',
      opened_at: '2026-06-01T10:00:00Z',
      closed_at: '2026-06-01T11:00:00Z',
      total: 1500,
      status: 'closed',
    }]), { status: 200 }));
    vi.stubGlobal('fetch', fetchMock);

    const api = createApiClient(() => auth, 'http://pos.local/api/v1');
    await expect(api.listClosedOrders({
      businessDateLocal: '2026-06-01',
      limit: 26,
      offset: 25,
    })).resolves.toHaveLength(1);

    expect(fetchMock.mock.calls[0][0]).toBe('http://pos.local/api/v1/orders/closed?business_date_local=2026-06-01&limit=26&offset=25');
  });

  it('uses backend-backed kitchen stop-list read and update endpoints without raw payload state', async () => {
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(new Response(JSON.stringify([{
        id: 'stop-1',
        catalog_item_id: 'dish-1',
        available_quantity: 0,
        source: 'edge_overlay_requires_manager_review',
        reason: 'sold out',
        active: true,
        updated_at: '2026-05-30T10:00:00Z',
        sync_state: 'pending',
        outbox_command_id: 'cmd-stop-1',
        outbox_status: 'pending',
        outbox_sequence_no: 7,
        outbox_attempts: 0,
      }]), { status: 200 }))
      .mockResolvedValueOnce(new Response(JSON.stringify({
        id: 'stop-1',
        warehouse_id: 'warehouse-main',
        event_type: 'StopListUpdated',
        replayed: false,
      }), { status: 201 }));
    vi.stubGlobal('fetch', fetchMock);

    const api = createApiClient(() => auth, 'http://pos.local/api/v1');
    await expect(api.listKitchenStopList()).resolves.toMatchObject([
      { id: 'stop-1', sync_state: 'pending', outbox_status: 'pending' },
    ]);
    await api.submitKitchenStopListUpdate({
      command_id: 'cmd-stop-ui',
      stop_list_id: 'stop-1',
      catalog_item_id: 'dish-1',
      available_quantity: 0,
      active: true,
      reason: 'sold out',
    });

    expect(fetchMock.mock.calls[0][0]).toBe('http://pos.local/api/v1/kitchen/stop-list');
    expect(fetchMock.mock.calls[1][0]).toBe('http://pos.local/api/v1/kitchen/stop-list-updates');
    expect(JSON.parse(String((fetchMock.mock.calls[1][1] as RequestInit).body))).toEqual({
      command_id: 'cmd-stop-ui',
      stop_list_id: 'stop-1',
      catalog_item_id: 'dish-1',
      available_quantity: 0,
      active: true,
      reason: 'sold out',
    });
  });

  it('uses provisioning endpoints for cloud registration and license pairing', async () => {
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(new Response(JSON.stringify({
        node_device_id: 'node-1',
        cloud_url: 'http://cloud.local',
        restaurant_id: 'restaurant-1',
        status: 'pending_admin_approval',
        paired: false,
      }), { status: 202 }))
      .mockResolvedValueOnce(new Response(JSON.stringify({
        node_device_id: 'node-1',
        cloud_url: 'http://cloud.local',
        restaurant_id: 'restaurant-1',
        status: 'paired',
        paired: true,
      }), { status: 200 }));
    vi.stubGlobal('fetch', fetchMock);

    const api = createApiClient(() => auth, 'http://pos.local/api/v1');
    await api.registerCloudProvisioning('http://cloud.local');
    await api.pairViaLicense('ABC-123');

    expect(fetchMock.mock.calls[0][0]).toBe('http://pos.local/api/v1/system/provisioning/register-cloud');
    expect(JSON.parse(String((fetchMock.mock.calls[0][1] as RequestInit).body))).toMatchObject({
      cloud_url: 'http://cloud.local',
      display_name: 'POS Terminal',
    });
    expect(fetchMock.mock.calls[1][0]).toBe('http://pos.local/api/v1/system/provisioning/pair-via-license');
    expect(JSON.parse(String((fetchMock.mock.calls[1][1] as RequestInit).body))).toEqual({
      pairing_code: 'ABC-123',
    });
  });
});
