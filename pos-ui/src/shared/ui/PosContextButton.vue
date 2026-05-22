<template>
  <span
    v-if="passive"
    class="context-button passive-context-button"
    :class="{ 'selected-item-button': selected, 'backlog-context-button': backlog }"
    :title="title"
    :aria-label="ariaLabel"
  >
    <q-icon v-if="icon" :name="icon" size="20px" />
    <span>{{ label }}</span>
  </span>
  <button
    v-else
    class="context-button"
    :class="{ 'selected-item-button': selected, 'backlog-context-button': backlog }"
    type="button"
    :disabled="disabled"
    :title="title"
    :aria-label="ariaLabel"
    @click="emit('click', $event)"
  >
    <q-icon v-if="icon" :name="icon" size="20px" />
    <span>{{ label }}</span>
  </button>
</template>

<script setup lang="ts">
withDefaults(defineProps<{
  label: string;
  icon?: string;
  disabled?: boolean;
  passive?: boolean;
  selected?: boolean;
  backlog?: boolean;
  title?: string;
  ariaLabel?: string;
}>(), {
  icon: undefined,
  disabled: false,
  passive: false,
  selected: false,
  backlog: false,
  title: undefined,
  ariaLabel: undefined,
});

const emit = defineEmits<{
  (event: 'click', value: MouseEvent): void;
}>();
</script>
