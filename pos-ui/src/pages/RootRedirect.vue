<template>
  <q-page class="center-page">
    <q-spinner color="primary" size="32px" />
  </q-page>
</template>

<script setup lang="ts">
import { useQuery } from '@tanstack/vue-query';
import { watchEffect } from 'vue';
import { useRouter } from 'vue-router';

import { getAuthSession, getPairingStatus } from '../shared/api';
import { useErrorHandling } from '../shared/errorHandling';
import { useAuthStore } from '../stores/auth';

const auth = useAuthStore();
const router = useRouter();
const { showBusinessError } = useErrorHandling();

const pairing = useQuery({
  queryKey: ['pairing-status'],
  queryFn: getPairingStatus,
});

const session = useQuery({
  queryKey: ['auth-session', auth.sessionId, auth.nodeDeviceId, auth.clientDeviceId],
  queryFn: getAuthSession,
  enabled: () => Boolean(auth.sessionId && auth.nodeDeviceId),
  retry: false,
});

watchEffect(() => {
  if (pairing.error.value) {
    showBusinessError(pairing.error.value);
    return;
  }
  if (session.error.value) {
    showBusinessError(session.error.value);
    void router.replace('/login');
    return;
  }
  if (pairing.data.value) {
    auth.applyPairing(pairing.data.value);
  }
  if (session.data.value) {
    auth.applySession(session.data.value.session, session.data.value.actor);
  }
  if (pairing.isPending.value) return;
  if (!pairing.data.value?.paired) {
    void router.replace('/pair');
    return;
  }
  if (auth.sessionId && session.isPending.value) return;
  if (auth.sessionId && session.data.value?.session.status === 'active') {
    void router.replace('/pos');
    return;
  }
  auth.clearSession();
  void router.replace('/login');
});
</script>
