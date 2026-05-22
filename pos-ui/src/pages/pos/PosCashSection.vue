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
        <PosPanel :title="terminal.t('pos.shift')">
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
        </PosPanel>

        <PosPanel :title="terminal.t('pos.cashSession')">
          <p>{{ terminal.t('pos.cashSessionBody') }}</p>
          <PosFormRow v-if="terminal.currentShift.data.value && !terminal.currentCashSession.data.value">
            <q-input v-model.number="terminal.openingCashAmount.value" outlined type="number" min="0" :step="terminal.currencyInputStep(terminal.currency.value)" :label="terminal.t('common.amount')" :suffix="terminal.currency.value" />
            <PosButton variant="primary" primary icon="point_of_sale" :label="terminal.t('actions.openCashSession')" :disabled="!terminal.canOpenCashSession.value" :loading="terminal.openCashMutation.isPending.value" @click="terminal.openCashMutation.mutate(terminal.openingCashAmount.value)" />
          </PosFormRow>
          <PosFormRow v-if="terminal.currentCashSession.data.value">
            <q-input v-model.number="terminal.closingCashAmount.value" outlined type="number" min="0" :step="terminal.currencyInputStep(terminal.currency.value)" :label="terminal.t('common.amount')" :suffix="terminal.currency.value" />
            <PosButton variant="secondary" mode="outline" icon="payments" :label="terminal.t('actions.closeCashSession')" :disabled="!terminal.canCloseCashSession.value" :loading="terminal.closeCashMutation.isPending.value" @click="terminal.closeCashMutation.mutate({ cashSessionId: terminal.currentCashSession.data.value.id, amount: terminal.closingCashAmount.value })" />
          </PosFormRow>
        </PosPanel>

        <PosPanel :title="terminal.t('pos.syncStatus')">
          <div class="sync-grid compact-sync-grid">
            <PosMetricCard size="compact" :label="terminal.t('pos.syncPending')" :value="terminal.syncStatus.data.value?.pending ?? 0" />
            <PosMetricCard size="compact" :label="terminal.t('pos.syncFailed')" :value="terminal.syncProblems.value" :tone="terminal.syncProblems.value > 0 ? 'warning' : 'neutral'" />
            <PosMetricCard size="compact" :label="terminal.t('pos.syncSent')" :value="terminal.syncStatus.data.value?.sent ?? 0" />
          </div>
        </PosPanel>
      </div>
    </main>

  </section>
</template>

<script setup lang="ts">
import { PosBanner, PosButton, PosFormRow, PosMetricCard, PosPanel, PosSectionHeader, PosStatusStrip } from '../../shared/ui';
import type { CashierTerminal } from './useCashierTerminal';

defineProps<{
  terminal: CashierTerminal;
}>();
</script>
