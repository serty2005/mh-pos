<template>
  <q-page class="pos-page">
    <section class="operator-strip">
      <div>
        <p class="eyebrow">{{ t('pos.actor') }}</p>
        <h1>{{ auth.actor?.name ?? auth.actor?.employee_id }}</h1>
      </div>
      <div class="session-box">
        <span>{{ t('pos.session') }}</span>
        <strong>{{ shortId(auth.sessionId) }}</strong>
      </div>
    </section>

    <section class="workspace-grid">
      <aside class="list-pane">
        <div class="pane-heading">
          <h2>{{ t('pos.halls') }}</h2>
          <q-btn flat dense round icon="refresh" :aria-label="t('actions.retry')" @click="refetchHalls" />
        </div>
        <q-skeleton v-if="halls.isPending.value" class="skeleton-row" />
        <q-banner v-else-if="halls.isError.value" class="error-banner" rounded>{{ t('common.error') }}</q-banner>
        <q-list v-else-if="halls.data.value?.length" separator>
          <q-item
            v-for="hall in halls.data.value"
            :key="hall.id"
            clickable
            :active="hall.id === selectedHallId"
            active-class="active-item"
            @click="selectedHallId = hall.id"
          >
            <q-item-section>{{ hall.name }}</q-item-section>
          </q-item>
        </q-list>
        <div v-else class="empty-state">{{ t('pos.noHalls') }}</div>
      </aside>

      <main class="table-pane">
        <div class="pane-heading">
          <h2>{{ t('pos.tables') }}</h2>
        </div>
        <div v-if="tables.isPending.value" class="table-grid">
          <q-skeleton v-for="n in 6" :key="n" class="table-tile" />
        </div>
        <q-banner v-else-if="tables.isError.value" class="error-banner" rounded>{{ t('common.error') }}</q-banner>
        <div v-else-if="tables.data.value?.length" class="table-grid">
          <button v-for="table in tables.data.value" :key="table.id" class="table-tile" type="button">
            <span>{{ table.name }}</span>
            <small>{{ table.seats }}</small>
          </button>
        </div>
        <div v-else class="empty-state wide">{{ t('pos.noTables') }}</div>
      </main>
    </section>
  </q-page>
</template>

<script setup lang="ts">
import { useQuery } from '@tanstack/vue-query';
import { computed, ref, watchEffect } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRouter } from 'vue-router';

import { getAuthSession, listHalls, listTables } from '../shared/api';
import { useAuthStore } from '../stores/auth';

const { t } = useI18n();
const auth = useAuthStore();
const router = useRouter();
const selectedHallId = ref('');

const session = useQuery({
  queryKey: ['auth-session', auth.sessionId, auth.nodeDeviceId, auth.clientDeviceId],
  queryFn: getAuthSession,
  enabled: () => Boolean(auth.sessionId && auth.nodeDeviceId),
  retry: false,
});

const halls = useQuery({
  queryKey: ['halls', auth.restaurantId],
  queryFn: () => listHalls(auth.restaurantId),
  enabled: () => Boolean(auth.restaurantId && auth.sessionId),
});

const activeHallId = computed(() => selectedHallId.value || halls.data.value?.[0]?.id || '');

const tables = useQuery({
  queryKey: ['tables', auth.restaurantId, activeHallId],
  queryFn: () => listTables(auth.restaurantId, activeHallId.value),
  enabled: () => Boolean(auth.restaurantId && activeHallId.value && auth.sessionId),
});

watchEffect(() => {
  if (!auth.nodeDeviceId) void router.replace('/pair');
  if (!auth.sessionId) void router.replace('/login');
  if (session.data.value) {
    auth.applySession(session.data.value.session, session.data.value.actor);
    if (session.data.value.session.status !== 'active') {
      void router.replace('/login');
    }
  }
  if (!selectedHallId.value && halls.data.value?.[0]) {
    selectedHallId.value = halls.data.value[0].id;
  }
});

function refetchHalls() {
  void halls.refetch();
}

function shortId(value: string) {
  return value.length > 10 ? `${value.slice(0, 8)}...` : value;
}
</script>
