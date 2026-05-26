<template>
  <PosDialog
    :model-value="terminal.cancelDialog.value"
    persistent
    :title="terminal.t('pos.cancelPrecheck')"
    body-class="form-stack pos-scrollarea-y pos-scrollbar-thin"
    @update:model-value="terminal.cancelDialog.value = $event"
  >
        <p class="dialog-copy">{{ terminal.t('pos.precheckCancelCopy') }}</p>
        <q-input v-model="terminal.managerPin.value" outlined :label="terminal.t('pos.managerPin')" type="password" inputmode="numeric" autocomplete="new-password" />
        <q-input v-model="terminal.cancelReason.value" outlined :label="terminal.t('pos.precheckCancelReason')" type="textarea" autogrow />
      <template #actions>
        <PosButton variant="neutral" mode="flat" :label="terminal.t('actions.cancel')" @click="terminal.closeCancelDialog" />
        <PosButton
          variant="danger"
          icon="undo"
          :label="terminal.t('pos.cancelPrecheck')"
          :loading="terminal.cancelPrecheckMutation.isPending.value"
          @click="terminal.submitCancelPrecheck"
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
