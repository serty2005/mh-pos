<template>
  <section class="cloud-workspace">
    <resource-list :ctx="ctx" />
    <resource-form :ctx="ctx" />
  </section>
</template>

<script setup lang="ts">
import ResourceForm from './ResourceForm.vue';
import ResourceList from './ResourceList.vue';

defineProps<{
  ctx: Record<string, any>;
}>();
</script>
