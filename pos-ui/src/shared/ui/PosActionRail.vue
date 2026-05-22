<template>
  <aside class="section-action-rail pos-action-rail" :aria-label="resolvedAriaLabel">
    <div v-if="eyebrow || title || $slots.header || $slots['header-side']" class="rail-head">
      <div>
        <p v-if="eyebrow" class="eyebrow">{{ eyebrow }}</p>
        <h2 v-if="title">{{ title }}</h2>
        <slot name="header" />
      </div>
      <slot name="header-side" />
    </div>

    <div v-if="empty" class="rail-empty">
      <slot name="empty" />
    </div>
    <template v-else>
      <div v-if="$slots.summary" class="rail-summary">
        <slot name="summary" />
      </div>
      <div v-if="$slots.default" class="pos-action-rail-content pos-scrollarea-y pos-scrollbar-thin">
        <slot />
      </div>
      <div v-if="$slots.actions" class="rail-actions integrated-action-bar">
        <slot name="actions" />
      </div>
    </template>
  </aside>
</template>

<script setup lang="ts">
import { computed } from 'vue';

const props = withDefaults(defineProps<{
  eyebrow?: string;
  title?: string;
  ariaLabel?: string;
  'aria-label'?: string;
  empty?: boolean;
}>(), {
  eyebrow: undefined,
  title: undefined,
  ariaLabel: undefined,
  'aria-label': undefined,
  empty: false,
});

const resolvedAriaLabel = computed(() => props.ariaLabel ?? props['aria-label']);
</script>
