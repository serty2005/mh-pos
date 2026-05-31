<template>
  <section class="cloud-panel cloud-plan-panel">
    <div class="section-head stacked">
      <p class="eyebrow">{{ t('cloud.readiness.inventory.status') }}</p>
      <h2>{{ t('cloud.resources.inventoryReadiness') }}</h2>
      <p class="cloud-copy">{{ t('cloud.readiness.inventory.copy') }}</p>
    </div>
    <cloud-safe-error-banner :ctx="ctx" target="inventory-readiness" />
    <div class="inventory-filter-grid">
      <q-select
        v-if="warehouseOptions.length"
        v-model="filters.warehouseId"
        dense
        outlined
        emit-value
        map-options
        clearable
        :options="warehouseOptions"
        :label="t('cloud.readiness.inventory.filters.warehouse')"
      />
      <q-input
        v-else
        v-model="filters.warehouseId"
        dense
        outlined
        :label="t('cloud.readiness.inventory.filters.warehouse')"
      />
      <q-select
        v-if="catalogItemOptions.length"
        v-model="filters.catalogItemId"
        dense
        outlined
        emit-value
        map-options
        clearable
        :options="catalogItemOptions"
        :label="t('cloud.readiness.inventory.filters.catalogItem')"
      />
      <q-input
        v-else
        v-model="filters.catalogItemId"
        dense
        outlined
        :label="t('cloud.readiness.inventory.filters.catalogItem')"
      />
      <q-input
        v-model="filters.businessDateTo"
        dense
        outlined
        mask="####-##-##"
        :label="t('cloud.readiness.inventory.filters.businessDateTo')"
      />
      <q-select
        v-model="filters.costingStatus"
        dense
        outlined
        emit-value
        map-options
        :options="costingStatusOptions"
        :label="t('cloud.readiness.inventory.filters.costingStatus')"
      />
      <q-btn
        color="primary"
        unelevated
        icon="refresh"
        :label="t('actions.refresh')"
        :loading="ctx.activeLoading.value"
        @click="refreshBalances"
      />
    </div>
    <div v-if="ctx.stopListReadiness.value" class="cloud-state-grid readiness-signal-grid">
      <div>
        <span>{{ t('cloud.readiness.inventory.signals.policy') }}</span>
        <strong>{{ ctx.stopListReadiness.value.default_conflict_policy }}</strong>
      </div>
      <div>
        <span>{{ t('cloud.readiness.inventory.signals.projection') }}</span>
        <strong>{{ ctx.stopListReadiness.value.projection_mode }}</strong>
      </div>
      <div>
        <span>{{ t('cloud.readiness.inventory.signals.stopLists') }}</span>
        <strong>{{ ctx.stopListReadiness.value.active_stop_list_entries }} / {{ ctx.stopListReadiness.value.total_stop_list_entries }}</strong>
      </div>
      <div>
        <span>{{ t('cloud.readiness.inventory.signals.packageAck') }}</span>
        <strong>{{ ctx.stopListReadiness.value.package_ack_status }}</strong>
      </div>
      <div>
        <span>{{ t('cloud.readiness.inventory.signals.publication') }}</span>
        <strong>{{ publicationLabel }}</strong>
      </div>
      <div>
        <span>{{ t('cloud.readiness.inventory.signals.edgeAck') }}</span>
        <strong>{{ edgeAckLabel }}</strong>
      </div>
      <div>
        <span>{{ t('cloud.readiness.inventory.signals.syncProblems') }}</span>
        <strong>{{ ctx.stopListReadiness.value.problem_events.total }}</strong>
      </div>
    </div>
    <div v-if="ctx.activeLoading.value" class="empty-state wide">
      {{ t('common.loading') }}
    </div>
    <div v-else-if="!ctx.inventoryStockBalances.value.length" class="empty-state wide">
      {{ t('cloud.readiness.inventory.emptyBalances') }}
    </div>
    <div v-else class="cloud-table-wrap">
      <table class="cloud-table">
        <thead>
          <tr>
            <th>{{ t('cloud.readiness.inventory.columns.warehouse') }}</th>
            <th>{{ t('cloud.readiness.inventory.columns.catalogItem') }}</th>
            <th>{{ t('cloud.readiness.inventory.columns.quantity') }}</th>
            <th>{{ t('cloud.readiness.inventory.columns.costingStatus') }}</th>
            <th>{{ t('cloud.readiness.inventory.columns.lastMovement') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="item in ctx.inventoryStockBalances.value" :key="`${item.restaurant_id}:${item.warehouse_id}:${item.catalog_item_id}:${item.unit_code}`">
            <td>{{ item.warehouse_id || '-' }}</td>
            <td>{{ ctx.safeOperationalValue(item.catalog_item_id) }}</td>
            <td>{{ item.quantity_on_hand }} {{ item.unit_code }}</td>
            <td>
              <span class="status-pill" :data-status="item.costing_status">
                {{ t(`cloud.readiness.inventory.costingStatuses.${item.costing_status}`) }}
              </span>
            </td>
            <td>{{ ctx.formatDate(item.last_movement_at) }}</td>
          </tr>
        </tbody>
      </table>
    </div>
    <ul class="cloud-gap-list">
      <li v-for="item in gaps" :key="item">{{ t(item) }}</li>
    </ul>
  </section>
</template>

<script setup lang="ts">
import { computed, reactive } from 'vue';
import { useI18n } from 'vue-i18n';

import CloudSafeErrorBanner from './CloudSafeErrorBanner.vue';

const { t } = useI18n();
const props = defineProps<{
  ctx: Record<string, any>;
}>();
const filters = reactive({
  warehouseId: '',
  catalogItemId: '',
  businessDateTo: '',
  costingStatus: '',
});
const costingStatusOptions = computed(() => [
  { label: t('cloud.readiness.inventory.costingStatuses.all'), value: '' },
  { label: t('cloud.readiness.inventory.costingStatuses.final'), value: 'final' },
  { label: t('cloud.readiness.inventory.costingStatuses.estimated'), value: 'estimated' },
  { label: t('cloud.readiness.inventory.costingStatuses.needs_recalculation'), value: 'needs_recalculation' },
  { label: t('cloud.readiness.inventory.costingStatuses.mixed'), value: 'mixed' },
  { label: t('cloud.readiness.inventory.costingStatuses.unknown'), value: 'unknown' },
]);
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
  return [...ids].map((id) => ({ label: props.ctx.safeOperationalValue(id), value: id }));
});
const gaps = [
  'cloud.readiness.inventory.gaps.stockDocuments',
  'cloud.readiness.inventory.gaps.costingEngine',
  'cloud.readiness.inventory.gaps.saleBlocking',
  'cloud.readiness.inventory.gaps.review',
];
async function refreshBalances() {
  await props.ctx.loadInventoryStockBalances({
    warehouseId: String(filters.warehouseId ?? '').trim(),
    catalogItemId: String(filters.catalogItemId ?? '').trim(),
    businessDateTo: String(filters.businessDateTo ?? '').trim(),
    costingStatus: String(filters.costingStatus ?? '').trim(),
  });
}
const publicationLabel = computed(() => {
  const publication = props.ctx.stopListReadiness.value?.latest_publication;
  return publication ? `${publication.version} / ${props.ctx.formatDate(publication.published_at)}` : '-';
});
const edgeAckLabel = computed(() => {
  const ack = props.ctx.stopListReadiness.value?.latest_stop_list_edge_ack;
  return ack ? `${ack.status} / ${props.ctx.safeOperationalValue(ack.event_id)}` : '-';
});
</script>
