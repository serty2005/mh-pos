<template>
  <section class="activity-workspace section-workspace">
    <main class="section-main-surface" :aria-label="terminal.t('pos.activityTitle')">
      <PosSectionHeader
        :eyebrow="terminal.t('pos.sections.activity')"
        :title="terminal.t('pos.activityTitle')"
        :refresh-label="terminal.t('actions.retry')"
        @refresh="() => terminal.closedOrders.refetch()"
      />

      <div class="compact-filter-bar" :aria-label="terminal.t('pos.activityFilters')">
        <q-input
          v-model="search"
          dense
          outlined
          clearable
          debounce="120"
          class="integrated-search"
          :label="terminal.t('pos.searchClosedOrders')"
        >
          <template #prepend>
            <q-icon name="search" />
          </template>
        </q-input>
        <q-input
          v-model="terminal.closedOrdersBusinessDate.value"
          dense
          outlined
          clearable
          type="date"
          class="date-filter"
          :label="terminal.t('pos.businessDate')"
        />
        <PosTabs :model-value="filter" :options="filterTabs" :accessibility-label="terminal.t('pos.activityFilters')" variant="chip" @update:model-value="setFilter" />
        <PosPagination
          :value="terminal.closedOrdersOffset.value + 1"
          :previous-label="terminal.t('actions.previousPage')"
          :next-label="terminal.t('actions.nextPage')"
          :previous-disabled="!terminal.closedOrdersHasPreviousPage.value"
          :next-disabled="!terminal.closedOrdersHasNextPage.value"
          @previous="terminal.previousClosedOrdersPage"
          @next="terminal.nextClosedOrdersPage"
        />
      </div>

      <PosBanner v-if="terminal.closedOrders.error.value" tone="error" :label="terminal.t(terminal.displayErrorMessageKey(terminal.closedOrders.error.value))" />

      <div v-if="terminal.closedOrders.isFetching.value" class="activity-list">
        <q-skeleton v-for="n in 8" :key="n" class="activity-order-item skeleton-row" />
      </div>

      <PosEmptyState v-else-if="!terminal.canViewClosedOrders.value" size="wide" :label="terminal.t('pos.noPermissionForClosedOrders')" />

      <div v-else-if="visibleOrders.length" class="activity-list">
        <button
          v-for="order in visibleOrders"
          :key="order.id"
          class="activity-order-item"
          :class="{ selected: order.id === selectedOrder?.id }"
          type="button"
          @click="selectedOrderId = order.id"
        >
          <span class="activity-order-main">
            <strong>{{ order.table_name || terminal.t('pos.table') }}</strong>
            <small>{{ terminal.t('pos.order') }} {{ terminal.shortId(order.id) }}</small>
          </span>
          <span class="activity-order-meta">
            <strong>{{ terminal.money(order.total, order.check?.currency_code ?? 'RUB') }}</strong>
            <small>{{ order.check?.business_date_local ?? terminal.statusLabel(order.status) }}</small>
          </span>
          <PosStatusStrip :value="paymentStateLabel(order)" :tone="paymentStateTone(order)" />
        </button>
      </div>

      <PosEmptyState v-else size="wide" :label="terminal.t('pos.noClosedOrders')" />
    </main>

    <aside class="activity-detail-rail section-action-rail" :aria-label="terminal.t('pos.closedOrderDetails')">
      <div class="rail-head">
        <div>
          <p class="eyebrow">{{ terminal.t('pos.finalCheck') }}</p>
          <h2>{{ selectedOrder ? terminal.t('pos.closedOrderDetails') : terminal.t('common.empty') }}</h2>
        </div>
      </div>

      <template v-if="selectedOrder">
        <div class="rail-summary">
          <div>
            <span>{{ terminal.t('pos.table') }}</span>
            <strong>{{ selectedOrder.table_name || terminal.t('common.empty') }}</strong>
          </div>
          <div>
            <span>{{ terminal.t('pos.total') }}</span>
            <strong>{{ terminal.money(selectedOrder.total, selectedOrder.check?.currency_code ?? 'RUB') }}</strong>
          </div>
          <div>
            <span>{{ terminal.t('common.status') }}</span>
            <strong>{{ terminal.statusLabel(selectedOrder.check?.status ?? selectedOrder.status) }}</strong>
          </div>
        </div>

        <div class="payment-list">
          <p class="eyebrow">{{ terminal.t('pos.payments') }}</p>
          <article v-for="payment in selectedOrder.check?.payments ?? []" :key="payment.id" class="payment-row">
            <span>
              <strong>{{ terminal.t(`pos.paymentMethods.${payment.method}`) }}</strong>
              <small>{{ terminal.statusLabel(payment.status) }}</small>
            </span>
            <strong>{{ terminal.money(payment.amount, payment.currency) }}</strong>
          </article>
          <PosEmptyState v-if="!(selectedOrder.check?.payments?.length)" size="small" :label="terminal.t('common.empty')" />
        </div>

        <div class="payment-list">
          <p class="eyebrow">{{ terminal.t('pos.financialOperations') }}</p>
          <PosBanner v-if="financialOperations.error.value" tone="error" :label="terminal.t(terminal.displayErrorMessageKey(financialOperations.error.value))" />
          <q-skeleton v-if="financialOperations.isFetching.value" class="order-skeleton" />
          <article v-for="operation in financialOperations.data.value ?? []" :key="operation.id" class="payment-row ledger-row">
            <span>
              <strong>{{ operationTitle(operation) }}</strong>
              <small>{{ operation.reason }}</small>
              <small>{{ terminal.t('pos.businessDate') }}: {{ operation.business_date_local }}</small>
              <small>{{ terminal.t('pos.inventoryDisposition') }}: {{ terminal.t(`pos.inventoryDispositions.${operation.inventory_disposition}`) }}</small>
              <small>{{ terminal.t('pos.createdByEmployee') }}: {{ terminal.shortId(operation.created_by_employee_id) }}</small>
              <small v-if="operation.approved_by_employee_id">{{ terminal.t('pos.approvedByEmployee') }}: {{ terminal.shortId(operation.approved_by_employee_id) }}</small>
            </span>
            <span class="ledger-row-side">
              <strong>{{ terminal.money(operation.amount, operation.currency) }}</strong>
              <small>{{ terminal.formatDate(operation.created_at) }}</small>
            </span>
          </article>
          <PosEmptyState v-if="!financialOperations.isFetching.value && !(financialOperations.data.value?.length)" size="small" :label="terminal.t('common.empty')" />
        </div>

        <div class="rail-actions integrated-action-bar">
          <PosButton
            variant="primary"
            primary
            icon="print"
            :label="terminal.t('actions.reprintCheck')"
            :disabled="!terminal.canReprintClosedCheck.value || !selectedOrder.check"
            :loading="terminal.reprintCheckMutation.isPending.value"
            @click="selectedOrder.check && terminal.reprintCheckMutation.mutate(selectedOrder.check.id)"
          />
          <PosButton
            variant="danger"
            icon="cancel"
            :label="terminal.t('pos.checkCancellation')"
            :disabled="!canCancelSelected"
            :loading="terminal.refundMutation.isPending.value && terminal.refundMode.value === 'check_cancellation'"
            @click="terminal.openCheckCancellationDialogForOrder(selectedOrder)"
          />
          <PosButton
            variant="danger"
            mode="outline"
            icon="undo"
            :label="terminal.t('pos.checkRefund')"
            :disabled="!canRefundSelected"
            :loading="terminal.refundMutation.isPending.value && terminal.refundMode.value === 'check_refund'"
            @click="terminal.openCheckRefundDialogForOrder(selectedOrder)"
          />
          <PosButton
            variant="danger"
            mode="flat"
            icon="payments"
            :label="terminal.t('pos.paymentRefund')"
            :disabled="!canRefundPaymentSelected"
            :loading="terminal.refundMutation.isPending.value && terminal.refundMode.value === 'payment_refund'"
            @click="terminal.openRefundDialogForOrder(selectedOrder)"
          />
        </div>
      </template>

      <div v-else class="rail-empty">
        <p>{{ terminal.t('pos.chooseClosedOrder') }}</p>
      </div>
    </aside>
  </section>
</template>

<script setup lang="ts">
import { useQuery } from '@tanstack/vue-query';
import { computed, ref, watch } from 'vue';

import { listFinancialOperationsByCheck } from '../../shared/api';
import type { ClosedOrder, FinancialOperation } from '../../shared/schemas';
import { PosBanner, PosButton, PosEmptyState, PosPagination, PosSectionHeader, PosStatusStrip, PosTabs, type PosTabOption } from '../../shared/ui';
import type { CashierTerminal } from './useCashierTerminal';

type ActivityFilter = 'all' | 'refundable';

const props = defineProps<{
  terminal: CashierTerminal;
}>();

const search = ref('');
const filter = ref<ActivityFilter>('all');
const selectedOrderId = ref('');

const filterOptions: Array<{ value: ActivityFilter; labelKey: string }> = [
  { value: 'all', labelKey: 'pos.allClosedOrders' },
  { value: 'refundable', labelKey: 'pos.refundableOrders' },
];

const filterTabs = computed<PosTabOption[]>(() => filterOptions.map((option) => ({
  value: option.value,
  label: props.terminal.t(option.labelKey),
})));

const orders = computed(() => props.terminal.closedOrders.data.value ?? []);

const visibleOrders = computed(() => {
  const query = search.value.trim().toLocaleLowerCase('ru-RU');
  return orders.value.filter((order) => {
    if (filter.value === 'refundable' && !hasCapturedPayment(order)) return false;
    if (!query) return true;
    return [order.id, order.table_name, order.check?.business_date_local]
      .filter(Boolean)
      .some((value) => String(value).toLocaleLowerCase('ru-RU').includes(query));
  });
});

const selectedOrder = computed(() => visibleOrders.value.find((order) => order.id === selectedOrderId.value) ?? visibleOrders.value[0] ?? null);
const selectedCheckId = computed(() => selectedOrder.value?.check?.id ?? '');

const financialOperations = useQuery({
  queryKey: ['financial-operations', selectedCheckId],
  queryFn: () => listFinancialOperationsByCheck(selectedCheckId.value),
  enabled: () => Boolean(selectedCheckId.value && props.terminal.canViewClosedOrders.value),
});

const canRefundSelected = computed(() => Boolean(
  selectedOrder.value
  && props.terminal.canRefundClosedOrder(selectedOrder.value),
));

const canCancelSelected = computed(() => Boolean(
  selectedOrder.value
  && props.terminal.canCancelClosedOrder(selectedOrder.value),
));

const canRefundPaymentSelected = computed(() => Boolean(
  selectedOrder.value
  && props.terminal.canRefundPaymentForOrder(selectedOrder.value),
));

watch(visibleOrders, (items) => {
  if (!items.some((order) => order.id === selectedOrderId.value)) {
    selectedOrderId.value = items[0]?.id ?? '';
  }
}, { immediate: true });

function hasCapturedPayment(order: ClosedOrder) {
  return Boolean(order.check?.payments?.some((payment) => payment.status === 'captured'));
}

function hasRefundedPayment(order: ClosedOrder) {
  return Boolean(order.check?.payments?.some((payment) => payment.status === 'refunded'));
}

function paymentStateLabel(order: ClosedOrder) {
  if (hasRefundedPayment(order)) return props.terminal.t('status.refunded');
  if (hasCapturedPayment(order)) return props.terminal.t('status.paid');
  return props.terminal.statusLabel(order.check?.status ?? order.status);
}

function paymentStateTone(order: ClosedOrder) {
  if (hasRefundedPayment(order)) return 'warning';
  if (hasCapturedPayment(order)) return 'good';
  return 'neutral';
}

function setFilter(value: string) {
  if (value === 'all' || value === 'refundable') {
    filter.value = value;
  }
}

function operationTitle(operation: FinancialOperation) {
  return `${props.terminal.t(`pos.operationTypes.${operation.operation_type}`)} · ${props.terminal.t(`pos.operationKinds.${operation.operation_kind}`)}`;
}
</script>
