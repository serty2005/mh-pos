import { describe, expect, it } from 'vitest';
import { navigationForEntitlements } from './navigation';

describe('licensed navigation', () => {
  it('fails closed for licensed sections and shows enabled modules', () => {
    expect(navigationForEntitlements({}).map((item) => item.route.id)).not.toContain('floor');
    expect(navigationForEntitlements({}).map((item) => item.route.id)).not.toContain('inventory');

    const enabled = navigationForEntitlements({ 'table-mode': true, 'warehouse-mode': true }).map((item) => item.route.id);
    expect(enabled).toContain('floor');
    expect(enabled).toContain('inventory');
  });
});
