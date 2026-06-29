export const productModules = [
  {
    id: 'cloud-subscription',
    labelKey: 'licenses.modules.cloudSubscription',
  },
  {
    id: 'table-mode',
    labelKey: 'licenses.modules.tableMode',
  },
  {
    id: 'kitchen-space',
    labelKey: 'licenses.modules.kitchenSpace',
  },
  {
    id: 'warehouse-mode',
    labelKey: 'licenses.modules.warehouseMode',
  },
  {
    id: 'waiter-space',
    labelKey: 'licenses.modules.waiterSpace',
  },
  {
    id: 'telegram-worker',
    labelKey: 'licenses.modules.telegramWorker',
  },
  {
    id: 'ticket-mode',
    labelKey: 'licenses.modules.ticketMode',
  },
] as const;

export type ProductModuleId = typeof productModules[number]['id'];

export const productModuleIds = productModules.map((module) => module.id);

export function hasEntitlement(entitlements: Record<string, boolean>, moduleId: ProductModuleId) {
  return entitlements[moduleId] === true;
}
