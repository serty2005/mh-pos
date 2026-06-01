import { describe, expect, it } from 'vitest';

import { t } from './index';

describe('pos-ui-g i18n labels', () => {
  it('contains labels used by bounded activity history and financial dialogs', () => {
    expect(t.activity.historyLimitHint).toContain('backend');
    expect(t.activity.businessDateFilter).toBeTruthy();
    expect(t.activity.allBusinessDates).toBeTruthy();
    expect(t.modals.refundReasonRequired).toBeTruthy();
    expect(t.modals.cashEventAmountRequired).toBeTruthy();
    expect(t.modals.paymentSubmit).toBeTruthy();
    expect(t.common.next).toBeTruthy();
  });
});
