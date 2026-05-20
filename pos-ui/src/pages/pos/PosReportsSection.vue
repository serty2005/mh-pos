<template>
  <section class="reports-workspace section-workspace">
    <main class="section-main-surface" :aria-label="terminal.t('pos.reportsTitle')">
      <div class="pos-section-head">
        <div>
          <p class="eyebrow">{{ terminal.t('pos.sections.reports') }}</p>
          <h1>{{ terminal.t('pos.reportsTitle') }}</h1>
        </div>
        <q-btn flat round icon="refresh" class="icon-touch" :aria-label="terminal.t('actions.retry')" @click="refreshReports" />
      </div>

      <div class="dashboard-metric-grid">
        <article class="dashboard-metric primary">
          <span>{{ terminal.t('pos.closedOrders') }}</span>
          <strong>{{ closedOrderCount }}</strong>
          <small>{{ terminal.t('pos.operationalDataOnly') }}</small>
        </article>
        <article class="dashboard-metric">
          <span>{{ terminal.t('pos.total') }}</span>
          <strong>{{ terminal.money(closedOrdersTotal, reportCurrency) }}</strong>
          <small>{{ terminal.t('pos.closedOrdersSummary') }}</small>
        </article>
        <article class="dashboard-metric" :class="{ warning: terminal.syncProblems.value > 0 }">
          <span>{{ terminal.t('pos.syncStatus') }}</span>
          <strong>{{ terminal.syncProblems.value }}</strong>
          <small>{{ terminal.syncProblems.value > 0 ? terminal.t('pos.syncFailed') : terminal.t('status.sent') }}</small>
        </article>
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
          <div class="status-strip large" :class="{ good: terminal.currentShift.data.value }">
            <span>{{ terminal.t('pos.shift') }}</span>
            <strong>{{ terminal.currentShift.data.value ? terminal.t('status.open') : terminal.t('pos.noShift') }}</strong>
          </div>
          <div class="status-strip large" :class="{ good: terminal.currentCashSession.data.value }">
            <span>{{ terminal.t('pos.cashSession') }}</span>
            <strong>{{ terminal.currentCashSession.data.value ? terminal.t('status.open') : terminal.t('pos.noCashSession') }}</strong>
          </div>
          <div class="status-strip large" :class="{ good: terminal.canViewSync.value && terminal.syncProblems.value === 0, warning: terminal.syncProblems.value > 0 }">
            <span>{{ terminal.t('pos.syncPending') }}</span>
            <strong>{{ terminal.syncStatus.data.value?.pending ?? 0 }}</strong>
          </div>
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
        <q-btn outline color="secondary" class="touch-button" icon="inventory_2" :label="terminal.t('pos.cashDrawer')" :disable="!terminal.canRecordCashDrawerEvent.value" @click="terminal.cashDrawerDialog.value = true" />
        <q-btn v-if="terminal.canViewSync.value" outline color="secondary" class="touch-button" icon="sync" :label="terminal.t('pos.syncStatus')" @click="terminal.syncDrawer.value = true" />
      </div>
    </aside>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue';

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
