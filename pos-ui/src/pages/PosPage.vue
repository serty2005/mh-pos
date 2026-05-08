<template>
  <q-page class="pos-page">
    <section class="operator-strip">
      <div class="operator-block">
        <p class="eyebrow">{{ t('pos.actor') }}</p>
        <h1>{{ auth.actor?.name ?? auth.actor?.employee_id }}</h1>
      </div>
      <div class="status-cluster">
        <div class="status-box">
          <span>{{ t('pos.pairing') }}</span>
          <strong>{{ pairing.data.value?.paired ? t('status.paired') : t('common.error') }}</strong>
        </div>
        <div class="status-box">
          <span>{{ t('common.node') }}</span>
          <strong>{{ shortId(auth.nodeDeviceId) }}</strong>
        </div>
        <div class="status-box">
          <span>{{ t('pos.session') }}</span>
          <strong>{{ shortId(auth.sessionId) }}</strong>
        </div>
        <q-btn flat dense round icon="lock" :aria-label="t('actions.lock')" @click="router.push('/lock')" />
      </div>
    </section>

    <section class="terminal-grid">
      <aside class="control-pane">
        <div class="section-head">
          <h2>{{ t('pos.shift') }}</h2>
          <q-btn flat dense round icon="refresh" :aria-label="t('actions.retry')" @click="refreshOps" />
        </div>

        <q-banner v-if="statusError" class="error-banner dense-banner" rounded>{{ statusError }}</q-banner>

        <div class="state-line">
          <span>{{ t('common.status') }}</span>
          <strong>{{ currentShift.data.value ? t('status.open') : t('pos.noShift') }}</strong>
        </div>
        <div v-if="currentShift.data.value" class="meta-line">
          <span>{{ shortId(currentShift.data.value.id) }}</span>
          <span>{{ formatDate(currentShift.data.value.opened_at) }}</span>
        </div>
        <q-btn
          v-if="!currentShift.data.value"
          color="primary"
          unelevated
          icon="schedule"
          :label="t('actions.openShift')"
          :disable="!canOpenShift"
          :loading="openShiftMutation.isPending.value"
          @click="openShiftMutation.mutate()"
        />
        <div v-if="!currentShift.data.value && recentShifts.data.value?.length" class="recent-shifts">
          <p class="eyebrow">{{ t('pos.recentPersonalShifts') }}</p>
          <div v-for="item in recentShifts.data.value" :key="item.id" class="meta-line">
            <span>{{ shortId(item.id) }}</span>
            <span>{{ formatDate(item.opened_at) }}</span>
            <span>{{ statusLabel(item.status) }}</span>
          </div>
        </div>

        <q-separator />

        <div class="section-head slim">
          <h2>{{ t('pos.cashSession') }}</h2>
        </div>
        <div class="state-line">
          <span>{{ t('common.status') }}</span>
          <strong>{{ currentCashSession.data.value ? t('status.open') : t('pos.noCashSession') }}</strong>
        </div>
        <div v-if="currentCashSession.data.value" class="meta-line">
          <span>{{ shortId(currentCashSession.data.value.id) }}</span>
          <span>{{ formatDate(currentCashSession.data.value.opened_at) }}</span>
        </div>
        <q-input
          v-if="!currentCashSession.data.value"
          v-model.number="openingCashAmount"
          dense
          outlined
          type="number"
          min="0"
          :step="currencyInputStep(currency)"
          :label="t('common.amount')"
          :suffix="currency"
          :disable="!currentShift.data.value"
        />
        <q-btn
          v-if="!currentCashSession.data.value"
          color="primary"
          outline
          icon="point_of_sale"
          :label="t('actions.openCashSession')"
          :disable="!currentShift.data.value || !canOpenCashSession"
          :loading="openCashMutation.isPending.value"
          @click="openCashMutation.mutate(openingCashAmount)"
        />

        <div v-if="currentCashSession.data.value" class="inline-action">
          <q-input
            v-model.number="closingCashAmount"
            dense
            outlined
            type="number"
            min="0"
            :step="currencyInputStep(currency)"
            :label="t('common.amount')"
            :suffix="currency"
          />
          <q-btn
            outline
            color="secondary"
            icon="payments"
            :label="t('actions.closeCashSession')"
            :disable="!canCloseCashSession"
            :loading="closeCashMutation.isPending.value"
            @click="closeCashMutation.mutate({ cashSessionId: currentCashSession.data.value.id, amount: closingCashAmount })"
          />
        </div>

        <div v-else-if="currentShift.data.value" class="inline-action">
          <q-btn
            outline
            color="secondary"
            icon="event_busy"
            :label="t('actions.closeShift')"
            :disable="!canCloseShift"
            :loading="closeShiftMutation.isPending.value"
            @click="closeShiftMutation.mutate(currentShift.data.value.id)"
          />
        </div>

        <q-separator />

        <div class="section-head slim">
          <h2>{{ t('pos.halls') }}</h2>
        </div>
        <q-skeleton v-if="halls.isPending.value" class="skeleton-row" />
        <q-banner v-else-if="halls.isError.value" class="error-banner dense-banner" rounded>{{ t('common.error') }}</q-banner>
        <q-list v-else-if="activeHalls.length" class="compact-list" separator>
          <q-item
            v-for="hall in activeHalls"
            :key="hall.id"
            clickable
            :active="hall.id === selectedHallId"
            active-class="active-item"
            @click="selectHall(hall.id)"
          >
            <q-item-section>{{ hall.name }}</q-item-section>
          </q-item>
        </q-list>
        <div v-else class="empty-state small">{{ t('pos.noHalls') }}</div>

        <div class="section-head slim">
          <h2>{{ t('pos.tables') }}</h2>
        </div>
        <div v-if="tables.isPending.value" class="table-list">
          <q-skeleton v-for="n in 6" :key="n" class="table-button skeleton-tile" />
        </div>
        <q-banner v-else-if="tables.isError.value" class="error-banner dense-banner" rounded>{{ t('common.error') }}</q-banner>
        <div v-else-if="activeTables.length" class="table-list">
          <button
            v-for="table in activeTables"
            :key="table.id"
            class="table-button"
            :class="{ selected: table.id === selectedTableId }"
            type="button"
            @click="selectTable(table.id)"
          >
            <span>{{ table.name }}</span>
            <small>{{ table.seats }}</small>
          </button>
        </div>
        <div v-else class="empty-state small">{{ t('pos.noTables') }}</div>
      </aside>

      <main class="order-pane">
        <div class="section-head">
          <div>
            <p class="eyebrow">{{ selectedTable?.name ? `${t('pos.selectedTable')} ${selectedTable.name}` : t('pos.chooseTable') }}</p>
            <h2>{{ t('pos.activeOrder') }}</h2>
          </div>
          <q-btn
            color="primary"
            unelevated
            icon="receipt_long"
            :label="t('actions.createOrder')"
            :disable="!canCreateOrder"
            :loading="createOrderMutation.isPending.value"
            @click="createOrderMutation.mutate()"
          />
        </div>

        <q-banner v-if="orderError" class="error-banner dense-banner" rounded>{{ orderError }}</q-banner>
        <q-skeleton v-if="orderLoading" class="order-skeleton" />

        <div v-else-if="activeOrder" class="order-workspace">
          <div class="order-summary">
            <div>
              <span>{{ t('pos.order') }}</span>
              <strong>{{ shortId(activeOrder.id) }}</strong>
            </div>
            <div>
              <span>{{ t('common.status') }}</span>
              <strong>{{ statusLabel(activeOrder.status) }}</strong>
            </div>
            <div>
              <span>{{ t('pos.total') }}</span>
              <strong>{{ money(activeOrder.total, orderCurrency) }}</strong>
            </div>
          </div>

          <q-banner v-if="activeOrder.status === 'locked'" class="info-banner" rounded>
            {{ t('pos.lockedOrder') }}
          </q-banner>
          <q-banner v-if="finalCheckData" class="success-banner" rounded>
            {{ t('pos.checkCreated') }}: {{ shortId(finalCheckData.id) }} · {{ money(finalCheckData.total, orderCurrency) }}
          </q-banner>

          <div class="section-head slim">
            <h2>{{ t('pos.orderLines') }}</h2>
          </div>
          <div v-if="activeLines.length" class="line-table">
            <div v-for="line in activeLines" :key="line.id" class="line-row">
              <div class="line-title">
                <strong>{{ line.name }}</strong>
                <span>{{ money(line.unit_price, orderCurrency) }}</span>
              </div>
              <div class="quantity-stepper">
                <q-btn
                  flat
                  dense
                  round
                  icon="remove"
                  :disable="!canChangeOrderLine || line.quantity <= 1"
                  @click="changeQuantity(line.id, line.quantity - 1)"
                />
                <span>{{ line.quantity }}</span>
                <q-btn flat dense round icon="add" :disable="!canChangeOrderLine" @click="changeQuantity(line.id, line.quantity + 1)" />
              </div>
              <strong class="line-total">{{ money(line.total_price, orderCurrency) }}</strong>
              <q-btn flat dense round icon="delete" color="negative" :disable="!canVoidOrderLine" :aria-label="t('actions.voidLine')" @click="voidLine(line.id)" />
            </div>
          </div>
          <div v-else class="empty-state">{{ t('pos.emptyOrder') }}</div>
        </div>

        <div v-else class="empty-state wide">{{ selectedTableId ? t('pos.noActiveOrder') : t('pos.chooseTable') }}</div>
      </main>

      <aside class="action-pane">
        <div class="section-head">
          <h2>{{ t('pos.menu') }}</h2>
          <q-btn flat dense round icon="refresh" :aria-label="t('actions.retry')" @click="refetchMenu" />
        </div>
        <q-skeleton v-if="menu.isPending.value" class="skeleton-row" />
        <q-banner v-else-if="menu.isError.value" class="error-banner dense-banner" rounded>{{ t('common.error') }}</q-banner>
        <div v-else-if="activeMenuItems.length" class="menu-list">
          <button
            v-for="item in activeMenuItems"
            :key="item.id"
            class="menu-button"
            type="button"
            :disabled="!canAddOrderLine"
            @click="addLineMutation.mutate(item.id)"
          >
            <span>{{ item.name }}</span>
            <strong>{{ money(item.price, item.currency) }}</strong>
          </button>
        </div>
        <div v-else class="empty-state">{{ t('pos.emptyMenu') }}</div>

        <q-separator />

        <div class="section-head slim">
          <h2>{{ t('pos.precheckActions') }}</h2>
        </div>
        <div v-if="activePrecheck" class="precheck-box">
          <div class="state-line">
            <span>{{ t('pos.precheck') }}</span>
            <strong>{{ t('pos.precheckIssued') }}</strong>
          </div>
          <div class="state-line">
            <span>{{ t('pos.total') }}</span>
            <strong>{{ money(activePrecheck.total, orderCurrency) }}</strong>
          </div>
          <div class="state-line">
            <span>{{ t('pos.payment') }}</span>
            <strong>{{ money(activePrecheck.paid_total, orderCurrency) }}</strong>
          </div>
          <q-btn outline color="negative" icon="undo" :label="t('pos.cancelPrecheck')" :disable="activePrecheck.paid_total > 0 || !canCancelPrecheck" @click="cancelDialog = true" />
        </div>
        <q-btn
          v-else
          color="primary"
          unelevated
          icon="request_quote"
          :label="t('actions.issuePrecheck')"
          :disable="!canIssuePrecheck"
          :loading="issuePrecheckMutation.isPending.value"
          @click="issuePrecheckMutation.mutate(activeOrder?.id ?? '')"
        />

        <div class="payment-box">
          <q-input
            v-model.number="paymentAmount"
            outlined
            dense
            type="number"
            min="0"
            :step="currencyInputStep(orderCurrency)"
            :label="t('pos.paymentAmount')"
            :suffix="orderCurrency"
            :disable="!activePrecheck"
          />
          <div class="payment-actions">
            <q-btn
              color="primary"
              unelevated
              icon="payments"
              :label="t('actions.payCash')"
              :disable="!canPayCash"
              :loading="paymentMutation.isPending.value"
              @click="pay('cash')"
            />
            <q-btn
              color="secondary"
              unelevated
              icon="credit_card"
              :label="t('actions.payCard')"
              :disable="!canPayCard"
              :loading="paymentMutation.isPending.value"
              @click="pay('card')"
            />
          </div>
        </div>
      </aside>
    </section>

    <q-dialog v-model="cancelDialog" persistent>
      <q-card class="dialog-card">
        <q-card-section>
          <h2>{{ t('pos.cancelPrecheck') }}</h2>
        </q-card-section>
        <q-card-section class="form-stack">
          <q-input v-model="managerEmployeeId" outlined :label="t('pos.managerEmployeeId')" autocomplete="off" />
          <q-input v-model="managerPin" outlined :label="t('pos.managerPin')" type="password" inputmode="numeric" autocomplete="new-password" />
          <q-input v-model="cancelReason" outlined :label="t('pos.precheckCancelReason')" type="textarea" autogrow />
        </q-card-section>
        <q-card-actions align="right">
          <q-btn flat :label="t('actions.cancel')" @click="closeCancelDialog" />
          <q-btn
            color="negative"
            unelevated
            icon="undo"
            :label="t('pos.cancelPrecheck')"
            :loading="cancelPrecheckMutation.isPending.value"
            @click="submitCancelPrecheck"
          />
        </q-card-actions>
      </q-card>
    </q-dialog>
  </q-page>
</template>

<script setup lang="ts">
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query';
import { computed, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRouter } from 'vue-router';

import {
  addOrderLine,
  cancelPrecheck,
  capturePrecheckPayment,
  changeOrderLineQuantity,
  closeCashSession,
  closeShift,
  createOrder,
  getAuthSession,
  getCurrentCashSession,
  getCurrentOrderByTable,
  getCurrentShift,
  getCheck,
  getOrder,
  getPairingStatus,
  issuePrecheck,
  listHalls,
  listMenuItems,
  listRecentShifts,
  listPrechecksByOrder,
  listTables,
  openCashSession,
  openShift,
  voidOrderLine,
} from '../shared/api';
import { resolveProtectedPosFallback } from '../shared/sessionGuards';
import { currencyInputStep, formatMinorCurrency, minorToMoney, moneyToMinor } from '../shared/currency';
import { displayErrorMessageKey, useErrorHandling } from '../shared/errorHandling';
import { hasPermission, permissionCatalog } from '../shared/rbac';
import type { Order } from '../shared/schemas';
import { useAuthStore } from '../stores/auth';

const { t } = useI18n();
const auth = useAuthStore();
const router = useRouter();
const queryClient = useQueryClient();
const { showBusinessError } = useErrorHandling();

const selectedHallId = ref('');
const selectedTableId = ref('');
const currentOrderId = ref('');
const openingCashAmount = ref(0);
const closingCashAmount = ref(0);
const paymentAmount = ref(0);
const cancelDialog = ref(false);
const managerEmployeeId = ref('');
const managerPin = ref('');
const cancelReason = ref('');
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
const activePrecheck = computed(() => prechecks.data.value?.find((item) => item.status === 'issued') ?? null);
const finalCheckData = computed(() => finalCheck.data.value ?? activeOrder.value?.check ?? null);
const currency = computed(() => activeMenuItems.value[0]?.currency ?? 'RUB');
const orderCurrency = computed(() => activeMenuItems.value.find((item) => activeLines.value.some((line) => line.menu_item_id === item.id))?.currency ?? currency.value);
const canCreateOrder = computed(() => Boolean(selectedTableId.value && currentShift.data.value && !activeOrder.value && hasPermission(grantedPermissions.value, permissionCatalog.orderCreate)));
const canAddOrderLine = computed(() => Boolean(activeOrder.value?.status === 'open' && !activePrecheck.value && !activeOrder.value.check && hasPermission(grantedPermissions.value, permissionCatalog.orderAddLine)));
const canChangeOrderLine = computed(() => Boolean(activeOrder.value?.status === 'open' && !activePrecheck.value && !activeOrder.value.check && hasPermission(grantedPermissions.value, permissionCatalog.orderChangeQuantity)));
const canVoidOrderLine = computed(() => Boolean(activeOrder.value?.status === 'open' && !activePrecheck.value && !activeOrder.value.check && hasPermission(grantedPermissions.value, permissionCatalog.orderVoidLine)));
const canIssuePrecheck = computed(() => Boolean(activeOrder.value?.status === 'open' && activeLines.value.length > 0 && !activePrecheck.value && hasPermission(grantedPermissions.value, permissionCatalog.precheckIssue)));
const canCancelPrecheck = computed(() => hasPermission(grantedPermissions.value, permissionCatalog.precheckCancelRequest));
const canPayCash = computed(() => Boolean(currentCashSession.data.value && activePrecheck.value && paymentAmount.value > 0 && paymentAmount.value <= remainingPayment.value && hasPermission(grantedPermissions.value, permissionCatalog.paymentCash)));
const canPayCard = computed(() => Boolean(currentCashSession.data.value && activePrecheck.value && paymentAmount.value > 0 && paymentAmount.value <= remainingPayment.value && hasPermission(grantedPermissions.value, permissionCatalog.paymentCardManual)));
const remainingPayment = computed(() => activePrecheck.value ? activePrecheck.value.total - activePrecheck.value.paid_total : 0);
const orderLoading = computed(() => tableOrder.isPending.value || order.isPending.value);
const statusError = computed(() => firstError([currentShift.error.value, currentCashSession.error.value]));
const orderError = computed(() => firstError([tableOrder.error.value, order.error.value, prechecks.error.value]));
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

const issuePrecheckMutation = useMutation({
  mutationFn: issuePrecheck,
  onSuccess: refreshOrder,
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
  currentOrderId.value = value?.id ?? '';
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

function refetchMenu() {
  void menu.refetch();
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
</script>
