<template>
  <section class="permission-matrix">
    <div class="section-head stacked">
      <p class="eyebrow">{{ t(field.labelKey) }}</p>
      <h2>{{ t('cloud.permissions.matrixTitle') }}</h2>
    </div>
    <div class="permission-group" v-for="group in ctx.permissionGroups" :key="group.key">
      <strong>{{ t(group.labelKey) }}</strong>
      <div class="permission-grid">
        <q-checkbox
          v-for="permission in group.permissions"
          :key="permission"
          :model-value="ctx.selectedPermissions.value.includes(permission)"
          dense
          :label="ctx.permissionLabel(permission)"
          @update:model-value="(checked) => ctx.togglePermission(permission, Boolean(checked))"
        />
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n';

const { t } = useI18n();

defineProps<{
  ctx: Record<string, any>;
  field: Record<string, any>;
}>();
</script>
