<template>
  <q-page class="center-page">
    <section class="status-panel">
      <p class="eyebrow">{{ t('pos.interfaceNotAvailable') }}</p>
      <h1>{{ t(titleKey) }}</h1>
      <p class="status-copy">{{ t(bodyKey) }}</p>
      <q-btn color="primary" unelevated class="touch-button" icon="point_of_sale" :label="t('pos.cashierTerminal')" to="/pos" />
    </section>
  </q-page>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRoute } from 'vue-router';

const { t } = useI18n();
const route = useRoute();

const titleKey = computed(() => String(route.meta.titleKey ?? 'pos.cashierTerminal'));
const bodyKey = computed(() => String(route.meta.bodyKey ?? 'pos.interfaceNotAvailableBody'));
</script>
