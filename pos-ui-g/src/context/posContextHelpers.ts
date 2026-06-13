import type { BackendPrecheck } from '../shared/schemas';

export function activeIssuedPrecheck(prechecks: BackendPrecheck[]): BackendPrecheck | null {
  return prechecks
    .filter((precheck) => precheck.status === 'issued')
    .sort((a, b) => b.version - a.version)[0] ?? null;
}

export function paymentChange(total: number, paidAmount: number): number {
  return Math.max(0, paidAmount - total);
}

export function canUsePermission(permissions: string[], permission: string): boolean {
  return permissions.includes(permission);
}

export function canUseAnyPermission(permissions: string[], requiredPermissions: string[]): boolean {
  return requiredPermissions.some((permission) => canUsePermission(permissions, permission));
}
