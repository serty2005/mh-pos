<template>
  <section class="simple-pos-section">
    <div class="pos-section-head">
      <div>
        <p class="eyebrow">{{ terminal.t('pos.sections.activity') }}</p>
        <h1>{{ terminal.t('pos.activityTitle') }}</h1>
      </div>
    </div>
    <div class="simple-section-grid">
      <article class="simple-section-panel">
        <q-icon name="history" size="36px" color="primary" />
        <h2>{{ terminal.t('pos.closedOrders') }}</h2>
        <p>{{ terminal.t('pos.activityBody') }}</p>
        <q-btn color="primary" unelevated class="touch-button" icon="history" :label="terminal.t('pos.closedOrders')" :disable="!terminal.canViewClosedOrders.value" @click="terminal.closedOrdersDrawer.value = true" />
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
