<template>
  <section class="cloud-panel cloud-plan-panel readiness-only-panel">
    <div class="section-head stacked">
      <p class="eyebrow">{{ t('cloud.readiness.status') }}</p>
      <h2>{{ t('cloud.resources.inventoryReadiness') }}</h2>
      <p class="cloud-copy">{{ t('cloud.readiness.inventory.copy') }}</p>
    </div>
    <cloud-safe-error-banner :ctx="ctx" target="inventory-readiness" />
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
    <ul class="cloud-gap-list">
      <li v-for="item in gaps" :key="item">{{ t(item) }}</li>
    </ul>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';

import CloudSafeErrorBanner from './CloudSafeErrorBanner.vue';

const { t } = useI18n();
const props = defineProps<{
  ctx: Record<string, any>;
}>();
const gaps = [
  'cloud.readiness.inventory.gaps.documents',
  'cloud.readiness.inventory.gaps.costing',
  'cloud.readiness.inventory.gaps.review',
];
const publicationLabel = computed(() => {
  const publication = props.ctx.stopListReadiness.value?.latest_publication;
  return publication ? `${publication.version} / ${props.ctx.formatDate(publication.published_at)}` : '-';
});
const edgeAckLabel = computed(() => {
  const ack = props.ctx.stopListReadiness.value?.latest_stop_list_edge_ack;
  return ack ? `${ack.status} / ${props.ctx.safeOperationalValue(ack.event_id)}` : '-';
});
</script>
