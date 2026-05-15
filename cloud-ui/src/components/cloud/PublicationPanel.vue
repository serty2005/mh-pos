<template>
  <section class="cloud-publication-grid">
    <div class="cloud-panel">
      <cloud-safe-error-banner :ctx="ctx" target="publications" />
      <div class="section-head">
        <h2>{{ t('cloud.publications.currentState') }}</h2>
        <q-btn flat dense icon="refresh" :label="t('actions.refresh')" :loading="ctx.isLoading('publication')" @click="ctx.loadPublication()" />
      </div>
      <div v-if="ctx.publication.value" class="cloud-state-grid publication-card-grid">
        <div v-for="field in ctx.publicationFields.value" :key="field.key">
          <span>{{ t(field.labelKey) }}</span>
          <strong>{{ field.value }}</strong>
        </div>
      </div>
      <div v-else class="empty-state">{{ t('cloud.empty.noPublication') }}</div>
      <div v-if="ctx.publication.value" class="cloud-counts publication-count-cards">
        <div v-for="[key, value] in ctx.publicationCounts.value" :key="key" class="cloud-count-row">
          <span>{{ key }}</span>
          <strong>{{ value }}</strong>
        </div>
      </div>
    </div>

    <form class="cloud-panel cloud-form-panel" @submit.prevent="ctx.publishSelectedRestaurant()">
      <div class="section-head">
        <h2>{{ t('cloud.publications.publish') }}</h2>
      </div>
      <q-input v-model="ctx.publishForm.published_by" dense outlined :label="t('cloud.fields.published_by')" />
      <q-select
        v-model="ctx.publishForm.node_device_id"
        dense
        outlined
        clearable
        emit-value
        map-options
        :label="t('cloud.fields.edge_node')"
        :options="ctx.knownNodeOptions.value"
      />
      <q-btn
        color="primary"
        unelevated
        icon="publish"
        type="submit"
        :loading="ctx.isLoading('publication-submit')"
        :label="t('cloud.publications.publishAction')"
      />
    </form>
  </section>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n';

import CloudSafeErrorBanner from './CloudSafeErrorBanner.vue';

const { t } = useI18n();

defineProps<{
  ctx: Record<string, any>;
}>();
</script>
