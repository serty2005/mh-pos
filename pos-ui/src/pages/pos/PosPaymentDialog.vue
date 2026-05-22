<template>
  <PosDialog
    :model-value="modelValue"
    card-class="payment-dialog-card"
    body-class="form-stack pos-scrollarea-y pos-scrollbar-thin"
    :eyebrow="terminal.t('pos.cashier')"
    :title="terminal.t('pos.payment')"
    @update:model-value="$emit('update:modelValue', $event)"
  >
        <div class="payment-metrics">
          <div>
            <span>{{ terminal.t('pos.amountDue') }}</span>
            <strong>{{ terminal.money(terminal.activePrecheck.value?.total ?? 0, terminal.orderCurrency.value) }}</strong>
          </div>
          <div>
            <span>{{ terminal.t('pos.remainingTotal') }}</span>
            <strong>{{ terminal.money(terminal.remainingPayment.value, terminal.orderCurrency.value) }}</strong>
          </div>
        </div>

        <div class="payment-readiness">
          <span :class="{ good: terminal.currentShift.data.value }">{{ terminal.t('pos.shift') }}: {{ terminal.currentShift.data.value ? terminal.t('status.open') : terminal.t('pos.noShift') }}</span>
          <span :class="{ good: terminal.currentCashSession.data.value }">{{ terminal.t('pos.cashSession') }}: {{ terminal.currentCashSession.data.value ? terminal.t('status.open') : terminal.t('pos.noCashSession') }}</span>
        </div>

        <PosBanner v-if="terminal.finalCheckData.value" tone="success" :label="terminal.t('pos.paymentCompleteCheckClosed')" />

        <template v-if="terminal.activePrecheck.value && !terminal.finalCheckData.value">
          <q-input
            v-model.number="terminal.paymentAmount.value"
            outlined
            type="number"
            min="0"
            :step="terminal.currencyInputStep(terminal.orderCurrency.value)"
            :label="terminal.t('pos.paymentAmount')"
            :suffix="terminal.orderCurrency.value"
          />
          <p v-if="terminal.paymentBlockedReasonKey.value" class="payment-hint">
            {{ terminal.t(terminal.paymentBlockedReasonKey.value) }}
          </p>
          <blocking-notice
            v-if="terminal.actionBlocker('pos.payment.cash', terminal.canPayCash.value) && !terminal.canPayCard.value"
            :terminal="terminal"
            :title="terminal.t(terminal.actionBlocker('pos.payment.cash', terminal.canPayCash.value)?.titleKey ?? '')"
            :reason="terminal.t(terminal.actionBlocker('pos.payment.cash', terminal.canPayCash.value)?.reasonKey ?? '')"
            :permission="terminal.actionBlocker('pos.payment.cash', terminal.canPayCash.value)?.permission"
            icon="payments"
          />
        </template>

        <PosEmptyState v-else-if="!terminal.finalCheckData.value" :label="terminal.t('pos.noActivePrecheck')" />

      <template #actions>
        <PosButton variant="neutral" mode="flat" :label="terminal.t('actions.close')" @click="$emit('update:modelValue', false)" />
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
        <template v-else>
          <PosButton
            variant="primary"
            icon="payments"
            :label="terminal.t('actions.payCash')"
            :disabled="!terminal.canPayCash.value"
            :loading="terminal.paymentMutation.isPending.value"
            @click="terminal.pay('cash')"
          />
          <PosButton
            variant="secondary"
            icon="credit_card"
            :label="terminal.t('actions.payCard')"
            :disabled="!terminal.canPayCard.value"
            :loading="terminal.paymentMutation.isPending.value"
            @click="terminal.pay('card')"
          />
        </template>
      </template>
  </PosDialog>
</template>

<script setup lang="ts">
import { PosBanner, PosButton, PosDialog, PosEmptyState } from '../../shared/ui';
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
