import { describe, expect, it } from 'vitest';

import { activeIssuedPrecheck, canUseAnyPermission, canUsePermission, paymentChange } from './posContextHelpers';
import type { BackendPrecheck } from '../shared/schemas';

describe('POS context helpers', () => {
  it('selects the latest issued precheck by version', () => {
    const prechecks: BackendPrecheck[] = [
      precheck('old', 'issued', 1),
      precheck('new', 'issued', 3),
      precheck('cancelled', 'cancelled', 4),
    ];

    expect(activeIssuedPrecheck(prechecks)?.id).toBe('new');
  });

  it('calculates payment change only for display after backend payment command', () => {
    expect(paymentChange(1000, 1500)).toBe(500);
    expect(paymentChange(1000, 900)).toBe(0);
  });

  it('checks permissions as UX gating only', () => {
    expect(canUsePermission(['pos.payment.cash'], 'pos.payment.cash')).toBe(true);
    expect(canUsePermission(['pos.order.view'], 'pos.payment.cash')).toBe(false);
    expect(canUseAnyPermission(['pos.payment.card.manual'], ['pos.payment.cash', 'pos.payment.card.manual'])).toBe(true);
    expect(canUseAnyPermission(['pos.order.view'], ['pos.payment.cash', 'pos.payment.card.manual'])).toBe(false);
  });
});

function precheck(id: string, status: BackendPrecheck['status'], version: number): BackendPrecheck {
  return {
    id,
    order_id: 'order-1',
    status,
    version,
    supersedes_precheck_id: null,
    currency_code: 'RUB',
    subtotal: 100,
    discount_total: 0,
    surcharge_total: 0,
    tax_total: 0,
    total: 100,
    paid_total: 0,
    remaining_total: 100,
    created_at: '2026-05-24T10:00:00Z',
    issued_at: '2026-05-24T10:00:00Z',
    closed_at: null,
    cancelled_by_employee_id: null,
    cancellation_reason: null,
  };
}
