import { describe, expect, it } from 'vitest';

import { inventoryStockBalanceSchema, inventoryStockLedgerEntrySchema, stopListReadinessSchema } from './schemas';

describe('cloud schemas inventory readiness contract', () => {
  it('accepts stop-list readiness response and defaults optional fields', () => {
    const parsed = stopListReadinessSchema.parse({
      restaurant_id: 'restaurant-1',
      default_conflict_policy: 'manager_review',
      projection_mode: 'cloud_authoritative',
      active_stop_list_entries: 2,
      total_stop_list_entries: 3,
      latest_inventory_package: {
        stream_name: 'inventory',
        cloud_version: 7,
        updated_at: '2026-05-30T10:00:00Z',
      },
      problem_events: {
        total: 1,
        by_error_code: [{
          error_code: 'SYNC_FAILED',
          count: 1,
        }],
      },
      package_ack_status: 'pending',
      package_ack_status_reason_key: 'cloud.readiness.stopList.pending',
    });

    expect(parsed.node_device_id).toBe('');
    expect(parsed.latest_inventory_package?.checkpoint_token).toBe('');
    expect(parsed.latest_inventory_package?.cloud_updated_at).toBeUndefined();
    expect(parsed.latest_publication).toBeUndefined();
    expect(parsed.latest_stop_list_edge_ack).toBeUndefined();
  });

  it('accepts inventory stock balance without warehouse_id and defaults it', () => {
    const parsed = inventoryStockBalanceSchema.parse({
      restaurant_id: 'restaurant-1',
      catalog_item_id: 'catalog-item-1',
      quantity_on_hand: '12.500',
      unit_code: 'kg',
      costing_status: 'estimated',
      needs_recalculation: false,
      last_movement_at: '2026-05-30T10:00:00Z',
      business_date_to: '2026-05-30',
    });

    expect(parsed.warehouse_id).toBe('');
  });

  it('rejects unknown inventory costing status', () => {
    const parsed = inventoryStockBalanceSchema.safeParse({
      restaurant_id: 'restaurant-1',
      warehouse_id: '',
      catalog_item_id: 'catalog-item-1',
      quantity_on_hand: '12.500',
      unit_code: 'kg',
      costing_status: 'not_supported',
      needs_recalculation: false,
      last_movement_at: '2026-05-30T10:00:00Z',
      business_date_to: '2026-05-30',
    });

    expect(parsed.success).toBe(false);
  });

  it('accepts inventory stock ledger entry and defaults optional ids', () => {
    const parsed = inventoryStockLedgerEntrySchema.parse({
      id: 'ledger-entry-1',
      restaurant_id: 'restaurant-1',
      stock_document_id: 'stock-document-1',
      source_event_id: 'source-event-1',
      source_event_type: 'OrderLineServed',
      catalog_item_id: 'catalog-item-1',
      movement_type: 'sale',
      quantity: '-2.000',
      unit_code: 'kg',
      unit_cost_minor: 125,
      total_cost_minor: -250,
      costing_status: 'estimated',
      occurred_at: '2026-05-30T10:00:00Z',
      business_date_local: '2026-05-30',
      created_at: '2026-05-30T10:01:00Z',
    });

    expect(parsed.warehouse_id).toBe('');
    expect(parsed.order_line_id).toBe('');
  });
});
