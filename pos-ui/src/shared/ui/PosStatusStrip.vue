<template>
  <div :class="[...posToneClasses('status-strip', tone), { large }]">
    <span v-if="label">{{ label }}</span>
    <strong>{{ value }}</strong>
  </div>
</template>

<script setup lang="ts">
import { posToneClasses, type PosStatusTone } from './uiTypes';

withDefaults(defineProps<{
  label?: string;
  value: string | number;
  tone?: PosStatusTone;
  large?: boolean;
}>(), {
  label: undefined,
  tone: 'neutral',
  large: false,
});
</script>
