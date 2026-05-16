<template>
  <section class="hall-orders-screen">
    <main class="hall-workspace" :aria-label="terminal.t('pos.floorPlan')">
      <q-banner v-if="terminal.statusError.value" class="error-banner dense-banner">{{ terminal.statusError.value }}</q-banner>

      <div v-if="terminal.tables.isPending.value" class="floor-table-grid">
        <q-skeleton v-for="n in 15" :key="n" class="floor-table-tile skeleton-tile" />
      </div>
      <q-banner v-else-if="terminal.tables.isError.value" class="error-banner dense-banner">{{ terminal.t('common.error') }}</q-banner>
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
      <div v-else class="empty-state small">{{ terminal.t('pos.noTables') }}</div>
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
      <div v-if="!activeOrderGroups.length" class="empty-state">{{ terminal.t('pos.noActiveOrder') }}</div>
    </aside>
  </section>
</template>

<script setup lang="ts">
import { computed, nextTick } from 'vue';

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

const tableCards = computed<TableCard[]>(() => props.terminal.activeTables.value.map((table, index) => {
  if (table.id === props.terminal.selectedTableId.value && props.terminal.activeOrder.value) {
    return {
      id: table.id,
      name: table.name,
      status: props.terminal.activePrecheck.value ? 'precheck' : props.terminal.finalCheckData.value ? 'paid' : 'open',
      orderNo: props.terminal.shortId(props.terminal.activeOrder.value.id),
      guests: props.terminal.activeOrder.value.guest_count,
      total: props.terminal.activeOrder.value.total,
      duration: durationFrom(props.terminal.activeOrder.value.opened_at),
    };
  }

  const status = mockStatus(index);
  return {
    id: table.id,
    name: table.name,
    status,
    orderNo: status === 'free' || status === 'unavailable' ? undefined : String(102 + index),
    guests: status === 'free' || status === 'unavailable' ? undefined : 2 + (index % 3),
    total: status === 'free' || status === 'unavailable' ? undefined : 125000 + index * 39000,
    duration: status === 'free' || status === 'unavailable' ? undefined : `${14 + index * 7} ${props.terminal.t('pos.minutesShort')}`,
  };
}));

const activeOrderGroups = computed(() => {
  const hallName = props.terminal.activeHalls.value.find((hall) => hall.id === props.terminal.selectedHallId.value)?.name ?? props.terminal.t('pos.currentHall');
  const orders: ActiveOrderCard[] = [];

  if (props.terminal.activeOrder.value && props.terminal.selectedTable.value) {
    orders.push({
      id: props.terminal.activeOrder.value.id,
      number: props.terminal.shortId(props.terminal.activeOrder.value.id),
      tableId: props.terminal.selectedTable.value.id,
      table: props.terminal.selectedTable.value.name,
      total: props.terminal.activeOrder.value.total,
      positions: props.terminal.activeLines.value.length,
      duration: durationFrom(props.terminal.activeOrder.value.opened_at),
      status: props.terminal.activePrecheck.value ? 'precheck' : 'open',
    });
  }

  for (const card of tableCards.value.filter((item) => item.status === 'open' || item.status === 'precheck').slice(0, 5)) {
    if (orders.some((order) => order.tableId === card.id)) continue;
    orders.push({
      id: `mock-${card.id}`,
      number: card.orderNo ?? props.terminal.shortId(card.id),
      tableId: card.id,
      table: card.name,
      total: card.total ?? 0,
      positions: 2 + (orders.length % 4),
      duration: card.duration ?? '',
      status: card.status === 'precheck' ? 'precheck' : 'open',
    });
  }

  return orders.length ? [{ hall: hallName, orders }] : [];
});

function mockStatus(index: number): TableStatus {
  const statuses: TableStatus[] = ['free', 'open', 'precheck', 'free', 'open', 'free', 'open', 'precheck', 'open', 'free', 'free', 'precheck', 'free', 'paid', 'precheck'];
  return statuses[index % statuses.length];
}

function openTable(card: TableCard) {
  props.terminal.selectTable(card.id);
  if (card.status === 'free') {
    void nextTick(() => {
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
