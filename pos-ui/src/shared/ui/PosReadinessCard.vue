<template>
  <article
    :class="cardClasses"
    :aria-disabled="passive ? 'true' : undefined"
  >
    <span v-if="icon" class="material-icons pos-readiness-card-icon" aria-hidden="true">{{ icon }}</span>
    <span class="pos-readiness-card-main">
      <small v-if="badge">{{ badge }}</small>
      <strong>{{ title }}</strong>
      <em v-if="description">{{ description }}</em>
    </span>
  </article>
</template>

<script setup lang="ts">
import { computed } from 'vue';

import { posToneClasses, type PosTone } from './uiTypes';

const props = withDefaults(defineProps<{
  title: string;
  description?: string;
  badge?: string;
  icon?: string;
  tone?: PosTone;
  passive?: boolean;
  compact?: boolean;
}>(), {
  description: undefined,
  badge: undefined,
  icon: undefined,
  tone: 'neutral',
  passive: false,
  compact: false,
});

const cardClasses = computed(() => [
  ...posToneClasses('pos-readiness-card', props.tone),
  props.icon ? 'has-icon' : '',
  props.passive ? 'passive' : '',
  props.compact ? 'compact' : '',
].filter(Boolean));
</script>
