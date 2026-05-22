<template>
  <PosDialog
    :model-value="modelValue"
    card-class="actions-dialog-card"
    body-class="form-stack pos-scrollarea-y pos-scrollbar-thin"
    :eyebrow="terminal.t('pos.activeOrder')"
    :title="terminal.t('pos.actions')"
    @update:model-value="$emit('update:modelValue', $event)"
  >
        <blocking-notice
          v-if="terminal.activePrecheck.value || terminal.activeOrder.value?.status === 'locked'"
          :terminal="terminal"
          :title="terminal.t('pos.blocking.lockedOrder.title')"
          :reason="terminal.t('pos.blocking.lockedOrder.reason')"
          :permission="terminal.canCancelPrecheck.value ? '' : 'pos.precheck.cancel.request'"
          icon="lock"
        />

        <div v-if="terminal.activeLines.value.length" class="action-line-list">
          <article v-for="line in terminal.activeLines.value" :key="line.id" class="action-line">
            <div>
              <strong>{{ line.name }}</strong>
              <span>{{ terminal.money(line.total_price, terminal.orderCurrency.value) }}</span>
            </div>
            <PosQuantityStepper
              compact
              :value="line.quantity"
              :label="line.name"
              :decrement-label="terminal.t('actions.remove')"
              :increment-label="terminal.t('actions.add')"
              :decrement-disabled="!terminal.canChangeOrderLine.value || line.quantity <= 1"
              :increment-disabled="!terminal.canChangeOrderLine.value"
              @decrement="terminal.changeQuantity(line.id, line.quantity - 1)"
              @increment="terminal.changeQuantity(line.id, line.quantity + 1)"
            />
            <PosButton variant="neutral" mode="flat" round dense compact icon="tune" class="stepper-button" :aria-label="terminal.t('actions.editModifiers')" :disabled="!terminal.canChangeOrderLine.value || !terminal.canEditLineModifiers(line.id)" @click="terminal.editLineModifiers(line.id)" />
            <PosButton variant="danger" mode="flat" round dense compact icon="delete" class="stepper-button" :aria-label="terminal.t('actions.voidLine')" :disabled="!terminal.canVoidOrderLine.value" @click="terminal.voidLine(line.id)" />
          </article>
        </div>
        <PosEmptyState v-else :label="terminal.t('pos.emptyOrder')" />

        <div class="supported-action-list">
          <PosButton
            v-if="terminal.latestPrecheck.value"
            variant="secondary"
            mode="outline"
            icon="print"
            :label="terminal.t('actions.reprintPrecheck')"
            :disabled="!terminal.canReprintPrecheck.value"
            :loading="terminal.reprintPrecheckMutation.isPending.value"
            @click="terminal.reprintPrecheckMutation.mutate(terminal.latestPrecheck.value.id)"
          />
          <PosButton
            v-if="terminal.finalCheckData.value"
            variant="secondary"
            mode="outline"
            icon="print"
            :label="terminal.t('actions.reprintCheck')"
            :disabled="!terminal.canReprintCheck.value"
            :loading="terminal.reprintCheckMutation.isPending.value"
            @click="terminal.reprintCheckMutation.mutate(terminal.finalCheckData.value.id)"
          />
        </div>

      <template #actions>
        <PosButton variant="neutral" mode="flat" :label="terminal.t('actions.close')" @click="$emit('update:modelValue', false)" />
      </template>
  </PosDialog>
</template>

<script setup lang="ts">
import { PosButton, PosDialog, PosEmptyState, PosQuantityStepper } from '../../shared/ui';
import BlockingNotice from './BlockingNotice.vue';
import type { CashierTerminal } from './useCashierTerminal';

defineProps<{
  terminal: CashierTerminal;
  modelValue: boolean;
}>();

defineEmits<{
  (event: 'update:modelValue', value: boolean): void;
}>();
</script>
