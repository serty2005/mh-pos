export const permissionCatalog = {
  shiftOpen: 'pos.shift.open',
  shiftClose: 'pos.shift.close',
  floorView: 'pos.floor.view',
  menuView: 'pos.menu.view',
  cashSessionOpen: 'pos.cash_session.open',
  cashSessionClose: 'pos.cash_session.close',
  orderCreate: 'pos.order.create',
  orderAddLine: 'pos.order.add_line',
  orderChangeQuantity: 'pos.order.change_quantity',
  orderVoidLine: 'pos.order.void_line',
  precheckIssue: 'pos.precheck.issue',
  precheckCancelRequest: 'pos.precheck.cancel.request',
  paymentCapture: 'pos.payment.capture',
} as const;

export type PermissionID = typeof permissionCatalog[keyof typeof permissionCatalog];

export function hasPermission(granted: string[] | undefined, required: PermissionID): boolean {
  return Boolean(granted?.includes(required));
}
