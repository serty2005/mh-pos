import { describe, expect, it, vi } from 'vitest';

vi.mock('quasar', () => ({
  useQuasar: () => ({ notify: vi.fn() }),
}));

import type { ClosedOrder } from '../../shared/schemas';
import {
  closedOrderCompensationUnavailableKey,
  invalidatePaymentConflictQueries,
  paymentConflictInvalidationQueryKeys,
} from './useCashierTerminal';

describe('useCashierTerminal payment conflict handling', () => {
  it('invalidates operational state after payment 409 without retrying the payment command', () => {
    const queryClient = {
      invalidateQueries: vi.fn(() => Promise.resolve()),
    };

    invalidatePaymentConflictQueries(queryClient);

    expect(queryClient.invalidateQueries).toHaveBeenCalledTimes(paymentConflictInvalidationQueryKeys.length);
    for (const queryKey of paymentConflictInvalidationQueryKeys) {
      expect(queryClient.invalidateQueries).toHaveBeenCalledWith({ queryKey });
    }
  });
});

describe('closed order compensation availability', () => {
  it('separates same-shift cancellation from after-shift refund paths', () => {
    const order = sampleClosedOrder('shift-original');
    const context = {
      canRefundPayment: true,
      canRecordCheckCancellation: true,
      currentCashSessionShiftId: 'shift-original',
    };

    expect(closedOrderCompensationUnavailableKey(order, 'check_cancellation', context)).toBe('');
    expect(closedOrderCompensationUnavailableKey(order, 'check_refund', context)).toBe('pos.compensationUnavailable.refundBoundary');
    expect(closedOrderCompensationUnavailableKey(order, 'payment_refund', context)).toBe('pos.compensationUnavailable.refundBoundary');

    const nextShiftContext = { ...context, currentCashSessionShiftId: 'shift-next' };
    expect(closedOrderCompensationUnavailableKey(order, 'check_cancellation', nextShiftContext)).toBe('pos.compensationUnavailable.cancellationBoundary');
    expect(closedOrderCompensationUnavailableKey(order, 'check_refund', nextShiftContext)).toBe('');
    expect(closedOrderCompensationUnavailableKey(order, 'payment_refund', nextShiftContext)).toBe('');
  });

  it('keeps compatibility payment refund unavailable without captured payment', () => {
    const order = sampleClosedOrder('shift-original', 'failed');

    expect(closedOrderCompensationUnavailableKey(order, 'payment_refund', {
      canRefundPayment: true,
      canRecordCheckCancellation: true,
      currentCashSessionShiftId: 'shift-next',
    })).toBe('pos.compensationUnavailable.noCapturedPayment');
  });
});

function sampleClosedOrder(paymentShiftId: string, paymentStatus: 'captured' | 'failed' = 'captured'): ClosedOrder {
  return {
    id: 'order-1',
    table_name: 'A1',
    opened_at: '2026-05-16T10:00:00Z',
    closed_at: '2026-05-16T10:30:00Z',
    total: 1000,
    status: 'closed',
    check: {
      id: 'check-1',
      order_id: 'order-1',
      status: 'paid',
      currency_code: 'RUB',
      subtotal: 900,
      discount_total: 0,
      surcharge_total: 0,
      tax_total: 100,
      total: 1000,
      paid_total: 1000,
      remaining_total: 0,
      business_date_local: '2026-05-16',
      closed_at: '2026-05-16T10:30:00Z',
      payments: [{
        id: 'payment-1',
        edge_payment_id: 'edge-payment-1',
        restaurant_id: 'restaurant-1',
        device_id: 'node-1',
        shift_id: paymentShiftId,
        precheck_id: 'precheck-1',
        method: 'cash',
        amount: 1000,
        currency: 'RUB',
        status: paymentStatus,
        business_date_local: '2026-05-16',
        provider_name: null,
        provider_transaction_id: null,
        provider_reference: null,
        fingerprint_hash: null,
        created_at: '2026-05-16T10:20:00Z',
        updated_at: '2026-05-16T10:20:00Z',
      }],
      created_at: '2026-05-16T10:30:00Z',
      updated_at: '2026-05-16T10:30:00Z',
    },
  };
}
