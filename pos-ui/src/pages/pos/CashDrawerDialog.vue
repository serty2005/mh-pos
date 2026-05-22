<template>
  <PosDialog
    :model-value="terminal.cashDrawerDialog.value"
    persistent
    :eyebrow="terminal.t('pos.cashSession')"
    :title="terminal.t('pos.cashDrawer')"
    body-class="form-stack pos-scrollarea-y pos-scrollbar-thin"
    @update:model-value="terminal.cashDrawerDialog.value = $event"
  >
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
      <template #actions>
        <PosButton variant="neutral" mode="flat" :label="terminal.t('actions.cancel')" @click="terminal.cashDrawerDialog.value = false" />
        <PosButton
          variant="secondary"
          icon="inventory_2"
          :label="terminal.t('actions.recordCashDrawerEvent')"
          :disabled="!terminal.canSubmitCashDrawerEvent.value"
          :loading="terminal.cashDrawerMutation.isPending.value"
          @click="terminal.cashDrawerMutation.mutate()"
        />
      </template>
  </PosDialog>
</template>

<script setup lang="ts">
import { PosButton, PosDialog } from '../../shared/ui';
import type { CashierTerminal } from './useCashierTerminal';

defineProps<{
  terminal: CashierTerminal;
}>();
</script>
