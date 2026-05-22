<template>
  <div :class="['cash-form-row', 'pos-form-row', { inline }]">
    <slot />
    <div v-if="$slots.actions" class="pos-form-row-actions">
      <slot name="actions" />
    </div>
  </div>
</template>

<script setup lang="ts">
withDefaults(defineProps<{
  inline?: boolean;
}>(), {
  inline: false,
});
</script>
