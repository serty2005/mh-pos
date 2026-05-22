<template>
  <q-dialog :model-value="terminal.refundDialog.value" persistent @update:model-value="terminal.refundDialog.value = $event">
    <q-card class="dialog-card">
      <q-card-section>
        <h2>{{ terminal.t(terminal.refundDialogTitleKey()) }}</h2>
      </q-card-section>
      <q-card-section class="form-stack">
        <p class="dialog-copy">{{ terminal.t(terminal.refundDialogCopyKey()) }}</p>
        <q-banner v-if="terminal.refundMode.value === 'payment_refund'" class="compatibility-banner" rounded>
          {{ terminal.t('pos.paymentRefundFallbackCopy') }}
        </q-banner>
        <q-input v-model="terminal.refundReason.value" outlined :label="terminal.t('pos.refundReason')" type="textarea" autogrow />
        <template v-if="terminal.refundDialogShowsLedgerControls()">
          <q-option-group
            v-model="terminal.refundScope.value"
            class="scope-toggle"
            color="primary"
            inline
            :options="terminal.ledgerScopeOptions.value"
          />
          <template v-if="terminal.refundScope.value === 'order_line'">
            <q-select
              v-model="terminal.refundOrderLineId.value"
              outlined
              emit-value
              map-options
              :options="terminal.refundLineOptions.value"
              :label="terminal.t('pos.ledgerLine')"
            />
            <q-input
              v-model.number="terminal.refundLineQuantity.value"
              outlined
              type="number"
              min="1"
              :max="terminal.maxRefundLineQuantity.value"
              :label="terminal.t('pos.quantity')"
              @blur="terminal.normalizeRefundLineQuantity"
            />
            <div class="ledger-line-summary">
              <span>{{ terminal.t('common.amount') }}</span>
              <strong>{{ terminal.money(terminal.refundLineAmount.value, terminal.selectedRefundLine.value?.currency_code ?? 'RUB') }}</strong>
            </div>
            <div v-if="terminal.refundLineTaxAmount.value > 0" class="ledger-line-summary">
              <span>{{ terminal.t('common.taxAmount') }}</span>
              <strong>{{ terminal.money(terminal.refundLineTaxAmount.value, terminal.selectedRefundLine.value?.currency_code ?? 'RUB') }}</strong>
            </div>
          </template>
          <q-select
            v-model="terminal.refundInventoryDisposition.value"
            outlined
            emit-value
            map-options
            :options="terminal.inventoryDispositionOptions.value"
            :label="terminal.t('pos.inventoryDisposition')"
          />
          <q-select
            v-model="terminal.refundOperationKind.value"
            outlined
            emit-value
            map-options
            :options="[{ label: terminal.t(`pos.operationKinds.${terminal.currentLedgerOperationKind.value}`), value: terminal.currentLedgerOperationKind.value }]"
            :label="terminal.t('pos.operationKind')"
            disable
          />
          <div class="unsupported-scope-panel">
            <p class="eyebrow">{{ terminal.t('pos.unsupportedLedgerScopes') }}</p>
            <div class="unsupported-scope-list">
              <q-chip v-for="scope in terminal.unsupportedLedgerScopeOptions.value" :key="scope" dense outline color="grey-7">
                {{ scope }}
              </q-chip>
            </div>
          </div>
        </template>
      </q-card-section>
      <q-card-actions align="right">
        <q-btn flat :label="terminal.t('actions.cancel')" @click="terminal.closeRefundDialog" />
        <q-btn
          color="negative"
          unelevated
          :icon="terminal.refundDialogIcon()"
          :label="terminal.t(terminal.refundDialogSubmitKey())"
          :loading="terminal.refundMutation.isPending.value"
          :disable="!terminal.refundReason.value.trim() || (terminal.refundScope.value === 'order_line' && !terminal.selectedRefundLine.value)"
          @click="terminal.refundMutation.mutate()"
        />
      </q-card-actions>
    </q-card>
  </q-dialog>
</template>

<script setup lang="ts">
import type { CashierTerminal } from './useCashierTerminal';

defineProps<{
  terminal: CashierTerminal;
}>();
</script>

<style scoped>
.scope-toggle {
  padding: 2px 0;
}

.ledger-line-summary {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  min-height: 40px;
  padding: 8px 12px;
  border: 1px solid rgba(36, 42, 54, 0.14);
  border-radius: 8px;
}

.ledger-line-summary span {
  color: #6d7280;
}

.compatibility-banner {
  background: rgba(143, 99, 0, 0.08);
  color: #5f4300;
}

.unsupported-scope-panel {
  display: grid;
  gap: 8px;
  padding: 12px;
  border: 1px solid rgba(36, 42, 54, 0.14);
  border-radius: 8px;
  background: rgba(36, 42, 54, 0.03);
}

.unsupported-scope-list {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
</style>
