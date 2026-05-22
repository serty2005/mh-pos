<template>
  <q-btn
    no-caps
    :color="posButtonColor(variant)"
    :disable="disabled"
    :loading="loading"
    :icon="icon"
    :label="label"
    :square="square && !round"
    :round="round"
    :dense="dense"
    :class="buttonClasses"
    v-bind="modeProps"
    @click="emit('click', $event)"
  />
</template>

<script setup lang="ts">
import { computed } from 'vue';

import { posButtonClasses, posButtonColor, posButtonModeProps, type PosButtonMode, type PosButtonVariant } from './uiTypes';

const props = withDefaults(defineProps<{
  variant?: PosButtonVariant;
  mode?: PosButtonMode;
  icon?: string;
  label?: string;
  disabled?: boolean;
  loading?: boolean;
  primary?: boolean;
  square?: boolean;
  round?: boolean;
  dense?: boolean;
  compact?: boolean;
}>(), {
  variant: 'secondary',
  mode: 'filled',
  icon: undefined,
  label: undefined,
  disabled: false,
  loading: false,
  primary: false,
  square: true,
  round: false,
  dense: false,
  compact: false,
});

const emit = defineEmits<{
  (event: 'click', value: Event): void;
}>();

const modeProps = computed(() => posButtonModeProps(props.mode));
const buttonClasses = computed(() => posButtonClasses({ primary: props.primary, compact: props.compact }));
</script>
