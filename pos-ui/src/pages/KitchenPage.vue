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
        <PosPanel class="kitchen-contract-panel" :eyebrow="t('pos.backendContracts')" :title="t('pos.kitchenMissingContracts')">
          <ul class="readiness-list">
            <li v-for="item in missingContracts" :key="item">{{ t(item) }}</li>
          </ul>
        </PosPanel>

        <PosPanel class="kitchen-lifecycle-panel" :eyebrow="t('pos.kdsLifecycle')" :title="t('pos.kitchenLifecycleSlots')">
          <div class="kds-lifecycle-map" :aria-label="t('pos.kdsLifecycle')">
            <span v-for="(status, index) in statuses" :key="status" class="kds-status-node">
              <small>{{ index + 1 }}</small>
              <strong>{{ t(status) }}</strong>
              <em>{{ index < statuses.length - 1 ? t('pos.kitchenLifecycleFutureStep') : t('pos.kitchenLifecycleTerminalStep') }}</em>
            </span>
          </div>
          <p class="kitchen-muted">{{ t('pos.kitchenLifecycleDisabled') }}</p>
          <div class="kitchen-disabled-actions" :aria-label="t('pos.kitchenDisabledActions')">
            <span v-for="action in disabledActions" :key="action" aria-disabled="true">
              {{ t(action) }}
            </span>
          </div>
        </PosPanel>

        <PosPanel :eyebrow="t('pos.plannedNext')" :title="t('pos.kitchenActivationGates')">
          <div class="kitchen-gate-list">
            <article v-for="gate in activationGates" :key="gate.titleKey" class="kitchen-gate-card">
              <span>{{ t(gate.statusKey) }}</span>
              <strong>{{ t(gate.titleKey) }}</strong>
              <small>{{ t(gate.copyKey) }}</small>
            </article>
          </div>
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

const activationGates = [
  {
    statusKey: 'pos.plannedBeforePilot',
    titleKey: 'pos.kitchenGates.ticketReadModel',
    copyKey: 'pos.kitchenGateDetails.ticketReadModel',
  },
  {
    statusKey: 'pos.plannedBeforePilot',
    titleKey: 'pos.kitchenGates.lifecyclePermissions',
    copyKey: 'pos.kitchenGateDetails.lifecyclePermissions',
  },
  {
    statusKey: 'pos.plannedNext',
    titleKey: 'pos.kitchenGates.syncEvents',
    copyKey: 'pos.kitchenGateDetails.syncEvents',
  },
];

const disabledActions = [
  'pos.kitchenActions.accept',
  'pos.kitchenActions.start',
  'pos.kitchenActions.hold',
  'pos.kitchenActions.ready',
  'pos.kitchenActions.served',
  'pos.kitchenActions.recall',
];
</script>
