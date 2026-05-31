<template>
  <div class="cloud-panel-stack">
    <cloud-safe-error-banner :ctx="ctx" target="financialOperations" />
    <div class="cloud-panel cloud-table-panel">
      <div class="section-head">
        <div>
          <p class="eyebrow">{{ t('cloud.reporting.financial.status') }}</p>
          <h2>{{ t('cloud.reporting.financial.title') }}</h2>
          <p>{{ t('cloud.reporting.financial.description') }}</p>
        </div>
        <q-btn flat icon="refresh" :loading="ctx.isLoading('financial-operations')" :label="t('actions.refresh')" @click="ctx.loadFinancialOperations()" />
      </div>

      <div class="financial-operations-filter-grid">
        <q-input v-model="ctx.financialOperationFilters.businessDateFrom" dense outlined :label="t('cloud.reporting.financial.filters.businessDateFrom')" mask="####-##-##" />
        <q-input v-model="ctx.financialOperationFilters.businessDateTo" dense outlined :label="t('cloud.reporting.financial.filters.businessDateTo')" mask="####-##-##" />
        <q-select
          v-model="ctx.financialOperationFilters.operationType"
          dense
          outlined
          emit-value
          map-options
          :label="t('cloud.reporting.financial.filters.operationType')"
          :options="operationTypeOptions"
        />
        <q-input v-model="ctx.financialOperationFilters.shiftId" dense outlined :label="t('cloud.fields.shift_id')" />
        <q-input v-model="ctx.financialOperationFilters.originalShiftId" dense outlined :label="t('cloud.fields.original_shift_id')" />
        <q-input v-model="ctx.financialOperationFilters.checkId" dense outlined :label="t('cloud.fields.check_id')" />
        <q-btn color="primary" unelevated icon="search" :label="t('cloud.reporting.financial.applyFilters')" :loading="ctx.isLoading('financial-operations')" @click="ctx.loadFinancialOperations()" />
      </div>

      <div class="cloud-signal-row">
        <span>{{ t('cloud.reporting.financial.signals.readOnly') }}</span>
        <span>{{ t('cloud.reporting.financial.signals.noRawPayload') }}</span>
        <span>{{ t('cloud.reporting.financial.signals.noCashierCommands') }}</span>
      </div>

      <div v-if="ctx.financialOperations.value.length === 0" class="empty-state wide">
        {{ t('cloud.reporting.financial.empty') }}
      </div>
      <template v-else>
        <div class="cloud-table-wrap">
          <table class="cloud-table">
            <thead>
              <tr>
                <th>{{ t('cloud.fields.business_date_local') }}</th>
                <th>{{ t('cloud.fields.operation_type') }}</th>
                <th>{{ t('cloud.fields.amount') }}</th>
                <th>{{ t('cloud.fields.check_id') }}</th>
                <th>{{ t('cloud.fields.shift_id') }}</th>
                <th>{{ t('cloud.fields.reason') }}</th>
                <th>{{ t('cloud.fields.inventory_disposition') }}</th>
                <th>{{ t('cloud.reporting.financial.payloadHash') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="operation in ctx.financialOperations.value" :key="operation.operation_id">
                <td>{{ operation.business_date_local }}</td>
                <td><span class="cloud-status published">{{ operationTypeLabel(operation.operation_type) }}</span></td>
                <td>{{ formatMinorAmount(operation.amount, operation.currency) }}</td>
                <td>{{ ctx.formatCell('check_id', operation.check_id) }}</td>
                <td>{{ ctx.formatCell('shift_id', operation.shift_id) }}</td>
                <td>{{ ctx.formatCell('reason', operation.reason) }}</td>
                <td>{{ ctx.formatCell('inventory_disposition', operation.inventory_disposition) }}</td>
                <td>{{ ctx.formatCell('raw_payload_sha256_hex', operation.raw_payload_sha256_hex) }}</td>
              </tr>
            </tbody>
          </table>
        </div>

        <div class="edge-event-card-list" :aria-label="t('cloud.reporting.financial.title')">
          <article v-for="operation in ctx.financialOperations.value" :key="operation.operation_id" class="edge-event-card">
            <span class="cloud-status published">{{ operationTypeLabel(operation.operation_type) }}</span>
            <strong>{{ formatMinorAmount(operation.amount, operation.currency) }}</strong>
            <small>{{ t('cloud.fields.business_date_local') }}: {{ operation.business_date_local }}</small>
            <small>{{ t('cloud.fields.check_id') }}: {{ ctx.formatCell('check_id', operation.check_id) }}</small>
            <small>{{ t('cloud.fields.shift_id') }}: {{ ctx.formatCell('shift_id', operation.shift_id) }}</small>
            <small>{{ t('cloud.fields.reason') }}: {{ ctx.formatCell('reason', operation.reason) }}</small>
            <small>{{ t('cloud.reporting.financial.payloadHash') }}: {{ ctx.formatCell('raw_payload_sha256_hex', operation.raw_payload_sha256_hex) }}</small>
          </article>
        </div>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';

import CloudSafeErrorBanner from './CloudSafeErrorBanner.vue';

const { t } = useI18n();

defineProps<{
  ctx: Record<string, any>;
}>();

const operationTypeOptions = computed(() => [
  { label: t('cloud.reporting.financial.operationTypes.all'), value: '' },
  { label: t('cloud.reporting.financial.operationTypes.refund'), value: 'refund' },
  { label: t('cloud.reporting.financial.operationTypes.cancellation'), value: 'cancellation' },
]);

function operationTypeLabel(value: string) {
  if (value === 'refund') return t('cloud.reporting.financial.operationTypes.refund');
  if (value === 'cancellation') return t('cloud.reporting.financial.operationTypes.cancellation');
  return value;
}

function formatMinorAmount(amount: number, currency: string) {
  return `${(amount / 100).toFixed(2)} ${currency}`;
}
</script>
