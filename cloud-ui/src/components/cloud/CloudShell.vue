<template>
  <q-layout view="hHh LpR fFf" class="cloud-layout">
    <q-header class="cloud-header">
      <q-toolbar class="cloud-toolbar">
        <q-toolbar-title class="cloud-brand">{{ t('app.title') }}</q-toolbar-title>
        <span class="cloud-api">{{ ctx.apiBaseLabel }}</span>
        <q-btn flat dense icon="refresh" :label="t('actions.refresh')" :loading="ctx.activeLoading.value" @click="ctx.reloadActive()" />
      </q-toolbar>
    </q-header>

    <q-page-container>
      <q-page class="cloud-page">
        <aside class="cloud-sidebar">
          <div class="cloud-sidebar-head">
            <p class="eyebrow">{{ t('cloud.scope') }}</p>
            <h1>{{ t('cloud.title') }}</h1>
          </div>

          <q-select
            v-model="ctx.selectedRestaurantId.value"
            dense
            outlined
            emit-value
            map-options
            :loading="ctx.isLoading('restaurants')"
            :label="t('cloud.restaurantFilter')"
            :options="ctx.restaurantOptions.value"
          />

          <section v-for="group in ctx.navGroups.value" :key="group.key" class="cloud-nav-group">
            <p class="cloud-nav-label">{{ t(group.labelKey) }}</p>
            <button
              v-for="item in group.items"
              :key="item.key"
              type="button"
              class="cloud-nav-item"
              :class="{ selected: ctx.activeKey.value === item.key }"
              @click="ctx.setActive(item.key)"
            >
              <span>{{ t(item.titleKey) }}</span>
              <strong>{{ ctx.navCount(item.key) }}</strong>
            </button>
          </section>
        </aside>

        <main class="cloud-main">
          <slot />
        </main>
      </q-page>
    </q-page-container>
  </q-layout>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n';

const { t } = useI18n();

defineProps<{
  ctx: Record<string, any>;
}>();
</script>
