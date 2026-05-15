<template>
  <q-banner class="blocking-notice" rounded>
    <div class="blocking-notice-content">
      <q-icon :name="icon" />
      <div>
        <strong>{{ title }}</strong>
        <span>{{ reason }}</span>
        <small v-if="permission">{{ terminal.t('pos.blocking.permission', { permission }) }}</small>
      </div>
    </div>
  </q-banner>
</template>

<script setup lang="ts">
import type { CashierTerminal } from './useCashierTerminal';

withDefaults(defineProps<{
  terminal: CashierTerminal;
  title: string;
  reason: string;
  permission?: string;
  icon?: string;
}>(), {
  permission: '',
  icon: 'info',
});
</script>
