<template>
  <div class="cloud-panel-stack">
    <cloud-safe-error-banner :ctx="ctx" target="olapExports" />
    <section class="cloud-panel cloud-table-panel">
      <div class="section-head">
        <div>
          <p class="eyebrow">{{ t('cloud.readiness.olap.status') }}</p>
          <h2>{{ t('cloud.resources.olapExports') }}</h2>
          <p>{{ t('cloud.readiness.olap.copy') }}</p>
        </div>
        <q-btn flat icon="refresh" :loading="ctx.isLoading('olap-operator')" :label="t('actions.refresh')" @click="ctx.loadOlapOperatorSurface()" />
      </div>

      <div class="cloud-signal-row">
        <span>{{ t('cloud.readiness.olap.signals.readOnly') }}</span>
        <span>{{ t('cloud.readiness.olap.signals.noRetry') }}</span>
        <span>{{ t('cloud.readiness.olap.signals.noRawPayload') }}</span>
      </div>

      <div v-if="ctx.olapExportStatuses.value.length" class="cloud-state-grid olap-status-grid">
        <div v-for="status in ctx.olapExportStatuses.value" :key="status.stream">
          <span>{{ streamLabel(status.stream) }}</span>
          <strong>{{ t('cloud.readiness.olap.statusLabels.checkpoint') }}: {{ ctx.safeOperationalValue(status.last_checkpoint || '-') }}</strong>
          <small>{{ t('cloud.readiness.olap.statusLabels.lastExported') }}: {{ lastExportedLabel(status) }}</small>
          <small>{{ t('cloud.readiness.olap.statusLabels.queue') }}: {{ status.pending_count }} / {{ status.processing_count }} / {{ status.failed_count }}</small>
          <small>{{ t('cloud.readiness.olap.statusLabels.retry') }}: {{ retryLabel(status) }}</small>
        </div>
      </div>
      <div v-else-if="ctx.isLoading('olap-operator')" class="empty-state wide">
        {{ t('common.loading') }}
      </div>
      <div v-else class="empty-state wide">
        {{ t('cloud.readiness.olap.emptyStatus') }}
      </div>

      <div class="olap-filter-grid">
        <q-input v-model="ctx.olapFilters.businessDateFrom" dense outlined mask="####-##-##" :label="t('cloud.readiness.olap.filters.businessDateFrom')" />
        <q-input v-model="ctx.olapFilters.businessDateTo" dense outlined mask="####-##-##" :label="t('cloud.readiness.olap.filters.businessDateTo')" />
        <q-select
          v-if="catalogItemOptions.length"
          v-model="ctx.olapFilters.catalogItemId"
          dense
          outlined
          emit-value
          map-options
          clearable
          :options="catalogItemOptions"
          :label="t('cloud.readiness.olap.filters.catalogItem')"
        />
        <q-input v-else v-model="ctx.olapFilters.catalogItemId" dense outlined :label="t('cloud.readiness.olap.filters.catalogItem')" />
        <q-select
          v-if="warehouseOptions.length"
          v-model="ctx.olapFilters.warehouseId"
          dense
          outlined
          emit-value
          map-options
          clearable
          :options="warehouseOptions"
          :label="t('cloud.readiness.olap.filters.warehouse')"
        />
        <q-input v-else v-model="ctx.olapFilters.warehouseId" dense outlined :label="t('cloud.readiness.olap.filters.warehouse')" />
        <q-input v-model="ctx.olapFilters.sourceEventType" dense outlined :label="t('cloud.readiness.olap.filters.sourceEventType')" />
        <q-select
          v-model="ctx.olapFilters.groupBy"
          dense
          outlined
          emit-value
          map-options
          :options="groupByOptions"
          :label="t('cloud.readiness.olap.filters.groupBy')"
        />
        <q-btn color="primary" unelevated icon="search" :label="t('cloud.readiness.olap.applyFilters')" :loading="ctx.isLoading('olap-operator')" @click="ctx.loadOlapOperatorSurface()" />
      </div>

      <div class="section-head compact">
        <div>
          <p class="eyebrow">{{ t('cloud.readiness.olap.movesEyebrow') }}</p>
          <h3>{{ t('cloud.readiness.olap.movesTitle') }}</h3>
        </div>
      </div>
      <div v-if="ctx.olapStockMoves.value.length === 0" class="empty-state wide">
        {{ t('cloud.readiness.olap.emptyMoves') }}
      </div>
      <template v-else>
        <div class="cloud-table-wrap">
          <table class="cloud-table">
            <thead>
              <tr>
                <th>{{ t('cloud.fields.business_date_local') }}</th>
                <th>{{ t('cloud.fields.source_event_type') }}</th>
                <th>{{ t('cloud.fields.catalog_item_id') }}</th>
                <th>{{ t('cloud.fields.warehouse_id') }}</th>
                <th>{{ t('cloud.fields.movement_type') }}</th>
                <th>{{ t('cloud.fields.quantity') }}</th>
                <th>{{ t('cloud.fields.costing_status') }}</th>
                <th>{{ t('cloud.fields.occurred_at') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="move in ctx.olapStockMoves.value" :key="move.ledger_entry_id">
                <td>{{ move.business_date_local }}</td>
                <td>{{ ctx.safeOperationalValue(move.source_event_type) }}</td>
                <td>{{ catalogItemLabel(move.catalog_item_id) }}</td>
                <td>{{ ctx.safeOperationalValue(move.warehouse_id || '-') }}</td>
                <td>{{ ctx.safeOperationalValue(move.movement_type) }}</td>
                <td>{{ move.quantity }} {{ move.unit_code }}</td>
                <td><span class="status-pill" :data-status="move.costing_status">{{ move.costing_status }}</span></td>
                <td>{{ ctx.formatDate(move.occurred_at) }}</td>
              </tr>
            </tbody>
          </table>
        </div>
        <div class="edge-event-card-list" :aria-label="t('cloud.readiness.olap.movesTitle')">
          <article v-for="move in ctx.olapStockMoves.value" :key="move.ledger_entry_id" class="edge-event-card">
            <span class="cloud-status published">{{ ctx.safeOperationalValue(move.movement_type) }}</span>
            <strong>{{ move.quantity }} {{ move.unit_code }}</strong>
            <small>{{ t('cloud.fields.business_date_local') }}: {{ move.business_date_local }}</small>
            <small>{{ t('cloud.fields.catalog_item_id') }}: {{ catalogItemLabel(move.catalog_item_id) }}</small>
            <small>{{ t('cloud.fields.warehouse_id') }}: {{ ctx.safeOperationalValue(move.warehouse_id || '-') }}</small>
            <small>{{ t('cloud.fields.source_event_type') }}: {{ ctx.safeOperationalValue(move.source_event_type) }}</small>
          </article>
        </div>
      </template>

      <div class="section-head compact">
        <div>
          <p class="eyebrow">{{ t('cloud.readiness.olap.summaryEyebrow') }}</p>
          <h3>{{ t('cloud.readiness.olap.summaryTitle') }}</h3>
        </div>
      </div>
      <div v-if="ctx.olapStockMoveSummary.value.length === 0" class="empty-state wide">
        {{ t('cloud.readiness.olap.emptySummary') }}
      </div>
      <template v-else>
        <div class="cloud-table-wrap">
          <table class="cloud-table">
            <thead>
              <tr>
                <th>{{ t('cloud.readiness.olap.columns.group') }}</th>
                <th>{{ t('cloud.readiness.olap.columns.moves') }}</th>
                <th>{{ t('cloud.readiness.olap.columns.in') }}</th>
                <th>{{ t('cloud.readiness.olap.columns.out') }}</th>
                <th>{{ t('cloud.readiness.olap.columns.net') }}</th>
                <th>{{ t('cloud.readiness.olap.columns.first') }}</th>
                <th>{{ t('cloud.readiness.olap.columns.last') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="item in ctx.olapStockMoveSummary.value" :key="`${item.group_by}:${item.group_key}`">
                <td>{{ summaryGroupLabel(item) }}</td>
                <td>{{ item.move_count }}</td>
                <td>{{ item.in_quantity }}</td>
                <td>{{ item.out_quantity }}</td>
                <td>{{ item.net_quantity }}</td>
                <td>{{ ctx.formatDate(item.first_occurred_at || '') }}</td>
                <td>{{ ctx.formatDate(item.last_occurred_at || '') }}</td>
              </tr>
            </tbody>
          </table>
        </div>
        <div class="edge-event-card-list" :aria-label="t('cloud.readiness.olap.summaryTitle')">
          <article v-for="item in ctx.olapStockMoveSummary.value" :key="`${item.group_by}:${item.group_key}`" class="edge-event-card">
            <span class="cloud-status published">{{ groupByLabel(item.group_by) }}</span>
            <strong>{{ summaryGroupLabel(item) }}</strong>
            <small>{{ t('cloud.readiness.olap.columns.moves') }}: {{ item.move_count }}</small>
            <small>{{ t('cloud.readiness.olap.columns.in') }}: {{ item.in_quantity }}</small>
            <small>{{ t('cloud.readiness.olap.columns.out') }}: {{ item.out_quantity }}</small>
            <small>{{ t('cloud.readiness.olap.columns.net') }}: {{ item.net_quantity }}</small>
          </article>
        </div>
      </template>

      <ul class="cloud-gap-list">
        <li v-for="item in gaps" :key="item">{{ t(item) }}</li>
      </ul>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';

import CloudSafeErrorBanner from './CloudSafeErrorBanner.vue';
import type { OlapExportStatus, OlapStockMoveSummary } from '../../shared/schemas';

const { t } = useI18n();

const props = defineProps<{
  ctx: Record<string, any>;
}>();

const catalogItemOptions = computed(() => (props.ctx.scopedRows.catalogItems ?? []).map((item: Record<string, unknown>) => ({
  label: `${String(item.name ?? item.sku ?? item.id)} (${props.ctx.safeOperationalValue(String(item.id ?? ''))})`,
  value: String(item.id ?? ''),
})));

const warehouseOptions = computed(() => {
  const ids = new Set<string>();
  for (const item of props.ctx.scopedRows.stopLists ?? []) {
    if (item.warehouse_id) ids.add(String(item.warehouse_id));
  }
  for (const item of props.ctx.inventoryStockBalances.value ?? []) {
    if (item.warehouse_id) ids.add(String(item.warehouse_id));
  }
  for (const item of props.ctx.olapStockMoves.value ?? []) {
    if (item.warehouse_id) ids.add(String(item.warehouse_id));
  }
  return [...ids].map((id) => ({ label: props.ctx.safeOperationalValue(id), value: id }));
});

const groupByOptions = computed(() => [
  { label: t('cloud.readiness.olap.groupBy.business_date'), value: 'business_date' },
  { label: t('cloud.readiness.olap.groupBy.catalog_item'), value: 'catalog_item' },
  { label: t('cloud.readiness.olap.groupBy.warehouse'), value: 'warehouse' },
]);

const gaps = [
  'cloud.readiness.olap.gaps.retryControl',
  'cloud.readiness.olap.gaps.backfill',
  'cloud.readiness.olap.gaps.analytics',
];

function streamLabel(stream: string) {
  return t(`cloud.readiness.olap.streams.${stream}`);
}

function lastExportedLabel(status: OlapExportStatus) {
  const id = status.last_exported_id ? props.ctx.safeOperationalValue(status.last_exported_id) : '-';
  const at = status.last_exported_at ? props.ctx.formatDate(status.last_exported_at) : '-';
  return `${id} / ${at}`;
}

function retryLabel(status: OlapExportStatus) {
  if (status.retry_blocked) return t('cloud.readiness.olap.retry.blocked');
  if (status.next_retry_at) return props.ctx.formatDate(status.next_retry_at);
  if (status.consecutive_failures > 0) return t('cloud.readiness.olap.retry.waiting', { count: status.consecutive_failures });
  return t('cloud.readiness.olap.retry.ready');
}

function catalogItemLabel(id: string) {
  const item = (props.ctx.scopedRows.catalogItems ?? []).find((row: Record<string, unknown>) => row.id === id);
  if (!item) return props.ctx.safeOperationalValue(id);
  return `${String(item.name ?? id)} (${props.ctx.safeOperationalValue(id)})`;
}

function groupByLabel(groupBy: string) {
  return t(`cloud.readiness.olap.groupBy.${groupBy}`);
}

function summaryGroupLabel(item: OlapStockMoveSummary) {
  if (item.group_by === 'catalog_item') return catalogItemLabel(item.catalog_item_id || item.group_key);
  if (item.group_by === 'warehouse') return props.ctx.safeOperationalValue(item.warehouse_id || item.group_key);
  return item.business_date_local || item.group_key;
}
</script>
