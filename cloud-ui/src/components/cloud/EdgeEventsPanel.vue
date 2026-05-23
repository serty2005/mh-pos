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
      <template v-else>
        <div class="cloud-table-wrap">
          <table class="cloud-table">
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
                <td>{{ ctx.formatCell('device_id', event.device_id) }}</td>
                <td>{{ event.aggregate_type }}</td>
                <td>{{ ctx.formatCell('aggregate_id', event.aggregate_id) }}</td>
                <td>{{ ctx.formatCell('raw_payload_sha256_hex', event.raw_payload_sha256_hex) }}</td>
              </tr>
            </tbody>
          </table>
        </div>

        <div class="edge-event-card-list" :aria-label="t('cloud.edgeEvents.title')">
          <article v-for="event in ctx.edgeEvents.value" :key="event.cloud_receipt_id" class="edge-event-card">
            <span class="cloud-status published">{{ event.event_type }}</span>
            <strong>{{ ctx.formatDate(event.cloud_received_at) }}</strong>
            <small>{{ t('cloud.fields.device_id') }}: {{ ctx.formatCell('device_id', event.device_id) }}</small>
            <small>{{ t('cloud.fields.aggregate_type') }}: {{ event.aggregate_type }}</small>
            <small>{{ t('cloud.fields.aggregate_id') }}: {{ ctx.formatCell('aggregate_id', event.aggregate_id) }}</small>
            <small>{{ t('cloud.fields.raw_payload_sha256_hex') }}: {{ ctx.formatCell('raw_payload_sha256_hex', event.raw_payload_sha256_hex) }}</small>
          </article>
        </div>
      </template>
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
