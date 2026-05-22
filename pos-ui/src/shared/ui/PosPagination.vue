<template>
  <div class="pagination-controls">
    <q-btn
      flat
      round
      icon="chevron_left"
      class="icon-touch"
      :aria-label="previousLabel"
      :disable="previousDisabled"
      @click="emit('previous')"
    />
    <span>{{ value }}</span>
    <q-btn
      flat
      round
      icon="chevron_right"
      class="icon-touch"
      :aria-label="nextLabel"
      :disable="nextDisabled"
      @click="emit('next')"
    />
  </div>
</template>

<script setup lang="ts">
defineProps<{
  value: string | number;
  previousLabel: string;
  nextLabel: string;
  previousDisabled: boolean;
  nextDisabled: boolean;
}>();

const emit = defineEmits<{
  (event: 'previous'): void;
  (event: 'next'): void;
}>();
</script>
