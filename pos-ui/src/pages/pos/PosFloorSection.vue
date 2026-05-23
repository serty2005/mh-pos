<template>
  <section class="hall-orders-screen">
    <main class="hall-workspace" :aria-label="terminal.t('pos.floorPlan')">
      <PosBanner v-if="terminal.statusError.value" tone="error" :label="terminal.statusError.value" />

      <blocking-notice
        v-if="!terminal.currentShift.data.value && terminal.currentBlockingNotice.value"
        :terminal="terminal"
        :title="terminal.t(terminal.currentBlockingNotice.value.titleKey)"
        :reason="terminal.t(terminal.currentBlockingNotice.value.reasonKey)"
        :permission="terminal.currentBlockingNotice.value.permission"
        icon="lock_clock"
      />
      <PosEmptyState v-else-if="!terminal.canViewFloor.value" size="small" :label="terminal.t('pos.noPermissionForFloor')" />
      <div v-else-if="terminal.tables.isPending.value" class="floor-table-grid">
        <PosSkeleton v-for="n in 15" :key="n" class="floor-table-tile skeleton-tile" />
      </div>
      <PosBanner v-else-if="terminal.tables.isError.value" tone="error" :label="terminal.t('common.error')" />
      <div v-else-if="tableCards.length" class="floor-table-grid">
        <button
          v-for="card in tableCards"
          :key="card.id"
          class="floor-table-tile"
          :class="[`status-${card.status}`, { selected: card.id === terminal.selectedTableId.value }]"
          type="button"
          :aria-pressed="card.id === terminal.selectedTableId.value"
          :disabled="card.status === 'unavailable'"
          @click="openTable(card)"
        >
          <strong class="table-number">{{ card.name }}</strong>
          <span class="table-status">{{ terminal.t(`pos.tableStatus.${card.status}`) }}</span>
          <small v-if="card.orderNo">{{ terminal.t('pos.orderNumber', { number: card.orderNo }) }}</small>
          <small v-if="card.guests">{{ terminal.t('pos.guestsShort', { count: card.guests }) }}</small>
          <strong v-if="card.total" class="table-total">{{ terminal.money(card.total, terminal.orderCurrency.value) }}</strong>
          <small v-if="card.duration">{{ card.duration }}</small>
        </button>
      </div>
      <PosEmptyState v-else size="small" :label="terminal.t('pos.noTables')" />
    </main>

    <aside class="active-orders-panel" :aria-label="terminal.t('pos.activeOrders')">
      <div v-for="group in activeOrderGroups" :key="group.hall" class="hall-order-group">
        <h2>{{ group.hall }}</h2>
        <button v-for="order in group.orders" :key="order.id" class="active-order-card" type="button" @click="openOrder(order.tableId)">
          <strong>{{ terminal.t('pos.orderShort', { number: order.number }) }}</strong>
          <span>{{ terminal.t('pos.tableWithName', { table: order.table }) }}</span>
          <span>{{ order.status === 'precheck' ? terminal.t('pos.precheck') : terminal.money(order.total, terminal.orderCurrency.value) }}</span>
          <span>{{ terminal.t('pos.positionsShort', { count: order.positions }) }}</span>
          <span>{{ order.duration }}</span>
        </button>
      </div>
      <PosEmptyState v-if="!activeOrderGroups.length" :label="terminal.t('pos.noActiveOrder')" />
    </aside>
  </section>
</template>

<script setup lang="ts">
import { computed, nextTick } from 'vue';

import { PosBanner, PosEmptyState, PosSkeleton } from '../../shared/ui';
import BlockingNotice from './BlockingNotice.vue';
import type { CashierTerminal } from './useCashierTerminal';

type TableStatus = 'free' | 'open' | 'precheck' | 'paid' | 'unavailable';

type TableCard = {
  id: string;
  name: string;
  status: TableStatus;
  orderNo?: string;
  guests?: number;
  total?: number;
  duration?: string;
};

type ActiveOrderCard = {
  id: string;
  number: string;
  tableId: string;
  table: string;
  total: number;
  positions: number;
  duration: string;
  status: 'open' | 'precheck';
};

const props = defineProps<{
  terminal: CashierTerminal;
}>();

const emit = defineEmits<{
  (event: 'open-orders'): void;
}>();

const tableCards = computed<TableCard[]>(() => props.terminal.activeTables.value.map((table) => {
  const order = props.terminal.activeOrders.value.find((item) => item.table_id === table.id);
  if (!order) {
    return { id: table.id, name: table.name, status: table.active ? 'free' : 'unavailable' };
  }

  const hasCheck = Boolean(order.check);
  return {
    id: table.id,
    name: table.name,
    status: hasCheck ? 'paid' : order.status === 'locked' ? 'precheck' : 'open',
    orderNo: props.terminal.shortId(order.id),
    guests: order.guest_count,
    total: order.total,
    duration: durationFrom(order.opened_at),
  };
}));

const activeOrderGroups = computed(() => {
  const hallName = props.terminal.activeHalls.value.find((hall) => hall.id === props.terminal.selectedHallId.value)?.name ?? props.terminal.t('pos.currentHall');
  const orders: ActiveOrderCard[] = [];

  for (const order of props.terminal.activeOrders.value) {
    orders.push({
      id: order.id,
      number: props.terminal.shortId(order.id),
      tableId: order.table_id,
      table: order.table_name,
      total: order.total,
      positions: order.lines.filter((line) => line.status === 'active').length,
      duration: durationFrom(order.opened_at),
      status: order.status === 'locked' ? 'precheck' : 'open',
    });
  }

  return orders.length ? [{ hall: hallName, orders }] : [];
});


function openTable(card: TableCard) {
  props.terminal.selectTable(card.id);
  if (card.status === 'free') {
    void nextTick(() => {
      if (!props.terminal.canCreateOrder.value) return;
      props.terminal.createOrderMutation.mutate();
      emit('open-orders');
    });
    return;
  }
  emit('open-orders');
}

function openOrder(tableId: string) {
  props.terminal.selectTable(tableId);
  emit('open-orders');
}

function durationFrom(value: string) {
  const opened = new Date(value).getTime();
  if (Number.isNaN(opened)) return '';
  const minutes = Math.max(1, Math.round((Date.now() - opened) / 60000));
  if (minutes < 60) return `${minutes} ${props.terminal.t('pos.minutesShort')}`;
  return `${Math.floor(minutes / 60)} ${props.terminal.t('pos.hoursShort')} ${minutes % 60} ${props.terminal.t('pos.minutesShort')}`;
}
</script>
