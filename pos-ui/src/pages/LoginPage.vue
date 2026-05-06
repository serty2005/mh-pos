<template>
  <q-page class="auth-page">
    <section class="auth-panel compact">
      <div>
        <p class="eyebrow">{{ t('common.node') }} {{ auth.nodeDeviceId }}</p>
        <h1>{{ t('login.title') }}</h1>
      </div>
      <q-form class="form-stack" @submit.prevent="submit">
        <q-input
          v-model="pin"
          outlined
          :label="t('login.pin')"
          :hint="t('login.pinHint')"
          type="password"
          inputmode="numeric"
          autocomplete="current-password"
        />
        <q-banner v-if="login.isError.value" class="error-banner" rounded>
          {{ errorMessage(login.error.value) }}
        </q-banner>
        <q-btn color="primary" unelevated icon="login" :label="t('actions.login')" type="submit" :loading="login.isPending.value" />
      </q-form>
    </section>
  </q-page>
</template>

<script setup lang="ts">
import { useMutation, useQuery } from '@tanstack/vue-query';
import { ref, watchEffect } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRouter } from 'vue-router';

import { getPairingStatus, pinLogin } from '../shared/api';
import { useAuthStore } from '../stores/auth';

const { t } = useI18n();
const auth = useAuthStore();
const router = useRouter();
const pin = ref('');

const pairing = useQuery({
  queryKey: ['pairing-status'],
  queryFn: getPairingStatus,
});

const login = useMutation({
  mutationFn: pinLogin,
  onSuccess(result) {
    auth.applySession(result.session, result.actor);
    pin.value = '';
    void router.replace('/pos');
  },
});

watchEffect(() => {
  if (pairing.data.value) {
    auth.applyPairing(pairing.data.value);
    if (!pairing.data.value.paired) {
      void router.replace('/pair');
    }
  }
});

function submit() {
  if (!pin.value.trim()) return;
  login.mutate(pin.value.trim());
}

function errorMessage(error: unknown) {
  return error instanceof Error ? error.message : t('common.error');
}
</script>
