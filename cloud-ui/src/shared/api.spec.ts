import { afterEach, describe, expect, it, vi } from 'vitest';

import { getStopListReadiness, listInventoryStockBalances, listInventoryStockLedger } from './api';

const stopListReadinessResponse = {
  restaurant_id: 'restaurant-1',
  default_conflict_policy: 'manager_review',
  projection_mode: 'cloud_authoritative',
  active_stop_list_entries: 2,
  total_stop_list_entries: 3,
  problem_events: {
    total: 0,
    by_error_code: [],
  },
  package_ack_status: 'pending',
  package_ack_status_reason_key: 'cloud.readiness.stopList.pending',
};

const inventoryStockBalanceResponse = [{
  restaurant_id: 'restaurant-1',
  catalog_item_id: 'catalog-item-1',
  quantity_on_hand: '12.500',
  unit_code: 'kg',
  costing_status: 'final',
  needs_recalculation: false,
  last_movement_at: '2026-05-30T10:00:00Z',
  business_date_to: '2026-05-30',
}];

const inventoryStockLedgerResponse = [{
  id: 'ledger-entry-1',
  restaurant_id: 'restaurant-1',
  warehouse_id: 'warehouse-1',
  stock_document_id: 'stock-document-1',
  source_event_id: 'source-event-1',
  source_event_type: 'OrderLineServed',
  catalog_item_id: 'catalog-item-1',
  order_line_id: 'order-line-1',
  movement_type: 'sale',
  quantity: '-2.000',
  unit_code: 'kg',
  unit_cost_minor: 125,
  total_cost_minor: -250,
  costing_status: 'estimated',
  occurred_at: '2026-05-30T10:00:00Z',
  business_date_local: '2026-05-30',
  created_at: '2026-05-30T10:01:00Z',
}];

function stubJsonFetch(payload: unknown) {
  const fetchMock = vi.fn(async () => new Response(JSON.stringify(payload), { status: 200 }));
  vi.stubGlobal('fetch', fetchMock);
  return fetchMock;
}

function requestedUrl(fetchMock: ReturnType<typeof vi.fn>) {
  const input = fetchMock.mock.calls[0]?.[0];
  expect(input).toBeDefined();
  return new URL(String(input));
}

afterEach(() => {
  vi.unstubAllGlobals();
});

describe('cloud api inventory readiness contract', () => {
  it('calls stop-list readiness endpoint with restaurant_id and node_device_id', async () => {
    const fetchMock = stubJsonFetch(stopListReadinessResponse);

    await getStopListReadiness('restaurant-1', 'node-1');

    const url = requestedUrl(fetchMock);
    expect(url.pathname).toBe('/api/v1/sync/readiness/stop-list');
    expect(url.searchParams.get('restaurant_id')).toBe('restaurant-1');
    expect(url.searchParams.get('node_device_id')).toBe('node-1');
  });

  it('omits optional node_device_id for stop-list readiness when it is empty', async () => {
    const fetchMock = stubJsonFetch(stopListReadinessResponse);

    await getStopListReadiness('restaurant-1');

    const url = requestedUrl(fetchMock);
    expect(url.searchParams.get('restaurant_id')).toBe('restaurant-1');
    expect(url.searchParams.has('node_device_id')).toBe(false);
  });

  it('maps inventory stock balance filters to snake_case query params', async () => {
    const fetchMock = stubJsonFetch(inventoryStockBalanceResponse);

    await listInventoryStockBalances('restaurant-1', {
      warehouseId: 'warehouse-1',
      catalogItemId: 'catalog-item-1',
      businessDateTo: '2026-05-30',
      costingStatus: 'final',
      limit: 25,
      offset: 10,
    });

    const url = requestedUrl(fetchMock);
    expect(url.pathname).toBe('/api/v1/inventory/stock-balances');
    expect(url.searchParams.get('restaurant_id')).toBe('restaurant-1');
    expect(url.searchParams.get('warehouse_id')).toBe('warehouse-1');
    expect(url.searchParams.get('catalog_item_id')).toBe('catalog-item-1');
    expect(url.searchParams.get('business_date_to')).toBe('2026-05-30');
    expect(url.searchParams.get('costing_status')).toBe('final');
    expect(url.searchParams.get('limit')).toBe('25');
    expect(url.searchParams.get('offset')).toBe('10');
  });

  it('uses default limit and offset for inventory stock balances', async () => {
    const fetchMock = stubJsonFetch(inventoryStockBalanceResponse);

    await listInventoryStockBalances('restaurant-1');

    const url = requestedUrl(fetchMock);
    expect(url.searchParams.get('limit')).toBe('50');
    expect(url.searchParams.get('offset')).toBe('0');
  });

  it('does not add empty optional inventory stock balance filters to query', async () => {
    const fetchMock = stubJsonFetch(inventoryStockBalanceResponse);

    await listInventoryStockBalances('restaurant-1', {
      warehouseId: '',
      catalogItemId: '',
      businessDateTo: '',
      costingStatus: '',
    });

    const url = requestedUrl(fetchMock);
    expect(url.searchParams.has('warehouse_id')).toBe(false);
    expect(url.searchParams.has('catalog_item_id')).toBe(false);
    expect(url.searchParams.has('business_date_to')).toBe(false);
    expect(url.searchParams.has('costing_status')).toBe(false);
    expect(url.searchParams.get('limit')).toBe('50');
    expect(url.searchParams.get('offset')).toBe('0');
  });

  it('maps inventory stock ledger filters to supported read-only query params', async () => {
    const fetchMock = stubJsonFetch(inventoryStockLedgerResponse);

    await listInventoryStockLedger('restaurant-1', {
      catalogItemId: 'catalog-item-1',
      sourceEventType: 'OrderLineServed',
      sourceEventId: 'source-event-1',
      orderLineId: 'order-line-1',
      limit: 25,
      offset: 10,
    });

    const url = requestedUrl(fetchMock);
    expect(url.pathname).toBe('/api/v1/inventory/stock-ledger');
    expect(url.searchParams.get('restaurant_id')).toBe('restaurant-1');
    expect(url.searchParams.get('catalog_item_id')).toBe('catalog-item-1');
    expect(url.searchParams.get('source_event_type')).toBe('OrderLineServed');
    expect(url.searchParams.get('source_event_id')).toBe('source-event-1');
    expect(url.searchParams.get('order_line_id')).toBe('order-line-1');
    expect(url.searchParams.get('limit')).toBe('25');
    expect(url.searchParams.get('offset')).toBe('10');
  });

  it('uses default limit and offset for inventory stock ledger', async () => {
    const fetchMock = stubJsonFetch(inventoryStockLedgerResponse);

    await listInventoryStockLedger('restaurant-1');

    const url = requestedUrl(fetchMock);
    expect(url.searchParams.get('limit')).toBe('50');
    expect(url.searchParams.get('offset')).toBe('0');
  });
});
