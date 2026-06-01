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
    expect(t.common.clear).toBeTruthy();
    expect(t.auth.pinSessionCopy).toBeTruthy();
    expect(t.auth.login).toBeTruthy();
    expect(t.auth.dbVersion).toBeTruthy();
    expect(t.ops.closedOrdersLoadFailed).toBeTruthy();
    expect(t.ops.employeeAuthorized('Оператор')).toContain('Оператор');
  });
});
