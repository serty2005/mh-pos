<template>
  <section class="recipe-version-grid">
    <div class="cloud-panel">
      <cloud-safe-error-banner :ctx="ctx" target="recipeVersions" />
      <div class="section-head">
        <div>
          <p class="eyebrow">{{ t('cloud.recipeVersions.editor') }}</p>
          <h2>{{ t('cloud.recipeVersions.title') }}</h2>
        </div>
        <q-btn flat dense icon="refresh" :label="t('actions.refresh')" :loading="ctx.isLoading('recipeVersions')" @click="ctx.loadRecipeVersions()" />
      </div>

      <div v-if="!ctx.selectedRestaurantId.value" class="empty-state wide">{{ t('cloud.empty.selectRestaurant') }}</div>
      <form v-else class="recipe-version-form" @submit.prevent="submit(false)">
        <q-select v-model="ownerCatalogItemId" dense outlined emit-value map-options :label="t('cloud.fields.owner_catalog_item_id')" :options="ownerOptions" />
        <q-input v-model="name" dense outlined :label="t('cloud.fields.name')" />
        <div class="recipe-version-form-row">
          <q-input v-model.number="yieldQuantity" dense outlined type="number" :label="t('cloud.fields.yield_quantity')" />
          <q-input v-model="yieldUnit" dense outlined :label="t('cloud.fields.yield_unit')" />
        </div>
        <div class="recipe-version-form-row">
          <q-select v-model="componentCatalogItemId" dense outlined emit-value map-options :label="t('cloud.fields.component_catalog_item_id')" :options="componentOptions" />
          <q-input v-model.number="quantity" dense outlined type="number" :label="t('cloud.fields.quantity')" />
        </div>
        <div class="recipe-version-form-row">
          <q-input v-model="unit" dense outlined :label="t('cloud.fields.unit_code')" />
          <q-input v-model.number="lossPercent" dense outlined type="number" :label="t('cloud.fields.loss_percent')" />
        </div>
        <q-input v-model="managerId" dense outlined :label="t('cloud.fields.reviewed_by_employee_id')" />
        <q-input v-model="reason" dense outlined type="textarea" autogrow :label="t('cloud.fields.reason')" />
        <div class="proposal-actions">
          <q-btn color="primary" unelevated icon="save" :label="t('cloud.recipeVersions.saveDraft')" :loading="ctx.isLoading('recipe-version-submit')" @click="submit(false)" />
          <q-btn flat icon="send" :label="t('cloud.recipeVersions.submitDraft')" :loading="ctx.isLoading('recipe-version-submit')" @click="submit(true)" />
        </div>
      </form>
    </div>

    <div class="cloud-panel">
      <div class="section-head">
        <div>
          <p class="eyebrow">{{ t('cloud.recipeVersions.current') }}</p>
          <h2>{{ t('cloud.recipeVersions.versions') }}</h2>
        </div>
      </div>
      <div v-if="ctx.isLoading('recipeVersions')" class="cloud-skeleton-list">
        <q-skeleton v-for="index in 4" :key="index" class="skeleton-row" />
      </div>
      <div v-else-if="ctx.recipeVersions.value.length === 0" class="empty-state">{{ t('cloud.recipeVersions.empty') }}</div>
      <div v-else class="recipe-version-list">
        <article v-for="item in ctx.recipeVersions.value" :key="item.version.id" class="recipe-version-row">
          <div>
            <span class="cloud-status" :class="item.version.status">{{ statusText(item.version.status) }}</span>
            <strong>{{ item.version.name }}</strong>
            <small>{{ t('cloud.fields.version') }}: {{ item.version.version }} · {{ ctx.formatDate(item.version.updated_at) }}</small>
          </div>
          <dl>
            <div v-for="line in item.lines" :key="line.id">
              <dt>{{ shortId(line.component_catalog_item_id) }}</dt>
              <dd>{{ line.quantity }} {{ line.unit }} · {{ line.loss_percent }}%</dd>
            </div>
          </dl>
        </article>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue';
import { useI18n } from 'vue-i18n';

import CloudSafeErrorBanner from './CloudSafeErrorBanner.vue';

const props = defineProps<{ ctx: Record<string, any> }>();
const { t } = useI18n();

const ownerCatalogItemId = ref('');
const componentCatalogItemId = ref('');
const name = ref('');
const yieldQuantity = ref(1);
const yieldUnit = ref('portion');
const quantity = ref(1);
const unit = ref('g');
const lossPercent = ref(0);
const managerId = ref('cloud-manager');
const reason = ref('');

const ownerOptions = computed(() => props.ctx.scopedRows.catalogItems
  .filter((item: Record<string, unknown>) => item.kind === 'dish' || item.kind === 'semi_finished')
  .map((item: Record<string, unknown>) => ({ label: String(item.name ?? item.id), value: String(item.id) })));
const componentOptions = computed(() => props.ctx.scopedRows.catalogItems
  .filter((item: Record<string, unknown>) => item.kind === 'good' || item.kind === 'semi_finished')
  .map((item: Record<string, unknown>) => ({ label: String(item.name ?? item.id), value: String(item.id) })));

async function submit(submitForReview: boolean) {
  await props.ctx.createRecipeVersionDraft({
    restaurant_id: props.ctx.selectedRestaurantId.value,
    owner_catalog_item_id: ownerCatalogItemId.value,
    name: name.value,
    yield_quantity: yieldQuantity.value,
    yield_unit: yieldUnit.value,
    created_by_employee_id: managerId.value,
    submit_for_review: submitForReview,
    reason: reason.value,
    lines: [{
      component_catalog_item_id: componentCatalogItemId.value,
      quantity: quantity.value,
      unit: unit.value,
      loss_percent: lossPercent.value,
    }],
  });
}

function statusText(status: string) {
  const key = `cloud.statuses.${status}`;
  const translated = t(key);
  return translated === key ? status : translated;
}

function shortId(value: string) {
  return value.length > 16 ? `${value.slice(0, 10)}...${value.slice(-4)}` : value;
}
</script>
