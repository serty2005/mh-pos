<template>
  <nav :class="['pos-tab-list', `pos-tab-list--${variant}`]" :aria-label="accessibilityLabel">
    <button
      v-for="option in options"
      :key="option.value"
      :class="['pos-tab-button', { active: option.value === modelValue }]"
      type="button"
      :disabled="option.disabled"
      @click="emit('update:modelValue', option.value)"
    >
      {{ option.label }}
    </button>
  </nav>
</template>

<script setup lang="ts">
export interface PosTabOption {
  value: string;
  label: string;
  disabled?: boolean;
}

withDefaults(defineProps<{
  modelValue: string;
  options: PosTabOption[];
  accessibilityLabel: string;
  variant?: 'underline' | 'chip';
}>(), {
  variant: 'underline',
});

const emit = defineEmits<{
  (event: 'update:modelValue', value: string): void;
}>();
</script>
