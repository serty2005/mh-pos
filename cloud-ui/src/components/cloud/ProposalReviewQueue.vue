<template>
  <section class="cloud-panel cloud-plan-panel readiness-only-panel">
    <div class="section-head stacked">
      <p class="eyebrow">{{ t('cloud.readiness.status') }}</p>
      <h2>{{ t('cloud.resources.proposalReview') }}</h2>
      <p class="cloud-copy">{{ t('cloud.readiness.proposalReview.copy') }}</p>
    </div>
    <ul class="cloud-gap-list">
      <li v-for="item in gaps" :key="item">{{ t(item) }}</li>
    </ul>
  </section>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n';

const { t } = useI18n();
const gaps = [
  'cloud.readiness.proposalReview.gaps.queue',
  'cloud.readiness.proposalReview.gaps.approval',
  'cloud.readiness.proposalReview.gaps.audit',
];
</script>
