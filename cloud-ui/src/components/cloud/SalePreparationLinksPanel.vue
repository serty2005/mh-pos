<template>
  <div class="cloud-panel-stack">
    <cloud-safe-error-banner :ctx="ctx" target="salePreparationLinks" />
    <div class="cloud-panel cloud-table-panel">
      <div class="section-head">
        <div>
          <p class="eyebrow">{{ t('cloud.salePreparation.status') }}</p>
          <h2>{{ t('cloud.salePreparation.title') }}</h2>
          <p>{{ t('cloud.salePreparation.description') }}</p>
        </div>
        <q-btn flat icon="refresh" :loading="ctx.isLoading('sale-preparation-links')" :label="t('actions.refresh')" @click="ctx.loadSalePreparationLinks()" />
      </div>

      <div class="sale-preparation-filter-grid">
        <q-input v-model="query" dense outlined :label="t('cloud.salePreparation.filters.search')" />
        <q-select
          v-model="statusFilter"
          dense
          outlined
          emit-value
          map-options
          :label="t('cloud.salePreparation.filters.status')"
          :options="statusOptions"
        />
        <q-toggle v-model="onlyIncomplete" :label="t('cloud.salePreparation.filters.onlyIncomplete')" />
      </div>

      <div class="cloud-signal-row">
        <span>{{ t('cloud.salePreparation.signals.readOnly') }}</span>
        <span>{{ t('cloud.salePreparation.signals.cloudMasterDataOnly') }}</span>
        <span>{{ t('cloud.salePreparation.signals.noCashierCommands') }}</span>
        <span>{{ t('cloud.salePreparation.signals.noRawPayload') }}</span>
      </div>

      <div v-if="ctx.isLoading('sale-preparation-links')" class="empty-state wide">
        {{ t('cloud.salePreparation.loading') }}
      </div>
      <div v-else-if="rows.length === 0" class="empty-state wide">
        {{ t('cloud.salePreparation.empty') }}
      </div>
      <template v-else>
        <p v-if="isBounded" class="cloud-muted-note">
          {{ t('cloud.salePreparation.boundedNote', { shown: boundedRows.length, total: rows.length }) }}
        </p>
        <div class="cloud-table-wrap">
          <table class="cloud-table">
            <thead>
              <tr>
                <th>{{ t('cloud.salePreparation.columns.menuItem') }}</th>
                <th>{{ t('cloud.salePreparation.columns.catalogItem') }}</th>
                <th>{{ t('cloud.salePreparation.columns.modifiers') }}</th>
                <th>{{ t('cloud.salePreparation.columns.pricing') }}</th>
                <th>{{ t('cloud.salePreparation.columns.readiness') }}</th>
                <th>{{ t('cloud.salePreparation.columns.hint') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="row in boundedRows" :key="row.menuItem.id">
                <td>
                  <strong>{{ row.menuItem.name }}</strong>
                  <small class="cloud-muted-cell">{{ safeId(row.menuItem.id) }}</small>
                  <span class="cloud-status" :class="row.menuItem.status">{{ statusText(row.menuItem.status) }}</span>
                </td>
                <td>
                  <template v-if="row.catalogItem">
                    <strong>{{ row.catalogItem.name }}</strong>
                    <small class="cloud-muted-cell">{{ catalogMeta(row.catalogItem) }}</small>
                    <span class="cloud-status" :class="row.catalogItem.status">{{ statusText(row.catalogItem.status) }}</span>
                  </template>
                  <template v-else>
                    <span class="cloud-status archived">{{ t('cloud.salePreparation.readiness.missing') }}</span>
                    <small class="cloud-muted-cell">{{ safeId(row.menuItem.catalog_item_id) }}</small>
                  </template>
                </td>
                <td>{{ modifierSummary(row) }}</td>
                <td>{{ pricingSummary }}</td>
                <td><span class="cloud-status" :class="statusClass(row.readiness)">{{ readinessLabel(row.readiness) }}</span></td>
                <td>{{ hint(row) }}</td>
              </tr>
            </tbody>
          </table>
        </div>

        <div class="edge-event-card-list" :aria-label="t('cloud.salePreparation.title')">
          <article v-for="row in boundedRows" :key="row.menuItem.id" class="edge-event-card">
            <span class="cloud-status" :class="statusClass(row.readiness)">{{ readinessLabel(row.readiness) }}</span>
            <strong>{{ row.menuItem.name }}</strong>
            <small>{{ t('cloud.salePreparation.columns.menuItem') }}: {{ safeId(row.menuItem.id) }} / {{ statusText(row.menuItem.status) }}</small>
            <small>{{ t('cloud.salePreparation.columns.catalogItem') }}: {{ row.catalogItem ? `${row.catalogItem.name} / ${statusText(row.catalogItem.status)}` : safeId(row.menuItem.catalog_item_id) }}</small>
            <small>{{ t('cloud.salePreparation.columns.modifiers') }}: {{ modifierSummary(row) }}</small>
            <small>{{ t('cloud.salePreparation.columns.pricing') }}: {{ pricingSummary }}</small>
            <small>{{ t('cloud.salePreparation.columns.hint') }}: {{ hint(row) }}</small>
          </article>
        </div>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue';
import { useI18n } from 'vue-i18n';

import CloudSafeErrorBanner from './CloudSafeErrorBanner.vue';
import type { CatalogItem, MenuItem, ModifierBinding, ModifierGroup, PricingPolicy } from '../../shared/schemas';

type Readiness = 'ready' | 'warning' | 'missing';

type SalePreparationRow = {
  menuItem: MenuItem;
  catalogItem?: CatalogItem;
  directMenuBindings: ModifierBinding[];
  directCatalogBindings: ModifierBinding[];
  readiness: Readiness;
  reason: string;
};

const props = defineProps<{
  ctx: Record<string, any>;
}>();

const { t } = useI18n();
const query = ref('');
const statusFilter = ref<'all' | Readiness>('all');
const onlyIncomplete = ref(false);
const previewLimit = 50;

const statusOptions = computed(() => [
  { label: t('cloud.salePreparation.filters.all'), value: 'all' },
  { label: t('cloud.salePreparation.readiness.ready'), value: 'ready' },
  { label: t('cloud.salePreparation.readiness.warning'), value: 'warning' },
  { label: t('cloud.salePreparation.readiness.missing'), value: 'missing' },
]);

const catalogItems = computed<CatalogItem[]>(() => props.ctx.scopedRows.catalogItems ?? []);
const menuItems = computed<MenuItem[]>(() => props.ctx.scopedRows.menuItems ?? []);
const modifierGroups = computed<ModifierGroup[]>(() => props.ctx.scopedRows.modifierGroups ?? []);
const modifierBindings = computed<ModifierBinding[]>(() => props.ctx.scopedRows.modifierBindings ?? []);
const pricingPolicies = computed<PricingPolicy[]>(() => props.ctx.scopedRows.pricingPolicies ?? []);

const catalogById = computed(() => new Map(catalogItems.value.map((item) => [item.id, item])));
const modifierGroupById = computed(() => new Map(modifierGroups.value.map((item) => [item.id, item])));
const publishedPricingPolicies = computed(() => pricingPolicies.value.filter((item) => item.status === 'published'));
const pricingSummary = computed(() => t('cloud.salePreparation.summaries.pricing', {
  published: publishedPricingPolicies.value.length,
  total: pricingPolicies.value.length,
}));

const rows = computed(() => {
  const needle = query.value.trim().toLowerCase();
  return menuItems.value
    .map((menuItem): SalePreparationRow => {
      const catalogItem = catalogById.value.get(menuItem.catalog_item_id);
      const directMenuBindings = publishedBindings('menu_item', menuItem.id);
      const directCatalogBindings = catalogItem ? publishedBindings('catalog_item', catalogItem.id) : [];
      const readiness = evaluateReadiness(menuItem, catalogItem, directMenuBindings, directCatalogBindings);
      return {
        menuItem,
        catalogItem,
        directMenuBindings,
        directCatalogBindings,
        readiness: readiness.status,
        reason: readiness.reason,
      };
    })
    .filter((row) => {
      if (statusFilter.value !== 'all' && row.readiness !== statusFilter.value) return false;
      if (onlyIncomplete.value && row.readiness === 'ready') return false;
      if (!needle) return true;
      const haystack = [
        row.menuItem.name,
        row.menuItem.id,
        row.menuItem.catalog_item_id,
        row.catalogItem?.name,
        row.catalogItem?.id,
        row.catalogItem?.sku,
      ].filter(Boolean).join(' ').toLowerCase();
      return haystack.includes(needle);
    });
});

const boundedRows = computed(() => rows.value.slice(0, previewLimit));
const isBounded = computed(() => rows.value.length > boundedRows.value.length);

function publishedBindings(targetType: string, targetId: string) {
  return modifierBindings.value.filter((binding) => binding.status === 'published' && binding.target_type === targetType && binding.target_id === targetId);
}

function evaluateReadiness(
  menuItem: MenuItem,
  catalogItem: CatalogItem | undefined,
  directMenuBindings: ModifierBinding[],
  directCatalogBindings: ModifierBinding[],
): { status: Readiness; reason: string } {
  if (!catalogItem) return { status: 'missing', reason: 'missingCatalogItem' };
  if (catalogItem.status !== 'published') return { status: 'missing', reason: 'catalogNotPublished' };
  if (menuItem.status !== 'published') return { status: 'missing', reason: 'menuItemNotPublished' };
  if (directMenuBindings.length + directCatalogBindings.length === 0) return { status: 'warning', reason: 'noModifierBindings' };
  if (publishedPricingPolicies.value.length === 0) return { status: 'warning', reason: 'noPricingPolicy' };
  return { status: 'ready', reason: 'ready' };
}

function modifierSummary(row: SalePreparationRow) {
  const total = row.directMenuBindings.length + row.directCatalogBindings.length;
  if (total === 0) return t('cloud.salePreparation.summaries.noModifiers');
  const names = [...row.directMenuBindings, ...row.directCatalogBindings]
    .map((binding) => modifierGroupById.value.get(binding.modifier_group_id)?.name || safeId(binding.modifier_group_id));
  return t('cloud.salePreparation.summaries.modifiers', {
    count: total,
    groups: names.slice(0, 3).join(', '),
  });
}

function hint(row: SalePreparationRow) {
  return t(`cloud.salePreparation.hints.${row.reason}`);
}

function readinessLabel(value: Readiness) {
  return t(`cloud.salePreparation.readiness.${value}`);
}

function statusClass(value: Readiness) {
  if (value === 'ready') return 'published';
  if (value === 'warning') return 'pending';
  return 'archived';
}

function statusText(value: string) {
  return props.ctx.statusText ? props.ctx.statusText(value) : t(`cloud.statuses.${value}`);
}

function catalogMeta(item: CatalogItem) {
  return `${item.kind} / ${safeId(item.id)}${item.sku ? ` / ${item.sku}` : ''}`;
}

function safeId(value: string) {
  return props.ctx.safeOperationalValue ? props.ctx.safeOperationalValue(value) : value;
}
</script>
