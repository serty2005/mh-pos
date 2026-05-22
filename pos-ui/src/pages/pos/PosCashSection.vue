<template>
  <section class="cash-workspace section-workspace">
    <main class="section-main-surface" :aria-label="terminal.t('pos.cashTitle')">
      <PosSectionHeader
        :eyebrow="terminal.t('pos.sections.cash')"
        :title="terminal.t('pos.cashTitle')"
        :refresh-label="terminal.t('actions.retry')"
        @refresh="terminal.refreshOps"
      />

      <PosBanner v-if="terminal.statusError.value" tone="error" :label="terminal.statusError.value" />

      <div class="cash-status-strip">
        <PosStatusStrip :label="terminal.t('pos.shift')" :value="terminal.currentShift.data.value ? terminal.t('status.open') : terminal.t('pos.noShift')" :tone="terminal.currentShift.data.value ? 'good' : 'neutral'" large />
        <PosStatusStrip :label="terminal.t('pos.cashSession')" :value="terminal.currentCashSession.data.value ? terminal.t('status.open') : terminal.t('pos.noCashSession')" :tone="terminal.currentCashSession.data.value ? 'good' : 'neutral'" large />
        <PosStatusStrip :label="terminal.t('pos.syncStatus')" :value="terminal.syncProblems.value > 0 ? terminal.syncProblems.value : terminal.t('status.sent')" :tone="terminal.syncProblems.value > 0 ? 'warning' : 'good'" large />
      </div>

      <div class="cash-operation-grid">
        <section class="integrated-panel">
          <div class="section-head slim">
            <h2>{{ terminal.t('pos.shift') }}</h2>
          </div>
          <p>{{ terminal.t('pos.employeeShiftBody') }}</p>
          <PosButton
            v-if="!terminal.currentShift.data.value"
            variant="primary"
            primary
            icon="schedule"
            :label="terminal.t('actions.openShift')"
            :disabled="!terminal.canOpenShift.value"
            :loading="terminal.openShiftMutation.isPending.value"
            @click="terminal.openShiftMutation.mutate()"
          />
          <PosButton
            v-else
            variant="secondary"
            mode="outline"
            icon="event_busy"
            :label="terminal.t('actions.closeShift')"
            :disabled="!terminal.canCloseShift.value"
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
            <PosButton variant="primary" primary icon="point_of_sale" :label="terminal.t('actions.openCashSession')" :disabled="!terminal.canOpenCashSession.value" :loading="terminal.openCashMutation.isPending.value" @click="terminal.openCashMutation.mutate(terminal.openingCashAmount.value)" />
          </div>
          <div v-if="terminal.currentCashSession.data.value" class="cash-form-row">
            <q-input v-model.number="terminal.closingCashAmount.value" outlined type="number" min="0" :step="terminal.currencyInputStep(terminal.currency.value)" :label="terminal.t('common.amount')" :suffix="terminal.currency.value" />
            <PosButton variant="secondary" mode="outline" icon="payments" :label="terminal.t('actions.closeCashSession')" :disabled="!terminal.canCloseCashSession.value" :loading="terminal.closeCashMutation.isPending.value" @click="terminal.closeCashMutation.mutate({ cashSessionId: terminal.currentCashSession.data.value.id, amount: terminal.closingCashAmount.value })" />
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

  </section>
</template>

<script setup lang="ts">
import { PosBanner, PosButton, PosSectionHeader, PosStatusStrip } from '../../shared/ui';
import type { CashierTerminal } from './useCashierTerminal';

defineProps<{
  terminal: CashierTerminal;
}>();
</script>
