<template>
  <q-dialog :model-value="terminal.cancelDialog.value" persistent @update:model-value="terminal.cancelDialog.value = $event">
    <q-card class="dialog-card">
      <q-card-section>
        <h2>{{ terminal.t('pos.cancelPrecheck') }}</h2>
      </q-card-section>
      <q-card-section class="form-stack">
        <p class="dialog-copy">{{ terminal.t('pos.precheckCancelCopy') }}</p>
        <q-input v-model="terminal.managerEmployeeId.value" outlined :label="terminal.t('pos.managerEmployeeId')" autocomplete="off" />
        <q-input v-model="terminal.managerPin.value" outlined :label="terminal.t('pos.managerPin')" type="password" inputmode="numeric" autocomplete="new-password" />
        <q-input v-model="terminal.cancelReason.value" outlined :label="terminal.t('pos.precheckCancelReason')" type="textarea" autogrow />
      </q-card-section>
      <q-card-actions align="right">
        <q-btn flat :label="terminal.t('actions.cancel')" @click="terminal.closeCancelDialog" />
        <q-btn
          color="negative"
          unelevated
          icon="undo"
          :label="terminal.t('pos.cancelPrecheck')"
          :loading="terminal.cancelPrecheckMutation.isPending.value"
          @click="terminal.submitCancelPrecheck"
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
