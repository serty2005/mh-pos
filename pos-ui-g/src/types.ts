export interface ModifierOption {
  id: string;
  name: string;
  price: number;
}

export interface ModifierGroup {
  id: string;
  name: string;
  minRequired: number;
  maxAllowed: number;
  options: ModifierOption[];
}

export interface MenuItem {
  id: string;
  catalogItemId: string;
  name: string;
  price: number;
  category: string;
  isAvailable: boolean;
  singleUnitPerLine?: boolean;
  stopListActive?: boolean;
  stopListBlocked?: boolean;
  stopListAvailableQuantity?: number;
  modifierGroups?: ModifierGroup[];
  hasImage?: boolean; // Waiter menu items
}

export interface SelectedModifier {
  groupId: string;
  groupName: string;
  optionId: string;
  optionName: string;
  price: number;
  quantity?: number;
}

export interface PricingPolicy {
  id: string;
  kind: 'discount' | 'surcharge';
  name: string;
  scope: 'line' | 'order';
  amountKind: 'fixed' | 'percentage';
  amount: number;
  valueBasisPoints: number;
  applicationIndex: number;
  requiresPermission?: string;
}

export interface OrderLine {
  id: string;
  itemId: string;
  name: string;
  price: number;
  totalPrice: number;
  quantity: number;
  selectedModifiers: SelectedModifier[];
  comment?: string;
  course?: number; // Подача: 1, 2, 3
}

export type OrderStatus = 'open' | 'precheck_issued' | 'paid' | 'closed';

export interface Payment {
  id: string;
  method: 'cash' | 'card';
  amount: number;
  timestamp: string;
}

export interface Order {
  id: string;
  shortId: string;
  tableId?: string;
  tableName?: string;
  hallName?: string;
  status: OrderStatus;
  lines: OrderLine[];
  subtotal: number;
  tax: number;
  discount: number;
  total: number;
  precheckTime?: string;
  precheckBy?: string;
  payments: Payment[];
  openedAt: string;
  waiterName: string;
}

export interface Table {
  id: string;
  number: number;
  hallId: string;
  status: 'free' | 'occupied' | 'reserved';
  currentOrderId?: string;
  activeOrderSum?: number;
  waiter?: string;
  guestsCount?: number;
}

export interface Hall {
  id: string;
  name: string;
}

export interface EmployeeShift {
  id: string;
  employeeName: string;
  role: 'cashier' | 'waiter' | 'kitchen' | 'manager' | 'support';
  permissions: string[];
  openTime: string;
  closeTime?: string;
  status: 'open' | 'closed';
}

export interface CashSession {
  id: string;
  openedAt: string;
  closedAt?: string;
  initialAmount: number;
  currentAmount: number;
  status: 'open' | 'closed';
  openedBy: string;
}

export interface CashDrawerEvent {
  id: string;
  timestamp: string;
  type: 'in' | 'out' | 'no_sale';
  amount: number;
  reason: string;
  operator: string;
}

export interface FinancialOperation {
  id: string;
  type: 'payment' | 'refund' | 'cancellation';
  kind: string; // e.g. "Наличные", "Карта", "Списание", "Отмена"
  amount: number;
  reason: string;
  employee: string;
  timestamp: string;
  disposition?: 'waste' | 'return'; // списание утилизацией или на склад
}

export interface ClosedOrder {
  id: string;
  shortId: string;
  tableName?: string;
  hallName?: string;
  total: number;
  paymentMethod: 'cash' | 'card';
  closedAt: string;
  operator: string;
  status: 'closed' | 'cancelled' | 'refunded' | 'partially_refunded';
  originalShiftId: string;
  lines: OrderLine[];
  refundReason?: string;
  operations: FinancialOperation[];
}

export type POSSection = 'floor' | 'order' | 'activity' | 'reports' | 'cash';

// Canonical permission catalog ids matching POS UI RBAC rules
export const permissions = {
  EMPLOYEE_SHIFT_OPEN: 'pos.employee_shift.open',
  EMPLOYEE_SHIFT_CLOSE: 'pos.employee_shift.close',
  EMPLOYEE_SHIFT_VIEW: 'pos.employee_shift.view_current',
  EMPLOYEE_SHIFT_RECENT: 'pos.employee_shift.recent',
  CASH_SESSION_OPEN: 'pos.cash_session.open',
  CASH_SESSION_CLOSE: 'pos.cash_session.close',
  CASH_SESSION_VIEW: 'pos.cash_session.view_current',
  CASH_DRAWER_RECORD: 'pos.cash_drawer.record_event',
  CATALOG_VIEW: 'pos.catalog.view',
  FLOOR_VIEW: 'pos.floor.view',
  MENU_VIEW: 'pos.menu.view',
  ORDER_CREATE: 'pos.order.create',
  ORDER_VIEW: 'pos.order.view',
  ORDER_ADD_LINE: 'pos.order.add_line',
  ORDER_CHANGE_QTY: 'pos.order.change_quantity',
  ORDER_VOID_LINE: 'pos.order.void_line',
  ORDER_CLOSE: 'pos.order.close',
  PRICING_VIEW: 'pos.pricing.view',
  PRICING_DISCOUNT_APPLY: 'pos.pricing.discount.apply',
  PRICING_SURCHARGE_APPLY: 'pos.pricing.surcharge.apply',
  PRECHECK_ISSUE: 'pos.precheck.issue',
  PRECHECK_VIEW: 'pos.precheck.view',
  PRECHECK_CANCEL_REQUEST: 'pos.precheck.cancel.request',
  PRECHECK_CANCEL: 'pos.precheck.cancel',
  PRECHECK_REPRINT: 'pos.precheck.reprint',
  PAYMENT_CASH: 'pos.payment.cash',
  PAYMENT_CARD: 'pos.payment.card.manual',
  PAYMENT_OTHER: 'pos.payment.other',
  PAYMENT_REFUND: 'pos.payment.refund',
  CHECK_VIEW: 'pos.check.view',
  CHECK_REPRINT: 'pos.check.reprint',
  SYNC_VIEW: 'pos.sync.view',
  SYNC_RETRY: 'pos.sync.retry_failed',
  KITCHEN_VIEW: 'pos.kitchen.view',
  KITCHEN_STATUS_CHANGE: 'pos.kitchen.status.change',
  KITCHEN_CATALOG_VIEW: 'pos.kitchen.catalog.view',
  KITCHEN_RECIPE_VIEW: 'pos.kitchen.recipe.view',
  KITCHEN_RECIPE_SUGGEST: 'pos.kitchen.recipe.suggest',
  KITCHEN_CATALOG_SUGGEST: 'pos.kitchen.catalog.suggest',
  KITCHEN_STOCK_RECEIPT: 'pos.kitchen.stock.receipt',
  KITCHEN_STOCK_INVENTORY_COUNT: 'pos.kitchen.stock.inventory_count',
  KITCHEN_STOCK_WRITE_OFF: 'pos.kitchen.stock.write_off',
  KITCHEN_PRODUCTION_COMPLETE: 'pos.kitchen.production.complete',
  KITCHEN_STOP_LIST_VIEW: 'pos.kitchen.stop_list.view',
  KITCHEN_STOP_LIST_UPDATE: 'pos.kitchen.stop_list.update',
} as const;
