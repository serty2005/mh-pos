<template>
  <q-dialog :model-value="terminal.refundDialog.value" persistent @update:model-value="terminal.refundDialog.value = $event">
    <q-card class="dialog-card">
      <q-card-section>
        <h2>{{ terminal.t(terminal.refundDialogTitleKey()) }}</h2>
      </q-card-section>
      <q-card-section class="form-stack">
        <p class="dialog-copy">{{ terminal.t(terminal.refundDialogCopyKey()) }}</p>
        <q-input v-model="terminal.refundReason.value" outlined :label="terminal.t('pos.refundReason')" type="textarea" autogrow />
        <template v-if="terminal.refundDialogShowsLedgerControls()">
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
            :options="[{ label: terminal.t('pos.operationKinds.full'), value: 'full' }]"
            :label="terminal.t('pos.operationKind')"
            disable
          />
          <div class="planned-scope-panel">
            <p class="eyebrow">{{ terminal.t('pos.partialLedgerScope') }}</p>
            <div class="planned-scope-list">
              <q-chip v-for="scope in terminal.plannedLedgerScopeOptions.value" :key="scope" dense outline color="grey-7">
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
          :disable="!terminal.refundReason.value.trim()"
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
.planned-scope-panel {
  display: grid;
  gap: 8px;
  padding: 12px;
  border: 1px solid rgba(36, 42, 54, 0.14);
  border-radius: 8px;
  background: rgba(36, 42, 54, 0.03);
}

.planned-scope-list {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
</style>
