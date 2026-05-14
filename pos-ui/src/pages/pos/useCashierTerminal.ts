import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query';
import { useQuasar } from 'quasar';
import { computed, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRouter } from 'vue-router';

import {
  addOrderLine,
  cancelPrecheck,
  capturePrecheckPayment,
  changeOrderLineQuantity,
  closeCashSession,
  closeOrder,
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
  listClosedOrders,
  listHalls,
  listLocalEvents,
  listMenuItems,
  listPrechecksByOrder,
  listRecentShifts,
  listSyncOutbox,
  listTables,
  openCashSession,
  openShift,
  recordCashDrawerEvent,
  refundPayment,
  reprintCheck,
  reprintPrecheck,
  retryFailedOutbox,
  voidOrderLine,
  type CashDrawerEventType,
} from '../../shared/api';
import { currencyInputStep, formatMinorCurrency, minorToMoney, moneyToMinor } from '../../shared/currency';
import { displayErrorMessageKey, useErrorHandling } from '../../shared/errorHandling';
import { hasPermission, permissionCatalog } from '../../shared/rbac';
import { resolveProtectedPosFallback } from '../../shared/sessionGuards';
import type { ClosedOrder, Order } from '../../shared/schemas';
import { useAuthStore } from '../../stores/auth';

export function useCashierTerminal() {
  const { t } = useI18n();
  const auth = useAuthStore();
  const router = useRouter();
  const $q = useQuasar();
  const queryClient = useQueryClient();
  const { showBusinessError } = useErrorHandling();

  const selectedHallId = ref('');
  const selectedTableId = ref('');
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
  const refundPaymentId = ref('');
  const refundReason = ref('');

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

  const cashDrawerTypeOptions = computed(() => [
    { label: t('pos.cashDrawerTypes.no_sale'), value: 'no_sale' },
    { label: t('pos.cashDrawerTypes.cash_in'), value: 'cash_in' },
    { label: t('pos.cashDrawerTypes.cash_out'), value: 'cash_out' },
    { label: t('pos.cashDrawerTypes.cash_count'), value: 'cash_count' },
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
  const activeOrder = computed<Order | null>(() => order.data.value ?? tableOrder.data.value ?? null);
  const finalCheckId = computed(() => activeOrder.value?.check?.id ?? '');
  const finalCheck = useQuery({
    queryKey: ['check', finalCheckId],
    queryFn: () => getCheck(finalCheckId.value),
    enabled: () => Boolean(finalCheckId.value && auth.sessionId && currentShift.data.value && hasPermission(grantedPermissions.value, permissionCatalog.checkView)),
  });
  const activeLines = computed(() => activeOrder.value?.lines.filter((line) => line.status === 'active') ?? []);
  const activeMenuItems = computed(() => menu.data.value?.filter((item) => item.active) ?? []);
  const visibleMenuItems = computed(() => {
    const query = menuSearch.value.trim().toLocaleLowerCase('ru-RU');
    if (!query) return activeMenuItems.value;
    return activeMenuItems.value.filter((item) => item.name.toLocaleLowerCase('ru-RU').includes(query));
  });

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
    queryKey: ['closed-orders'],
    queryFn: () => listClosedOrders(50),
    enabled: () => Boolean(auth.nodeDeviceId && auth.sessionId && canViewClosedOrders.value && closedOrdersDrawer.value),
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
  const canPayCash = computed(() => Boolean(currentCashSession.data.value && activePrecheck.value && paymentAmount.value > 0 && paymentAmount.value <= remainingPayment.value && hasPermission(grantedPermissions.value, permissionCatalog.paymentCash)));
  const canPayCard = computed(() => Boolean(currentCashSession.data.value && activePrecheck.value && paymentAmount.value > 0 && paymentAmount.value <= remainingPayment.value && hasPermission(grantedPermissions.value, permissionCatalog.paymentCardManual)));
  const canCloseOrder = computed(() => Boolean(activeOrder.value?.status === 'open' && finalCheckData.value?.paid_total === finalCheckData.value?.total && hasPermission(grantedPermissions.value, permissionCatalog.orderClose)));
  const canSubmitCashDrawerEvent = computed(() => {
    if (!canRecordCashDrawerEvent.value) return false;
    if (cashDrawerType.value === 'no_sale') return true;
    return cashDrawerAmount.value >= 0;
  });
  const remainingPayment = computed(() => activePrecheck.value?.remaining_total ?? 0);
  const orderLoading = computed(() => tableOrder.isFetching.value || order.isFetching.value);
  const statusError = computed(() => firstError([currentShift.error.value, currentCashSession.error.value]));
  const orderError = computed(() => firstError([tableOrder.error.value, order.error.value, prechecks.error.value]));
  const actorName = computed(() => auth.actor?.name || auth.actor?.employee_id || '');
  const syncProblems = computed(() => (syncStatus.data.value?.failed ?? 0) + (syncStatus.data.value?.suspended ?? 0));

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
    mutationFn: (menuItemId: string) => addOrderLine(activeOrder.value?.id ?? '', menuItemId, 1),
    onSuccess: refreshOrder,
    onError: showBusinessError,
  });

  const quantityMutation = useMutation({
    mutationFn: (payload: { lineId: string; quantity: number }) => changeOrderLineQuantity(activeOrder.value?.id ?? '', payload.lineId, payload.quantity),
    onSuccess: refreshOrder,
    onError: showBusinessError,
  });

  const voidLineMutation = useMutation({
    mutationFn: (lineId: string) => voidOrderLine(activeOrder.value?.id ?? '', lineId, 'cashier_void'),
    onSuccess: refreshOrder,
    onError: showBusinessError,
  });

  const closeOrderMutation = useMutation({
    mutationFn: closeOrder,
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
    onSuccess() {
      paymentAmount.value = 0;
      void refreshOrder();
    },
    onError: showBusinessError,
  });

  const refundMutation = useMutation({
    mutationFn: () => refundPayment(refundPaymentId.value, refundReason.value),
    onSuccess() {
      refundDialog.value = false;
      refundPaymentId.value = '';
      refundReason.value = '';
      void queryClient.invalidateQueries({ queryKey: ['closed-orders'] });
      void queryClient.invalidateQueries({ queryKey: ['sync-outbox'] });
      void queryClient.invalidateQueries({ queryKey: ['sync-status'] });
      $q.notify({
        message: t('pos.refundSuccess'),
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

  watch(remainingPayment, (value) => {
    paymentAmount.value = minorToMoney(value, orderCurrency.value);
  });

  function selectHall(id: string) {
    selectedHallId.value = id;
    selectedTableId.value = '';
    currentOrderId.value = '';
  }

  function selectTable(id: string) {
    selectedTableId.value = id;
    currentOrderId.value = '';
  }

  function changeQuantity(lineId: string, quantity: number) {
    if (quantity < 1) return;
    quantityMutation.mutate({ lineId, quantity });
  }

  function voidLine(lineId: string) {
    voidLineMutation.mutate(lineId);
  }

  function pay(method: 'cash' | 'card') {
    paymentMutation.mutate(method);
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
    refundPaymentId.value = paymentId;
    refundReason.value = '';
    refundDialog.value = true;
  }

  function openRefundDialogForOrder(orderItem: ClosedOrder) {
    const payment = orderItem.check?.payments?.find((item) => item.status === 'captured');
    if (payment) openRefundDialog(payment.id);
  }

  function refreshOps() {
    void queryClient.invalidateQueries({ queryKey: ['current-shift'] });
    void queryClient.invalidateQueries({ queryKey: ['recent-shifts'] });
    void queryClient.invalidateQueries({ queryKey: ['current-cash-session'] });
  }

  function refreshOrder() {
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

  function firstError(errors: unknown[]) {
    const found = errors.find(Boolean);
    return found ? t(displayErrorMessageKey(found)) : '';
  }

  return {
    t,
    auth,
    selectedHallId,
    selectedTableId,
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
    refundReason,
    canOpenShift,
    canCloseShift,
    canOpenCashSession,
    canCloseCashSession,
    canRecordCashDrawerEvent,
    canViewSync,
    canRetrySync,
    canViewClosedOrders,
    canRefundPayment,
    canCreateOrder,
    canAddOrderLine,
    canChangeOrderLine,
    canVoidOrderLine,
    canIssuePrecheck,
    canCancelPrecheck,
    canReprintPrecheck,
    canReprintCheck,
    canPayCash,
    canPayCard,
    canCloseOrder,
    canSubmitCashDrawerEvent,
    cashDrawerTypeOptions,
    pairing,
    currentShift,
    recentShifts,
    currentCashSession,
    halls,
    tables,
    menu,
    syncStatus,
    syncOutbox,
    localEvents,
    closedOrders,
    activeHalls,
    activeTables,
    selectedTable,
    activeOrder,
    activeLines,
    activeMenuItems,
    visibleMenuItems,
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
    openShiftMutation,
    closeShiftMutation,
    openCashMutation,
    closeCashMutation,
    cashDrawerMutation,
    createOrderMutation,
    addLineMutation,
    closeOrderMutation,
    issuePrecheckMutation,
    reprintPrecheckMutation,
    reprintCheckMutation,
    cancelPrecheckMutation,
    paymentMutation,
    refundMutation,
    retrySyncMutation,
    selectHall,
    selectTable,
    changeQuantity,
    voidLine,
    pay,
    submitCancelPrecheck,
    closeCancelDialog,
    openRefundDialogForOrder,
    refreshOps,
    refreshSync,
    refetchMenu,
    lockTerminal,
    money,
    formatDate,
    shortId,
    statusLabel,
    currencyInputStep,
    displayErrorMessageKey,
  };
}

export type CashierTerminal = ReturnType<typeof useCashierTerminal>;
