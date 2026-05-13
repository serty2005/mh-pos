<template>
  <aside class="control-pane floor-pane" :aria-label="terminal.t('pos.floorPlan')">
    <div class="pane-scroll">
      <div class="section-head">
        <div>
          <p class="eyebrow">{{ terminal.t('pos.halls') }}</p>
          <h2>{{ terminal.t('pos.tables') }}</h2>
        </div>
        <q-btn flat round icon="refresh" class="icon-touch" :aria-label="terminal.t('actions.retry')" @click="terminal.refreshOps" />
      </div>

      <q-banner v-if="terminal.statusError.value" class="error-banner dense-banner" rounded>{{ terminal.statusError.value }}</q-banner>

      <q-skeleton v-if="terminal.halls.isPending.value" class="skeleton-row" />
      <q-banner v-else-if="terminal.halls.isError.value" class="error-banner dense-banner" rounded>{{ terminal.t('common.error') }}</q-banner>
      <nav v-else-if="terminal.activeHalls.value.length" class="hall-tabs" :aria-label="terminal.t('pos.halls')">
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

      <div v-if="terminal.tables.isPending.value" class="table-list">
        <q-skeleton v-for="n in 8" :key="n" class="table-button skeleton-tile" />
      </div>
      <q-banner v-else-if="terminal.tables.isError.value" class="error-banner dense-banner" rounded>{{ terminal.t('common.error') }}</q-banner>
      <div v-else-if="terminal.activeTables.value.length" class="table-list">
        <button
          v-for="table in terminal.activeTables.value"
          :key="table.id"
          class="table-button"
          :class="{ selected: table.id === terminal.selectedTableId.value }"
          type="button"
          :aria-pressed="table.id === terminal.selectedTableId.value"
          @click="terminal.selectTable(table.id)"
        >
          <span>{{ table.name }}</span>
          <small>{{ terminal.t('pos.guestCount') }} {{ table.seats }}</small>
        </button>
      </div>
      <div v-else class="empty-state small">{{ terminal.t('pos.noTables') }}</div>
    </div>
  </aside>
</template>

<script setup lang="ts">
import type { CashierTerminal } from './useCashierTerminal';

defineProps<{
  terminal: CashierTerminal;
}>();
</script>
