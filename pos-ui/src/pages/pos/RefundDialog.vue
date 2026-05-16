<template>
  <q-dialog :model-value="terminal.refundDialog.value" persistent @update:model-value="terminal.refundDialog.value = $event">
    <q-card class="dialog-card">
      <q-card-section>
        <h2>{{ terminal.t(terminal.refundDialogTitleKey()) }}</h2>
      </q-card-section>
      <q-card-section class="form-stack">
        <p class="dialog-copy">{{ terminal.t(terminal.refundDialogCopyKey()) }}</p>
        <q-input v-model="terminal.refundReason.value" outlined :label="terminal.t('pos.refundReason')" type="textarea" autogrow />
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
