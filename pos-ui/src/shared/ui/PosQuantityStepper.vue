<template>
  <div v-if="compact" class="quantity-stepper compact-stepper" :aria-label="label">
    <q-btn
      flat
      round
      class="stepper-button"
      icon="remove"
      :aria-label="decrementLabel"
      :disable="decrementIsDisabled"
      @click="emit('decrement')"
    />
    <span>{{ displayValue }}</span>
    <q-btn
      flat
      round
      class="stepper-button"
      icon="add"
      :aria-label="incrementLabel"
      :disable="incrementIsDisabled"
      @click="emit('increment')"
    />
  </div>
  <div v-else class="quantity-control" :class="{ 'quantity-control-with-edit': showEdit }" :aria-label="label">
    <q-btn
      flat
      square
      icon="remove"
      class="quantity-button"
      :aria-label="decrementLabel"
      :disable="decrementIsDisabled"
      @click="emit('decrement')"
    />
    <button class="quantity-value" type="button" :disabled="valueDisabled" @click="emit('edit-value')">
      {{ displayValue }}
    </button>
    <q-btn
      flat
      square
      icon="add"
      class="quantity-button"
      :aria-label="incrementLabel"
      :disable="incrementIsDisabled"
      @click="emit('increment')"
    />
    <q-btn
      v-if="showEdit"
      flat
      square
      icon="tune"
      class="quantity-button"
      :aria-label="editLabel"
      :disable="editIsDisabled"
      @click="emit('edit')"
    />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';

const props = withDefaults(defineProps<{
  value: number | string;
  label: string;
  decrementLabel: string;
  incrementLabel: string;
  valueLabel?: string;
  min?: number;
  disabled?: boolean;
  decrementDisabled?: boolean;
  incrementDisabled?: boolean;
  editable?: boolean;
  showEdit?: boolean;
  editLabel?: string;
  editDisabled?: boolean;
  compact?: boolean;
}>(), {
  valueLabel: undefined,
  min: undefined,
  disabled: false,
  decrementDisabled: false,
  incrementDisabled: false,
  editable: false,
  showEdit: false,
  editLabel: undefined,
  editDisabled: false,
  compact: false,
});

const emit = defineEmits<{
  (event: 'decrement'): void;
  (event: 'increment'): void;
  (event: 'edit'): void;
  (event: 'edit-value'): void;
}>();

const displayValue = computed(() => props.valueLabel ?? String(props.value));
const minBlocksDecrement = computed(() => typeof props.value === 'number' && props.min !== undefined && props.value <= props.min);
const decrementIsDisabled = computed(() => props.disabled || props.decrementDisabled || minBlocksDecrement.value);
const incrementIsDisabled = computed(() => props.disabled || props.incrementDisabled);
const editIsDisabled = computed(() => props.disabled || props.editDisabled);
const valueDisabled = computed(() => props.disabled || !props.editable);
</script>
