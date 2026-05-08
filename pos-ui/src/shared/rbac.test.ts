import { describe, expect, it } from 'vitest';

import { hasAnyPermission, hasPermission, permissionCatalog } from './rbac';

describe('rbac helpers', () => {
  it('returns true when permission is granted', () => {
    expect(hasPermission([permissionCatalog.orderCreate], permissionCatalog.orderCreate)).toBe(true);
  });

  it('returns false when permission is missing', () => {
    expect(hasPermission([permissionCatalog.orderCreate], permissionCatalog.precheckIssue)).toBe(false);
  });

  it('maps employee shift and payment permissions to backend canonical ids', () => {
    expect(permissionCatalog.employeeShiftOpen).toBe('pos.employee_shift.open');
    expect(permissionCatalog.employeeShiftViewCurrent).toBe('pos.employee_shift.view_current');
    expect(permissionCatalog.paymentCash).toBe('pos.payment.cash');
    expect(permissionCatalog.paymentCardManual).toBe('pos.payment.card.manual');
  });

  it('checks alternatives for role visibility decisions', () => {
    expect(hasAnyPermission([permissionCatalog.syncView], [permissionCatalog.paymentCash, permissionCatalog.syncView])).toBe(true);
    expect(hasAnyPermission([permissionCatalog.syncView], [permissionCatalog.paymentCash, permissionCatalog.paymentCardManual])).toBe(false);
  });
});
