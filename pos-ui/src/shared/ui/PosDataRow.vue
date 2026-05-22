<template>
  <component
    :is="as"
    :type="as === 'button' ? 'button' : undefined"
    :class="rowClasses"
    @click="emit('click', $event)"
  >
    <span class="pos-data-row-main">
      <slot name="main">
        <strong v-if="label">{{ label }}</strong>
        <small v-if="meta">{{ meta }}</small>
      </slot>
    </span>
    <span v-if="$slots.side || value !== undefined || sideMeta" class="pos-data-row-side">
      <slot name="side">
        <strong v-if="value !== undefined">{{ value }}</strong>
        <small v-if="sideMeta">{{ sideMeta }}</small>
      </slot>
    </span>
    <slot />
  </component>
</template>

<script setup lang="ts">
import { computed } from 'vue';

import { posToneClasses, type PosTone } from './uiTypes';

const props = withDefaults(defineProps<{
  as?: 'article' | 'button' | 'div';
  label?: string;
  meta?: string;
  value?: string | number;
  sideMeta?: string;
  layout?: 'default' | 'ledger';
  tone?: PosTone;
  selected?: boolean;
  interactive?: boolean;
}>(), {
  as: 'article',
  label: undefined,
  meta: undefined,
  value: undefined,
  sideMeta: undefined,
  layout: 'default',
  tone: 'neutral',
  selected: false,
  interactive: false,
});

const emit = defineEmits<{
  (event: 'click', value: Event): void;
}>();

const rowClasses = computed(() => [
  ...posToneClasses('pos-data-row', props.tone),
  'payment-row',
  props.layout === 'ledger' ? 'ledger-row' : '',
  props.as === 'button' || props.interactive ? 'interactive' : '',
  props.selected ? 'selected' : '',
].filter(Boolean));
</script>
