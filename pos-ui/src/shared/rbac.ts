/** Канонические backend permission ids для POS UI visibility guards. */
export const permissionCatalog = {
  employeeShiftOpen: 'pos.employee_shift.open',
  employeeShiftClose: 'pos.employee_shift.close',
  employeeShiftViewCurrent: 'pos.employee_shift.view_current',
  employeeShiftRecent: 'pos.employee_shift.recent',
  catalogView: 'pos.catalog.view',
  floorView: 'pos.floor.view',
  menuView: 'pos.menu.view',
  cashSessionOpen: 'pos.cash_session.open',
  cashSessionClose: 'pos.cash_session.close',
  cashSessionViewCurrent: 'pos.cash_session.view_current',
  cashDrawerRecordEvent: 'pos.cash_drawer.record_event',
  orderCreate: 'pos.order.create',
  orderView: 'pos.order.view',
  orderAddLine: 'pos.order.add_line',
  orderChangeQuantity: 'pos.order.change_quantity',
  orderVoidLine: 'pos.order.void_line',
  orderClose: 'pos.order.close',
  precheckIssue: 'pos.precheck.issue',
  precheckView: 'pos.precheck.view',
  precheckCancelRequest: 'pos.precheck.cancel.request',
  precheckCancel: 'pos.precheck.cancel',
  paymentCash: 'pos.payment.cash',
  paymentCardManual: 'pos.payment.card.manual',
  paymentOther: 'pos.payment.other',
  checkView: 'pos.check.view',
  syncView: 'pos.sync.view',
  syncRetryFailed: 'pos.sync.retry_failed',
} as const;

/** PermissionID является frontend-зеркалом canonical backend permission id. */
export type PermissionID = typeof permissionCatalog[keyof typeof permissionCatalog];

/** Проверяет, есть ли у текущего actor точное право для guarded action. */
export function hasPermission(granted: string[] | undefined, required: PermissionID): boolean {
  return Boolean(granted?.includes(required));
}

/** Проверяет, есть ли у текущего actor хотя бы одно право из набора альтернатив. */
export function hasAnyPermission(granted: string[] | undefined, required: PermissionID[]): boolean {
  return required.some((permission) => hasPermission(granted, permission));
}
