<template>
  <q-page class="pair-page">
    <section class="pair-shell">
      <header class="pair-header">
        <p class="eyebrow">{{ auth.clientDeviceId }}</p>
        <h1>{{ t('pair.title') }}</h1>
        <p class="pair-lead">{{ t('pair.subtitle') }}</p>
      </header>

      <q-tabs v-model="mode" dense align="left" class="pair-tabs" active-color="primary" indicator-color="primary">
        <q-tab name="cloud" icon="cloud_done" :label="t('pair.cloudMode')" />
        <q-tab name="license" icon="vpn_key" :label="t('pair.licenseMode')" />
      </q-tabs>

      <q-tab-panels v-model="mode" animated class="pair-panels">
        <q-tab-panel name="cloud">
          <div class="pair-grid">
            <div class="pair-status">
              <div>
                <p class="meta-label">{{ t('common.node') }}</p>
                <p class="mono-value">{{ provisioning.data.value?.node_device_id ?? t('common.loading') }}</p>
              </div>
              <div>
                <p class="meta-label">{{ t('pair.cloudUrl') }}</p>
                <p class="mono-value">{{ provisioning.data.value?.cloud_url || t('pair.cloudUrlEmpty') }}</p>
              </div>
              <div>
                <p class="meta-label">{{ t('common.status') }}</p>
                <q-badge outline color="primary" :label="statusLabel" />
              </div>
              <p class="status-copy">{{ t('pair.pendingCopy') }}</p>
            </div>
            <div class="pair-actions">
              <q-btn unelevated color="primary" icon="refresh" :label="t('pair.retryRegister')" :loading="registering.isPending.value" @click="registering.mutate()" />
              <q-btn flat icon="sync" :label="t('actions.retry')" :loading="provisioning.isFetching.value" @click="void provisioning.refetch()" />
            </div>
          </div>
        </q-tab-panel>

        <q-tab-panel name="license">
          <q-form class="license-form" @submit.prevent="submitLicense">
            <q-input v-model="code" outlined :label="t('pair.licenseCode')" autocomplete="off" input-class="text-uppercase" />
            <q-btn color="primary" unelevated icon="link" :label="t('actions.pair')" type="submit" :loading="licensePairing.isPending.value" :disable="licensePairing.isPending.value" />
          </q-form>
        </q-tab-panel>
      </q-tab-panels>
    </section>
  </q-page>
</template>

<script setup lang="ts">
import { useMutation, useQuery } from '@tanstack/vue-query';
import { computed, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRouter } from 'vue-router';

import { getProvisioningStatus, pairViaLicense, registerCloudProvisioning } from '../shared/api';
import { useErrorHandling } from '../shared/errorHandling';
import { useAuthStore } from '../stores/auth';

const { t } = useI18n();
const auth = useAuthStore();
const router = useRouter();
const { showBusinessError } = useErrorHandling();
const mode = ref<'cloud' | 'license'>('cloud');
const code = ref('');

const provisioning = useQuery({
  queryKey: ['provisioning-status'],
  queryFn: getProvisioningStatus,
  refetchInterval: 2500,
});

const registering = useMutation({
  mutationFn: () => registerCloudProvisioning(provisioning.data.value?.cloud_url),
  onSuccess(status) {
    auth.applyProvisioning(status);
    if (status.paired) void router.replace('/login');
    void provisioning.refetch();
  },
  onError: showBusinessError,
});

const licensePairing = useMutation({
  mutationFn: pairViaLicense,
  onSuccess(status) {
    auth.applyProvisioning(status);
    void router.replace('/login');
  },
  onError: showBusinessError,
});

const statusLabel = computed(() => t(`pair.status.${provisioning.data.value?.status ?? 'not_configured'}`));

watch(
  () => provisioning.data.value,
  (status) => {
    if (!status) return;
    auth.applyProvisioning(status);
    if (status.paired) void router.replace('/login');
  },
  { immediate: true },
);

function submitLicense() {
  if (!code.value.trim()) return;
  licensePairing.mutate(code.value.trim().toUpperCase());
}
</script>
