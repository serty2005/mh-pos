<template>
  <form class="cloud-panel cloud-form-panel" @submit.prevent="ctx.submitForm()">
    <div class="section-head">
      <h2>{{ t(ctx.mode.value === 'create' ? 'cloud.form.create' : 'cloud.form.edit') }}</h2>
      <q-btn v-if="ctx.mode.value === 'edit'" flat dense icon="add" :label="t('cloud.form.new')" @click="ctx.startCreate()" />
    </div>

    <template v-for="field in ctx.visibleFields.value" :key="field.key">
      <q-checkbox v-if="field.type === 'checkbox'" v-model="ctx.form[field.key]" :label="t(field.labelKey)" />
      <template v-else-if="field.type === 'permissionMatrix'" />
      <template v-else-if="field.options">
        <q-select
          :key="`${field.key}-${ctx.selectOptions(field).length}`"
          :model-value="ctx.form[field.key]"
          @update:model-value="(value) => ctx.setFormValue(field.key, value)"
          dense
          outlined
          clearable
          emit-value
          map-options
          :disable="ctx.isSelectDisabled(field)"
          :label="t(field.labelKey)"
          :options="ctx.selectOptions(field)"
        />
        <p v-if="ctx.isSelectDisabled(field)" class="cloud-field-hint">{{ t('cloud.form.selectDataFirst') }}</p>
      </template>
      <q-input
        v-else
        :model-value="ctx.inputModelValue(field.key)"
        @update:model-value="(value) => ctx.setFormValue(field.key, value)"
        dense
        outlined
        :type="field.type === 'textarea' ? 'textarea' : field.type === 'number' ? 'number' : 'text'"
        :rows="field.type === 'textarea' ? field.rows ?? 4 : undefined"
        :label="t(field.labelKey)"
      />
    </template>
    <role-permission-matrix v-for="field in ctx.permissionFields.value" :key="field.key" :ctx="ctx" :field="field" />

    <div class="cloud-form-actions">
      <q-btn
        color="primary"
        unelevated
        icon="save"
        type="submit"
        :loading="ctx.isLoading('submit')"
        :disable="!ctx.canSubmitActive.value"
        :label="t(ctx.mode.value === 'create' ? 'cloud.form.createAction' : 'actions.save')"
      />
      <q-btn
        v-if="ctx.mode.value === 'edit' && ctx.activeConfig.value?.archive"
        flat
        color="negative"
        icon="archive"
        :loading="ctx.isLoading('archive')"
        :label="t('cloud.actions.archive')"
        @click="ctx.archiveSelected()"
      />
    </div>

    <div v-if="ctx.activeKey.value === 'employees' && ctx.mode.value === 'edit'" class="cloud-extra-actions">
      <q-separator />
      <div class="inline-action horizontal">
        <q-btn flat icon="pause" :label="t('cloud.actions.suspend')" @click="ctx.employeeAction('suspend')" />
        <q-btn flat icon="play_arrow" :label="t('cloud.actions.activate')" @click="ctx.employeeAction('activate')" />
      </div>
      <q-btn flat color="negative" icon="archive" :label="t('cloud.actions.archive')" @click="ctx.employeeAction('archive')" />
      <q-btn flat icon="assignment_ind" :label="t('cloud.actions.assignRole')" @click="ctx.assignSelectedEmployeeRole()" />
      <div class="inline-action horizontal">
        <q-input v-model="ctx.actionPin.value" dense outlined type="password" :label="t('cloud.fields.pin')" />
        <q-btn flat icon="vpn_key" :label="t('cloud.actions.rotatePin')" @click="ctx.rotateSelectedEmployeePIN()" />
      </div>
    </div>
  </form>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n';

import RolePermissionMatrix from './RolePermissionMatrix.vue';

const { t } = useI18n();

defineProps<{
  ctx: Record<string, any>;
}>();
</script>
