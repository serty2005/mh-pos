<template>
  <q-dialog :model-value="terminal.refundDialog.value" persistent @update:model-value="terminal.refundDialog.value = $event">
    <q-card class="dialog-card">
      <q-card-section>
        <h2>{{ terminal.t('pos.refund') }}</h2>
      </q-card-section>
      <q-card-section class="form-stack">
        <p class="dialog-copy">{{ terminal.t('pos.refundCopy') }}</p>
        <q-input v-model="terminal.refundReason.value" outlined :label="terminal.t('pos.refundReason')" type="textarea" autogrow />
      </q-card-section>
      <q-card-actions align="right">
        <q-btn flat :label="terminal.t('actions.cancel')" @click="terminal.refundDialog.value = false" />
        <q-btn
          color="negative"
          unelevated
          icon="undo"
          :label="terminal.t('pos.refund')"
          :loading="terminal.refundMutation.isPending.value"
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
