<template>
  <section class="simple-pos-section">
    <div class="pos-section-head">
      <div>
        <p class="eyebrow">{{ terminal.t('pos.sections.reports') }}</p>
        <h1>{{ terminal.t('pos.reportsTitle') }}</h1>
      </div>
    </div>
    <div class="simple-section-grid">
      <article class="simple-section-panel">
        <q-icon name="monitoring" size="36px" color="primary" />
        <h2>{{ terminal.t('pos.interfaceNotAvailable') }}</h2>
        <p>{{ terminal.t('pos.reportsPlaceholder') }}</p>
      </article>
    </div>
  </section>
</template>

<script setup lang="ts">
import type { CashierTerminal } from './useCashierTerminal';

defineProps<{
  terminal: CashierTerminal;
}>();
</script>
