<template>
  <q-dialog :model-value="modelValue" @update:model-value="$emit('update:modelValue', $event)">
    <q-card class="dialog-card payment-dialog-card">
      <q-card-section>
        <p class="eyebrow">{{ terminal.t('pos.cashier') }}</p>
        <h2>{{ terminal.t('pos.payment') }}</h2>
      </q-card-section>

      <q-card-section class="form-stack">
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

        <q-banner v-if="terminal.finalCheckData.value" class="success-banner" rounded>
          {{ terminal.t('pos.paymentCompleteCheckClosed') }}
        </q-banner>

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

        <div v-else-if="!terminal.finalCheckData.value" class="empty-state">
          {{ terminal.t('pos.noActivePrecheck') }}
        </div>
      </q-card-section>

      <q-card-actions align="right" class="dialog-actions">
        <q-btn flat :label="terminal.t('actions.close')" @click="$emit('update:modelValue', false)" />
        <q-btn
          v-if="terminal.finalCheckData.value"
          outline
          color="secondary"
          icon="print"
          :label="terminal.t('actions.reprintCheck')"
          :disable="!terminal.canReprintCheck.value"
          :loading="terminal.reprintCheckMutation.isPending.value"
          @click="terminal.reprintCheckMutation.mutate(terminal.finalCheckData.value.id)"
        />
        <template v-else>
          <q-btn
            color="primary"
            unelevated
            icon="payments"
            :label="terminal.t('actions.payCash')"
            :disable="!terminal.canPayCash.value"
            :loading="terminal.paymentMutation.isPending.value"
            @click="terminal.pay('cash')"
          />
          <q-btn
            color="secondary"
            unelevated
            icon="credit_card"
            :label="terminal.t('actions.payCard')"
            :disable="!terminal.canPayCard.value"
            :loading="terminal.paymentMutation.isPending.value"
            @click="terminal.pay('card')"
          />
        </template>
      </q-card-actions>
    </q-card>
  </q-dialog>
</template>

<script setup lang="ts">
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
