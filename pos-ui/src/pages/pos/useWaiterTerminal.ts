import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query';
import { useQuasar } from 'quasar';
import { computed, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRouter } from 'vue-router';

import {
  addOrderLine,
  ApiError,
  changeOrderLineQuantity,
  createOrder,
  getAuthSession,
  getCurrentOrderByTable,
  getCurrentShift,
  getOrder,
  getPairingStatus,
  issuePrecheck,
  listActiveOrdersByHall,
  listHalls,
  listMenuItems,
  listPrechecksByOrder,
  listTables,
  openShift,
  reprintPrecheck,
  type SelectedModifierPayload,
  voidOrderLine,
} from '../../shared/api';
import { formatMinorCurrency } from '../../shared/currency';
import { displayErrorMessageKey, displayErrorSupportCode, useErrorHandling } from '../../shared/errorHandling';
import { hasPermission, permissionCatalog } from '../../shared/rbac';
import { resolveProtectedPosFallback } from '../../shared/sessionGuards';
import type { MenuItem, Order } from '../../shared/schemas';
import { useAuthStore } from '../../stores/auth';

export function useWaiterTerminal() {
  const { t } = useI18n();
  const auth = useAuthStore();
  const router = useRouter();
  const queryClient = useQueryClient();
  const { showBusinessError } = useErrorHandling();
  const $q = useQuasar();

  const selectedHallId = ref('');
  const selectedTableId = ref('');
  const currentOrderId = ref('');
  const selectedOrderLineId = ref('');
  const menuSearch = ref('');
  const modifierDialog = ref(false);
  const modifierMenuItem = ref<MenuItem | null>(null);
  const modifierQuantities = ref<Record<string, number>>({});
  const modifierValidationKey = ref('');

  const grantedPermissions = computed(() => auth.actor?.permissions ?? []);
  const canViewFloor = computed(() => hasPermission(grantedPermissions.value, permissionCatalog.floorView));
  const canViewMenu = computed(() => hasPermission(grantedPermissions.value, permissionCatalog.menuView));
  const canViewCurrentShift = computed(() => hasPermission(grantedPermissions.value, permissionCatalog.employeeShiftViewCurrent));
  const canOpenShift = computed(() => hasPermission(grantedPermissions.value, permissionCatalog.employeeShiftOpen));
  const canViewOrder = computed(() => hasPermission(grantedPermissions.value, permissionCatalog.orderView));
  const canCreateOrderPermission = computed(() => hasPermission(grantedPermissions.value, permissionCatalog.orderCreate));
  const canAddLinePermission = computed(() => hasPermission(grantedPermissions.value, permissionCatalog.orderAddLine));
  const canChangeLinePermission = computed(() => hasPermission(grantedPermissions.value, permissionCatalog.orderChangeQuantity));
  const canVoidLinePermission = computed(() => hasPermission(grantedPermissions.value, permissionCatalog.orderVoidLine));
  const canIssuePrecheckPermission = computed(() => hasPermission(grantedPermissions.value, permissionCatalog.precheckIssue));
  const canViewPrecheck = computed(() => hasPermission(grantedPermissions.value, permissionCatalog.precheckView));
  const canReprintPrecheckPermission = computed(() => hasPermission(grantedPermissions.value, permissionCatalog.precheckReprint));

  const pairing = useQuery({
    queryKey: ['waiter-pairing-status'],
    queryFn: getPairingStatus,
  });

  const session = useQuery({
    queryKey: ['waiter-auth-session', auth.sessionId, auth.nodeDeviceId, auth.clientDeviceId],
    queryFn: getAuthSession,
    enabled: () => Boolean(auth.sessionId && auth.nodeDeviceId),
    retry: false,
  });

  const currentShift = useQuery({
    queryKey: ['waiter-current-shift', auth.nodeDeviceId],
    queryFn: getCurrentShift,
    enabled: () => Boolean(auth.nodeDeviceId && auth.sessionId && canViewCurrentShift.value),
  });

  const halls = useQuery({
    queryKey: ['waiter-halls', auth.restaurantId],
    queryFn: () => listHalls(auth.restaurantId),
    enabled: () => Boolean(auth.restaurantId && auth.sessionId && currentShift.data.value && canViewFloor.value),
  });

  const activeHallId = computed(() => selectedHallId.value || halls.data.value?.find((hall) => hall.active)?.id || '');

  const tables = useQuery({
    queryKey: ['waiter-tables', auth.restaurantId, activeHallId],
    queryFn: () => listTables(auth.restaurantId, activeHallId.value),
    enabled: () => Boolean(auth.restaurantId && activeHallId.value && auth.sessionId && currentShift.data.value && canViewFloor.value),
  });

  const activeOrdersQuery = useQuery({
    queryKey: ['waiter-active-orders', activeHallId],
    queryFn: () => listActiveOrdersByHall(activeHallId.value),
    enabled: () => Boolean(activeHallId.value && auth.sessionId && currentShift.data.value && canViewOrder.value),
  });

  const tableOrder = useQuery({
    queryKey: ['waiter-current-order', selectedTableId],
    queryFn: () => getCurrentOrderByTable(selectedTableId.value),
    enabled: () => Boolean(selectedTableId.value && auth.sessionId && currentShift.data.value && canViewOrder.value),
  });

  const order = useQuery({
    queryKey: ['waiter-order', currentOrderId],
    queryFn: () => getOrder(currentOrderId.value),
    enabled: () => Boolean(currentOrderId.value && auth.sessionId && currentShift.data.value && canViewOrder.value),
  });

  const prechecks = useQuery({
    queryKey: ['waiter-prechecks', currentOrderId],
    queryFn: () => listPrechecksByOrder(currentOrderId.value),
    enabled: () => Boolean(currentOrderId.value && auth.sessionId && currentShift.data.value && canViewPrecheck.value),
  });

  const menu = useQuery({
    queryKey: ['waiter-menu-items'],
    queryFn: listMenuItems,
    enabled: () => Boolean(auth.sessionId && currentShift.data.value && canViewMenu.value),
  });

  const activeHalls = computed(() => halls.data.value?.filter((hall) => hall.active) ?? []);
  const activeTables = computed(() => tables.data.value?.filter((table) => table.active) ?? []);
  const activeOrders = computed(() => activeOrdersQuery.data.value ?? []);
  const selectedTable = computed(() => activeTables.value.find((table) => table.id === selectedTableId.value));
  const activeOrder = computed<Order | null>(() => order.data.value ?? tableOrder.data.value ?? activeOrders.value.find((item) => item.table_id === selectedTableId.value) ?? null);
  const activeLines = computed(() => activeOrder.value?.lines.filter((line) => line.status === 'active') ?? []);
  const selectedOrderLine = computed(() => activeLines.value.find((line) => line.id === selectedOrderLineId.value) ?? activeLines.value[0] ?? null);
  const activeMenuItems = computed(() => menu.data.value?.filter((item) => item.active && item.item_type !== 'service') ?? []);
  const visibleMenuItems = computed(() => {
    const query = menuSearch.value.trim().toLocaleLowerCase('ru-RU');
    if (!query) return activeMenuItems.value;
    return activeMenuItems.value.filter((item) => item.name.toLocaleLowerCase('ru-RU').includes(query));
  });
  const activePrecheck = computed(() => prechecks.data.value?.find((item) => item.status === 'issued') ?? null);
  const orderIsLocked = computed(() => Boolean(activePrecheck.value || activeOrder.value?.status === 'locked' || activeOrder.value?.check));
  const latestPrecheck = computed(() => {
    const items = prechecks.data.value ?? [];
    return items.reduce((latest, item) => (latest === null || item.version > latest.version ? item : latest), null as (typeof items)[number] | null);
  });
  const orderCurrency = computed(() => activeMenuItems.value.find((item) => activeLines.value.some((line) => line.menu_item_id === item.id))?.currency ?? activeMenuItems.value[0]?.currency ?? 'RUB');
  const modifierGroupsForDialog = computed(() => modifierMenuItem.value?.modifier_groups.filter((group) => group.active && group.options.some((option) => option.active)) ?? []);
  const selectedModifierPayload = computed<SelectedModifierPayload[]>(() => {
    const payload: SelectedModifierPayload[] = [];
    for (const group of modifierGroupsForDialog.value) {
      for (const option of group.options) {
        const quantity = modifierQuantities.value[option.id] ?? 0;
        if (quantity > 0) payload.push({ modifier_group_id: group.id, modifier_option_id: option.id, quantity });
      }
    }
    return payload;
  });
  const canCreateOrder = computed(() => Boolean(selectedTableId.value && currentShift.data.value && !activeOrder.value && canCreateOrderPermission.value));
  const canAddOrderLine = computed(() => Boolean(activeOrder.value?.status === 'open' && !orderIsLocked.value && canAddLinePermission.value));
  const canChangeOrderLine = computed(() => Boolean(activeOrder.value?.status === 'open' && !orderIsLocked.value && canChangeLinePermission.value));
  const canVoidOrderLine = computed(() => Boolean(activeOrder.value?.status === 'open' && !orderIsLocked.value && canVoidLinePermission.value));
  const canIssuePrecheck = computed(() => Boolean(activeOrder.value?.status === 'open' && activeLines.value.length > 0 && !activePrecheck.value && canIssuePrecheckPermission.value));
  const canReprintPrecheck = computed(() => Boolean(latestPrecheck.value && canReprintPrecheckPermission.value));
  const canSubmitModifierSelection = computed(() => Boolean(canAddOrderLine.value && modifierMenuItem.value && !addLineMutation.isPending.value));
  const statusError = computed(() => firstError([currentShift.error.value, halls.error.value, tables.error.value]));
  const orderError = computed(() => firstError([activeOrdersQuery.error.value, tableOrder.error.value, order.error.value, prechecks.error.value, menu.error.value]));
  const actorName = computed(() => auth.actor?.name || auth.actor?.employee_id || '');

  const openShiftMutation = useMutation({
    mutationFn: openShift,
    onSuccess: refreshOps,
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

  const quantityMutation = useMutation({
    mutationFn: (payload: { lineId: string; quantity: number }) => changeOrderLineQuantity(activeOrder.value?.id ?? '', payload.lineId, payload.quantity),
    onSuccess: refreshOrder,
    onError: showBusinessError,
  });

  const voidLineMutation = useMutation({
    mutationFn: (lineId: string) => voidOrderLine(activeOrder.value?.id ?? '', lineId, 'waiter_void'),
    onSuccess: refreshOrder,
    onError: showBusinessError,
  });

  const issuePrecheckMutation = useMutation({
    mutationFn: () => issuePrecheck(activeOrder.value?.id ?? ''),
    onSuccess: refreshOrder,
    onError: showBusinessError,
  });

  const reprintPrecheckMutation = useMutation({
    mutationFn: () => reprintPrecheck(latestPrecheck.value?.id ?? ''),
    onSuccess() {
      $q.notify({ type: 'positive', message: `${t('pos.reprintReady')} · ${t('pos.reprintCopy')}` });
    },
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
      if (!value.paired) void router.replace('/pair');
    }
  }, { immediate: true });

  watch(session.data, (value) => {
    if (value) {
      auth.applySession(value.session, value.actor);
      if (value.session.status !== 'active') void router.replace('/login');
    }
  }, { immediate: true });

  watch(() => [auth.nodeDeviceId, auth.sessionId], () => {
    const fallback = resolveProtectedPosFallback({ nodeDeviceId: auth.nodeDeviceId, sessionId: auth.sessionId });
    if (fallback) void router.replace(fallback);
  }, { immediate: true });

  watch(activeHalls, (items) => {
    if (!selectedHallId.value && items[0]) selectedHallId.value = items[0].id;
  });

  watch(tableOrder.data, (value) => {
    if (value?.id) currentOrderId.value = value.id;
  });

  watch(activeLines, (lines) => {
    if (lines.some((line) => line.id === selectedOrderLineId.value)) return;
    selectedOrderLineId.value = lines[0]?.id ?? '';
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

  function selectOrder(orderId: string) {
    const found = activeOrders.value.find((item) => item.id === orderId);
    currentOrderId.value = orderId;
    selectedTableId.value = found?.table_id ?? selectedTableId.value;
  }

  function selectOrderLine(id: string) {
    selectedOrderLineId.value = id;
  }

  function openMenuItem(item: MenuItem) {
    if (!canAddOrderLine.value) return;
    const groups = item.modifier_groups.filter((group) => group.active && group.options.some((option) => option.active));
    if (groups.length === 0) {
      addLineMutation.mutate({ menuItemId: item.id });
      return;
    }
    modifierMenuItem.value = item;
    modifierQuantities.value = {};
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
    modifierQuantities.value = { ...modifierQuantities.value, [optionId]: Math.max(0, nextQuantity) };
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
    addLineMutation.mutate({ menuItemId: modifierMenuItem.value.id, selectedModifiers: selectedModifierPayload.value });
  }

  function closeModifierDialog() {
    modifierDialog.value = false;
    modifierMenuItem.value = null;
    modifierQuantities.value = {};
    modifierValidationKey.value = '';
  }

  function changeQuantity(lineId: string, quantity: number) {
    if (quantity < 1 || !canChangeOrderLine.value) return;
    quantityMutation.mutate({ lineId, quantity });
  }

  function voidLine(lineId: string) {
    if (!canVoidOrderLine.value) return;
    voidLineMutation.mutate(lineId);
  }

  function refreshOps() {
    void queryClient.invalidateQueries({ queryKey: ['waiter-current-shift'] });
  }

  function refreshOrder() {
    void queryClient.invalidateQueries({ queryKey: ['waiter-active-orders'] });
    void queryClient.invalidateQueries({ queryKey: ['waiter-current-order'] });
    void queryClient.invalidateQueries({ queryKey: ['waiter-order'] });
    void queryClient.invalidateQueries({ queryKey: ['waiter-prechecks'] });
  }

  function money(value: number, code: string) {
    return formatMinorCurrency(value, code, 'ru-RU');
  }

  function statusLabel(status: string) {
    return t(`status.${status}`);
  }

  function firstError(errors: unknown[]) {
    const found = errors.find(Boolean);
    if (!found) return '';
    const code = displayErrorSupportCode(found);
    const message = found instanceof ApiError ? t(displayErrorMessageKey(found)) : t('common.error');
    return code ? `${message} · ${t('errors.supportCode')}: ${code}` : message;
  }

  return {
    t,
    auth,
    selectedHallId,
    selectedTableId,
    currentOrderId,
    selectedOrderLineId,
    menuSearch,
    modifierDialog,
    modifierMenuItem,
    modifierQuantities,
    modifierValidationKey,
    canViewFloor,
    canViewMenu,
    canViewCurrentShift,
    canOpenShift,
    canViewOrder,
    canCreateOrderPermission,
    canAddLinePermission,
    canChangeLinePermission,
    canVoidLinePermission,
    canIssuePrecheckPermission,
    canViewPrecheck,
    canReprintPrecheckPermission,
    currentShift,
    halls,
    tables,
    activeOrdersQuery,
    tableOrder,
    order,
    prechecks,
    menu,
    activeHalls,
    activeTables,
    activeOrders,
    selectedTable,
    activeOrder,
    activeLines,
    selectedOrderLine,
    activeMenuItems,
    visibleMenuItems,
    activePrecheck,
    orderIsLocked,
    latestPrecheck,
    orderCurrency,
    modifierGroupsForDialog,
    canCreateOrder,
    canAddOrderLine,
    canChangeOrderLine,
    canVoidOrderLine,
    canIssuePrecheck,
    canReprintPrecheck,
    canSubmitModifierSelection,
    statusError,
    orderError,
    actorName,
    openShiftMutation,
    createOrderMutation,
    addLineMutation,
    quantityMutation,
    voidLineMutation,
    issuePrecheckMutation,
    reprintPrecheckMutation,
    selectHall,
    selectTable,
    selectOrder,
    selectOrderLine,
    openMenuItem,
    modifierGroupCount,
    changeModifierQuantity,
    submitModifierSelection,
    closeModifierDialog,
    changeQuantity,
    voidLine,
    refreshOrder,
    money,
    statusLabel,
  };
}

export type WaiterTerminal = ReturnType<typeof useWaiterTerminal>;
