<template>
  <article :class="[...posToneClasses('dashboard-metric', tone), sizeClass]">
    <span>{{ label }}</span>
    <strong>{{ value }}</strong>
    <small v-if="hint">{{ hint }}</small>
  </article>
</template>

<script setup lang="ts">
import { computed } from 'vue';

import { posSizeClass, posToneClasses, type PosSize, type PosTone } from './uiTypes';

const props = withDefaults(defineProps<{
  label: string;
  value: string | number;
  hint?: string;
  tone?: PosTone;
  size?: PosSize;
}>(), {
  hint: undefined,
  tone: 'neutral',
  size: 'regular',
});

const sizeClass = computed(() => posSizeClass(props.size));
</script>
