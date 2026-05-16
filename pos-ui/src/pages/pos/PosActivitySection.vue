<template>
  <section class="activity-workspace section-workspace">
    <main class="section-main-surface" :aria-label="terminal.t('pos.activityTitle')">
      <div class="pos-section-head">
        <div>
          <p class="eyebrow">{{ terminal.t('pos.sections.activity') }}</p>
          <h1>{{ terminal.t('pos.activityTitle') }}</h1>
        </div>
        <q-btn flat round icon="refresh" class="icon-touch" :aria-label="terminal.t('actions.retry')" @click="() => terminal.closedOrders.refetch()" />
      </div>

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
        <button
          v-for="option in filterOptions"
          :key="option.value"
          class="menu-filter-chip"
          :class="{ active: filter === option.value }"
          type="button"
          @click="filter = option.value"
        >
          {{ terminal.t(option.labelKey) }}
        </button>
      </div>

      <q-banner v-if="terminal.closedOrders.error.value" class="error-banner dense-banner" rounded>
        {{ terminal.t(terminal.displayErrorMessageKey(terminal.closedOrders.error.value)) }}
      </q-banner>

      <div v-if="terminal.closedOrders.isFetching.value" class="activity-list">
        <q-skeleton v-for="n in 8" :key="n" class="activity-order-item skeleton-row" />
      </div>

      <div v-else-if="!terminal.canViewClosedOrders.value" class="empty-state wide">
        {{ terminal.t('pos.noPermissionForClosedOrders') }}
      </div>

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
          <span class="status-strip" :class="{ good: hasCapturedPayment(order), warning: hasRefundedPayment(order) }">
            {{ paymentStateLabel(order) }}
          </span>
        </button>
      </div>

      <div v-else class="empty-state wide">{{ terminal.t('pos.noClosedOrders') }}</div>
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
          <div v-if="!(selectedOrder.check?.payments?.length)" class="empty-state small">{{ terminal.t('common.empty') }}</div>
        </div>

        <div class="rail-actions integrated-action-bar">
          <q-btn
            color="primary"
            unelevated
            class="touch-button primary-action"
            icon="print"
            :label="terminal.t('actions.reprintCheck')"
            :disable="!terminal.canReprintClosedCheck.value || !selectedOrder.check"
            :loading="terminal.reprintCheckMutation.isPending.value"
            @click="selectedOrder.check && terminal.reprintCheckMutation.mutate(selectedOrder.check.id)"
          />
          <q-btn
            color="negative"
            outline
            class="touch-button"
            icon="undo"
            :label="terminal.t('pos.refund')"
            :disable="!canRefundSelected"
            :loading="terminal.refundMutation.isPending.value"
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
import { computed, ref, watch } from 'vue';

import type { ClosedOrder } from '../../shared/schemas';
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

const canRefundSelected = computed(() => Boolean(
  selectedOrder.value
  && hasCapturedPayment(selectedOrder.value)
  && props.terminal.canRefundPayment.value
  && props.terminal.currentCashSession.data.value,
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
</script>
