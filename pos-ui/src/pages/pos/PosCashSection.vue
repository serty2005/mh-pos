<template>
  <section class="cash-workspace section-workspace">
    <main class="section-main-surface" :aria-label="terminal.t('pos.cashTitle')">
      <div class="pos-section-head">
        <div>
          <p class="eyebrow">{{ terminal.t('pos.sections.cash') }}</p>
          <h1>{{ terminal.t('pos.cashTitle') }}</h1>
        </div>
        <q-btn flat round icon="refresh" class="icon-touch" :aria-label="terminal.t('actions.retry')" @click="terminal.refreshOps" />
      </div>

      <q-banner v-if="terminal.statusError.value" class="error-banner dense-banner" rounded>{{ terminal.statusError.value }}</q-banner>

      <div class="cash-status-strip">
        <div class="status-strip large" :class="{ good: terminal.currentShift.data.value }">
          <span>{{ terminal.t('pos.shift') }}</span>
          <strong>{{ terminal.currentShift.data.value ? terminal.t('status.open') : terminal.t('pos.noShift') }}</strong>
        </div>
        <div class="status-strip large" :class="{ good: terminal.currentCashSession.data.value }">
          <span>{{ terminal.t('pos.cashSession') }}</span>
          <strong>{{ terminal.currentCashSession.data.value ? terminal.t('status.open') : terminal.t('pos.noCashSession') }}</strong>
        </div>
        <div class="status-strip large" :class="{ good: terminal.syncProblems.value === 0, warning: terminal.syncProblems.value > 0 }">
          <span>{{ terminal.t('pos.syncStatus') }}</span>
          <strong>{{ terminal.syncProblems.value > 0 ? terminal.syncProblems.value : terminal.t('status.sent') }}</strong>
        </div>
      </div>

      <div class="cash-operation-grid">
        <section class="integrated-panel">
          <div class="section-head slim">
            <h2>{{ terminal.t('pos.shift') }}</h2>
          </div>
          <p>{{ terminal.t('pos.employeeShiftBody') }}</p>
          <q-btn
            v-if="!terminal.currentShift.data.value"
            color="primary"
            unelevated
            class="touch-button primary-action"
            icon="schedule"
            :label="terminal.t('actions.openShift')"
            :disable="!terminal.canOpenShift.value"
            :loading="terminal.openShiftMutation.isPending.value"
            @click="terminal.openShiftMutation.mutate()"
          />
          <q-btn
            v-else
            outline
            color="secondary"
            class="touch-button"
            icon="event_busy"
            :label="terminal.t('actions.closeShift')"
            :disable="!terminal.canCloseShift.value"
            :loading="terminal.closeShiftMutation.isPending.value"
            @click="terminal.closeShiftMutation.mutate(terminal.currentShift.data.value.id)"
          />
        </section>

        <section class="integrated-panel">
          <div class="section-head slim">
            <h2>{{ terminal.t('pos.cashSession') }}</h2>
          </div>
          <p>{{ terminal.t('pos.cashSessionBody') }}</p>
          <div v-if="terminal.currentShift.data.value && !terminal.currentCashSession.data.value" class="cash-form-row">
            <q-input v-model.number="terminal.openingCashAmount.value" outlined type="number" min="0" :step="terminal.currencyInputStep(terminal.currency.value)" :label="terminal.t('common.amount')" :suffix="terminal.currency.value" />
            <q-btn color="primary" unelevated class="touch-button primary-action" icon="point_of_sale" :label="terminal.t('actions.openCashSession')" :disable="!terminal.canOpenCashSession.value" :loading="terminal.openCashMutation.isPending.value" @click="terminal.openCashMutation.mutate(terminal.openingCashAmount.value)" />
          </div>
          <div v-if="terminal.currentCashSession.data.value" class="cash-form-row">
            <q-input v-model.number="terminal.closingCashAmount.value" outlined type="number" min="0" :step="terminal.currencyInputStep(terminal.currency.value)" :label="terminal.t('common.amount')" :suffix="terminal.currency.value" />
            <q-btn outline color="secondary" class="touch-button" icon="payments" :label="terminal.t('actions.closeCashSession')" :disable="!terminal.canCloseCashSession.value" :loading="terminal.closeCashMutation.isPending.value" @click="terminal.closeCashMutation.mutate({ cashSessionId: terminal.currentCashSession.data.value.id, amount: terminal.closingCashAmount.value })" />
          </div>
        </section>

        <section class="integrated-panel">
          <div class="section-head slim">
            <h2>{{ terminal.t('pos.syncStatus') }}</h2>
          </div>
          <div class="sync-grid compact-sync-grid">
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
        </section>
      </div>
    </main>

    <aside class="section-action-rail cash-action-rail" :aria-label="terminal.t('pos.terminalActions')">
      <div class="rail-head">
        <div>
          <p class="eyebrow">{{ terminal.t('pos.primaryOperations') }}</p>
          <h2>{{ terminal.t('pos.terminalActions') }}</h2>
        </div>
      </div>
      <div class="rail-actions integrated-action-bar">
        <q-btn color="primary" unelevated class="touch-button primary-action" icon="inventory_2" :label="terminal.t('pos.cashDrawer')" :disable="!terminal.canRecordCashDrawerEvent.value" @click="terminal.cashDrawerDialog.value = true" />
        <q-btn v-if="terminal.canViewSync.value" outline color="secondary" class="touch-button" icon="sync" :label="terminal.t('pos.syncStatus')" @click="terminal.syncDrawer.value = true" />
        <q-btn v-if="terminal.canRetrySync.value" outline color="secondary" class="touch-button" icon="published_with_changes" :label="terminal.t('actions.retrySync')" :disable="!terminal.syncProblems.value" :loading="terminal.retrySyncMutation.isPending.value" @click="terminal.retrySyncMutation.mutate()" />
      </div>
      <div class="planned-block">
        <p class="eyebrow">{{ terminal.t('pos.supportOperations') }}</p>
        <p>{{ terminal.t('pos.cashPlannedBody') }}</p>
      </div>
      <div class="rail-actions integrated-action-bar">
        <q-btn outline color="secondary" class="touch-button" icon="lock" :label="terminal.t('actions.lock')" @click="terminal.lockTerminal" />
        <q-btn outline color="secondary" class="touch-button" icon="logout" :label="terminal.t('actions.logout')" :loading="terminal.logoutMutation.isPending.value" @click="terminal.logoutMutation.mutate()" />
      </div>
    </aside>
  </section>
</template>

<script setup lang="ts">
import type { CashierTerminal } from './useCashierTerminal';

defineProps<{
  terminal: CashierTerminal;
}>();
</script>
