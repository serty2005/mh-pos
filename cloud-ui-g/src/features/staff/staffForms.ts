import type { Employee, Role } from '../../shared/api/schemas';

export type EmployeeStatus = 'active' | 'suspended' | 'archived';

export type RoleFormValues = {
  name: string;
  active: boolean;
  permission_ids: string[];
};

export type EmployeeCreateFormValues = {
  name: string;
  role_id: string;
  pin: string;
};

export type EmployeeUpdateFormValues = {
  name: string;
  role_id: string;
  status: EmployeeStatus;
};

export const defaultRoleValues: RoleFormValues = {
  name: '',
  active: true,
  permission_ids: [],
};

export const defaultEmployeeCreateValues: EmployeeCreateFormValues = {
  name: '',
  role_id: '',
  pin: '',
};

export const defaultEmployeeUpdateValues: EmployeeUpdateFormValues = {
  name: '',
  role_id: '',
  status: 'active',
};

// permissionsJsonFromIds кодирует deterministic object shape, который backend валидирует как permissions_json.
export function permissionsJsonFromIds(permissionIds: string[]): string {
  const ids = Array.from(new Set(permissionIds.map((id) => id.trim()).filter(Boolean))).sort();
  if (ids.length === 0) return '{}';
  return `{${ids.map((id) => `"${id}":true`).join(',')}}`;
}

// parsePermissionIds поддерживает текущий object shape и legacy shape с массивом permissions.
export function parsePermissionIds(body: string): string[] {
  try {
    const parsed = JSON.parse(body) as unknown;
    if (!parsed || typeof parsed !== 'object') return [];
    const record = parsed as Record<string, unknown>;
    const rawArray = Array.isArray(record.permissions) ? record.permissions : null;
    const ids = rawArray
      ? rawArray.filter((value): value is string => typeof value === 'string')
      : Object.entries(record).filter(([, value]) => value === true).map(([key]) => key);
    return Array.from(new Set(ids.map((id) => id.trim()).filter(Boolean))).sort();
  } catch {
    return [];
  }
}

// roleToFormValues переводит backend role snapshot в значения формы матрицы прав.
export function roleToFormValues(role: Role): RoleFormValues {
  return {
    name: role.name,
    active: role.active,
    permission_ids: parsePermissionIds(role.permissions_json),
  };
}

// employeeToUpdateValues готовит редактирование карточки без PIN material.
export function employeeToUpdateValues(employee: Employee): EmployeeUpdateFormValues {
  return {
    name: employee.name,
    role_id: employee.role_id,
    status: employee.status,
  };
}

// buildCreateRolePayload исключает restaurant_id: его добавляет StaffPage из выбранного scope.
export function buildCreateRolePayload(values: RoleFormValues) {
  return {
    name: values.name.trim(),
    permissions_json: permissionsJsonFromIds(values.permission_ids),
  };
}

// buildUpdateRolePayload передает active для archive/reactivate сценариев роли.
export function buildUpdateRolePayload(values: RoleFormValues) {
  return {
    name: values.name.trim(),
    active: values.active,
    permissions_json: permissionsJsonFromIds(values.permission_ids),
  };
}

// buildCreateEmployeePayload отправляет plaintext PIN только в create command.
export function buildCreateEmployeePayload(values: EmployeeCreateFormValues) {
  return {
    name: values.name.trim(),
    role_id: values.role_id,
    pin: values.pin.trim(),
  };
}

// buildUpdateEmployeePayload намеренно не принимает PIN, чтобы не слить credential в обычный PATCH.
export function buildUpdateEmployeePayload(values: EmployeeUpdateFormValues) {
  return {
    name: values.name.trim(),
    role_id: values.role_id,
    status: values.status,
  };
}
