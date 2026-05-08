<template>
  <q-page class="auth-page">
    <section class="auth-panel">
      <div>
        <p class="eyebrow">{{ auth.clientDeviceId }}</p>
        <h1>{{ t('pair.title') }}</h1>
      </div>
      <q-form class="form-stack" @submit.prevent="submit">
        <q-input v-model="code" outlined :label="t('pair.code')" :hint="t('pair.hint')" autocomplete="off" />
        <q-btn color="primary" unelevated icon="link" :label="t('actions.pair')" type="submit" :loading="pairing.isPending.value" :disable="pairing.isPending.value" />
      </q-form>
    </section>
  </q-page>
</template>

<script setup lang="ts">
import { useMutation, useQueryClient } from '@tanstack/vue-query';
import { ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRouter } from 'vue-router';

import { pairEdgeNodeAndRefresh } from '../shared/api';
import { useErrorHandling } from '../shared/errorHandling';
import { useAuthStore } from '../stores/auth';

const { t } = useI18n();
const auth = useAuthStore();
const router = useRouter();
const queryClient = useQueryClient();
const { showBusinessError } = useErrorHandling();
const code = ref('');

const pairing = useMutation({
  mutationFn: pairEdgeNodeAndRefresh,
  onSuccess(status) {
    auth.applyPairing(status);
    void queryClient.invalidateQueries({ queryKey: ['pairing-status'] });
    void router.replace('/login');
  },
  onError: showBusinessError,
});

function submit() {
  if (!code.value.trim()) return;
  pairing.mutate(code.value.trim());
}

</script>
