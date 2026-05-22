<template>
  <q-page class="kitchen-page">
    <section class="kitchen-readiness">
      <div class="kitchen-readiness-head">
        <p class="eyebrow">{{ t('pos.plannedNext') }}</p>
        <h1>{{ t('pos.kitchenDisplay') }}</h1>
        <p>{{ t('pos.kitchenReadinessCopy') }}</p>
      </div>

      <PosBanner tone="warning" :label="t('pos.kitchenNoRuntime')" />

      <div class="kitchen-readiness-grid">
        <PosPanel :eyebrow="t('pos.backendContracts')" :title="t('pos.kitchenMissingContracts')">
          <ul class="readiness-list">
            <li v-for="item in missingContracts" :key="item">{{ t(item) }}</li>
          </ul>
        </PosPanel>

        <PosPanel :eyebrow="t('pos.kdsLifecycle')" :title="t('pos.kitchenLifecycleSlots')">
          <div class="kds-status-grid" :aria-label="t('pos.kdsLifecycle')">
            <span v-for="status in statuses" :key="status">{{ t(status) }}</span>
          </div>
          <p class="kitchen-muted">{{ t('pos.kitchenLifecycleDisabled') }}</p>
        </PosPanel>

        <PosPanel :eyebrow="t('pos.syncStatus')" :title="t('pos.kitchenSyncReadiness')">
          <p class="kitchen-muted">{{ t('pos.kitchenSyncReadinessCopy') }}</p>
          <template #footer>
            <PosButton variant="primary" icon="point_of_sale" :label="t('pos.cashierTerminal')" @click="router.push('/pos')" />
          </template>
        </PosPanel>
      </div>
    </section>
  </q-page>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n';
import { useRouter } from 'vue-router';

import { PosBanner, PosButton, PosPanel } from '../shared/ui';

const { t } = useI18n();
const router = useRouter();

const missingContracts = [
  'pos.kitchenContracts.tickets',
  'pos.kitchenContracts.lifecycle',
  'pos.kitchenContracts.stationGrouping',
  'pos.kitchenContracts.recall',
  'pos.kitchenContracts.printer',
];

const statuses = [
  'pos.kdsStatuses.new',
  'pos.kdsStatuses.accepted',
  'pos.kdsStatuses.in_progress',
  'pos.kdsStatuses.hold',
  'pos.kdsStatuses.ready',
  'pos.kdsStatuses.served',
  'pos.kdsStatuses.recall',
  'pos.kdsStatuses.cancelled',
];
</script>
