import { describe, expect, it } from 'vitest';

import {
  catalogSuggestionSchema,
  inventoryStockBalanceSchema,
  inventoryStockLedgerEntrySchema,
  reviewAssignmentResponseSchema,
  salesKitchenSummaryGroupBySchema,
  salesKitchenSummaryItemSchema,
  stopListReadinessSchema,
} from './schemas';

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

  it('accepts sales/kitchen summary row and strips raw payload fields', () => {
    const parsed = salesKitchenSummaryItemSchema.parse({
      group_by: 'business_date',
      group_key: '2026-05-30',
      event_count: 3,
      stock_move_count: 2,
      sale_event_count: 1,
      kitchen_event_count: 2,
      out_quantity: '2.000',
      in_quantity: '0.000',
      net_quantity: '-2.000',
      total_cost_minor: -500,
      payload: '{"unsafe":true}',
      raw_payload_sha256_hex: 'hash',
      snapshot_json: '{}',
    });

    expect(parsed.business_date_local).toBe('');
    expect(parsed.first_occurred_at).toBe('');
    expect(parsed.last_occurred_at).toBe('');
    expect('payload' in parsed).toBe(false);
    expect('raw_payload_sha256_hex' in parsed).toBe(false);
    expect('snapshot_json' in parsed).toBe(false);
  });

  it('rejects unsupported sales/kitchen group_by values', () => {
    expect(salesKitchenSummaryGroupBySchema.safeParse('business_date').success).toBe(true);
    expect(salesKitchenSummaryGroupBySchema.safeParse('event_type').success).toBe(true);
    expect(salesKitchenSummaryGroupBySchema.safeParse('source_event_type').success).toBe(true);
    expect(salesKitchenSummaryGroupBySchema.safeParse('catalog_item').success).toBe(true);
    expect(salesKitchenSummaryGroupBySchema.safeParse('warehouse').success).toBe(false);
  });

  it('accepts safe review assignment response without raw payload fields', () => {
    const parsed = reviewAssignmentResponseSchema.parse({
      review_type: 'catalog_suggestion',
      id: 'review-1',
      status: 'pending',
      assigned_to_employee_id: 'manager-2',
      assigned_by_employee_id: 'manager-1',
      assigned_at: '2026-06-01T10:00:00Z',
      assignment_note: 'take ownership',
    });

    expect(parsed.assigned_to_employee_id).toBe('manager-2');
    expect('payload_json' in parsed).toBe(false);
  });

  it('defaults safe assignment metadata on catalog suggestion rows', () => {
    const parsed = catalogSuggestionSchema.parse({
      id: 'catalog-review-1',
      suggestion_id: 'suggestion-1',
      restaurant_id: 'restaurant-1',
      action: 'create_item',
      status: 'pending',
      suggested_at: '2026-06-01T10:00:00Z',
      cloud_received_at: '2026-06-01T10:00:01Z',
      created_at: '2026-06-01T10:00:01Z',
      updated_at: '2026-06-01T10:00:01Z',
    });

    expect(parsed.assigned_to_employee_id).toBe('');
    expect(parsed.assignment_note).toBe('');
  });
});
