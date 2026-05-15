import { describe, expect, it, vi } from 'vitest';

vi.mock('quasar', () => ({
  useQuasar: () => ({ notify: vi.fn() }),
}));

import { invalidatePaymentConflictQueries, paymentConflictInvalidationQueryKeys } from './useCashierTerminal';

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
