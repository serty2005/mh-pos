import { describe, expect, it } from 'vitest';

import { hasPermission, permissionCatalog } from './rbac';

describe('rbac helpers', () => {
  it('returns true when permission is granted', () => {
    expect(hasPermission([permissionCatalog.orderCreate], permissionCatalog.orderCreate)).toBe(true);
  });

  it('returns false when permission is missing', () => {
    expect(hasPermission([permissionCatalog.orderCreate], permissionCatalog.precheckIssue)).toBe(false);
  });
});
