import { useMutation, useQuery, useQueryClient, type QueryClient } from '@tanstack/vue-query';
import { useQuasar } from 'quasar';
import { computed, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRouter } from 'vue-router';

import {
  addOrderLine,
  ApiError,
  cancelPrecheck,
  capturePrecheckPayment,
  changeOrderLineQuantity,
  closeCashSession,
  closeShift,
  createOrder,
  getAuthSession,
  getCheck,
  getCurrentCashSession,
  getCurrentOrderByTable,
  getCurrentShift,
  getOrder,
  getPairingStatus,
  getSyncStatus,
  issuePrecheck,
  listActiveOrdersByHall,
  listClosedOrders,
  listHalls,
  listLocalEvents,
  listMenuItems,
  listPrechecksByOrder,
  listRecentShifts,
  listSyncOutbox,
  listTables,
  logout,
  openCashSession,
  openShift,
  recordCashDrawerEvent,
  recordCheckCancellation,
  recordCheckRefund,
  refundPayment,
  reprintCheck,
  reprintPrecheck,
  retryFailedOutbox,
  updateOrderLineDetails,
  updateOrderLineModifiers,
  voidOrderLine,
  type CashDrawerEventType,
  type FinancialOperationItemPayload,
  type FinancialOperationKind,
  type InventoryDisposition,
  type SelectedModifierPayload,
} from '../../shared/api';
import { currencyInputStep, formatMinorCurrency, minorToMoney, moneyToMinor } from '../../shared/currency';
import { displayErrorMessageKey, useErrorHandling } from '../../shared/errorHandling';
import { hasPermission, permissionCatalog } from '../../shared/rbac';
import { resolveProtectedPosFallback } from '../../shared/sessionGuards';
import { checkSnapshotSchema, type CheckSnapshotLine, type ClosedOrder, type MenuItem, type Order } from '../../shared/schemas';
import { useAuthStore } from '../../stores/auth';

export const paymentConflictInvalidationQueryKeys = [
  ['current-cash-session'],
  ['current-order'],
  ['order'],
  ['prechecks'],
  ['check'],
  ['closed-orders'],
] as const;

export function invalidatePaymentConflictQueries(queryClient: Pick<QueryClient, 'invalidateQueries'>) {
  for (const queryKey of paymentConflictInvalidationQueryKeys) {
    void queryClient.invalidateQueries({ queryKey });
  }
}

type FlowStepState = 'ready' | 'active' | 'blocked' | 'pending';
type CompensationMode = 'payment_refund' | 'check_refund' | 'check_cancellation';
type LedgerScope = 'whole_check' | 'order_line';
type ModifierDialogMode = 'add' | 'edit';
export type ClosedOrderCompensationAction = 'check_cancellation' | 'check_refund' | 'payment_refund';
type BlockingNotice = {
  titleKey: string;
  reasonKey: string;
  permission?: string;
};

export type ClosedOrderCompensationContext = {
  canRefundPayment: boolean;
  canRecordCheckCancellation: boolean;
  currentCashSessionShiftId: string;
};

export function closedOrderOriginalPaymentShiftID(orderItem: ClosedOrder) {
  return orderItem.check?.payments?.find((payment) => payment.status === 'captured')?.shift_id ?? orderItem.check?.payments?.[0]?.shift_id ?? '';
}

export function closedOrderHasCapturedPayment(orderItem: ClosedOrder) {
  return Boolean(orderItem.check?.payments?.some((payment) => payment.status === 'captured'));
}

export function closedOrderCompensationUnavailableKey(orderItem: ClosedOrder, action: ClosedOrderCompensationAction, context: ClosedOrderCompensationContext) {
  if (!orderItem.check) return 'pos.compensationUnavailable.noFinalCheck';
  if (!context.currentCashSessionShiftId) return 'pos.compensationUnavailable.noCashSession';
  if (action === 'check_cancellation') {
    if (!context.canRecordCheckCancellation) return 'pos.compensationUnavailable.noCancellationPermission';
    if (closedOrderOriginalPaymentShiftID(orderItem) !== context.currentCashSessionShiftId) return 'pos.compensationUnavailable.cancellationBoundary';
    return '';
  }
  if (!closedOrderHasCapturedPayment(orderItem)) return 'pos.compensationUnavailable.noCapturedPayment';
  if (!context.canRefundPayment) return 'pos.compensationUnavailable.noRefundPermission';
  const originalShiftID = closedOrderOriginalPaymentShiftID(orderItem);
  if (originalShiftID && originalShiftID === context.currentCashSessionShiftId) return 'pos.compensationUnavailable.refundBoundary';
  return '';
}

export function useCashierTerminal() {
  const { t } = useI18n();
  const auth = useAuthStore();
  const router = useRouter();
  const $q = useQuasar();
  const queryClient = useQueryClient();
  const { showBusinessError } = useErrorHandling();

  const selectedHallId = ref('');
  const selectedTableId = ref('');
  const selectedOrderLineId = ref('');
  const currentOrderId = ref('');
  const openingCashAmount = ref(0);
  const closingCashAmount = ref(0);
  const paymentAmount = ref(0);
  const menuSearch = ref('');
  const cashDrawerType = ref<CashDrawerEventType>('no_sale');
  const cashDrawerAmount = ref(0);
  const cashDrawerReason = ref('');
  const cashDrawerNote = ref('');
  const localEventFilter = ref('');
  const cancelDialog = ref(false);
  const managerEmployeeId = ref('');
  const managerPin = ref('');
  const cancelReason = ref('');
  const closedOrdersDrawer = ref(false);
  const cashDrawerDialog = ref(false);
  const syncDrawer = ref(false);
  const refundDialog = ref(false);
  const refundMode = ref<CompensationMode>('payment_refund');
  const refundPaymentId = ref('');
  const refundCheckId = ref('');
  const refundOrder = ref<ClosedOrder | null>(null);
  const refundReason = ref('');
  const refundInventoryDisposition = ref<InventoryDisposition>('no_stock_effect');
  const refundOperationKind = ref<FinancialOperationKind>('full');
  const refundScope = ref<LedgerScope>('whole_check');
  const refundOrderLines = ref<CheckSnapshotLine[]>([]);
  const refundOrderLineId = ref('');
  const refundLineQuantity = ref(1);
  const closedOrdersLimit = ref(50);
  const closedOrdersOffset = ref(0);
  const closedOrdersBusinessDate = ref('');
  const modifierDialog = ref(false);
  const modifierDialogMode = ref<ModifierDialogMode>('add');
  const modifierLineId = ref('');
  const modifierMenuItem = ref<MenuItem | null>(null);
  const modifierQuantities = ref<Record<string, number>>({});
  const modifierValidationKey = ref('');
  const lineCourseDraft = ref('');
  const lineCommentDraft = ref('');

  const grantedPermissions = computed(() => auth.actor?.permissions ?? []);
  const canViewFloor = computed(() => hasPermission(grantedPermissions.value, permissionCatalog.floorView));
  const canViewMenu = computed(() => hasPermission(grantedPermissions.value, permissionCatalog.menuView));
  const canViewCurrentShift = computed(() => hasPermission(grantedPermissions.value, permissionCatalog.employeeShiftViewCurrent));
  const canViewRecentShifts = computed(() => hasPermission(grantedPermissions.value, permissionCatalog.employeeShiftRecent));
  const canOpenShift = computed(() => hasPermission(grantedPermissions.value, permissionCatalog.employeeShiftOpen));
  const canCloseShift = computed(() => hasPermission(grantedPermissions.value, permissionCatalog.employeeShiftClose));
  const canOpenCashSession = computed(() => hasPermission(grantedPermissions.value, permissionCatalog.cashSessionOpen));
  const canCloseCashSession = computed(() => hasPermission(grantedPermissions.value, permissionCatalog.cashSessionClose));
  const canViewCurrentCashSession = computed(() => hasPermission(grantedPermissions.value, permissionCatalog.cashSessionViewCurrent));
  const canViewSync = computed(() => hasPermission(grantedPermissions.value, permissionCatalog.syncView));
  const canRetrySync = computed(() => hasPermission(grantedPermissions.value, permissionCatalog.syncRetryFailed));
  const canViewClosedOrders = computed(() => hasPermission(grantedPermissions.value, permissionCatalog.checkView));
  const canRefundPayment = computed(() => hasPermission(grantedPermissions.value, permissionCatalog.paymentRefund));
  const canRecordCheckCancellation = computed(() => hasPermission(grantedPermissions.value, permissionCatalog.precheckCancel));

  const cashDrawerTypeOptions = computed(() => [
    { label: t('pos.cashDrawerTypes.no_sale'), value: 'no_sale' },
    { label: t('pos.cashDrawerTypes.cash_in'), value: 'cash_in' },
    { label: t('pos.cashDrawerTypes.cash_out'), value: 'cash_out' },
    { label: t('pos.cashDrawerTypes.cash_count'), value: 'cash_count' },
  ]);

  const inventoryDispositionOptions = computed<Array<{ label: string; value: InventoryDisposition }>>(() => [
    { label: t('pos.inventoryDispositions.no_stock_effect'), value: 'no_stock_effect' },
    { label: t('pos.inventoryDispositions.return_to_stock'), value: 'return_to_stock' },
    { label: t('pos.inventoryDispositions.write_off_waste'), value: 'write_off_waste' },
    { label: t('pos.inventoryDispositions.manual_review'), value: 'manual_review' },
  ]);

  const ledgerScopeOptions = computed<Array<{ label: string; value: LedgerScope }>>(() => {
    const options: Array<{ label: string; value: LedgerScope }> = [
      { label: t('pos.ledgerScopes.whole_check'), value: 'whole_check' },
    ];
    if (refundOrderLines.value.length > 0) {
      options.push({ label: t('pos.ledgerScopes.order_line'), value: 'order_line' });
    }
    return options;
  });

  const refundLineOptions = computed(() => refundOrderLines.value.map((line) => ({
    label: `${line.name} · ${line.quantity} × ${money(line.unit_price_minor, line.currency_code)}`,
    value: line.order_line_id,
  })));

  const selectedRefundLine = computed(() => refundOrderLines.value.find((line) => line.order_line_id === refundOrderLineId.value) ?? refundOrderLines.value[0] ?? null);

  const maxRefundLineQuantity = computed(() => Math.max(1, selectedRefundLine.value?.quantity ?? 1));

  const refundLineAmount = computed(() => {
    const line = selectedRefundLine.value;
    if (!line) return 0;
    return proportionalMinor(line.total_minor, line.quantity, refundLineQuantity.value);
  });

  const refundLineTaxAmount = computed(() => {
    const line = selectedRefundLine.value;
    if (!line) return 0;
    return proportionalMinor(line.tax_total_minor, line.quantity, refundLineQuantity.value);
  });

  const currentLedgerOperationKind = computed<FinancialOperationKind>(() => {
    if (refundScope.value === 'whole_check') return 'full';
    const checkTotal = refundOrder.value?.check?.total ?? 0;
    return checkTotal > 0 && refundLineAmount.value >= checkTotal ? 'full' : 'partial';
  });

  const unsupportedLedgerScopeOptions = computed(() => [
    t('pos.ledgerScopes.modifier_line'),
    t('pos.ledgerScopes.service_charge'),
    t('pos.ledgerScopes.tip'),
  ]);

  const pairing = useQuery({
    queryKey: ['pairing-status'],
    queryFn: getPairingStatus,
  });

  const session = useQuery({
    queryKey: ['auth-session', auth.sessionId, auth.nodeDeviceId, auth.clientDeviceId],
    queryFn: getAuthSession,
    enabled: () => Boolean(auth.sessionId && auth.nodeDeviceId),
    retry: false,
  });

  const currentShift = useQuery({
    queryKey: ['current-shift', auth.nodeDeviceId],
    queryFn: getCurrentShift,
    enabled: () => Boolean(auth.nodeDeviceId && auth.sessionId && canViewCurrentShift.value),
  });

  const recentShifts = useQuery({
    queryKey: ['recent-shifts', auth.sessionId],
    queryFn: listRecentShifts,
    enabled: () => Boolean(auth.nodeDeviceId && auth.sessionId && !currentShift.data.value && canViewRecentShifts.value),
  });

  const currentCashSession = useQuery({
    queryKey: ['current-cash-session', auth.nodeDeviceId],
    queryFn: getCurrentCashSession,
    enabled: () => Boolean(auth.nodeDeviceId && auth.sessionId && currentShift.data.value && canViewCurrentCashSession.value),
  });

  const halls = useQuery({
    queryKey: ['halls', auth.restaurantId],
    queryFn: () => listHalls(auth.restaurantId),
    enabled: () => Boolean(auth.restaurantId && auth.sessionId && currentShift.data.value && canViewFloor.value),
  });

  const activeHallId = computed(() => selectedHallId.value || halls.data.value?.find((hall) => hall.active)?.id || '');

  const tables = useQuery({
    queryKey: ['tables', auth.restaurantId, activeHallId],
    queryFn: () => listTables(auth.restaurantId, activeHallId.value),
    enabled: () => Boolean(auth.restaurantId && activeHallId.value && auth.sessionId && currentShift.data.value && canViewFloor.value),
  });

  const activeOrdersQuery = useQuery({
    queryKey: ['active-orders', activeHallId],
    queryFn: () => listActiveOrdersByHall(activeHallId.value),
    enabled: () => Boolean(activeHallId.value && auth.nodeDeviceId && auth.sessionId && currentShift.data.value && hasPermission(grantedPermissions.value, permissionCatalog.orderView)),
  });

  const tableOrder = useQuery({
    queryKey: ['current-order', selectedTableId],
    queryFn: () => getCurrentOrderByTable(selectedTableId.value),
    enabled: () => Boolean(selectedTableId.value && auth.nodeDeviceId && auth.sessionId && currentShift.data.value && hasPermission(grantedPermissions.value, permissionCatalog.orderView)),
  });

  const order = useQuery({
    queryKey: ['order', currentOrderId],
    queryFn: () => getOrder(currentOrderId.value),
    enabled: () => Boolean(currentOrderId.value && auth.sessionId && currentShift.data.value && hasPermission(grantedPermissions.value, permissionCatalog.orderView)),
  });

  const prechecks = useQuery({
    queryKey: ['prechecks', currentOrderId],
    queryFn: () => listPrechecksByOrder(currentOrderId.value),
    enabled: () => Boolean(currentOrderId.value && auth.sessionId && currentShift.data.value && hasPermission(grantedPermissions.value, permissionCatalog.precheckView)),
  });

  const menu = useQuery({
    queryKey: ['menu-items'],
    queryFn: listMenuItems,
    enabled: () => Boolean(auth.sessionId && currentShift.data.value && canViewMenu.value),
  });

  const activeHalls = computed(() => halls.data.value?.filter((hall) => hall.active) ?? []);
  const activeTables = computed(() => tables.data.value?.filter((table) => table.active) ?? []);
  const selectedTable = computed(() => activeTables.value.find((table) => table.id === selectedTableId.value));
  const activeOrders = computed(() => activeOrdersQuery.data.value ?? []);
  const activeOrder = computed<Order | null>(() => order.data.value ?? tableOrder.data.value ?? activeOrders.value.find((item) => item.table_id === selectedTableId.value) ?? null);
  const finalCheckId = computed(() => activeOrder.value?.check?.id ?? '');
  const finalCheck = useQuery({
    queryKey: ['check', finalCheckId],
    queryFn: () => getCheck(finalCheckId.value),
    enabled: () => Boolean(finalCheckId.value && auth.sessionId && currentShift.data.value && hasPermission(grantedPermissions.value, permissionCatalog.checkView)),
  });
  const activeLines = computed(() => activeOrder.value?.lines.filter((line) => line.status === 'active') ?? []);
  const selectedOrderLine = computed(() => activeLines.value.find((line) => line.id === selectedOrderLineId.value) ?? activeLines.value[0] ?? null);
  const activeMenuItems = computed(() => menu.data.value?.filter((item) => item.active) ?? []);
  const regularMenuItems = computed(() => activeMenuItems.value.filter((item) => item.item_type !== 'service'));
  const serviceMenuItems = computed(() => activeMenuItems.value.filter((item) => item.item_type === 'service'));
  const visibleMenuItems = computed(() => {
    const query = menuSearch.value.trim().toLocaleLowerCase('ru-RU');
    if (!query) return regularMenuItems.value;
    return regularMenuItems.value.filter((item) => item.name.toLocaleLowerCase('ru-RU').includes(query));
  });
  const visibleServiceItems = computed(() => {
    const query = menuSearch.value.trim().toLocaleLowerCase('ru-RU');
    if (!query) return serviceMenuItems.value;
    return serviceMenuItems.value.filter((item) => item.name.toLocaleLowerCase('ru-RU').includes(query));
  });
  const modifierGroupsForDialog = computed(() => modifierMenuItem.value?.modifier_groups.filter((group) => group.active && group.options.some((option) => option.active)) ?? []);
  const selectedModifierPayload = computed<SelectedModifierPayload[]>(() => {
    const item = modifierMenuItem.value;
    if (!item) return [];
    const payload: SelectedModifierPayload[] = [];
    for (const group of modifierGroupsForDialog.value) {
      for (const option of group.options) {
        const quantity = modifierQuantities.value[option.id] ?? 0;
        if (quantity > 0) {
          payload.push({
            modifier_group_id: group.id,
            modifier_option_id: option.id,
            quantity,
          });
        }
      }
    }
    return payload;
  });
  const selectedModifierTotal = computed(() => {
    let total = 0;
    for (const group of modifierGroupsForDialog.value) {
      for (const option of group.options) {
        total += (modifierQuantities.value[option.id] ?? 0) * option.price_minor;
      }
    }
    return total;
  });
  const canSubmitModifierSelection = computed(() => Boolean(modifierMenuItem.value && validateModifierSelection() === ''));

  const syncStatus = useQuery({
    queryKey: ['sync-status', auth.sessionId],
    queryFn: getSyncStatus,
    enabled: () => Boolean(auth.nodeDeviceId && auth.sessionId && canViewSync.value),
    refetchInterval: 30_000,
  });

  const syncOutbox = useQuery({
    queryKey: ['sync-outbox', auth.sessionId],
    queryFn: () => listSyncOutbox(5),
    enabled: () => Boolean(auth.nodeDeviceId && auth.sessionId && canViewSync.value),
  });

  const localEvents = useQuery({
    queryKey: ['local-events', auth.sessionId, localEventFilter],
    queryFn: () => listLocalEvents(5, localEventFilter.value),
    enabled: () => Boolean(auth.nodeDeviceId && auth.sessionId && canViewSync.value),
  });

  const closedOrders = useQuery({
    queryKey: ['closed-orders', closedOrdersLimit, closedOrdersOffset, closedOrdersBusinessDate],
    queryFn: () => listClosedOrders({
      businessDateLocal: closedOrdersBusinessDate.value,
      limit: closedOrdersLimit.value,
      offset: closedOrdersOffset.value,
    }),
    enabled: () => Boolean(auth.nodeDeviceId && auth.sessionId && canViewClosedOrders.value),
  });

  const activePrecheck = computed(() => prechecks.data.value?.find((item) => item.status === 'issued') ?? null);
  const latestPrecheck = computed(() => {
    const items = prechecks.data.value ?? [];
    return items.reduce((latest, item) => (latest === null || item.version > latest.version ? item : latest), null as (typeof items)[number] | null);
  });
  const finalCheckData = computed(() => finalCheck.data.value ?? activeOrder.value?.check ?? null);
  const currency = computed(() => activeMenuItems.value[0]?.currency ?? 'RUB');
  const orderCurrency = computed(() => activeMenuItems.value.find((item) => activeLines.value.some((line) => line.menu_item_id === item.id))?.currency ?? currency.value);
  const canRecordCashDrawerEvent = computed(() => Boolean(currentCashSession.data.value && hasPermission(grantedPermissions.value, permissionCatalog.cashDrawerRecordEvent)));
  const canCreateOrder = computed(() => Boolean(selectedTableId.value && currentShift.data.value && !activeOrder.value && hasPermission(grantedPermissions.value, permissionCatalog.orderCreate)));
  const canAddOrderLine = computed(() => Boolean(activeOrder.value?.status === 'open' && !activePrecheck.value && !activeOrder.value.check && hasPermission(grantedPermissions.value, permissionCatalog.orderAddLine)));
  const canChangeOrderLine = computed(() => Boolean(activeOrder.value?.status === 'open' && !activePrecheck.value && !activeOrder.value.check && hasPermission(grantedPermissions.value, permissionCatalog.orderChangeQuantity)));
  const canVoidOrderLine = computed(() => Boolean(activeOrder.value?.status === 'open' && !activePrecheck.value && !activeOrder.value.check && hasPermission(grantedPermissions.value, permissionCatalog.orderVoidLine)));
  const canIssuePrecheck = computed(() => Boolean(activeOrder.value?.status === 'open' && activeLines.value.length > 0 && !activePrecheck.value && hasPermission(grantedPermissions.value, permissionCatalog.precheckIssue)));
  const canCancelPrecheck = computed(() => hasPermission(grantedPermissions.value, permissionCatalog.precheckCancelRequest));
  const canReprintPrecheck = computed(() => Boolean(latestPrecheck.value && hasPermission(grantedPermissions.value, permissionCatalog.precheckReprint)));
  const canReprintCheck = computed(() => Boolean(finalCheckData.value && hasPermission(grantedPermissions.value, permissionCatalog.checkReprint)));
  const canReprintClosedCheck = computed(() => hasPermission(grantedPermissions.value, permissionCatalog.checkReprint));
  const canPayCash = computed(() => Boolean(currentCashSession.data.value && activePrecheck.value && paymentAmount.value > 0 && paymentAmount.value <= remainingPayment.value && hasPermission(grantedPermissions.value, permissionCatalog.paymentCash)));
  const canPayCard = computed(() => Boolean(currentCashSession.data.value && activePrecheck.value && paymentAmount.value > 0 && paymentAmount.value <= remainingPayment.value && hasPermission(grantedPermissions.value, permissionCatalog.paymentCardManual)));
  const paymentBlockedReasonKey = computed(() => {
    if (!activePrecheck.value) return '';
    if (!currentCashSession.data.value) return 'pos.openCashSessionToPay';
    return '';
  });
  const canSubmitCashDrawerEvent = computed(() => {
    if (!canRecordCashDrawerEvent.value) return false;
    if (cashDrawerType.value === 'no_sale') return true;
    return cashDrawerAmount.value >= 0;
  });
  const remainingPayment = computed(() => activePrecheck.value?.remaining_total ?? 0);
  const orderLoading = computed(() => activeOrdersQuery.isFetching.value || tableOrder.isFetching.value || order.isFetching.value);
  const statusError = computed(() => firstError([currentShift.error.value, currentCashSession.error.value]));
  const orderError = computed(() => firstError([activeOrdersQuery.error.value, tableOrder.error.value, order.error.value, prechecks.error.value]));
  const actorName = computed(() => auth.actor?.name || auth.actor?.employee_id || '');
  const syncProblems = computed(() => (syncStatus.data.value?.failed ?? 0) + (syncStatus.data.value?.suspended ?? 0));
  const closedOrdersHasPreviousPage = computed(() => closedOrdersOffset.value > 0);
  const closedOrdersHasNextPage = computed(() => (closedOrders.data.value?.length ?? 0) >= closedOrdersLimit.value);
  const primaryFlowSteps = computed(() => {
    const shiftReady = Boolean(currentShift.data.value && currentCashSession.data.value);
    const tableReady = Boolean(selectedTableId.value);
    const orderReady = Boolean(activeOrder.value);
    const precheckReady = Boolean(activePrecheck.value || latestPrecheck.value || finalCheckData.value);
    const paymentReady = Boolean(finalCheckData.value);
    return [
      flowStep('readiness', 1, shiftReady, !shiftReady),
      flowStep('table', 2, tableReady, shiftReady && !tableReady),
      flowStep('order', 3, orderReady, shiftReady && tableReady && !orderReady),
      flowStep('precheck', 4, precheckReady, orderReady && !precheckReady),
      flowStep('payment', 5, paymentReady, precheckReady && !paymentReady),
    ];
  });
  const currentBlockingNotice = computed<BlockingNotice | null>(() => {
    if (!currentShift.data.value) {
      return {
        titleKey: 'pos.blocking.noShift.title',
        reasonKey: canOpenShift.value ? 'pos.blocking.noShift.reason' : 'pos.blocking.noShift.permissionReason',
        permission: canOpenShift.value ? '' : permissionCatalog.employeeShiftOpen,
      };
    }
    if (!currentCashSession.data.value) {
      return {
        titleKey: 'pos.blocking.noCashSession.title',
        reasonKey: canOpenCashSession.value ? 'pos.blocking.noCashSession.reason' : 'pos.blocking.noCashSession.permissionReason',
        permission: canOpenCashSession.value ? '' : permissionCatalog.cashSessionOpen,
      };
    }
    if (activeOrder.value?.status === 'locked' || activePrecheck.value) {
      return {
        titleKey: 'pos.blocking.lockedOrder.title',
        reasonKey: 'pos.blocking.lockedOrder.reason',
        permission: canCancelPrecheck.value ? '' : permissionCatalog.precheckCancelRequest,
      };
    }
    return null;
  });

  const openShiftMutation = useMutation({
    mutationFn: openShift,
    onSuccess: refreshOps,
    onError: showBusinessError,
  });

  const closeShiftMutation = useMutation({
    mutationFn: closeShift,
    onSuccess: refreshOps,
    onError: showBusinessError,
  });

  const openCashMutation = useMutation({
    mutationFn: (amount: number) => openCashSession(moneyToMinor(amount, currency.value)),
    onSuccess: refreshOps,
    onError: showBusinessError,
  });

  const closeCashMutation = useMutation({
    mutationFn: (payload: { cashSessionId: string; amount: number }) => closeCashSession(payload.cashSessionId, moneyToMinor(payload.amount, currency.value)),
    onSuccess: refreshOps,
    onError: showBusinessError,
  });

  const cashDrawerMutation = useMutation({
    mutationFn: () => recordCashDrawerEvent(
      currentCashSession.data.value?.id ?? '',
      cashDrawerType.value,
      cashDrawerType.value === 'no_sale' ? 0 : moneyToMinor(cashDrawerAmount.value, currency.value),
      cashDrawerReason.value.trim(),
      cashDrawerNote.value.trim(),
    ),
    onSuccess() {
      cashDrawerDialog.value = false;
      cashDrawerAmount.value = 0;
      cashDrawerReason.value = '';
      cashDrawerNote.value = '';
      void refreshSync();
    },
    onError: showBusinessError,
  });

  const createOrderMutation = useMutation({
    mutationFn: () => createOrder(selectedTableId.value, selectedTable.value?.name ?? '', 1),
    onSuccess(result) {
      currentOrderId.value = result.id;
      void refreshOrder();
    },
    onError: showBusinessError,
  });

  const addLineMutation = useMutation({
    mutationFn: (payload: { menuItemId: string; selectedModifiers?: SelectedModifierPayload[] }) => addOrderLine(activeOrder.value?.id ?? '', payload.menuItemId, 1, payload.selectedModifiers ?? []),
    onSuccess(result) {
      selectedOrderLineId.value = result.id;
      closeModifierDialog();
      void refreshOrder();
    },
    onError: showBusinessError,
  });

  const modifierUpdateMutation = useMutation({
    mutationFn: (payload: { lineId: string; selectedModifiers: SelectedModifierPayload[] }) => updateOrderLineModifiers(activeOrder.value?.id ?? '', payload.lineId, payload.selectedModifiers),
    onSuccess(result) {
      selectedOrderLineId.value = result.id;
      closeModifierDialog();
      void refreshOrder();
    },
    onError: showBusinessError,
  });

  const quantityMutation = useMutation({
    mutationFn: (payload: { lineId: string; quantity: number }) => changeOrderLineQuantity(activeOrder.value?.id ?? '', payload.lineId, payload.quantity),
    onSuccess: refreshOrder,
    onError: showBusinessError,
  });

  const lineDetailsMutation = useMutation({
    mutationFn: () => updateOrderLineDetails(activeOrder.value?.id ?? '', selectedOrderLine.value?.id ?? '', lineCourseDraft.value, lineCommentDraft.value),
    onSuccess: refreshOrder,
    onError: showBusinessError,
  });

  const voidLineMutation = useMutation({
    mutationFn: (lineId: string) => voidOrderLine(activeOrder.value?.id ?? '', lineId, 'cashier_void'),
    onSuccess: refreshOrder,
    onError: showBusinessError,
  });

  const issuePrecheckMutation = useMutation({
    mutationFn: issuePrecheck,
    onSuccess: refreshOrder,
    onError: showBusinessError,
  });

  const reprintPrecheckMutation = useMutation({
    mutationFn: reprintPrecheck,
    onSuccess: showReprintReady,
    onError: showBusinessError,
  });

  const reprintCheckMutation = useMutation({
    mutationFn: reprintCheck,
    onSuccess: showReprintReady,
    onError: showBusinessError,
  });

  const cancelPrecheckMutation = useMutation({
    mutationFn: () => cancelPrecheck(activePrecheck.value?.id ?? '', managerEmployeeId.value.trim(), managerPin.value, cancelReason.value.trim()),
    onSuccess() {
      closeCancelDialog();
      void refreshOrder();
    },
    onSettled() {
      managerPin.value = '';
    },
    onError: showBusinessError,
  });

  const paymentMutation = useMutation({
    mutationFn: (method: 'cash' | 'card') => capturePrecheckPayment(activePrecheck.value?.id ?? '', method, moneyToMinor(paymentAmount.value, orderCurrency.value), orderCurrency.value),
    retry: false,
    onSuccess() {
      paymentAmount.value = 0;
      void refreshOrder();
    },
    onError: handlePaymentError,
  });

  const refundMutation = useMutation<unknown, unknown, void>({
    mutationFn: () => {
      const operationPayload = buildCheckLedgerPayload();
      if (refundMode.value === 'check_refund') {
        return recordCheckRefund(refundCheckId.value, operationPayload);
      }
      if (refundMode.value === 'check_cancellation') {
        return recordCheckCancellation(refundCheckId.value, operationPayload);
      }
      return refundPayment(refundPaymentId.value, { reason: refundReason.value });
    },
    onSuccess() {
      const messageKey = refundMode.value === 'check_cancellation' ? 'pos.cancellationSuccess' : 'pos.refundSuccess';
      closeRefundDialog();
      refreshCompensationState();
      $q.notify({
        message: t(messageKey),
        type: 'positive',
        position: 'top',
      });
    },
    onError: showBusinessError,
  });

  const retrySyncMutation = useMutation({
    mutationFn: retryFailedOutbox,
    onSuccess: refreshSync,
    onError: showBusinessError,
  });

  const logoutMutation = useMutation({
    mutationFn: logout,
    onSuccess() {
      auth.clearSession();
      void router.replace('/login');
    },
    onSettled() {
      queryClient.removeQueries({ queryKey: ['auth-session'] });
    },
    onError(error) {
      showBusinessError(error);
      if (!(error instanceof ApiError && (error.category === 'network' || error.category === 'timeout'))) {
        auth.clearSession();
        void router.replace('/login');
      }
    },
  });

  watch(pairing.error, (error) => {
    if (error) showBusinessError(error);
  });

  watch(session.error, (error) => {
    if (error) showBusinessError(error);
  });

  watch(pairing.data, (value) => {
    if (value) {
      auth.applyPairing(value);
      if (!value.paired) {
        void router.replace('/pair');
      }
    }
  }, { immediate: true });

  watch(session.data, (value) => {
    if (value) {
      auth.applySession(value.session, value.actor);
      if (value.session.status !== 'active') {
        void router.replace('/login');
      }
    }
  }, { immediate: true });

  watch(() => [auth.nodeDeviceId, auth.sessionId], () => {
    const fallback = resolveProtectedPosFallback({ nodeDeviceId: auth.nodeDeviceId, sessionId: auth.sessionId });
    if (fallback) {
      void router.replace(fallback);
    }
  }, { immediate: true });

  watch(activeHalls, (items) => {
    if (!selectedHallId.value && items[0]) {
      selectedHallId.value = items[0].id;
    }
  });

  watch(tableOrder.data, (value) => {
    // После оплаты current-order по столу становится пустым, но прямой order query еще нужен для финального чека.
    if (value?.id) {
      currentOrderId.value = value.id;
    }
  });

  watch(activeLines, (lines) => {
    if (lines.some((line) => line.id === selectedOrderLineId.value)) return;
    selectedOrderLineId.value = lines[0]?.id ?? '';
  });

  watch(remainingPayment, (value) => {
    paymentAmount.value = minorToMoney(value, orderCurrency.value);
  });

  watch(selectedRefundLine, () => {
    normalizeRefundLineQuantity();
  });

  watch(currentLedgerOperationKind, (kind) => {
    refundOperationKind.value = kind;
  }, { immediate: true });

  watch(closedOrdersBusinessDate, () => {
    closedOrdersOffset.value = 0;
  });

  function selectHall(id: string) {
    selectedHallId.value = id;
    selectedTableId.value = '';
    selectedOrderLineId.value = '';
    currentOrderId.value = '';
  }

  function selectTable(id: string) {
    selectedTableId.value = id;
    selectedOrderLineId.value = '';
    currentOrderId.value = '';
  }

  function selectOrderLine(id: string) {
    selectedOrderLineId.value = id;
    primeLineDetailsDraft();
  }

  function primeLineDetailsDraft() {
    lineCourseDraft.value = selectedOrderLine.value?.course ?? '';
    lineCommentDraft.value = selectedOrderLine.value?.comment ?? '';
  }

  function saveSelectedLineDetails() {
    if (!selectedOrderLine.value || !activeOrder.value) return;
    lineDetailsMutation.mutate();
  }

  function changeQuantity(lineId: string, quantity: number) {
    if (quantity < 1) return;
    quantityMutation.mutate({ lineId, quantity });
  }

  function voidLine(lineId: string) {
    voidLineMutation.mutate(lineId);
  }

  function openMenuItem(item: MenuItem) {
    const groups = item.modifier_groups.filter((group) => group.active && group.options.some((option) => option.active));
    if (groups.length === 0) {
      addLineMutation.mutate({ menuItemId: item.id });
      return;
    }
    modifierDialogMode.value = 'add';
    modifierLineId.value = '';
    modifierMenuItem.value = item;
    modifierQuantities.value = {};
    modifierValidationKey.value = '';
    modifierDialog.value = true;
  }

  function canEditLineModifiers(lineId: string) {
    const line = activeLines.value.find((item) => item.id === lineId);
    const item = line ? menu.data.value?.find((menuItem) => menuItem.id === line.menu_item_id) : null;
    return Boolean(item?.modifier_groups.some((group) => group.active && group.options.some((option) => option.active)));
  }

  function editLineModifiers(lineId: string) {
    const line = activeLines.value.find((item) => item.id === lineId);
    if (!line) return;
    const item = menu.data.value?.find((menuItem) => menuItem.id === line.menu_item_id);
    if (!item || item.modifier_groups.filter((group) => group.active && group.options.some((option) => option.active)).length === 0) {
      $q.notify({ type: 'warning', message: t('pos.modifierEditUnavailable') });
      return;
    }
    const quantities: Record<string, number> = {};
    for (const modifier of line.modifiers) {
      quantities[modifier.modifier_option_id] = modifier.quantity;
    }
    modifierDialogMode.value = 'edit';
    modifierLineId.value = line.id;
    modifierMenuItem.value = item;
    modifierQuantities.value = quantities;
    modifierValidationKey.value = '';
    modifierDialog.value = true;
  }

  function modifierGroupCount(groupId: string) {
    const group = modifierGroupsForDialog.value.find((item) => item.id === groupId);
    if (!group) return 0;
    return group.options.reduce((sum, option) => sum + (modifierQuantities.value[option.id] ?? 0), 0);
  }

  function changeModifierQuantity(optionId: string, nextQuantity: number) {
    modifierValidationKey.value = '';
    modifierQuantities.value = {
      ...modifierQuantities.value,
      [optionId]: Math.max(0, nextQuantity),
    };
  }

  function validateModifierSelection() {
    for (const group of modifierGroupsForDialog.value) {
      const count = modifierGroupCount(group.id);
      if (group.required && count === 0) return 'pos.modifierRequired';
      if (count < group.min_count) return 'pos.modifierMin';
      if (group.max_count > 0 && count > group.max_count) return 'pos.modifierMax';
    }
    return '';
  }

  function submitModifierSelection() {
    const validationKey = validateModifierSelection();
    if (validationKey) {
      modifierValidationKey.value = validationKey;
      return;
    }
    if (!modifierMenuItem.value) return;
    if (modifierDialogMode.value === 'edit') {
      if (!modifierLineId.value) return;
      modifierUpdateMutation.mutate({ lineId: modifierLineId.value, selectedModifiers: selectedModifierPayload.value });
      return;
    }
    addLineMutation.mutate({ menuItemId: modifierMenuItem.value.id, selectedModifiers: selectedModifierPayload.value });
  }

  function closeModifierDialog() {
    modifierDialog.value = false;
    modifierDialogMode.value = 'add';
    modifierLineId.value = '';
    modifierMenuItem.value = null;
    modifierQuantities.value = {};
    modifierValidationKey.value = '';
  }

  function pay(method: 'cash' | 'card') {
    paymentMutation.mutate(method);
  }

  function handlePaymentError(error: unknown) {
    if (error instanceof ApiError && error.status === 409) {
      invalidatePaymentConflictQueries(queryClient);
    }
    showBusinessError(error);
  }

  function showReprintReady() {
    $q.notify({
      type: 'positive',
      message: `${t('pos.reprintReady')} · ${t('pos.reprintCopy')}`,
    });
  }

  function submitCancelPrecheck() {
    if (!managerEmployeeId.value.trim() || !managerPin.value || !cancelReason.value.trim()) return;
    cancelPrecheckMutation.mutate();
  }

  function closeCancelDialog() {
    cancelDialog.value = false;
    managerEmployeeId.value = '';
    managerPin.value = '';
    cancelReason.value = '';
  }

  function openRefundDialog(paymentId: string) {
    refundMode.value = 'payment_refund';
    refundPaymentId.value = paymentId;
    refundCheckId.value = '';
    refundOrder.value = null;
    refundReason.value = '';
    refundInventoryDisposition.value = 'no_stock_effect';
    refundOperationKind.value = 'partial';
    refundScope.value = 'whole_check';
    refundOrderLines.value = [];
    refundOrderLineId.value = '';
    refundLineQuantity.value = 1;
    refundDialog.value = true;
  }

  function openRefundDialogForOrder(orderItem: ClosedOrder) {
    const payment = orderItem.check?.payments?.find((item) => item.status === 'captured');
    if (payment) openRefundDialog(payment.id);
  }

  function openCheckRefundDialogForOrder(orderItem: ClosedOrder) {
    if (!orderItem.check) return;
    refundMode.value = 'check_refund';
    refundPaymentId.value = '';
    refundCheckId.value = orderItem.check.id;
    refundOrder.value = orderItem;
    refundReason.value = '';
    refundInventoryDisposition.value = 'no_stock_effect';
    refundOperationKind.value = 'full';
    primeLedgerScope(orderItem);
    refundDialog.value = true;
  }

  function openCheckCancellationDialogForOrder(orderItem: ClosedOrder) {
    if (!orderItem.check) return;
    refundMode.value = 'check_cancellation';
    refundPaymentId.value = '';
    refundCheckId.value = orderItem.check.id;
    refundOrder.value = orderItem;
    refundReason.value = '';
    refundInventoryDisposition.value = 'no_stock_effect';
    refundOperationKind.value = 'full';
    primeLedgerScope(orderItem);
    refundDialog.value = true;
  }

  function closeRefundDialog() {
    refundDialog.value = false;
    refundMode.value = 'payment_refund';
    refundPaymentId.value = '';
    refundCheckId.value = '';
    refundOrder.value = null;
    refundReason.value = '';
    refundInventoryDisposition.value = 'no_stock_effect';
    refundOperationKind.value = 'full';
    refundScope.value = 'whole_check';
    refundOrderLines.value = [];
    refundOrderLineId.value = '';
    refundLineQuantity.value = 1;
  }

  function closedOrderCompensationContext(): ClosedOrderCompensationContext {
    return {
      canRefundPayment: canRefundPayment.value,
      canRecordCheckCancellation: canRecordCheckCancellation.value,
      currentCashSessionShiftId: currentCashSession.data.value?.shift_id ?? '',
    };
  }

  function closedOrderCancellationUnavailableKey(orderItem: ClosedOrder) {
    return closedOrderCompensationUnavailableKey(orderItem, 'check_cancellation', closedOrderCompensationContext());
  }

  function closedOrderRefundUnavailableKey(orderItem: ClosedOrder) {
    return closedOrderCompensationUnavailableKey(orderItem, 'check_refund', closedOrderCompensationContext());
  }

  function closedOrderPaymentRefundUnavailableKey(orderItem: ClosedOrder) {
    return closedOrderCompensationUnavailableKey(orderItem, 'payment_refund', closedOrderCompensationContext());
  }

  function canCancelClosedOrder(orderItem: ClosedOrder) {
    return closedOrderCancellationUnavailableKey(orderItem) === '';
  }

  function canRefundClosedOrder(orderItem: ClosedOrder) {
    return closedOrderRefundUnavailableKey(orderItem) === '';
  }

  function canRefundPaymentForOrder(orderItem: ClosedOrder) {
    return closedOrderPaymentRefundUnavailableKey(orderItem) === '';
  }

  function refundDialogTitleKey() {
    if (refundMode.value === 'check_cancellation') return 'pos.checkCancellation';
    if (refundMode.value === 'check_refund') return 'pos.checkRefund';
    return 'pos.paymentRefund';
  }

  function refundDialogCopyKey() {
    if (refundMode.value === 'check_cancellation') return 'pos.checkCancellationCopy';
    if (refundMode.value === 'check_refund') return 'pos.checkRefundCopy';
    return 'pos.paymentRefundCopy';
  }

  function refundDialogSubmitKey() {
    if (refundMode.value === 'check_cancellation') return 'pos.recordCheckCancellation';
    if (refundMode.value === 'check_refund') return 'pos.recordCheckRefund';
    return 'pos.recordPaymentRefund';
  }

  function refundDialogIcon() {
    return refundMode.value === 'check_cancellation' ? 'cancel' : 'undo';
  }

  function refundDialogShowsLedgerControls() {
    return refundMode.value === 'check_cancellation' || refundMode.value === 'check_refund';
  }

  function buildCheckLedgerPayload() {
    const items = refundScope.value === 'order_line' ? buildOrderLineLedgerItems() : undefined;
    refundOperationKind.value = currentLedgerOperationKind.value;
    return {
      operationKind: currentLedgerOperationKind.value,
      inventoryDisposition: refundInventoryDisposition.value,
      reason: refundReason.value,
      items,
    };
  }

  function buildOrderLineLedgerItems(): FinancialOperationItemPayload[] | undefined {
    const line = selectedRefundLine.value;
    if (!line) return undefined;
    const quantity = clampQuantity(refundLineQuantity.value, line.quantity);
    return [{
      scope: 'order_line',
      orderLineId: line.order_line_id,
      quantity,
      amount: proportionalMinor(line.total_minor, line.quantity, quantity),
      currency: line.currency_code,
      taxAmount: proportionalMinor(line.tax_total_minor, line.quantity, quantity),
    }];
  }

  function primeLedgerScope(orderItem: ClosedOrder) {
    refundScope.value = 'whole_check';
    refundOrderLines.value = extractCheckSnapshotLines(orderItem);
    refundOrderLineId.value = refundOrderLines.value[0]?.order_line_id ?? '';
    refundLineQuantity.value = 1;
  }

  function extractCheckSnapshotLines(orderItem: ClosedOrder) {
    const parsed = checkSnapshotSchema.safeParse(orderItem.check?.snapshot);
    if (!parsed.success) return [];
    return parsed.data.precheck_snapshot?.lines.filter((line) => line.quantity > 0 && line.total_minor > 0 && line.order_line_id.trim()) ?? [];
  }

  function clampQuantity(value: number, max: number) {
    if (!Number.isFinite(value)) return 1;
    return Math.min(Math.max(1, Math.trunc(value)), Math.max(1, max));
  }

  function proportionalMinor(total: number, quantity: number, selectedQuantity: number) {
    if (quantity <= 0 || selectedQuantity >= quantity) return total;
    return Math.round((total * clampQuantity(selectedQuantity, quantity)) / quantity);
  }

  function normalizeRefundLineQuantity() {
    refundLineQuantity.value = clampQuantity(refundLineQuantity.value, maxRefundLineQuantity.value);
  }

  function previousClosedOrdersPage() {
    closedOrdersOffset.value = Math.max(0, closedOrdersOffset.value - closedOrdersLimit.value);
  }

  function nextClosedOrdersPage() {
    if (!closedOrdersHasNextPage.value) return;
    closedOrdersOffset.value += closedOrdersLimit.value;
  }

  function refreshCompensationState() {
    void queryClient.invalidateQueries({ queryKey: ['closed-orders'] });
    void queryClient.invalidateQueries({ queryKey: ['order'] });
    void queryClient.invalidateQueries({ queryKey: ['check'] });
    void queryClient.invalidateQueries({ queryKey: ['sync-outbox'] });
    void queryClient.invalidateQueries({ queryKey: ['sync-status'] });
    void queryClient.invalidateQueries({ queryKey: ['local-events'] });
  }

  function refreshOps() {
    void queryClient.invalidateQueries({ queryKey: ['current-shift'] });
    void queryClient.invalidateQueries({ queryKey: ['recent-shifts'] });
    void queryClient.invalidateQueries({ queryKey: ['current-cash-session'] });
  }

  function refreshOrder() {
    void queryClient.invalidateQueries({ queryKey: ['active-orders'] });
    void queryClient.invalidateQueries({ queryKey: ['current-order'] });
    void queryClient.invalidateQueries({ queryKey: ['order'] });
    void queryClient.invalidateQueries({ queryKey: ['prechecks'] });
    void queryClient.invalidateQueries({ queryKey: ['check'] });
  }

  function refreshSync() {
    void queryClient.invalidateQueries({ queryKey: ['sync-status'] });
    void queryClient.invalidateQueries({ queryKey: ['sync-outbox'] });
    void queryClient.invalidateQueries({ queryKey: ['local-events'] });
  }

  function refetchMenu() {
    void menu.refetch();
  }

  function lockTerminal() {
    void router.push('/lock');
  }

  function money(value: number, code: string) {
    return formatMinorCurrency(value, code, 'ru-RU');
  }

  function formatDate(value: string) {
    return new Intl.DateTimeFormat('ru-RU', { hour: '2-digit', minute: '2-digit' }).format(new Date(value));
  }

  function shortId(value: string) {
    return value.length > 10 ? `${value.slice(0, 8)}...` : value;
  }

  function statusLabel(status: string) {
    return t(`status.${status}`);
  }

  function flowStep(key: string, index: number, ready: boolean, active: boolean) {
    let state: FlowStepState = 'pending';
    if (ready) state = 'ready';
    else if (active) state = 'active';
    else if (currentBlockingNotice.value) state = 'blocked';
    return {
      key,
      index,
      state,
      titleKey: `pos.primaryFlow.steps.${key}.title`,
      descriptionKey: `pos.primaryFlow.steps.${key}.description`,
    };
  }

  function actionBlocker(permission: string, allowed: boolean) {
    if (allowed) return null;
    if (!currentShift.data.value) {
      return currentBlockingNotice.value ?? {
        titleKey: 'pos.blocking.noShift.title',
        reasonKey: 'pos.blocking.noShift.permissionReason',
        permission: permissionCatalog.employeeShiftOpen,
      };
    }
    const cashSessionPermissions: string[] = [permissionCatalog.paymentCash, permissionCatalog.paymentCardManual, permissionCatalog.cashDrawerRecordEvent, permissionCatalog.paymentRefund];
    if (!currentCashSession.data.value && cashSessionPermissions.includes(permission)) {
      return currentBlockingNotice.value ?? {
        titleKey: 'pos.blocking.noCashSession.title',
        reasonKey: 'pos.blocking.noCashSession.permissionReason',
        permission: permissionCatalog.cashSessionOpen,
      };
    }
    if (activeOrder.value?.status === 'locked' || activePrecheck.value) {
      return { titleKey: 'pos.blocking.lockedOrder.title', reasonKey: 'pos.blocking.lockedOrder.reason', permission };
    }
    return {
      titleKey: 'pos.blocking.permissionDenied.title',
      reasonKey: 'pos.blocking.permissionDenied.reason',
      permission,
    };
  }

  function firstError(errors: unknown[]) {
    const found = errors.find(Boolean);
    return found ? t(displayErrorMessageKey(found)) : '';
  }

  return {
    t,
    auth,
    selectedHallId,
    selectedTableId,
    selectedOrderLineId,
    openingCashAmount,
    closingCashAmount,
    paymentAmount,
    menuSearch,
    cashDrawerType,
    cashDrawerAmount,
    cashDrawerReason,
    cashDrawerNote,
    localEventFilter,
    cancelDialog,
    managerEmployeeId,
    managerPin,
    cancelReason,
    closedOrdersDrawer,
    cashDrawerDialog,
    syncDrawer,
    refundDialog,
    refundMode,
    refundReason,
    refundInventoryDisposition,
    refundOperationKind,
    refundScope,
    refundOrderLineId,
    refundLineQuantity,
    closedOrdersLimit,
    closedOrdersOffset,
    closedOrdersBusinessDate,
    modifierDialog,
    modifierDialogMode,
    modifierLineId,
    modifierMenuItem,
    modifierQuantities,
    modifierValidationKey,
    lineCourseDraft,
    lineCommentDraft,
    canOpenShift,
    canCloseShift,
    canOpenCashSession,
    canCloseCashSession,
    canRecordCashDrawerEvent,
    canViewSync,
    canRetrySync,
    canViewClosedOrders,
    canRefundPayment,
    canRecordCheckCancellation,
    canCreateOrder,
    canAddOrderLine,
    canChangeOrderLine,
    canVoidOrderLine,
    canIssuePrecheck,
    canCancelPrecheck,
    canReprintPrecheck,
    canReprintCheck,
    canReprintClosedCheck,
    canPayCash,
    canPayCard,
    canSubmitCashDrawerEvent,
    paymentBlockedReasonKey,
    cashDrawerTypeOptions,
    inventoryDispositionOptions,
    ledgerScopeOptions,
    refundLineOptions,
    selectedRefundLine,
    maxRefundLineQuantity,
    refundLineAmount,
    refundLineTaxAmount,
    currentLedgerOperationKind,
    unsupportedLedgerScopeOptions,
    pairing,
    currentShift,
    recentShifts,
    currentCashSession,
    halls,
    tables,
    activeOrdersQuery,
    menu,
    syncStatus,
    syncOutbox,
    localEvents,
    closedOrders,
    activeHalls,
    activeTables,
    selectedTable,
    activeOrders,
    activeOrder,
    activeLines,
    selectedOrderLine,
    activeMenuItems,
    regularMenuItems,
    serviceMenuItems,
    visibleMenuItems,
    visibleServiceItems,
    modifierGroupsForDialog,
    selectedModifierTotal,
    canSubmitModifierSelection,
    activePrecheck,
    latestPrecheck,
    finalCheckData,
    currency,
    orderCurrency,
    remainingPayment,
    orderLoading,
    statusError,
    orderError,
    actorName,
    syncProblems,
    closedOrdersHasPreviousPage,
    closedOrdersHasNextPage,
    primaryFlowSteps,
    currentBlockingNotice,
    openShiftMutation,
    closeShiftMutation,
    openCashMutation,
    closeCashMutation,
    cashDrawerMutation,
    createOrderMutation,
    addLineMutation,
    modifierUpdateMutation,
    issuePrecheckMutation,
    reprintPrecheckMutation,
    reprintCheckMutation,
    cancelPrecheckMutation,
    paymentMutation,
    refundMutation,
    lineDetailsMutation,
    retrySyncMutation,
    logoutMutation,
    selectHall,
    selectTable,
    selectOrderLine,
    primeLineDetailsDraft,
    saveSelectedLineDetails,
    openMenuItem,
    canEditLineModifiers,
    editLineModifiers,
    modifierGroupCount,
    changeModifierQuantity,
    submitModifierSelection,
    closeModifierDialog,
    changeQuantity,
    voidLine,
    pay,
    submitCancelPrecheck,
    closeCancelDialog,
    openRefundDialogForOrder,
    openCheckRefundDialogForOrder,
    openCheckCancellationDialogForOrder,
    closeRefundDialog,
    canCancelClosedOrder,
    canRefundClosedOrder,
    canRefundPaymentForOrder,
    closedOrderCancellationUnavailableKey,
    closedOrderRefundUnavailableKey,
    closedOrderPaymentRefundUnavailableKey,
    refundDialogTitleKey,
    refundDialogCopyKey,
    refundDialogSubmitKey,
    refundDialogIcon,
    refundDialogShowsLedgerControls,
    normalizeRefundLineQuantity,
    previousClosedOrdersPage,
    nextClosedOrdersPage,
    refreshOps,
    refreshSync,
    refetchMenu,
    lockTerminal,
    money,
    formatDate,
    shortId,
    statusLabel,
    actionBlocker,
    currencyInputStep,
    displayErrorMessageKey,
  };
}

export type CashierTerminal = ReturnType<typeof useCashierTerminal>;
