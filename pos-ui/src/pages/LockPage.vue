<template>
  <q-page class="center-page">
    <section class="status-panel">
      <q-icon name="lock" size="40px" color="primary" />
      <h1>{{ t('lock.title') }}</h1>
      <p>{{ t('lock.body') }}</p>
      <q-banner v-if="logoutMutation.isError.value" class="error-banner" rounded>
        {{ errorMessage(logoutMutation.error.value) }}
      </q-banner>
      <q-btn color="primary" unelevated icon="login" :label="t('actions.backToLogin')" :loading="logoutMutation.isPending.value" @click="goLogin" />
    </section>
  </q-page>
</template>

<script setup lang="ts">
import { useMutation, useQueryClient } from '@tanstack/vue-query';
import { onMounted } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRouter } from 'vue-router';

import { logout } from '../shared/api';
import { useAuthStore } from '../stores/auth';

const { t } = useI18n();
const auth = useAuthStore();
const router = useRouter();
const queryClient = useQueryClient();

const logoutMutation = useMutation({
  mutationFn: logout,
  onSettled() {
    auth.clearSession();
    queryClient.removeQueries({ queryKey: ['auth-session'] });
  },
});

onMounted(() => {
  if (auth.sessionId && !logoutMutation.isPending.value) {
    logoutMutation.mutate();
  }
});

function goLogin() {
  auth.clearSession();
  void router.replace('/login');
}

function errorMessage(error: unknown) {
  return error instanceof Error ? error.message : t('common.error');
}
</script>
