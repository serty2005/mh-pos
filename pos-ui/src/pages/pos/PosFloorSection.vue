<template>
  <section class="pos-floor-screen">
    <main class="floor-main" :aria-label="terminal.t('pos.floorPlan')">
      <div class="pos-section-head">
        <div>
          <p class="eyebrow">{{ terminal.t('pos.sections.floor') }}</p>
          <h1>{{ terminal.t('pos.tables') }}</h1>
        </div>
        <q-btn flat round icon="refresh" class="icon-touch" :aria-label="terminal.t('actions.retry')" @click="terminal.refreshOps" />
      </div>

      <q-banner v-if="terminal.statusError.value" class="error-banner dense-banner" rounded>{{ terminal.statusError.value }}</q-banner>

      <q-skeleton v-if="terminal.halls.isPending.value" class="skeleton-row" />
      <q-banner v-else-if="terminal.halls.isError.value" class="error-banner dense-banner" rounded>{{ terminal.t('common.error') }}</q-banner>
      <nav v-else-if="terminal.activeHalls.value.length" class="hall-tabs floor-hall-tabs" :aria-label="terminal.t('pos.halls')">
        <button
          v-for="hall in terminal.activeHalls.value"
          :key="hall.id"
          class="hall-chip"
          :class="{ selected: hall.id === terminal.selectedHallId.value }"
          type="button"
          :aria-pressed="hall.id === terminal.selectedHallId.value"
          @click="terminal.selectHall(hall.id)"
        >
          {{ hall.name }}
        </button>
      </nav>
      <div v-else class="empty-state small">{{ terminal.t('pos.noHalls') }}</div>

      <div v-if="terminal.tables.isPending.value" class="floor-table-grid">
        <q-skeleton v-for="n in 12" :key="n" class="floor-table-tile skeleton-tile" />
      </div>
      <q-banner v-else-if="terminal.tables.isError.value" class="error-banner dense-banner" rounded>{{ terminal.t('common.error') }}</q-banner>
      <div v-else-if="terminal.activeTables.value.length" class="floor-table-grid">
        <button
          v-for="table in terminal.activeTables.value"
          :key="table.id"
          class="floor-table-tile"
          :class="{ selected: table.id === terminal.selectedTableId.value }"
          type="button"
          :aria-pressed="table.id === terminal.selectedTableId.value"
          @click="selectTable(table.id)"
        >
          <span>{{ table.name }}</span>
          <small>{{ terminal.t('pos.guestCount') }} {{ table.seats }}</small>
        </button>
      </div>
      <div v-else class="empty-state small">{{ terminal.t('pos.noTables') }}</div>
    </main>

    <aside class="floor-order-rail" :aria-label="terminal.t('pos.activeOrders')">
      <div class="rail-head">
        <div>
          <p class="eyebrow">{{ terminal.t('pos.sections.orders') }}</p>
          <h2>{{ terminal.t('pos.activeOrders') }}</h2>
        </div>
      </div>
      <article v-if="terminal.activeOrder.value" class="floor-active-order" @click="$emit('open-orders')">
        <span>{{ terminal.selectedTable.value?.name ?? terminal.t('pos.table') }}</span>
        <strong>{{ terminal.money(terminal.activeOrder.value.total, terminal.orderCurrency.value) }}</strong>
        <small>{{ terminal.statusLabel(terminal.activeOrder.value.status) }} · {{ terminal.shortId(terminal.activeOrder.value.id) }}</small>
      </article>
      <div v-else class="empty-state">{{ terminal.selectedTableId.value ? terminal.t('pos.noActiveOrder') : terminal.t('pos.chooseTable') }}</div>
      <q-btn
        color="primary"
        unelevated
        class="touch-button primary-action"
        icon="receipt_long"
        :label="terminal.activeOrder.value ? terminal.t('pos.openOrder') : terminal.t('actions.createOrder')"
        :disable="terminal.activeOrder.value ? false : !terminal.canCreateOrder.value"
        :loading="terminal.createOrderMutation.isPending.value"
        @click="openOrCreateOrder"
      />
    </aside>
  </section>
</template>

<script setup lang="ts">
import type { CashierTerminal } from './useCashierTerminal';

const props = defineProps<{
  terminal: CashierTerminal;
}>();

const emit = defineEmits<{
  (event: 'open-orders'): void;
}>();

function selectTable(id: string) {
  props.terminal.selectTable(id);
  emit('open-orders');
}

function openOrCreateOrder() {
  if (props.terminal.activeOrder.value) {
    emit('open-orders');
    return;
  }
  props.terminal.createOrderMutation.mutate();
  emit('open-orders');
}
</script>
