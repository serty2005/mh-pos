<template>
  <q-dialog :model-value="modelValue" @update:model-value="$emit('update:modelValue', $event)">
    <q-card class="dialog-card actions-dialog-card">
      <q-card-section>
        <p class="eyebrow">{{ terminal.t('pos.activeOrder') }}</p>
        <h2>{{ terminal.t('pos.actions') }}</h2>
      </q-card-section>

      <q-card-section class="form-stack">
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
            <div class="quantity-stepper compact-stepper" :aria-label="line.name">
              <q-btn flat round class="stepper-button" icon="remove" :aria-label="terminal.t('actions.remove')" :disable="!terminal.canChangeOrderLine.value || line.quantity <= 1" @click="terminal.changeQuantity(line.id, line.quantity - 1)" />
              <span>{{ line.quantity }}</span>
              <q-btn flat round class="stepper-button" icon="add" :aria-label="terminal.t('actions.add')" :disable="!terminal.canChangeOrderLine.value" @click="terminal.changeQuantity(line.id, line.quantity + 1)" />
            </div>
            <q-btn flat round color="negative" icon="delete" class="stepper-button" :aria-label="terminal.t('actions.voidLine')" :disable="!terminal.canVoidOrderLine.value" @click="terminal.voidLine(line.id)" />
          </article>
        </div>
        <div v-else class="empty-state">{{ terminal.t('pos.emptyOrder') }}</div>

        <div class="supported-action-list">
          <q-btn
            v-if="terminal.latestPrecheck.value"
            outline
            color="secondary"
            class="touch-button"
            icon="print"
            :label="terminal.t('actions.reprintPrecheck')"
            :disable="!terminal.canReprintPrecheck.value"
            :loading="terminal.reprintPrecheckMutation.isPending.value"
            @click="terminal.reprintPrecheckMutation.mutate(terminal.latestPrecheck.value.id)"
          />
          <q-btn
            v-if="terminal.finalCheckData.value"
            outline
            color="secondary"
            class="touch-button"
            icon="print"
            :label="terminal.t('actions.reprintCheck')"
            :disable="!terminal.canReprintCheck.value"
            :loading="terminal.reprintCheckMutation.isPending.value"
            @click="terminal.reprintCheckMutation.mutate(terminal.finalCheckData.value.id)"
          />
        </div>
      </q-card-section>

      <q-card-actions align="right" class="dialog-actions">
        <q-btn flat :label="terminal.t('actions.close')" @click="$emit('update:modelValue', false)" />
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
