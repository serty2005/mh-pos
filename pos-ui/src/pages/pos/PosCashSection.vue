<template>
  <section class="simple-pos-section">
    <div class="pos-section-head">
      <div>
        <p class="eyebrow">{{ terminal.t('pos.sections.cash') }}</p>
        <h1>{{ terminal.t('pos.cashTitle') }}</h1>
      </div>
      <q-btn flat round icon="refresh" class="icon-touch" :aria-label="terminal.t('actions.retry')" @click="terminal.refreshOps" />
    </div>

    <q-banner v-if="terminal.statusError.value" class="error-banner dense-banner" rounded>{{ terminal.statusError.value }}</q-banner>

    <div class="cash-section-grid">
      <section class="cash-section-panel">
        <h2>{{ terminal.t('pos.shift') }}</h2>
        <p>{{ terminal.currentShift.data.value ? terminal.t('status.open') : terminal.t('pos.noShift') }}</p>
        <q-btn
          v-if="!terminal.currentShift.data.value"
          color="primary"
          unelevated
          class="touch-button"
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

      <section class="cash-section-panel">
        <h2>{{ terminal.t('pos.cashSession') }}</h2>
        <p>{{ terminal.currentCashSession.data.value ? terminal.t('status.open') : terminal.t('pos.noCashSession') }}</p>
        <div v-if="terminal.currentShift.data.value && !terminal.currentCashSession.data.value" class="cash-form-row">
          <q-input v-model.number="terminal.openingCashAmount.value" outlined type="number" min="0" :step="terminal.currencyInputStep(terminal.currency.value)" :label="terminal.t('common.amount')" :suffix="terminal.currency.value" />
          <q-btn color="primary" unelevated class="touch-button" icon="point_of_sale" :label="terminal.t('actions.openCashSession')" :disable="!terminal.canOpenCashSession.value" :loading="terminal.openCashMutation.isPending.value" @click="terminal.openCashMutation.mutate(terminal.openingCashAmount.value)" />
        </div>
        <div v-if="terminal.currentCashSession.data.value" class="cash-form-row">
          <q-input v-model.number="terminal.closingCashAmount.value" outlined type="number" min="0" :step="terminal.currencyInputStep(terminal.currency.value)" :label="terminal.t('common.amount')" :suffix="terminal.currency.value" />
          <q-btn outline color="secondary" class="touch-button" icon="payments" :label="terminal.t('actions.closeCashSession')" :disable="!terminal.canCloseCashSession.value" :loading="terminal.closeCashMutation.isPending.value" @click="terminal.closeCashMutation.mutate({ cashSessionId: terminal.currentCashSession.data.value.id, amount: terminal.closingCashAmount.value })" />
        </div>
      </section>

      <section class="cash-section-panel">
        <h2>{{ terminal.t('pos.terminalActions') }}</h2>
        <div class="supported-action-list">
          <q-btn outline color="secondary" class="touch-button" icon="inventory_2" :label="terminal.t('pos.cashDrawer')" :disable="!terminal.canRecordCashDrawerEvent.value" @click="terminal.cashDrawerDialog.value = true" />
          <q-btn v-if="terminal.canViewSync.value" outline color="secondary" class="touch-button" icon="sync" :label="terminal.t('pos.syncStatus')" @click="terminal.syncDrawer.value = true" />
          <q-btn outline color="secondary" class="touch-button" icon="lock" :label="terminal.t('actions.lock')" @click="terminal.lockTerminal" />
        </div>
      </section>
    </div>
  </section>
</template>

<script setup lang="ts">
import type { CashierTerminal } from './useCashierTerminal';

defineProps<{
  terminal: CashierTerminal;
}>();
</script>
