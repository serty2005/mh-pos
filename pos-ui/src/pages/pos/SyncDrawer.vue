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
          <q-btn flat round icon="refresh" class="icon-touch" :aria-label="terminal.t('actions.retry')" @click="terminal.refreshSync" />
          <q-btn flat round icon="close" class="icon-touch" :aria-label="terminal.t('actions.close')" @click="terminal.syncDrawer.value = false" />
        </div>
      </div>

      <q-banner v-if="terminal.syncStatus.isError.value" class="error-banner dense-banner" rounded>{{ terminal.t('common.error') }}</q-banner>
      <div v-else class="sync-grid">
        <div class="sync-metric">
          <span>{{ terminal.t('pos.syncPending') }}</span>
          <strong>{{ terminal.syncStatus.data.value?.pending ?? 0 }}</strong>
        </div>
        <div class="sync-metric" :class="{ active: terminal.syncProblems.value > 0 }">
          <span>{{ terminal.t('pos.syncFailed') }}</span>
          <strong>{{ terminal.syncProblems.value }}</strong>
        </div>
        <div class="sync-metric">
          <span>{{ terminal.t('pos.syncSent') }}</span>
          <strong>{{ terminal.syncStatus.data.value?.sent ?? 0 }}</strong>
        </div>
      </div>

      <q-btn
        v-if="terminal.canRetrySync.value"
        outline
        color="secondary"
        class="touch-button"
        icon="sync"
        :label="terminal.t('actions.retrySync')"
        :disable="!terminal.syncProblems.value"
        :loading="terminal.retrySyncMutation.isPending.value"
        @click="terminal.retrySyncMutation.mutate()"
      />

      <div class="event-feed">
        <p class="eyebrow">{{ terminal.t('pos.lastOutbox') }}</p>
        <div v-for="item in terminal.syncOutbox.data.value ?? []" :key="item.id" class="event-row">
          <strong>{{ item.command_type }}</strong>
          <span>{{ terminal.statusLabel(item.status) }} · #{{ item.sequence_no }}</span>
        </div>
      </div>

      <q-input v-model="terminal.localEventFilter.value" dense outlined clearable :label="terminal.t('pos.eventFilter')" />
      <div class="event-feed">
        <p class="eyebrow">{{ terminal.t('pos.localEvents') }}</p>
        <div v-for="item in terminal.localEvents.data.value ?? []" :key="item.id" class="event-row">
          <strong>{{ item.event_type }}</strong>
          <span>{{ terminal.formatDate(item.occurred_at) }} · {{ terminal.shortId(item.aggregate_id) }}</span>
        </div>
      </div>
    </section>
  </q-drawer>
</template>

<script setup lang="ts">
import type { CashierTerminal } from './useCashierTerminal';

defineProps<{
  terminal: CashierTerminal;
}>();
</script>
