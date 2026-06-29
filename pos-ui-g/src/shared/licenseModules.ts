export const productModules = [
  { id: 'cloud-subscription' },
  { id: 'table-mode' },
  { id: 'kitchen-space' },
  { id: 'warehouse-mode' },
  { id: 'waiter-space' },
  { id: 'telegram-worker' },
  { id: 'ticket-mode' },
] as const;

export type ProductModuleId = typeof productModules[number]['id'];

export function hasEntitlement(entitlements: Record<string, boolean>, moduleId: ProductModuleId) {
  return entitlements[moduleId] === true;
}
