<template>
  <q-dialog :model-value="modelValue" :persistent="persistent" @update:model-value="emit('update:modelValue', $event)">
    <q-card :class="['dialog-card', 'pos-square-dialog', cardClass]">
      <q-card-section v-if="eyebrow || title || $slots.header || $slots['header-side']" class="dialog-head">
        <div>
          <p v-if="eyebrow" class="eyebrow">{{ eyebrow }}</p>
          <h2 v-if="title">{{ title }}</h2>
          <slot name="header" />
        </div>
        <slot name="header-side" />
      </q-card-section>
      <q-card-section v-if="$slots.default" :class="bodyClass">
        <slot />
      </q-card-section>
      <q-card-actions v-if="$slots.actions" align="right" class="dialog-actions">
        <slot name="actions" />
      </q-card-actions>
    </q-card>
  </q-dialog>
</template>

<script setup lang="ts">
withDefaults(defineProps<{
  modelValue: boolean;
  title?: string;
  eyebrow?: string;
  persistent?: boolean;
  cardClass?: string;
  bodyClass?: string;
}>(), {
  title: undefined,
  eyebrow: undefined,
  persistent: false,
  cardClass: '',
  bodyClass: '',
});

const emit = defineEmits<{
  (event: 'update:modelValue', value: boolean): void;
}>();
</script>
