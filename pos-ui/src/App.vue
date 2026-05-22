<template>
  <q-layout view="hHh lpR fFf" class="app-shell">
    <q-header v-if="showHeader" class="app-header">
      <q-toolbar class="toolbar">
        <q-toolbar-title class="brand">{{ t('app.title') }}</q-toolbar-title>
        <div class="header-meta">
          <span v-if="auth.nodeDeviceId">{{ t('common.node') }} {{ shortId(auth.nodeDeviceId) }}</span>
          <span>{{ t('common.client') }} {{ shortId(auth.clientDeviceId) }}</span>
        </div>
        <q-btn
          v-if="auth.sessionId && route.path.startsWith('/pos')"
          flat
          class="icon-touch"
          icon="lock"
          :aria-label="t('actions.lock')"
          @click="router.push('/lock')"
        />
      </q-toolbar>
    </q-header>
    <q-page-container>
      <router-view />
    </q-page-container>
    <q-dialog v-model="errorDialog.open" persistent>
      <q-card class="dialog-card">
        <q-card-section>
          <p class="eyebrow">{{ t(`errors.severity.${errorDialog.severity}`) }}</p>
          <h2>{{ t(errorDialog.titleKey) }}</h2>
        </q-card-section>
        <q-card-section class="form-stack">
          <p>{{ t(errorDialog.messageKey) }}</p>
          <p class="meta-line">{{ t(errorDialog.recommendationKey) }}</p>
          <p v-if="errorDialog.supportCode" class="meta-line">
            {{ t('errors.supportCode') }}: {{ errorDialog.supportCode }}
          </p>
        </q-card-section>
        <q-card-actions align="right">
          <q-btn flat :label="t('actions.close')" @click="errorDialog.close" />
          <q-btn
            v-if="errorDialog.primaryAction === 'login'"
            color="primary"
            unelevated
            :label="t('actions.backToLogin')"
            @click="goLoginFromError"
          />
        </q-card-actions>
      </q-card>
    </q-dialog>
  </q-layout>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRoute, useRouter } from 'vue-router';

import { useErrorDialogStore } from './stores/errorDialog';
import { useAuthStore } from './stores/auth';

const { t } = useI18n();
const auth = useAuthStore();
const errorDialog = useErrorDialogStore();
const route = useRoute();
const router = useRouter();

const showHeader = computed(() => route.path !== '/pair' && route.path !== '/login' && !route.path.startsWith('/pos'));

function shortId(value: string) {
  return value.length > 10 ? `${value.slice(0, 8)}...` : value;
}

function goLoginFromError() {
  errorDialog.close();
  void router.replace('/login');
}
</script>
