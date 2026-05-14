import { describe, expect, it } from 'vitest';

import { checkSchema, orderSchema, precheckSchema } from './schemas';

const createdAt = '2026-05-09T10:00:00Z';

describe('runtime contract schemas', () => {
  it('parses backend precheck payload with current totals and currency fields', () => {
    const parsed = precheckSchema.parse({
      id: 'precheck-1',
      order_id: 'order-1',
      status: 'issued',
      version: 1,
      supersedes_precheck_id: null,
      currency_code: 'RUB',
      subtotal: 1000,
      discount_total: 100,
      surcharge_total: 50,
      tax_total: 150,
      total: 1100,
      paid_total: 200,
      remaining_total: 900,
      snapshot: {
        document_type: 'precheck',
        breakdown: {
          currency_code: 'RUB',
          surcharge_total_minor: 50,
          grand_total_minor: 1100,
        },
      },
      created_at: createdAt,
      issued_at: createdAt,
      closed_at: null,
      cancelled_by_employee_id: null,
      cancellation_reason: null,
    });

    expect(parsed.currency_code).toBe('RUB');
    expect(parsed.surcharge_total).toBe(50);
    expect(parsed.remaining_total).toBe(900);
  });

  it('parses backend check payload with current totals and nested payments', () => {
    const parsed = checkSchema.parse({
      id: 'check-1',
      order_id: 'order-1',
      status: 'paid',
      currency_code: 'RUB',
      subtotal: 1000,
      discount_total: 100,
      surcharge_total: 50,
      tax_total: 150,
      total: 1100,
      paid_total: 1100,
      remaining_total: 0,
      business_date_local: '2026-05-09',
      closed_at: createdAt,
      snapshot: { document_type: 'check' },
      payments: [{
        id: 'payment-1',
        edge_payment_id: 'edge-payment-1',
        restaurant_id: 'restaurant-1',
        device_id: 'device-1',
        shift_id: 'shift-1',
        precheck_id: 'precheck-1',
        method: 'cash',
        amount: 1100,
        currency: 'RUB',
        status: 'captured',
        business_date_local: '2026-05-09',
        provider_name: null,
        provider_transaction_id: null,
        provider_reference: null,
        fingerprint_hash: null,
        created_at: createdAt,
        updated_at: createdAt,
      }],
      created_at: createdAt,
      updated_at: createdAt,
    });

    expect(parsed.currency_code).toBe('RUB');
    expect(parsed.surcharge_total).toBe(50);
    expect(parsed.remaining_total).toBe(0);
    expect(parsed.payments?.[0]?.currency).toBe('RUB');
  });

  it('parses backend order payload with order line currency and tax profile fields', () => {
    const parsed = orderSchema.parse({
      id: 'order-1',
      edge_order_id: 'edge-order-1',
      restaurant_id: 'restaurant-1',
      device_id: 'device-1',
      shift_id: 'shift-1',
      status: 'open',
      table_id: 'table-1',
      table_name: 'A1',
      guest_count: 2,
      subtotal: 1000,
      discount_total: 0,
      tax_total: 0,
      total: 1000,
      opened_at: createdAt,
      closed_at: null,
      created_at: createdAt,
      updated_at: createdAt,
      lines: [{
        id: 'line-1',
        order_id: 'order-1',
        menu_item_id: 'menu-1',
        catalog_item_id: 'catalog-1',
        name: 'Tea',
        quantity: 1,
        unit_price: 1000,
        total_price: 1000,
        currency_code: 'RUB',
        tax_profile_id: null,
        status: 'active',
        created_at: createdAt,
        updated_at: createdAt,
      }],
    });

    expect(parsed.lines[0]?.currency_code).toBe('RUB');
    expect(parsed.lines[0]?.tax_profile_id).toBeNull();
  });
});
