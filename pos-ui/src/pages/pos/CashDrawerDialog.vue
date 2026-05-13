<template>
  <q-dialog :model-value="terminal.cashDrawerDialog.value" persistent @update:model-value="terminal.cashDrawerDialog.value = $event">
    <q-card class="dialog-card">
      <q-card-section>
        <p class="eyebrow">{{ terminal.t('pos.cashSession') }}</p>
        <h2>{{ terminal.t('pos.cashDrawer') }}</h2>
      </q-card-section>
      <q-card-section class="form-stack">
        <q-select
          v-model="terminal.cashDrawerType.value"
          outlined
          emit-value
          map-options
          :options="terminal.cashDrawerTypeOptions.value"
          :label="terminal.t('pos.cashDrawerEvent')"
        />
        <q-input
          v-model.number="terminal.cashDrawerAmount.value"
          outlined
          type="number"
          min="0"
          :step="terminal.currencyInputStep(terminal.currency.value)"
          :label="terminal.t('common.amount')"
          :suffix="terminal.currency.value"
          :disable="terminal.cashDrawerType.value === 'no_sale'"
        />
        <q-input v-model="terminal.cashDrawerReason.value" outlined :label="terminal.t('pos.cashDrawerReason')" />
        <q-input v-model="terminal.cashDrawerNote.value" outlined :label="terminal.t('pos.cashDrawerNote')" />
      </q-card-section>
      <q-card-actions align="right">
        <q-btn flat :label="terminal.t('actions.cancel')" @click="terminal.cashDrawerDialog.value = false" />
        <q-btn
          color="secondary"
          unelevated
          icon="inventory_2"
          :label="terminal.t('actions.recordCashDrawerEvent')"
          :disable="!terminal.canSubmitCashDrawerEvent.value"
          :loading="terminal.cashDrawerMutation.isPending.value"
          @click="terminal.cashDrawerMutation.mutate()"
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
