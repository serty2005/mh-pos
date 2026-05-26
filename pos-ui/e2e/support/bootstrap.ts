import fs from 'node:fs';

export const bootstrapEnvName = 'POS_E2E_BOOTSTRAP_JSON';

export function loadBootstrapJson() {
  const source = process.env[bootstrapEnvName]?.trim() ?? '';
  if (!source) return '';
  if (source.startsWith('{') || source.startsWith('[')) return normalizeBootstrapJson(source);
  if (fs.existsSync(source)) return normalizeBootstrapJson(fs.readFileSync(source, 'utf-8'));
  if (source.endsWith('.json') || source.startsWith('/') || source.startsWith('.')) return '';
  return source;
}

export function bootstrapRequiredMessage() {
  return 'Run stack bootstrap and set POS_E2E_BOOTSTRAP_JSON to JSON content or /workspace/myhoreca-pos/.e2e/bootstrap.json';
}

function normalizeBootstrapJson(raw: string) {
  try {
    const data = JSON.parse(raw);
    if (!data || typeof data !== 'object' || Array.isArray(data)) return raw;
    const normalized = data as Record<string, unknown>;
    const pins = recordValue(normalized.pins);
    const employees = recordValue(normalized.employee_ids);
    const roles = recordValue(normalized.role_ids);
    for (const key of ['cashier_pin', 'manager_pin', 'waiter_pin', 'senior_cashier_pin', 'kitchen_pin', 'support_pin']) {
      if (normalized[key] === undefined && pins[key] !== undefined) normalized[key] = pins[key];
    }
    for (const [role, id] of Object.entries(employees)) {
      const key = `${role}_employee_id`;
      if (normalized[key] === undefined) normalized[key] = id;
    }
    for (const [role, id] of Object.entries(roles)) {
      const key = `${role}_role_id`;
      if (normalized[key] === undefined) normalized[key] = id;
    }
    firstArrayValue(normalized, 'table_ids', 'table_id');
    firstArrayValue(normalized, 'hall_ids', 'hall_id');
    firstArrayValue(normalized, 'menu_item_ids', 'menu_item_id');
    firstArrayValue(normalized, 'catalog_item_ids', 'catalog_item_id');
    firstArrayValue(normalized, 'modifier_group_ids', 'modifier_group_id');
    firstArrayValue(normalized, 'modifier_option_ids', 'modifier_option_id');
    return JSON.stringify(normalized);
  } catch {
    return raw;
  }
}

function recordValue(value: unknown): Record<string, unknown> {
  if (!value || typeof value !== 'object' || Array.isArray(value)) return {};
  return value as Record<string, unknown>;
}

function firstArrayValue(data: Record<string, unknown>, arrayKey: string, scalarKey: string) {
  if (data[scalarKey] !== undefined) return;
  const value = data[arrayKey];
  if (Array.isArray(value) && value.length > 0) data[scalarKey] = value[0];
}
