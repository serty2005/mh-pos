import type {
  CashDrawerEvent,
  CashSession,
  ClosedOrder,
  EmployeeShift,
  FinancialOperation,
  Hall,
  MenuItem,
  Order,
  OrderLine,
  Payment,
  PricingPolicy,
  SelectedModifier,
  Table,
} from '../types';
import type {
  BackendActorContext,
  BackendCashDrawerEvent,
  BackendCashSession,
  BackendClosedOrder,
  BackendFinancialOperation,
  BackendHall,
  BackendMenuItem,
  BackendOrder,
  BackendOrderLine,
  BackendPayment,
  BackendPrecheck,
  BackendPricingPolicy,
  BackendShift,
  BackendSyncStatus,
  BackendTable,
} from './schemas';

export type InventoryDisposition = 'no_stock_effect' | 'return_to_stock' | 'write_off_waste' | 'manual_review';

export function mapHall(hall: BackendHall): Hall {
  return {
    id: hall.id,
    name: hall.name,
  };
}

export function mapTable(table: BackendTable, activeOrder?: BackendOrder | null): Table {
  return {
    id: table.id,
    number: extractTableNumber(table.name),
    hallId: table.hall_id,
    status: activeOrder ? 'occupied' : 'free',
    currentOrderId: activeOrder?.id,
    activeOrderSum: activeOrder?.total ?? 0,
    guestsCount: activeOrder?.guest_count ?? 0,
  };
}

export function mapMenuItem(item: BackendMenuItem): MenuItem {
  return {
    id: item.id,
    name: item.name,
    price: item.price,
    category: item.item_type === 'service' ? 'services' : item.item_type,
    isAvailable: item.active,
    modifierGroups: item.modifier_groups
      .filter((group) => group.active)
      .map((group) => ({
        id: group.id,
        name: group.name,
        minRequired: group.required ? Math.max(1, group.min_count) : group.min_count,
        maxAllowed: group.max_count,
        options: group.options
          .filter((option) => option.active)
          .map((option) => ({
            id: option.id,
            name: option.name,
            price: option.price_minor,
          })),
      })),
  };
}

export function mapPricingPolicy(policy: BackendPricingPolicy): PricingPolicy {
  return {
    id: policy.id,
    kind: policy.kind,
    name: policy.name,
    scope: policy.scope,
    amountKind: policy.amount_kind,
    amount: policy.amount_minor,
    valueBasisPoints: policy.value_basis_points,
    applicationIndex: policy.application_index,
    requiresPermission: policy.requires_permission || undefined,
  };
}

export function mapOrder(order: BackendOrder, activePrecheck?: BackendPrecheck | null): Order {
  const precheckIssued = order.status === 'locked' || activePrecheck?.status === 'issued';
  return {
    id: order.id,
    shortId: shortId(order.edge_order_id || order.id),
    tableId: order.table_id,
    tableName: order.table_name,
    status: precheckIssued ? 'precheck_issued' : order.status === 'closed' ? 'closed' : 'open',
    lines: order.lines.filter((line) => line.status === 'active').map(mapOrderLine),
    subtotal: order.subtotal,
    tax: order.tax_total,
    discount: order.discount_total,
    total: order.total,
    precheckTime: activePrecheck?.issued_at,
    precheckBy: activePrecheck?.cancelled_by_employee_id ?? undefined,
    payments: order.check?.payments?.map(mapPayment) ?? [],
    openedAt: formatTime(order.opened_at),
    waiterName: order.shift_id,
  };
}

export function mapClosedOrder(order: BackendClosedOrder, operations: BackendFinancialOperation[] = []): ClosedOrder {
  const check = order.check;
  const payment = check?.payments?.find((item) => item.status === 'captured') ?? check?.payments?.[0];
  const status = check?.status === 'refunded'
    ? 'refunded'
    : order.status === 'cancelled'
      ? 'cancelled'
      : 'closed';
  return {
    id: check?.id ?? order.id,
    shortId: shortId(check?.id ?? order.id),
    tableName: order.table_name,
    total: check?.total ?? order.total,
    paymentMethod: payment?.method === 'card' ? 'card' : 'cash',
    closedAt: order.closed_at ?? check?.closed_at ?? order.opened_at,
    operator: payment?.shift_id ?? '',
    status,
    originalShiftId: payment?.shift_id ?? '',
    lines: extractClosedOrderLines(check?.snapshot),
    operations: operations.map(mapFinancialOperation),
  };
}

export function mapPayment(payment: BackendPayment): Payment {
  return {
    id: payment.id,
    method: payment.method === 'card' ? 'card' : 'cash',
    amount: payment.amount,
    timestamp: payment.created_at,
  };
}

export function mapCashSession(session: BackendCashSession | null, actor?: BackendActorContext | null): CashSession | null {
  if (!session) return null;
  return {
    id: session.id,
    openedAt: session.opened_at,
    closedAt: session.closed_at ?? undefined,
    initialAmount: session.opening_cash_amount,
    currentAmount: session.closing_cash_amount ?? session.opening_cash_amount,
    status: session.status,
    openedBy: actor?.name ?? session.opened_by_employee_id,
  };
}

export function mapCashDrawerEvent(event: BackendCashDrawerEvent, actor?: BackendActorContext | null): CashDrawerEvent {
  return {
    id: event.id,
    timestamp: event.occurred_at,
    type: event.event_type === 'cash_out' ? 'out' : 'in',
    amount: event.amount,
    reason: event.reason ?? '',
    operator: actor?.name ?? event.created_by_employee_id,
  };
}

export function mapOperator(actor: BackendActorContext | null, shift: BackendShift | null): EmployeeShift | null {
  if (!actor || !shift) return null;
  return {
    id: shift.id,
    employeeName: actor.name,
    role: roleFromPermissions(actor.permissions),
    openTime: shift.opened_at,
    closeTime: shift.closed_at ?? undefined,
    status: shift.status,
  };
}

export function mapSyncStatus(status: BackendSyncStatus | null): 'online' | 'offline' {
  if (!status) return 'offline';
  return status.failed > 0 || status.suspended > 0 ? 'offline' : 'online';
}

export function outboxCount(status: BackendSyncStatus | null): number {
  if (!status) return 0;
  return status.pending + status.processing + status.failed + status.suspended;
}

export function selectedModifiersToPayload(selectedModifiers: SelectedModifier[]) {
  return selectedModifiers.map((modifier) => ({
    modifier_group_id: modifier.groupId,
    modifier_option_id: modifier.optionId,
    quantity: modifier.quantity ?? 1,
  }));
}

export function dispositionToBackend(disposition: 'waste' | 'return'): InventoryDisposition {
  return disposition === 'waste' ? 'write_off_waste' : 'return_to_stock';
}

function mapOrderLine(line: BackendOrderLine): OrderLine {
  return {
    id: line.id,
    itemId: line.menu_item_id,
    name: line.name,
    price: line.unit_price,
    quantity: line.quantity,
    selectedModifiers: line.modifiers.map((modifier) => ({
      groupId: modifier.modifier_group_id,
      groupName: modifier.modifier_group_id,
      optionId: modifier.modifier_option_id,
      optionName: modifier.name,
      price: modifier.unit_price,
      quantity: modifier.quantity,
    })),
    comment: line.comment ?? undefined,
    course: Number(line.course) || 1,
  };
}

function mapFinancialOperation(operation: BackendFinancialOperation): FinancialOperation {
  return {
    id: operation.id,
    type: operation.operation_type,
    kind: operation.operation_kind === 'full' ? 'Полная операция' : 'Частичная операция',
    amount: operation.amount,
    reason: operation.reason,
    employee: operation.created_by_employee_id,
    timestamp: operation.created_at,
    disposition: operation.inventory_disposition === 'write_off_waste' ? 'waste' : 'return',
  };
}

function extractClosedOrderLines(snapshot: unknown): OrderLine[] {
  if (!snapshot || typeof snapshot !== 'object') return [];
  const root = snapshot as { precheck_snapshot?: { lines?: unknown[] } };
  const lines = root.precheck_snapshot?.lines;
  if (!Array.isArray(lines)) return [];
  return lines.map((raw) => {
    const line = raw as Record<string, unknown>;
    return {
      id: String(line.order_line_id ?? ''),
      itemId: String(line.menu_item_id ?? line.catalog_item_id ?? ''),
      name: String(line.name ?? ''),
      price: Number(line.unit_price_minor ?? 0),
      quantity: Number(line.quantity ?? 0),
      selectedModifiers: [],
      course: 1,
    };
  }).filter((line) => line.id && line.quantity > 0);
}

function extractTableNumber(value: string): number {
  const match = value.match(/\d+/);
  return match ? Number(match[0]) : 0;
}

function shortId(value: string): string {
  return value.length > 8 ? value.slice(0, 8) : value;
}

function formatTime(value: string): string {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return new Intl.DateTimeFormat('ru-RU', { hour: '2-digit', minute: '2-digit' }).format(date);
}

function roleFromPermissions(permissions: string[]): EmployeeShift['role'] {
  if (permissions.includes('pos.precheck.cancel') || permissions.includes('pos.payment.refund')) return 'manager';
  if (permissions.includes('pos.payment.cash') || permissions.includes('pos.payment.card.manual')) return 'cashier';
  return 'waiter';
}
