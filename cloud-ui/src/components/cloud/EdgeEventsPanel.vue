<template>
  <div class="cloud-panel-stack">
    <cloud-safe-error-banner :ctx="ctx" />
    <div class="cloud-panel cloud-table-panel">
      <div class="section-head">
        <div>
          <h2>{{ t('cloud.edgeEvents.title') }}</h2>
          <p>{{ t('cloud.edgeEvents.description') }}</p>
        </div>
        <q-btn flat icon="refresh" :loading="ctx.isLoading('edge-events')" :label="t('actions.refresh')" @click="ctx.loadEdgeEvents()" />
      </div>

      <div v-if="ctx.edgeEvents.value.length === 0" class="empty-state">
        {{ t('cloud.edgeEvents.empty') }}
      </div>
      <div v-else class="cloud-table-scroll">
        <table class="resource-table">
          <thead>
            <tr>
              <th>{{ t('cloud.fields.cloud_received_at') }}</th>
              <th>{{ t('cloud.fields.event_type') }}</th>
              <th>{{ t('cloud.fields.device_id') }}</th>
              <th>{{ t('cloud.fields.aggregate_type') }}</th>
              <th>{{ t('cloud.fields.aggregate_id') }}</th>
              <th>{{ t('cloud.fields.raw_payload_sha256_hex') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="event in ctx.edgeEvents.value" :key="event.cloud_receipt_id">
              <td>{{ ctx.formatDate(event.cloud_received_at) }}</td>
              <td><span class="cloud-status published">{{ event.event_type }}</span></td>
              <td>{{ event.device_id }}</td>
              <td>{{ event.aggregate_type }}</td>
              <td>{{ ctx.formatCell('aggregate_id', event.aggregate_id) }}</td>
              <td>{{ ctx.formatCell('raw_payload_sha256_hex', event.raw_payload_sha256_hex) }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n';

import CloudSafeErrorBanner from './CloudSafeErrorBanner.vue';

const { t } = useI18n();

defineProps<{
  ctx: Record<string, any>;
}>();
</script>
