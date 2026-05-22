<template>
  <q-drawer
    :model-value="terminal.syncDrawer.value"
    side="right"
    overlay
    bordered
    :width="460"
    class="utility-drawer"
    @update:model-value="terminal.syncDrawer.value = $event"
  >
    <section class="drawer-body">
      <div class="section-head">
        <div>
          <p class="eyebrow">{{ terminal.t('pos.managerTools') }}</p>
          <h2>{{ terminal.t('pos.syncStatus') }}</h2>
        </div>
        <div class="drawer-actions">
          <PosButton variant="neutral" mode="flat" round dense compact icon="refresh" class="icon-touch" :aria-label="terminal.t('actions.retry')" @click="terminal.refreshSync" />
          <PosButton variant="neutral" mode="flat" round dense compact icon="close" class="icon-touch" :aria-label="terminal.t('actions.close')" @click="terminal.syncDrawer.value = false" />
        </div>
      </div>

      <PosBanner v-if="terminal.syncStatus.isError.value" tone="error" :label="terminal.t('common.error')" />
      <div v-else class="sync-grid">
        <PosMetricCard size="compact" :label="terminal.t('pos.syncPending')" :value="terminal.syncStatus.data.value?.pending ?? 0" />
        <PosMetricCard size="compact" :label="terminal.t('pos.syncFailed')" :value="terminal.syncProblems.value" :tone="terminal.syncProblems.value > 0 ? 'warning' : 'neutral'" />
        <PosMetricCard size="compact" :label="terminal.t('pos.syncSent')" :value="terminal.syncStatus.data.value?.sent ?? 0" />
      </div>

      <PosButton
        v-if="terminal.canRetrySync.value"
        variant="secondary"
        mode="outline"
        icon="sync"
        :label="terminal.t('actions.retrySync')"
        :disabled="!terminal.syncProblems.value"
        :loading="terminal.retrySyncMutation.isPending.value"
        @click="terminal.retrySyncMutation.mutate()"
      />

      <div class="event-feed">
        <p class="eyebrow">{{ terminal.t('pos.lastOutbox') }}</p>
        <PosDataRow v-for="item in terminal.syncOutbox.data.value ?? []" :key="item.id" :label="item.command_type" :meta="`${terminal.statusLabel(item.status)} · #${item.sequence_no}`" />
      </div>

      <q-input v-model="terminal.localEventFilter.value" dense outlined clearable :label="terminal.t('pos.eventFilter')" />
      <div class="event-feed">
        <p class="eyebrow">{{ terminal.t('pos.localEvents') }}</p>
        <PosDataRow v-for="item in terminal.localEvents.data.value ?? []" :key="item.id" :label="item.event_type" :meta="`${terminal.formatDate(item.occurred_at)} · ${terminal.shortId(item.aggregate_id)}`" />
      </div>
    </section>
  </q-drawer>
</template>

<script setup lang="ts">
import { PosBanner, PosButton, PosDataRow, PosMetricCard } from '../../shared/ui';
import type { CashierTerminal } from './useCashierTerminal';

defineProps<{
  terminal: CashierTerminal;
}>();
</script>
