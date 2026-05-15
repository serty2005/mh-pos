<template>
  <section class="primary-flow-strip" :aria-label="terminal.t('pos.primaryFlow.title')">
    <div class="primary-flow-head">
      <p class="eyebrow">{{ terminal.t('pos.primaryFlow.kicker') }}</p>
      <h2>{{ terminal.t('pos.primaryFlow.title') }}</h2>
    </div>
    <ol class="flow-steps">
      <li v-for="step in terminal.primaryFlowSteps.value" :key="step.key" :class="step.state">
        <span>{{ step.index }}</span>
        <div>
          <strong>{{ terminal.t(step.titleKey) }}</strong>
          <small>{{ terminal.t(step.descriptionKey) }}</small>
        </div>
      </li>
    </ol>
    <blocking-notice
      v-if="terminal.currentBlockingNotice.value"
      :terminal="terminal"
      :title="terminal.t(terminal.currentBlockingNotice.value.titleKey)"
      :reason="terminal.t(terminal.currentBlockingNotice.value.reasonKey)"
      :permission="terminal.currentBlockingNotice.value.permission"
      icon="lock_clock"
    />
  </section>
</template>

<script setup lang="ts">
import BlockingNotice from './BlockingNotice.vue';
import type { CashierTerminal } from './useCashierTerminal';

defineProps<{
  terminal: CashierTerminal;
}>();
</script>
