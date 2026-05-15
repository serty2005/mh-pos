<template>
  <aside class="pos-order-rail" :aria-label="terminal.t('pos.currentOrderRail')">
    <div class="rail-head">
      <div>
        <p class="eyebrow">{{ terminal.selectedTable.value?.name ? `${terminal.t('pos.table')} ${terminal.selectedTable.value.name}` : terminal.t('pos.chooseTable') }}</p>
        <h2>{{ terminal.t('pos.activeOrder') }}</h2>
      </div>
      <q-btn flat round icon="table_restaurant" class="icon-touch" :aria-label="terminal.t('pos.sections.floor')" @click="$emit('open-floor')" />
    </div>

    <q-banner v-if="terminal.orderError.value" class="error-banner dense-banner" rounded>{{ terminal.orderError.value }}</q-banner>
    <q-skeleton v-if="terminal.orderLoading.value" class="order-skeleton rail-skeleton" />

    <template v-else-if="terminal.activeOrder.value">
      <div class="rail-summary">
        <div>
          <span>{{ terminal.t('pos.order') }}</span>
          <strong>{{ terminal.shortId(terminal.activeOrder.value.id) }}</strong>
        </div>
        <div>
          <span>{{ terminal.t('common.status') }}</span>
          <strong>{{ terminal.statusLabel(terminal.activeOrder.value.status) }}</strong>
        </div>
        <div>
          <span>{{ terminal.t('pos.precheck') }}</span>
          <strong>{{ terminal.activePrecheck.value ? terminal.t('pos.precheckIssued') : terminal.t('pos.noPrecheck') }}</strong>
        </div>
      </div>

      <q-banner v-if="terminal.finalCheckData.value" class="success-banner" rounded>
        {{ terminal.t('pos.paymentCompleteCheckClosed') }}
      </q-banner>

      <blocking-notice
        v-if="terminal.activeOrder.value.status === 'locked' || terminal.activePrecheck.value"
        :terminal="terminal"
        :title="terminal.t('pos.blocking.lockedOrder.title')"
        :reason="terminal.t('pos.blocking.lockedOrder.reason')"
        :permission="terminal.canCancelPrecheck.value ? '' : 'pos.precheck.cancel.request'"
        icon="lock"
      />

      <div class="rail-lines">
        <div v-if="terminal.activeLines.value.length" class="rail-line-list">
          <article v-for="line in terminal.activeLines.value" :key="line.id" class="rail-line">
            <div class="rail-line-title">
              <strong>{{ line.name }}</strong>
              <span>{{ terminal.money(line.unit_price, terminal.orderCurrency.value) }}</span>
              <ul v-if="line.modifiers.length" class="line-modifiers">
                <li v-for="modifier in line.modifiers" :key="modifier.id">
                  <span>{{ modifier.name }} x {{ modifier.quantity }}</span>
                  <strong>{{ terminal.money(modifier.total_price, terminal.orderCurrency.value) }}</strong>
                </li>
              </ul>
            </div>
            <div class="rail-line-controls">
              <div class="quantity-stepper compact-stepper" :aria-label="line.name">
                <q-btn flat round class="stepper-button" icon="remove" :aria-label="terminal.t('actions.remove')" :disable="!terminal.canChangeOrderLine.value || line.quantity <= 1" @click="terminal.changeQuantity(line.id, line.quantity - 1)" />
                <span>{{ line.quantity }}</span>
                <q-btn flat round class="stepper-button" icon="add" :aria-label="terminal.t('actions.add')" :disable="!terminal.canChangeOrderLine.value" @click="terminal.changeQuantity(line.id, line.quantity + 1)" />
              </div>
              <strong>{{ terminal.money(line.total_price, terminal.orderCurrency.value) }}</strong>
              <q-btn flat round class="stepper-button" icon="delete" color="negative" :disable="!terminal.canVoidOrderLine.value" :aria-label="terminal.t('actions.voidLine')" @click="terminal.voidLine(line.id)" />
            </div>
          </article>
        </div>
        <div v-else class="empty-state">{{ terminal.t('pos.emptyOrder') }}</div>
      </div>

      <div class="rail-total">
        <span>{{ terminal.t('pos.total') }}</span>
        <strong>{{ terminal.money(terminal.activeOrder.value.total, terminal.orderCurrency.value) }}</strong>
      </div>

      <div class="rail-actions">
        <template v-if="terminal.activePrecheck.value">
          <q-btn color="primary" unelevated class="touch-button primary-action" icon="point_of_sale" :label="terminal.t('pos.cashier')" :disable="terminal.remainingPayment.value <= 0" @click="$emit('open-payment')" />
          <q-btn outline color="negative" class="touch-button" icon="undo" :label="terminal.t('pos.cancelPrecheck')" :disable="terminal.activePrecheck.value.paid_total > 0 || !terminal.canCancelPrecheck.value" @click="terminal.cancelDialog.value = true" />
        </template>
        <template v-else>
          <q-btn outline color="secondary" class="touch-button" icon="more_horiz" :label="terminal.t('pos.actions')" :disable="!terminal.activeLines.value.length" @click="$emit('open-actions')" />
          <q-btn color="primary" unelevated class="touch-button primary-action" icon="request_quote" :label="terminal.t('actions.issuePrecheck')" :disable="!terminal.canIssuePrecheck.value" :loading="terminal.issuePrecheckMutation.isPending.value" @click="terminal.issuePrecheckMutation.mutate(terminal.activeOrder.value.id)" />
        </template>
      </div>
    </template>

    <div v-else class="rail-empty">
      <p>{{ terminal.selectedTableId.value ? terminal.t('pos.noActiveOrder') : terminal.t('pos.chooseTable') }}</p>
      <q-btn color="primary" unelevated class="touch-button primary-action" icon="receipt_long" :label="terminal.t('actions.createOrder')" :disable="!terminal.canCreateOrder.value" :loading="terminal.createOrderMutation.isPending.value" @click="terminal.createOrderMutation.mutate()" />
      <blocking-notice
        v-if="terminal.selectedTableId.value && terminal.actionBlocker('pos.order.create', terminal.canCreateOrder.value)"
        :terminal="terminal"
        :title="terminal.t(terminal.actionBlocker('pos.order.create', terminal.canCreateOrder.value)?.titleKey ?? '')"
        :reason="terminal.t(terminal.actionBlocker('pos.order.create', terminal.canCreateOrder.value)?.reasonKey ?? '')"
        :permission="terminal.actionBlocker('pos.order.create', terminal.canCreateOrder.value)?.permission"
        icon="lock"
      />
    </div>
  </aside>
</template>

<script setup lang="ts">
import BlockingNotice from './BlockingNotice.vue';
import type { CashierTerminal } from './useCashierTerminal';

defineProps<{
  terminal: CashierTerminal;
}>();

defineEmits<{
  (event: 'open-actions'): void;
  (event: 'open-payment'): void;
  (event: 'open-floor'): void;
}>();
</script>
