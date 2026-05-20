<template>
  <footer class="pos-bottom-bar" :aria-label="terminal.t('pos.quickAccess')">
    <button class="bottom-section-button" :class="{ active: menuOpen }" type="button" @click="$emit('toggle-menu')">
      <q-icon name="apps" size="24px" />
      <span>{{ terminal.t(activeSectionLabelKey) }}</span>
    </button>

    <div v-if="activeSection === 'order'" class="bottom-status-grid order-status-grid">
      <span class="bottom-status-cell">
        <small>{{ terminal.t('pos.orderNumberLabel') }}</small>
        <strong>{{ terminal.activeOrder.value ? terminal.shortId(terminal.activeOrder.value.id) : '-' }}</strong>
      </span>
      <span class="bottom-status-cell">
        <small>{{ terminal.t('pos.hallTable') }}</small>
        <strong>{{ hallTableLabel }}</strong>
      </span>
      <span class="bottom-status-cell">
        <small>{{ terminal.t('pos.waiter') }}</small>
        <strong>{{ terminal.actorName.value || '-' }}</strong>
      </span>
      <button class="bottom-status-cell two-line-cell" type="button" @click="$emit('open-discounts')">
        <small>{{ openedLabel }}</small>
        <strong>{{ discountLabel }}</strong>
      </button>
    </div>

    <div v-else-if="activeSection === 'floor'" class="bottom-status-grid floor-status-grid">
      <span class="bottom-status-cell">
        <small>{{ terminal.t('pos.shiftTotal') }}</small>
        <strong>{{ terminal.money(shiftTotal, terminal.orderCurrency.value) }}</strong>
      </span>
      <span class="bottom-status-cell">
        <small>{{ terminal.t('pos.averageCheck') }}</small>
        <strong>{{ terminal.money(averageCheck, terminal.orderCurrency.value) }}</strong>
      </span>
      <span class="bottom-status-cell">
        <small>{{ terminal.t('pos.ordersCount') }}</small>
        <strong>{{ ordersCount }}</strong>
      </span>
      <span class="bottom-status-cell">
        <small>{{ terminal.t('pos.completedCount') }}</small>
        <strong>{{ completedCount }}</strong>
      </span>
    </div>

    <div v-else class="bottom-status-grid">
      <span class="bottom-status-cell">
        <small>{{ terminal.t(activeSectionLabelKey) }}</small>
        <strong>{{ sectionStatusLabel }}</strong>
      </span>
    </div>

    <q-btn flat square icon="lock" class="bottom-lock-button" :aria-label="terminal.t('actions.lock')" @click="terminal.lockTerminal" />
  </footer>
</template>

<script setup lang="ts">
import { computed } from 'vue';

import type { CashierTerminal } from './useCashierTerminal';

const props = defineProps<{
  terminal: CashierTerminal;
  activeSection: string;
  menuOpen: boolean;
}>();

defineEmits<{
  (event: 'toggle-menu'): void;
  (event: 'open-discounts'): void;
}>();

const activeSectionLabelKey = computed(() => `pos.sections.${props.activeSection}`);
const hallTableLabel = computed(() => {
  const hall = props.terminal.activeHalls.value.find((item) => item.id === props.terminal.selectedHallId.value)?.name ?? props.terminal.t('pos.currentHall');
  const table = props.terminal.selectedTable.value?.name ?? '-';
  return `${hall} / ${table}`;
});
const openedLabel = computed(() => {
  const openedAt = props.terminal.activeOrder.value?.opened_at;
  if (!openedAt) return props.terminal.t('pos.openedEmpty');
  return props.terminal.t('pos.openedAt', { value: formatOpenedAt(openedAt) });
});
const discountLabel = computed(() => props.terminal.t('pos.discountPercent', { value: 0 }));
const shiftTotal = computed(() => (props.terminal.closedOrders.data.value ?? []).reduce((sum, order) => sum + order.total, 0) + (props.terminal.activeOrder.value?.total ?? 0));
const ordersCount = computed(() => (props.terminal.closedOrders.data.value ?? []).length + (props.terminal.activeOrder.value ? 1 : 0));
const averageCheck = computed(() => ordersCount.value > 0 ? Math.round(shiftTotal.value / ordersCount.value) : 0);
const completedCount = computed(() => (props.terminal.closedOrders.data.value ?? []).length);
const sectionStatusLabel = computed(() => {
  if (props.activeSection === 'shift') {
    return props.terminal.currentCashSession.data.value ? props.terminal.t('pos.cashSessionOpen') : props.terminal.t('pos.noCashSession');
  }
  if (props.activeSection === 'analytics') {
    return props.terminal.t('pos.closedOrdersCount', { count: completedCount.value });
  }
  return props.terminal.t(activeSectionLabelKey.value);
});

function formatOpenedAt(value: string) {
  const date = new Date(value);
  const now = new Date();
  const sameDay = date.toDateString() === now.toDateString();
  return new Intl.DateTimeFormat('ru-RU', sameDay ? { hour: '2-digit', minute: '2-digit' } : { day: '2-digit', month: '2-digit', hour: '2-digit', minute: '2-digit' }).format(date);
}
</script>
