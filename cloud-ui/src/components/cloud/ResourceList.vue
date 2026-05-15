<template>
  <div class="cloud-panel cloud-list-panel">
    <cloud-safe-error-banner :ctx="ctx" :target="ctx.activeKey.value" />
    <div class="cloud-list-tools">
      <q-input v-model="ctx.search.value" dense outlined clearable debounce="120" :label="t('cloud.search')" />
      <span>{{ t('cloud.rows') }}: {{ ctx.filteredRows.value.length }}</span>
    </div>
    <div v-if="ctx.activeConfig.value?.commandOnly" class="empty-state wide">
      {{ t('cloud.empty.commandOnly') }}
    </div>
    <div v-else-if="ctx.activeLoading.value" class="cloud-skeleton-list">
      <q-skeleton v-for="index in 6" :key="index" class="skeleton-row" />
    </div>
    <div v-else-if="ctx.filteredRows.value.length === 0" class="empty-state wide">
      {{ t('common.empty') }}
    </div>
    <div v-else>
      <div class="cloud-table-wrap">
        <table class="cloud-table">
          <thead>
            <tr>
              <th v-for="column in ctx.activeColumns.value" :key="column.key">{{ t(column.labelKey) }}</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="row in ctx.filteredRows.value"
              :key="ctx.rowKey(row)"
              :class="{ selected: ctx.selectedRowId.value === ctx.rowKey(row) }"
              @click="ctx.selectRow(row)"
            >
              <td v-for="column in ctx.activeColumns.value" :key="column.key">
                <span v-if="column.key === 'status'" class="cloud-status" :class="String(row[column.key])">
                  {{ ctx.statusText(row[column.key]) }}
                </span>
                <span v-else>{{ ctx.formatCell(column.key, row[column.key]) }}</span>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <div class="resource-card-list">
        <button
          v-for="row in ctx.filteredRows.value"
          :key="ctx.rowKey(row)"
          type="button"
          class="resource-card"
          :class="{ selected: ctx.selectedRowId.value === ctx.rowKey(row) }"
          @click="ctx.selectRow(row)"
        >
          <strong>{{ ctx.formatCell('name', row.name ?? row.id) }}</strong>
          <span v-for="column in ctx.activeColumns.value.slice(0, 4)" :key="column.key">
            {{ t(column.labelKey) }}: {{ ctx.formatCell(column.key, row[column.key]) }}
          </span>
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n';

import CloudSafeErrorBanner from './CloudSafeErrorBanner.vue';

const { t } = useI18n();

defineProps<{
  ctx: Record<string, any>;
}>();
</script>
