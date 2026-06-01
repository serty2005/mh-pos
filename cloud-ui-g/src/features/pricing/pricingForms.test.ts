import { describe, expect, it } from 'vitest';
import {
  buildPricingPolicyPackagePayload,
  findDuplicateApplicationIndex,
  type PricingPolicyFormValues,
  type TaxPackageDraft,
} from './pricingForms';

const basePolicy: PricingPolicyFormValues = {
  name: 'Manager discount',
  kind: 'discount',
  scope: 'order',
  amount_kind: 'percentage',
  amount_minor: 0,
  value_basis_points: 500,
  application_index: 10,
  manual: true,
  requires_permission: 'pos.pricing.discount.apply',
  status: 'published',
};

describe('pricingForms', () => {
  it('detects duplicate application indexes among active pricing policies', () => {
    expect(findDuplicateApplicationIndex([
      { ...basePolicy, application_index: 10 },
      { ...basePolicy, name: 'Manager surcharge', kind: 'surcharge', application_index: 10 },
    ])).toBe(10);
  });

  it('ignores archived pricing policies when checking duplicate indexes', () => {
    expect(findDuplicateApplicationIndex([
      { ...basePolicy, application_index: 10 },
      { ...basePolicy, name: 'Old discount', application_index: 10, status: 'archived' },
    ])).toBeNull();
  });

  it('serializes tax and service-charge package rows into pricing_policy payload_json', () => {
    const draft: TaxPackageDraft = {
      node_device_id: 'node-1',
      restaurant_id: 'restaurant-1',
      sync_mode: 'full_snapshot',
      full_snapshot_reason: 'terminal_restaurant_changed',
      cloud_version: 2,
      tax_profiles: [{ id: 'tax-profile-1', name: 'VAT 20', rate_basis_points: 2000 }],
      tax_rules: [{ id: 'tax-rule-1', tax_profile_id: 'tax-profile-1', target_type: 'order' }],
      service_charge_rules: [{ id: 'service-1', name: 'Service 10', value_basis_points: 1000 }],
    };

    expect(buildPricingPolicyPackagePayload(draft)).toEqual({
      node_device_id: 'node-1',
      restaurant_id: 'restaurant-1',
      sync_mode: 'full_snapshot',
      full_snapshot_reason: 'terminal_restaurant_changed',
      cloud_version: 2,
      payload_json: {
        tax_profiles: draft.tax_profiles,
        tax_rules: draft.tax_rules,
        service_charge_rules: draft.service_charge_rules,
        pricing_policies: [],
      },
    });
  });
});
