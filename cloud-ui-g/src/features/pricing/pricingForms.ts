import type { PricingPolicy } from '../../shared/api/schemas';

export type LifecycleStatus = 'draft' | 'published' | 'archived';
export type PricingPolicyKind = 'discount' | 'surcharge';
export type PricingAmountKind = 'fixed' | 'percentage';

export type PricingPolicyFormValues = {
  name: string;
  kind: PricingPolicyKind;
  scope: string;
  amount_kind: PricingAmountKind;
  amount_minor: number;
  value_basis_points: number;
  application_index: number;
  manual: boolean;
  requires_permission: string;
  status: LifecycleStatus;
};

export type TaxPackageRow = Record<string, unknown>;

export type TaxPackageDraft = {
  node_device_id: string;
  restaurant_id: string;
  sync_mode: string;
  full_snapshot_reason: string;
  cloud_version: number;
  tax_profiles: TaxPackageRow[];
  tax_rules: TaxPackageRow[];
  service_charge_rules: TaxPackageRow[];
};

export const defaultPricingPolicyValues: PricingPolicyFormValues = {
  name: '',
  kind: 'discount',
  scope: 'order',
  amount_kind: 'percentage',
  amount_minor: 0,
  value_basis_points: 0,
  application_index: 10,
  manual: true,
  requires_permission: '',
  status: 'published',
};

export const defaultTaxPackageDraft: TaxPackageDraft = {
  node_device_id: '',
  restaurant_id: '',
  sync_mode: 'full_snapshot',
  full_snapshot_reason: 'terminal_restaurant_changed',
  cloud_version: 1,
  tax_profiles: [],
  tax_rules: [],
  service_charge_rules: [],
};

export function buildCreatePricingPolicyPayload(values: PricingPolicyFormValues) {
  const { status: _status, ...payload } = normalizePricingPolicyValues(values);
  return payload;
}

export function toPricingPolicyValues(policy: PricingPolicy): PricingPolicyFormValues {
  return {
    name: policy.name,
    kind: policy.kind,
    scope: policy.scope,
    amount_kind: policy.amount_kind,
    amount_minor: policy.amount_minor,
    value_basis_points: policy.value_basis_points,
    application_index: policy.application_index,
    manual: policy.manual,
    requires_permission: policy.requires_permission,
    status: policy.status,
  };
}

export function normalizePricingPolicyValues(values: PricingPolicyFormValues): PricingPolicyFormValues {
  return {
    ...values,
    name: values.name.trim(),
    scope: values.kind === 'surcharge' ? 'order' : values.scope.trim(),
    requires_permission: values.requires_permission.trim(),
    amount_minor: values.amount_kind === 'fixed' ? values.amount_minor : 0,
    value_basis_points: values.amount_kind === 'percentage' ? values.value_basis_points : 0,
  };
}

export function findDuplicateApplicationIndex(values: PricingPolicyFormValues[]): number | null {
  const seen = new Set<number>();
  for (const value of values) {
    if (value.status === 'archived') continue;
    if (seen.has(value.application_index)) return value.application_index;
    seen.add(value.application_index);
  }
  return null;
}

export function buildPricingPolicyPackagePayload(draft: TaxPackageDraft) {
  return {
    node_device_id: draft.node_device_id.trim(),
    restaurant_id: draft.restaurant_id.trim(),
    sync_mode: draft.sync_mode,
    full_snapshot_reason: draft.full_snapshot_reason,
    cloud_version: draft.cloud_version,
    payload_json: {
      tax_profiles: draft.tax_profiles,
      tax_rules: draft.tax_rules,
      service_charge_rules: draft.service_charge_rules,
      pricing_policies: [],
    },
  };
}

export function parseRows(value: string): TaxPackageRow[] {
  const parsed = JSON.parse(value) as unknown;
  if (!Array.isArray(parsed)) {
    throw new Error('Expected JSON array');
  }
  return parsed.map((row) => (row && typeof row === 'object' && !Array.isArray(row) ? row as TaxPackageRow : { value: row }));
}
