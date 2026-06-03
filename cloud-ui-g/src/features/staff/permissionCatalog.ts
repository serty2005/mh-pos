export type PermissionGroupId = 'shift' | 'cash' | 'sales' | 'pricing' | 'payments' | 'kitchen' | 'sync';

export type PermissionDefinition = {
  id: string;
  group: PermissionGroupId;
  labelKey: string;
};

// permissionCatalog повторяет backend-known permission IDs, чтобы role permissions_json не уходил с неизвестными правами.
export const permissionCatalog: PermissionDefinition[] = [
  { id: 'pos.employee_shift.open', group: 'shift', labelKey: 'staff.permissions.items.employeeShiftOpen' },
  { id: 'pos.employee_shift.close', group: 'shift', labelKey: 'staff.permissions.items.employeeShiftClose' },
  { id: 'pos.employee_shift.view_current', group: 'shift', labelKey: 'staff.permissions.items.employeeShiftViewCurrent' },
  { id: 'pos.employee_shift.recent', group: 'shift', labelKey: 'staff.permissions.items.employeeShiftRecent' },
  { id: 'pos.cash_session.open', group: 'cash', labelKey: 'staff.permissions.items.cashSessionOpen' },
  { id: 'pos.cash_session.close', group: 'cash', labelKey: 'staff.permissions.items.cashSessionClose' },
  { id: 'pos.cash_session.view_current', group: 'cash', labelKey: 'staff.permissions.items.cashSessionViewCurrent' },
  { id: 'pos.cash_drawer.record_event', group: 'cash', labelKey: 'staff.permissions.items.cashDrawerEvent' },
  { id: 'pos.catalog.view', group: 'sales', labelKey: 'staff.permissions.items.catalogView' },
  { id: 'pos.floor.view', group: 'sales', labelKey: 'staff.permissions.items.floorView' },
  { id: 'pos.menu.view', group: 'sales', labelKey: 'staff.permissions.items.menuView' },
  { id: 'pos.order.create', group: 'sales', labelKey: 'staff.permissions.items.orderCreate' },
  { id: 'pos.order.view', group: 'sales', labelKey: 'staff.permissions.items.orderView' },
  { id: 'pos.order.add_line', group: 'sales', labelKey: 'staff.permissions.items.orderAddLine' },
  { id: 'pos.order.change_quantity', group: 'sales', labelKey: 'staff.permissions.items.orderChangeQuantity' },
  { id: 'pos.order.void_line', group: 'sales', labelKey: 'staff.permissions.items.orderVoidLine' },
  { id: 'pos.order.close', group: 'sales', labelKey: 'staff.permissions.items.orderClose' },
  { id: 'pos.pricing.view', group: 'pricing', labelKey: 'staff.permissions.items.pricingView' },
  { id: 'pos.pricing.discount.apply', group: 'pricing', labelKey: 'staff.permissions.items.pricingDiscountApply' },
  { id: 'pos.pricing.surcharge.apply', group: 'pricing', labelKey: 'staff.permissions.items.pricingSurchargeApply' },
  { id: 'pos.precheck.issue', group: 'payments', labelKey: 'staff.permissions.items.precheckIssue' },
  { id: 'pos.precheck.view', group: 'payments', labelKey: 'staff.permissions.items.precheckView' },
  { id: 'pos.precheck.reprint', group: 'payments', labelKey: 'staff.permissions.items.precheckReprint' },
  { id: 'pos.precheck.cancel.request', group: 'payments', labelKey: 'staff.permissions.items.precheckCancelRequest' },
  { id: 'pos.precheck.cancel', group: 'payments', labelKey: 'staff.permissions.items.precheckCancel' },
  { id: 'pos.payment.cash', group: 'payments', labelKey: 'staff.permissions.items.paymentCash' },
  { id: 'pos.payment.card.manual', group: 'payments', labelKey: 'staff.permissions.items.paymentCardManual' },
  { id: 'pos.payment.other', group: 'payments', labelKey: 'staff.permissions.items.paymentOther' },
  { id: 'pos.payment.refund', group: 'payments', labelKey: 'staff.permissions.items.paymentRefund' },
  { id: 'pos.check.view', group: 'payments', labelKey: 'staff.permissions.items.checkView' },
  { id: 'pos.check.reprint', group: 'payments', labelKey: 'staff.permissions.items.checkReprint' },
  { id: 'pos.kitchen.view', group: 'kitchen', labelKey: 'staff.permissions.items.kitchenView' },
  { id: 'pos.kitchen.status.change', group: 'kitchen', labelKey: 'staff.permissions.items.kitchenStatusChange' },
  { id: 'pos.kitchen.catalog.view', group: 'kitchen', labelKey: 'staff.permissions.items.kitchenCatalogView' },
  { id: 'pos.kitchen.recipe.view', group: 'kitchen', labelKey: 'staff.permissions.items.kitchenRecipeView' },
  { id: 'pos.kitchen.recipe.suggest', group: 'kitchen', labelKey: 'staff.permissions.items.kitchenRecipeSuggest' },
  { id: 'pos.kitchen.catalog.suggest', group: 'kitchen', labelKey: 'staff.permissions.items.kitchenCatalogSuggest' },
  { id: 'pos.kitchen.stock.receipt', group: 'kitchen', labelKey: 'staff.permissions.items.kitchenStockReceipt' },
  { id: 'pos.kitchen.stock.inventory_count', group: 'kitchen', labelKey: 'staff.permissions.items.kitchenStockInventoryCount' },
  { id: 'pos.kitchen.stock.write_off', group: 'kitchen', labelKey: 'staff.permissions.items.kitchenStockWriteOff' },
  { id: 'pos.kitchen.production.complete', group: 'kitchen', labelKey: 'staff.permissions.items.kitchenProductionComplete' },
  { id: 'pos.kitchen.stop_list.view', group: 'kitchen', labelKey: 'staff.permissions.items.kitchenStopListView' },
  { id: 'pos.kitchen.stop_list.update', group: 'kitchen', labelKey: 'staff.permissions.items.kitchenStopListUpdate' },
  { id: 'pos.sync.view', group: 'sync', labelKey: 'staff.permissions.items.syncView' },
  { id: 'pos.sync.retry_failed', group: 'sync', labelKey: 'staff.permissions.items.syncRetryFailed' },
];

export const permissionGroupIds: PermissionGroupId[] = ['shift', 'cash', 'sales', 'pricing', 'payments', 'kitchen', 'sync'];

export const knownPermissionIds = new Set(permissionCatalog.map((permission) => permission.id));
