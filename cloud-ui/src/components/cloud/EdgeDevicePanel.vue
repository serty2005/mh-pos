<template>
  <section class="cloud-edge-grid">
    <div class="cloud-panel cloud-list-panel">
      <cloud-safe-error-banner :ctx="ctx" target="edgeDevices" />
      <div class="cloud-list-tools">
        <q-input v-model="ctx.search.value" dense outlined clearable debounce="120" :label="t('cloud.search')" />
        <span>{{ t('cloud.rows') }}: {{ ctx.filteredEdgeDevices.value.length }}</span>
      </div>
      <div v-if="ctx.isLoading('edge-devices')" class="cloud-skeleton-list">
        <q-skeleton v-for="index in 4" :key="index" class="skeleton-row" />
      </div>
      <div v-else-if="ctx.filteredEdgeDevices.value.length === 0" class="empty-state wide">{{ t('cloud.empty.noEdgeDevices') }}</div>
      <div v-else class="edge-device-list">
        <button
          v-for="node in ctx.filteredEdgeDevices.value"
          :key="node.node_device_id"
          type="button"
          class="edge-device-card"
          :class="{ selected: ctx.selectedEdgeNodeId.value === node.node_device_id }"
          @click="ctx.selectEdgeNode(node)"
        >
          <span class="cloud-status" :class="node.status">{{ ctx.edgeStatusText(node.status) }}</span>
          <strong>{{ node.display_name }}</strong>
          <small>{{ node.node_device_id }}</small>
          <span>{{ t('cloud.fields.app_version') }}: {{ node.app_version || '-' }}</span>
          <span>{{ t('cloud.fields.last_seen_at') }}: {{ ctx.formatDate(node.last_seen_at) }}</span>
        </button>
      </div>
    </div>

    <form class="cloud-panel cloud-form-panel" @submit.prevent="ctx.assignSelectedEdgeDevice()">
      <div class="section-head stacked">
        <p class="eyebrow">{{ t('cloud.edgeDevices.claimedFlow') }}</p>
        <h2>{{ t('cloud.edgeDevices.assignTitle') }}</h2>
      </div>
      <q-select
        v-model="ctx.selectedEdgeNodeId.value"
        dense
        outlined
        emit-value
        map-options
        :label="t('cloud.fields.edge_node')"
        :options="ctx.unassignedEdgeOptions.value"
        :disable="ctx.unassignedEdgeOptions.value.length === 0"
      />
      <q-btn color="primary" unelevated icon="link" type="submit" :disable="!ctx.selectedRestaurantId.value || !ctx.selectedEdgeNodeId.value" :loading="ctx.isLoading('edge-assign')" :label="t('cloud.edgeDevices.assignAction')" />
      <q-btn flat icon="manage_search" :disable="!ctx.selectedEdgeNodeId.value" :loading="ctx.isLoading('edge-status')" :label="t('cloud.edgeDevices.checkStatus')" @click="ctx.loadSelectedAssignmentStatus()" />
      <div v-if="ctx.assignmentResult.value" class="cloud-result-box">
        <span>{{ t('cloud.fields.status') }}: {{ ctx.assignmentResult.value.status }}</span>
        <span>{{ t('cloud.fields.snapshot_url') }}: {{ ctx.assignmentResult.value.snapshot_url }}</span>
      </div>
      <div v-if="ctx.assignmentStatus.value" class="cloud-result-box muted">
        <span>{{ t('cloud.fields.status') }}: {{ ctx.assignmentStatus.value.status }}</span>
        <span>{{ t('cloud.fields.cloud_url') }}: {{ ctx.assignmentStatus.value.cloud_url || '-' }}</span>
      </div>
      <q-separator />
      <div class="section-head stacked">
        <p class="eyebrow">{{ t('cloud.edgeDevices.licenseFlow') }}</p>
        <h2>{{ t('cloud.edgeDevices.pairingTitle') }}</h2>
      </div>
      <q-input v-model="ctx.pairingForm.display_name" dense outlined :label="t('cloud.fields.display_name')" />
      <q-input v-model.number="ctx.pairingForm.expires_in_minutes" dense outlined type="number" :label="t('cloud.fields.expires_in_minutes')" />
      <q-btn flat icon="password" :disable="!ctx.selectedRestaurantId.value" :loading="ctx.isLoading('pairing-code')" :label="t('cloud.edgeDevices.generatePairing')" @click="ctx.generateSelectedPairingCode()" />
      <div v-if="ctx.pairingResult.value" class="pairing-code-box">
        <span>{{ t('cloud.edgeDevices.pairingCode') }}</span>
        <strong>{{ ctx.pairingResult.value.pairing_code }}</strong>
        <small>{{ t('cloud.fields.edge_node') }}: {{ ctx.nodeDisplayName(ctx.pairingResult.value.node_device_id) }}</small>
        <small>{{ t('cloud.fields.expires_at') }}: {{ ctx.formatDate(ctx.pairingResult.value.expires_at) }}</small>
      </div>
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
