<template>
  <q-skeleton :type="type" :width="width" :height="height" :class="skeletonClasses" />
</template>

<script setup lang="ts">
import { computed } from 'vue';

const props = withDefaults(defineProps<{
  kind?: 'row' | 'list-row' | 'card' | 'rail' | 'order';
  type?: 'QAvatar' | 'QBadge' | 'QBtn' | 'QCheckbox' | 'QChip' | 'QInput' | 'QRadio' | 'QRange' | 'QSlider' | 'QToggle' | 'QToolbar' | 'circle' | 'rect' | 'text';
  width?: string;
  height?: string;
}>(), {
  kind: 'row',
  type: undefined,
  width: undefined,
  height: undefined,
});

const skeletonClasses = computed(() => [
  'pos-skeleton',
  props.kind === 'list-row' ? 'skeleton-row' : '',
  props.kind === 'card' ? 'dish-card dish-card-skeleton' : '',
  props.kind === 'rail' ? 'order-skeleton rail-skeleton' : '',
  props.kind === 'order' ? 'order-skeleton' : '',
].filter(Boolean));
</script>
