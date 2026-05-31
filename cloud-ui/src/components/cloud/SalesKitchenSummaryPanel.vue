<template>
  <div class="cloud-panel-stack">
    <cloud-safe-error-banner :ctx="ctx" target="salesKitchenSummary" />
    <div class="cloud-panel cloud-table-panel">
      <div class="section-head">
        <div>
          <p class="eyebrow">{{ t('cloud.reporting.salesKitchen.status') }}</p>
          <h2>{{ t('cloud.reporting.salesKitchen.title') }}</h2>
          <p>{{ t('cloud.reporting.salesKitchen.description') }}</p>
        </div>
        <q-btn flat icon="refresh" :loading="ctx.isLoading('sales-kitchen-summary')" :label="t('actions.refresh')" @click="ctx.loadSalesKitchenSummary()" />
      </div>

      <div class="sales-kitchen-filter-grid">
        <q-input v-model="ctx.salesKitchenSummaryFilters.businessDateFrom" dense outlined :label="t('cloud.reporting.salesKitchen.filters.businessDateFrom')" mask="####-##-##" />
        <q-input v-model="ctx.salesKitchenSummaryFilters.businessDateTo" dense outlined :label="t('cloud.reporting.salesKitchen.filters.businessDateTo')" mask="####-##-##" />
        <q-select
          v-model="ctx.salesKitchenSummaryFilters.groupBy"
          dense
          outlined
          emit-value
          map-options
          :label="t('cloud.reporting.salesKitchen.filters.groupBy')"
          :options="groupByOptions"
        />
        <q-btn color="primary" unelevated icon="search" :label="t('cloud.reporting.salesKitchen.applyFilters')" :loading="ctx.isLoading('sales-kitchen-summary')" @click="ctx.loadSalesKitchenSummary()" />
      </div>

      <div class="cloud-signal-row">
        <span>{{ t('cloud.reporting.salesKitchen.signals.readOnly') }}</span>
        <span>{{ t('cloud.reporting.salesKitchen.signals.noRawPayload') }}</span>
        <span>{{ t('cloud.reporting.salesKitchen.signals.noCostingBi') }}</span>
        <span>{{ t('cloud.reporting.salesKitchen.signals.noCharts') }}</span>
      </div>

      <div v-if="ctx.salesKitchenSummaryRows.value.length === 0" class="empty-state wide">
        {{ t('cloud.reporting.salesKitchen.empty') }}
      </div>
      <template v-else>
        <div class="cloud-table-wrap">
          <table class="cloud-table">
            <thead>
              <tr>
                <th>{{ t('cloud.reporting.salesKitchen.columns.group') }}</th>
                <th>{{ t('cloud.fields.business_date_local') }}</th>
                <th>{{ t('cloud.fields.event_type') }}</th>
                <th>{{ t('cloud.fields.source_event_type') }}</th>
                <th>{{ t('cloud.fields.catalog_item_id') }}</th>
                <th>{{ t('cloud.reporting.salesKitchen.columns.events') }}</th>
                <th>{{ t('cloud.reporting.salesKitchen.columns.stockMoves') }}</th>
                <th>{{ t('cloud.reporting.salesKitchen.columns.quantities') }}</th>
                <th>{{ t('cloud.reporting.salesKitchen.columns.stockAmount') }}</th>
                <th>{{ t('cloud.reporting.salesKitchen.columns.period') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="item in ctx.salesKitchenSummaryRows.value" :key="`${item.group_by}:${item.group_key}`">
                <td>
                  <span class="cloud-status published">{{ groupByLabel(item.group_by) }}</span>
                  <small class="cloud-muted-cell">{{ formatGroupKey(item.group_key) }}</small>
                </td>
                <td>{{ ctx.formatCell('business_date_local', item.business_date_local) }}</td>
                <td>{{ ctx.formatCell('event_type', item.event_type) }}</td>
                <td>{{ ctx.formatCell('source_event_type', item.source_event_type) }}</td>
                <td>{{ ctx.formatCell('catalog_item_id', item.catalog_item_id) }}</td>
                <td>{{ eventCounts(item) }}</td>
                <td>{{ item.stock_move_count }}</td>
                <td>{{ quantitySummary(item) }}</td>
                <td>{{ item.total_cost_minor }}</td>
                <td>{{ periodSummary(item) }}</td>
              </tr>
            </tbody>
          </table>
        </div>

        <div class="edge-event-card-list" :aria-label="t('cloud.reporting.salesKitchen.title')">
          <article v-for="item in ctx.salesKitchenSummaryRows.value" :key="`${item.group_by}:${item.group_key}`" class="edge-event-card">
            <span class="cloud-status published">{{ groupByLabel(item.group_by) }}</span>
            <strong>{{ formatGroupKey(item.group_key) }}</strong>
            <small>{{ t('cloud.fields.business_date_local') }}: {{ ctx.formatCell('business_date_local', item.business_date_local) }}</small>
            <small>{{ t('cloud.reporting.salesKitchen.columns.events') }}: {{ eventCounts(item) }}</small>
            <small>{{ t('cloud.reporting.salesKitchen.columns.stockMoves') }}: {{ item.stock_move_count }}</small>
            <small>{{ t('cloud.reporting.salesKitchen.columns.quantities') }}: {{ quantitySummary(item) }}</small>
            <small>{{ t('cloud.reporting.salesKitchen.columns.period') }}: {{ periodSummary(item) }}</small>
          </article>
        </div>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';

import CloudSafeErrorBanner from './CloudSafeErrorBanner.vue';
import type { SalesKitchenSummaryItem } from '../../shared/schemas';

const { t } = useI18n();

defineProps<{
  ctx: Record<string, any>;
}>();

const groupByOptions = computed(() => [
  { label: t('cloud.reporting.salesKitchen.groupBy.businessDate'), value: 'business_date' },
  { label: t('cloud.reporting.salesKitchen.groupBy.eventType'), value: 'event_type' },
  { label: t('cloud.reporting.salesKitchen.groupBy.sourceEventType'), value: 'source_event_type' },
  { label: t('cloud.reporting.salesKitchen.groupBy.catalogItem'), value: 'catalog_item' },
]);

function groupByLabel(value: string) {
  if (value === 'business_date') return t('cloud.reporting.salesKitchen.groupBy.businessDate');
  if (value === 'event_type') return t('cloud.reporting.salesKitchen.groupBy.eventType');
  if (value === 'source_event_type') return t('cloud.reporting.salesKitchen.groupBy.sourceEventType');
  if (value === 'catalog_item') return t('cloud.reporting.salesKitchen.groupBy.catalogItem');
  return value;
}

function formatGroupKey(value: string) {
  return value.trim() || '-';
}

function eventCounts(item: SalesKitchenSummaryItem) {
  return t('cloud.reporting.salesKitchen.counts.events', {
    total: item.event_count,
    sales: item.sale_event_count,
    kitchen: item.kitchen_event_count,
  });
}

function quantitySummary(item: SalesKitchenSummaryItem) {
  return t('cloud.reporting.salesKitchen.counts.quantities', {
    out: item.out_quantity,
    in: item.in_quantity,
    net: item.net_quantity,
  });
}

function periodSummary(item: SalesKitchenSummaryItem) {
  const first = item.first_occurred_at ? new Date(item.first_occurred_at) : null;
  const last = item.last_occurred_at ? new Date(item.last_occurred_at) : null;
  const format = (value: Date | null, fallback: string) => {
    if (!value || Number.isNaN(value.getTime())) return fallback;
    return new Intl.DateTimeFormat('ru-RU', { dateStyle: 'short', timeStyle: 'short' }).format(value);
  };
  return t('cloud.reporting.salesKitchen.counts.period', {
    first: format(first, '-'),
    last: format(last, '-'),
  });
}
</script>
