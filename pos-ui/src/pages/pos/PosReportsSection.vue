<template>
  <section class="reports-workspace section-workspace">
    <main class="section-main-surface" :aria-label="terminal.t('pos.reportsTitle')">
      <PosSectionHeader
        :eyebrow="terminal.t('pos.sections.reports')"
        :title="terminal.t('pos.reportsTitle')"
        :refresh-label="terminal.t('actions.retry')"
        @refresh="refreshReports"
      />

      <div class="dashboard-metric-grid">
        <PosMetricCard :label="terminal.t('pos.closedOrders')" :value="closedOrderCount" :hint="terminal.t('pos.operationalDataOnly')" tone="primary" />
        <PosMetricCard :label="terminal.t('pos.total')" :value="terminal.money(closedOrdersTotal, reportCurrency)" :hint="terminal.t('pos.closedOrdersSummary')" />
        <PosMetricCard :label="terminal.t('pos.syncStatus')" :value="terminal.syncProblems.value" :hint="terminal.syncProblems.value > 0 ? terminal.t('pos.syncFailed') : terminal.t('status.sent')" :tone="terminal.syncProblems.value > 0 ? 'warning' : 'neutral'" />
      </div>

      <section class="integrated-panel">
        <div class="section-head slim">
          <h2>{{ terminal.t('pos.paymentSummary') }}</h2>
        </div>
        <div class="payment-summary-grid">
          <div v-for="item in paymentTotals" :key="item.method" class="payment-summary-cell">
            <span>{{ terminal.t(`pos.paymentMethods.${item.method}`) }}</span>
            <strong>{{ terminal.money(item.total, reportCurrency) }}</strong>
          </div>
        </div>
      </section>

      <section class="integrated-panel">
        <div class="section-head slim">
          <h2>{{ terminal.t('pos.shiftReadiness') }}</h2>
        </div>
        <div class="status-strip-grid">
          <PosStatusStrip :label="terminal.t('pos.shift')" :value="terminal.currentShift.data.value ? terminal.t('status.open') : terminal.t('pos.noShift')" :tone="terminal.currentShift.data.value ? 'good' : 'neutral'" large />
          <PosStatusStrip :label="terminal.t('pos.cashSession')" :value="terminal.currentCashSession.data.value ? terminal.t('status.open') : terminal.t('pos.noCashSession')" :tone="terminal.currentCashSession.data.value ? 'good' : 'neutral'" large />
          <PosStatusStrip :label="terminal.t('pos.syncPending')" :value="terminal.syncStatus.data.value?.pending ?? 0" :tone="terminal.syncProblems.value > 0 ? 'warning' : terminal.canViewSync.value ? 'good' : 'neutral'" large />
        </div>
      </section>
    </main>

    <aside class="section-action-rail" :aria-label="terminal.t('pos.reportScope')">
      <div class="rail-head">
        <div>
          <p class="eyebrow">{{ terminal.t('pos.currentData') }}</p>
          <h2>{{ terminal.t('pos.reportScope') }}</h2>
        </div>
      </div>
      <div class="rail-summary">
        <div>
          <span>{{ terminal.t('pos.cashSession') }}</span>
          <strong>{{ terminal.currentCashSession.data.value ? terminal.t('status.open') : terminal.t('pos.noCashSession') }}</strong>
        </div>
        <div>
          <span>{{ terminal.t('pos.lastOutbox') }}</span>
          <strong>{{ terminal.syncStatus.data.value?.total ?? 0 }}</strong>
        </div>
      </div>
      <div class="rail-actions integrated-action-bar">
        <PosButton variant="secondary" mode="outline" icon="inventory_2" :label="terminal.t('pos.cashDrawer')" :disabled="!terminal.canRecordCashDrawerEvent.value" @click="terminal.cashDrawerDialog.value = true" />
        <PosButton v-if="terminal.canViewSync.value" variant="secondary" mode="outline" icon="sync" :label="terminal.t('pos.syncStatus')" @click="terminal.syncDrawer.value = true" />
      </div>
    </aside>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue';

import { PosButton, PosMetricCard, PosSectionHeader, PosStatusStrip } from '../../shared/ui';
import type { CashierTerminal } from './useCashierTerminal';

const props = defineProps<{
  terminal: CashierTerminal;
}>();

const closedOrderCount = computed(() => props.terminal.closedOrders.data.value?.length ?? 0);
const reportCurrency = computed(() => props.terminal.closedOrders.data.value?.find((order) => order.check?.currency_code)?.check?.currency_code ?? props.terminal.currency.value);
const closedOrdersTotal = computed(() => (props.terminal.closedOrders.data.value ?? []).reduce((sum, order) => sum + order.total, 0));
const paymentTotals = computed(() => {
  const totals = new Map<string, number>([
    ['cash', 0],
    ['card', 0],
    ['other', 0],
  ]);
  for (const order of props.terminal.closedOrders.data.value ?? []) {
    for (const payment of order.check?.payments ?? []) {
      totals.set(payment.method, (totals.get(payment.method) ?? 0) + payment.amount);
    }
  }
  return Array.from(totals, ([method, total]) => ({ method, total }));
});

function refreshReports() {
  void props.terminal.closedOrders.refetch();
  props.terminal.refreshOps();
  props.terminal.refreshSync();
}
</script>
