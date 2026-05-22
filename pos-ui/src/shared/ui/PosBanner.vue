<template>
  <q-banner :class="bannerClasses" rounded>
    <slot>{{ label }}</slot>
  </q-banner>
</template>

<script setup lang="ts">
import { computed } from 'vue';

const props = withDefaults(defineProps<{
  label?: string;
  tone?: 'error' | 'success' | 'info' | 'warning';
  dense?: boolean;
}>(), {
  label: '',
  tone: 'info',
  dense: true,
});

const bannerClasses = computed(() => [
  props.tone === 'error' ? 'error-banner' : '',
  props.tone === 'success' ? 'success-banner' : '',
  props.tone === 'info' ? 'info-banner' : '',
  props.tone === 'warning' ? 'warning-banner' : '',
  props.dense ? 'dense-banner' : '',
].filter(Boolean));
</script>
