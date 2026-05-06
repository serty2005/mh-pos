<template>
  <q-layout view="hHh lpR fFf" class="app-shell">
    <q-header v-if="showHeader" class="app-header">
      <q-toolbar class="toolbar">
        <q-toolbar-title class="brand">MyHoReCa POS</q-toolbar-title>
        <div class="header-meta">
          <span v-if="auth.nodeDeviceId">{{ t('common.node') }} {{ shortId(auth.nodeDeviceId) }}</span>
          <span>{{ t('common.client') }} {{ shortId(auth.clientDeviceId) }}</span>
        </div>
        <q-btn
          v-if="auth.sessionId && route.path === '/pos'"
          flat
          dense
          icon="lock"
          :aria-label="t('actions.lock')"
          @click="router.push('/lock')"
        />
      </q-toolbar>
    </q-header>
    <q-page-container>
      <router-view />
    </q-page-container>
  </q-layout>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRoute, useRouter } from 'vue-router';

import { useAuthStore } from './stores/auth';

const { t } = useI18n();
const auth = useAuthStore();
const route = useRoute();
const router = useRouter();

const showHeader = computed(() => route.path !== '/pair' && route.path !== '/login');

function shortId(value: string) {
  return value.length > 10 ? `${value.slice(0, 8)}...` : value;
}
</script>
