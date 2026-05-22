<template>
  <div class="pos-section-head">
    <div>
      <p v-if="eyebrow" class="eyebrow">{{ eyebrow }}</p>
      <h1>{{ title }}</h1>
    </div>
    <slot name="actions">
      <PosButton
        v-if="refreshLabel"
        variant="neutral"
        mode="flat"
        :icon="refreshIcon"
        :aria-label="refreshLabel"
        round
        class="icon-touch"
        @click="emit('refresh')"
      />
    </slot>
  </div>
</template>

<script setup lang="ts">
import PosButton from './PosButton.vue';

withDefaults(defineProps<{
  title: string;
  eyebrow?: string;
  refreshLabel?: string;
  refreshIcon?: string;
}>(), {
  eyebrow: undefined,
  refreshLabel: undefined,
  refreshIcon: 'refresh',
});

const emit = defineEmits<{
  (event: 'refresh'): void;
}>();
</script>
