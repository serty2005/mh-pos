<template>
  <div class="empty-state" :class="size">
    <slot>{{ label }}</slot>
  </div>
</template>

<script setup lang="ts">
withDefaults(defineProps<{
  label?: string;
  size?: 'small' | 'wide' | '';
}>(), {
  label: '',
  size: '',
});
</script>
