<template>
  <article :class="posToneClasses('dashboard-metric', tone)">
    <span>{{ label }}</span>
    <strong>{{ value }}</strong>
    <small v-if="hint">{{ hint }}</small>
  </article>
</template>

<script setup lang="ts">
import { posToneClasses, type PosStatusTone } from './uiTypes';

withDefaults(defineProps<{
  label: string;
  value: string | number;
  hint?: string;
  tone?: PosStatusTone;
}>(), {
  hint: undefined,
  tone: 'neutral',
});
</script>
