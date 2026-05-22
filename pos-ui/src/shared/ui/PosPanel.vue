<template>
  <section class="integrated-panel pos-panel">
    <div v-if="eyebrow || title || $slots.header || $slots['header-side']" :class="['section-head', { slim }]">
      <div>
        <p v-if="eyebrow" class="eyebrow">{{ eyebrow }}</p>
        <h2 v-if="title">{{ title }}</h2>
        <slot name="header" />
      </div>
      <slot name="header-side" />
    </div>
    <slot />
    <div v-if="$slots.footer" class="pos-panel-footer">
      <slot name="footer" />
    </div>
  </section>
</template>

<script setup lang="ts">
withDefaults(defineProps<{
  eyebrow?: string;
  title?: string;
  slim?: boolean;
}>(), {
  eyebrow: undefined,
  title: undefined,
  slim: true,
});
</script>
